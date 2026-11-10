//! Resolvedor de gameplay que lê a configuração viva **no momento da chamada**.
//!
//! O `_MSG_UseItem` de refino chega por `cpsock::dispatch` (`Opcode::UseItem`) →
//! `SelectionStore::use_item` → [`Gameplay::use_item`]. A **morte** de monstro chega por
//! `Opcode::Attack*` → [`Gameplay::loot_at_death`], que sorteia um roll por linha da tabela do
//! snapshot vigente no instante do abate. Caixa no chão e pickup ainda não têm handler: os
//! layouts (`_MSG_CreateItem` 0x026E, `_MSG_GetItem` 0x0270, `_MSG_CNFGetItem` 0x0171,
//! `_MSG_DecayItem` 0x016F) estão levantados em `docs/cliente/pacotes.md` §“Caixa de item”.
//! O módulo não guarda cópia de boot: cada
//! chamada pega o `Arc<GameConfig>` publicado por
//! [`LiveConfig`]. Assim, uma edição do portal aplicada pela thread `wyd-config` muda a próxima
//! tentativa **no mesmo processo**, sem reiniciar.
//!
//! - **Refino**: valida o pó e o nível com [`RefinePolicy::LEGACY`], lê a taxa de
//!   [`GameConfig::powder_rate_bps`] (SancRate + piedade) e usa o sorteio do `wyd-sim`.
//! - **Loot**: um sorteio por item da tabela do template (`docs/wyd/sistemas/drops.md` §1),
//!   com o multiplicador global de drop aplicado ao peso e teto na escala de 1..1.000.000.
//!
//! Nenhuma taxa é constante aqui e nenhum valor inválido vira zero: snapshot inválido não é
//! publicado, então a chamada continua usando o último snapshot bom.

use std::fmt;
use std::sync::Arc;

use wyd_db::ItemRow;
use wyd_protocol::UseItem;
use wyd_sim::formula::drop::{drops, effective_rate_with, DROP_ROLL_MAX};
use wyd_sim::formula::refine::{
    pity_gain, powder_next_level, powder_roll_succeeds, Powder, RefinePolicy, PITY_MAX, REFINE_MAX,
};
use wyd_sim::roll::XorShift64;
use wyd_sim::units::Bps;

use crate::config::{DropRule, GameConfig, LiveConfig};

/// Uma tentativa de refino com pó, já com os sorteios do chamador.
///
/// `roll` é `rand()%100` (0..=99); `pity_roll` é `rand()%4`, usado só quando falha.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct RefineRequest {
    /// `item_index` do item alvo.
    pub item_index: u32,
    /// Refino atual (`+0`..`+15`).
    pub level: u8,
    /// Pó usado (Ori, Lac ou Âmago).
    pub powder: Powder,
    /// Item selado em `+7..+9` pode ser destruído na falha.
    pub sealed: bool,
    /// Pontos de piedade acumulados antes desta tentativa.
    pub pity: u8,
    /// Sorteio de sucesso (`rand()%100`).
    pub roll: u8,
    /// Sorteio de piedade (`rand()%4`).
    pub pity_roll: u8,
}

/// O que a falha fez com o item.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RefineFailure {
    /// Item intocado; só o pó foi consumido.
    Kept,
    /// Item destruído (selado em `+7..+9`).
    Destroyed,
}

/// Resultado auditável de uma tentativa de refino.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct RefineResolution {
    pub success: bool,
    /// Taxa efetiva da célula + piedade, em basis points.
    pub rate_bps: u16,
    pub from_level: u8,
    pub to_level: u8,
    pub next_pity: u8,
    /// `None` no sucesso.
    pub failure: Option<RefineFailure>,
}

/// Motivo de recusa antes de qualquer mutação.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RefineError {
    /// Nível acima de `+15`.
    LevelOutOfRange { level: u8 },
    /// O pó não alcança esse nível/item (tetos de `RefineRate.txt`).
    NotAllowed {
        powder: Powder,
        level: u8,
        item_index: u32,
    },
    /// A célula de SancRate não existe no snapshot.
    MissingRate { powder: Powder, level: u8 },
    /// Sorteio fora de `0..=99`.
    RollOutOfRange { roll: u8 },
}

impl fmt::Display for RefineError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::LevelOutOfRange { level } => write!(f, "refino +{level} acima do máximo"),
            Self::NotAllowed {
                powder,
                level,
                item_index,
            } => write!(
                f,
                "pó {powder:?} não refina item {item_index} no nível +{level}"
            ),
            Self::MissingRate { powder, level } => {
                write!(f, "SancRate ausente para {powder:?} no nível +{level}")
            }
            Self::RollOutOfRange { roll } => write!(f, "sorteio de refino fora de 0..=99: {roll}"),
        }
    }
}

impl std::error::Error for RefineError {}

/// Item que caiu de um template.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct LootDrop {
    pub item_index: i32,
    pub qty: i16,
}

/// Sorteio de morte concluído: os rolls usados e os itens que passaram.
///
/// Guardar os rolls junto do resultado é o que torna o abate auditável: com a seed do `die`
/// qualquer pessoa reproduz a mesma decisão a partir da mesma tabela.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct LootResolution {
    pub template: String,
    /// `(item_index, roll)` na ordem do snapshot, roll em `1..=1.000.000`.
    pub rolls: Vec<(i32, u32)>,
    pub drops: Vec<LootDrop>,
}

/// Motivo de recusa do resolvedor de loot.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum LootError {
    /// O template não tem tabela no snapshot.
    NoTable { template: String },
    /// Falta o sorteio de um item da tabela.
    MissingRoll { item_index: i32 },
    /// Sorteio fora da escala `1..=1.000.000`.
    InvalidRoll { item_index: i32, roll: u32 },
    /// Peso não positivo (o snapshot já valida, mas a defesa fica aqui).
    InvalidWeight { item_index: i32, weight: i64 },
}

impl fmt::Display for LootError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::NoTable { template } => write!(f, "template sem tabela de drop: {template}"),
            Self::MissingRoll { item_index } => {
                write!(f, "sem sorteio de drop para o item {item_index}")
            }
            Self::InvalidRoll { item_index, roll } => {
                write!(
                    f,
                    "sorteio de drop fora de 1..=1000000 para {item_index}: {roll}"
                )
            }
            Self::InvalidWeight { item_index, weight } => {
                write!(f, "peso de drop inválido para {item_index}: {weight}")
            }
        }
    }
}

impl std::error::Error for LootError {}

/// Sorteios de uma tentativa de refino via `_MSG_UseItem`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct UseItemRolls {
    /// `rand() % 100` do legado, em `0..=99`.
    pub roll: u8,
    /// `rand() % 4` do legado, usado só na falha para a piedade.
    pub pity_roll: u8,
}

/// Resultado de um `_MSG_UseItem` de refino já aplicado ao inventário.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RefineUse {
    /// Slot persistido do pó consumido (`16 + Carry[i]`).
    pub source_slot: i64,
    /// Slot persistido do item alvo.
    pub target_slot: i64,
    pub powder: Powder,
    pub resolution: RefineResolution,
    /// Quantidade de pó que restou (0 = linha removida).
    pub powder_left: i64,
    /// `true` quando a falha destruiu o item alvo.
    pub destroyed: bool,
}

/// Recusa de `_MSG_UseItem` antes de qualquer mutação.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum UseItemError {
    /// O pó precisa estar na mochila (`ITEM_PLACE_CARRY`).
    SourceNotCarry { place: i32 },
    /// Alvo pode estar na mochila ou equipado; cofre (`ITEM_PLACE_CARGO`) não tem sessão.
    DestinationUnsupported { place: i32 },
    /// `SourPos`/`DestPos` fora da faixa do lugar declarado.
    SlotOutOfRange { place: i32, pos: i32 },
    /// Linha de origem ausente ou com quantidade zero.
    EmptySource { slot: i64 },
    /// Linha de destino ausente.
    EmptyTarget { slot: i64 },
    /// Origem e destino iguais.
    SameSlot { slot: i64 },
    /// O item de origem não é Pó de Ori/Lac nem Lactolerium 100.
    NotPowder { item_index: i64 },
    /// O alvo é pó, rosto ou item fora da faixa de refino.
    NotRefinable { item_index: i64, slot: i64 },
    /// O catálogo marca o item com `EF_NOSANC` (126): não é erro de protocolo, é veto
    /// de dado (FASE-03-16). Recusa sem mutação, como o legado faz na sanitização.
    BlockedByCatalog { item_index: i64 },
    /// Refino persistido fora de `0..=15`.
    InvalidTargetLevel { level: i64 },
    /// Regra de refino recusou (taxa ausente, teto do pó, sorteio inválido).
    Refine(RefineError),
}

impl fmt::Display for UseItemError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::SourceNotCarry { place } => {
                write!(f, "pó precisa vir da mochila, veio do lugar {place}")
            }
            Self::DestinationUnsupported { place } => {
                write!(f, "destino no lugar {place} não tem sessão neste processo")
            }
            Self::SlotOutOfRange { place, pos } => {
                write!(f, "posição {pos} fora da faixa do lugar {place}")
            }
            Self::EmptySource { slot } => write!(f, "slot de origem {slot} vazio"),
            Self::EmptyTarget { slot } => write!(f, "slot de destino {slot} vazio"),
            Self::SameSlot { slot } => write!(f, "origem e destino são o mesmo slot {slot}"),
            Self::NotPowder { item_index } => write!(f, "item {item_index} não é pó de refino"),
            Self::BlockedByCatalog { item_index } => {
                write!(f, "item {item_index} bloqueado pelo catálogo (EF_NOSANC)")
            }
            Self::NotRefinable { item_index, slot } => {
                write!(f, "item {item_index} no slot {slot} não é refinável")
            }
            Self::InvalidTargetLevel { level } => {
                write!(f, "refino persistido {level} fora de 0..=15")
            }
            Self::Refine(error) => write!(f, "{error}"),
        }
    }
}

impl std::error::Error for UseItemError {}

impl From<RefineError> for UseItemError {
    fn from(error: RefineError) -> Self {
        Self::Refine(error)
    }
}

/// `ITEM_PLACE_CARRY`/`ITEM_PLACE_EQUIP` (`Basedef.h:146-148`).
const ITEM_PLACE_EQUIP: i32 = 0;
const ITEM_PLACE_CARRY: i32 = 1;
const CARRY_FIRST: i32 = 16;
const CARRY_SLOTS: i32 = 64;
const EQUIP_SLOTS: i32 = 16;

/// Itens de pó (`EF_VOLATILE` 4/5) e o Lactolerium 100 que força a linha do Âmago.
const ITEM_POWDER_ORI: i64 = 412;
const ITEM_POWDER_LAC: i64 = 413;
const ITEM_POWDER_AMAGO: i64 = 4_141;

/// Faixas de item selado do legado (`_MSG_UseItem.cpp:224`): falha em +7..+9 destrói.
fn is_sealed(item_index: i64) -> bool {
    matches!(
        item_index,
        1_234..=1_237 | 1_369..=1_372 | 1_519..=1_522 | 1_669..=1_672 | 1_901..=1_910 | 1_714
    )
}

fn powder_for(item_index: i64) -> Option<Powder> {
    match item_index {
        ITEM_POWDER_ORI => Some(Powder::Ori),
        ITEM_POWDER_LAC => Some(Powder::Lac),
        ITEM_POWDER_AMAGO => Some(Powder::Amago),
        _ => None,
    }
}

/// Converte `(lugar, posição)` do fio no slot persistido.
///
/// O legado usa `ITEM_PLACE_EQUIP = 0`, `ITEM_PLACE_CARRY = 1`, `ITEM_PLACE_CARGO = 2`
/// (`Basedef.h:146-148`), e o banco usa `0..=15` para equipamento e `16..=79` para a
/// mochila (`character_items.slot`).
fn inventory_slot(place: i32, pos: i32) -> Option<i64> {
    match place {
        ITEM_PLACE_EQUIP if (0..EQUIP_SLOTS).contains(&pos) => Some(i64::from(pos)),
        ITEM_PLACE_CARRY if (0..CARRY_SLOTS).contains(&pos) => Some(i64::from(CARRY_FIRST + pos)),
        _ => None,
    }
}

/// Resolvedor compartilhado pelo processo: sempre lê o snapshot vigente.
#[derive(Debug, Clone)]
pub struct Gameplay {
    config: Arc<LiveConfig>,
}

impl Gameplay {
    #[must_use]
    pub fn new(config: Arc<LiveConfig>) -> Self {
        Self { config }
    }

    /// Revisão usada pela última leitura (observabilidade/teste).
    #[must_use]
    pub fn revision(&self) -> i64 {
        self.config.revision()
    }

    /// Resolve uma tentativa de refino com a taxa do snapshot atual.
    pub fn refine(&self, request: &RefineRequest) -> Result<RefineResolution, RefineError> {
        let config = self.config.snapshot();
        refine_with(&config, request)
    }

    /// Recompensa de EXP de um abate, com o multiplicador `exp` do snapshot vigente.
    ///
    /// É a **mesma** função que `SELECTION` usa em `gain_exp`: o valor provado no teste é o
    /// que entra no personagem. Base não positiva volta intacta (recusa acontece depois).
    #[must_use]
    pub fn exp_reward(&self, base: i64) -> i64 {
        if base <= 0 {
            return base;
        }
        let config = self.config.snapshot();
        let base = u64::try_from(base).unwrap_or(u64::MAX);
        i64::try_from(config.apply_multiplier("exp", base)).unwrap_or(i64::MAX)
    }

    /// Resolve uma tabela de drop do snapshot atual com sorteios já dados.
    ///
    /// É a forma pura do resolvedor (teste/golden): `rolls` precisa ter um sorteio por item
    /// da tabela; a ausência é erro explícito. `citizen` liga o bônus de cidadania do canal
    /// (FASE-03-10) — desligado, a taxa é exatamente a do legado.
    pub fn loot(
        &self,
        template: &str,
        rolls: &[(i32, u32)],
        citizen: bool,
    ) -> Result<Vec<LootDrop>, LootError> {
        let config = self.config.snapshot();
        loot_with(&config, template, rolls, citizen)
    }

    /// **Morte de monstro**: tabela do snapshot vigente + os rolls do momento.
    ///
    /// Um roll por item da tabela (`docs/wyd/sistemas/drops.md` §1 — roll por item, nunca
    /// ponderado), tirado do `die` do chamador para o resultado ser reproduzível pela seed.
    /// Lê o snapshot **uma vez** por abate: a tabela e o multiplicador de drop são os mesmos
    /// para todos os itens daquele kill, mesmo que o portal publique outra revisão no meio.
    pub fn loot_at_death(
        &self,
        template: &str,
        die: &mut XorShift64,
        citizen: bool,
    ) -> Result<LootResolution, LootError> {
        let config = self.config.snapshot();
        let table = config.drops_for(template);
        let rolls: Vec<(i32, u32)> = table
            .iter()
            .map(|(item_index, _)| {
                let roll =
                    1 + u32::try_from(die.next_u64() % u64::from(DROP_ROLL_MAX)).unwrap_or(0);
                (*item_index, roll)
            })
            .collect();
        let drops = loot_with(&config, template, &rolls, citizen)?;
        Ok(LootResolution {
            template: template.to_string(),
            rolls,
            drops,
        })
    }

    /// Resolve um `_MSG_UseItem` de refino e aplica o resultado ao inventário.
    ///
    /// A semântica de origem/destino vem do legado (`_MSG_UseItem.cpp:42-152`):
    /// `SourType/SourPos` apontam o pó, `DestType/DestPos` o item alvo. O pó é consumido
    /// sempre; sucesso sobe um nível, falha comum mantém o item e falha de selado em +7..+9
    /// destrói o alvo. A taxa é lida **nesta chamada** por [`Self::refine`], nunca no boot.
    ///
    /// `pity` é a piedade persistida do alvo (tabela `refine_pity`, FASE-03-16): quem chama
    /// lê antes e grava `resolution.next_pity` depois. A sessão passa a lista completa de
    /// linhas; o chamador persiste depois. Recusa não consome pó nem altera o alvo.
    pub fn use_item(
        &self,
        items: &mut Vec<ItemRow>,
        packet: &UseItem,
        rolls: UseItemRolls,
        pity: u8,
    ) -> Result<RefineUse, UseItemError> {
        if packet.sour_type != ITEM_PLACE_CARRY {
            return Err(UseItemError::SourceNotCarry {
                place: packet.sour_type,
            });
        }
        if packet.dest_type != ITEM_PLACE_EQUIP && packet.dest_type != ITEM_PLACE_CARRY {
            return Err(UseItemError::DestinationUnsupported {
                place: packet.dest_type,
            });
        }
        let source_slot = inventory_slot(ITEM_PLACE_CARRY, packet.sour_pos).ok_or(
            UseItemError::SlotOutOfRange {
                place: packet.sour_type,
                pos: packet.sour_pos,
            },
        )?;
        let target_slot = inventory_slot(packet.dest_type, packet.dest_pos).ok_or(
            UseItemError::SlotOutOfRange {
                place: packet.dest_type,
                pos: packet.dest_pos,
            },
        )?;
        if source_slot == target_slot {
            return Err(UseItemError::SameSlot { slot: source_slot });
        }
        // O rosto (`Equip[0]`) nunca é alvo de pó no legado (`_MSG_UseItem.cpp:363`).
        if packet.dest_type == ITEM_PLACE_EQUIP && packet.dest_pos == 0 {
            return Err(UseItemError::NotRefinable {
                item_index: 0,
                slot: target_slot,
            });
        }

        let source = items
            .iter()
            .find(|item| item.slot == source_slot)
            .copied()
            .filter(|item| item.qty > 0)
            .ok_or(UseItemError::EmptySource { slot: source_slot })?;
        let target = items
            .iter()
            .find(|item| item.slot == target_slot)
            .copied()
            .filter(|item| item.item_id != 0)
            .ok_or(UseItemError::EmptyTarget { slot: target_slot })?;
        let powder = powder_for(source.item_id).ok_or(UseItemError::NotPowder {
            item_index: source.item_id,
        })?;
        if powder_for(target.item_id).is_some() {
            return Err(UseItemError::NotRefinable {
                item_index: target.item_id,
                slot: target_slot,
            });
        }
        let level = u8::try_from(target.refine)
            .ok()
            .filter(|level| *level <= REFINE_MAX)
            .ok_or(UseItemError::InvalidTargetLevel {
                level: target.refine,
            })?;
        let item_index = u32::try_from(target.item_id).map_err(|_| UseItemError::NotRefinable {
            item_index: target.item_id,
            slot: target_slot,
        })?;

        let resolution = self.refine(&RefineRequest {
            item_index,
            level,
            powder,
            sealed: is_sealed(target.item_id),
            pity: pity.min(PITY_MAX),
            roll: rolls.roll,
            pity_roll: rolls.pity_roll,
        })?;

        if resolution.success {
            if let Some(target) = items.iter_mut().find(|item| item.slot == target_slot) {
                target.refine = i64::from(resolution.to_level);
            }
        } else if resolution.failure == Some(RefineFailure::Destroyed) {
            items.retain(|item| item.slot != target_slot);
        }

        let powder_left =
            if let Some(source) = items.iter_mut().find(|item| item.slot == source_slot) {
                source.qty = source.qty.saturating_sub(1);
                source.qty
            } else {
                0
            };
        if powder_left <= 0 {
            items.retain(|item| item.slot != source_slot);
        }

        Ok(RefineUse {
            source_slot,
            target_slot,
            powder,
            resolution,
            powder_left,
            destroyed: resolution.failure == Some(RefineFailure::Destroyed),
        })
    }
}

fn refine_with(
    config: &GameConfig,
    request: &RefineRequest,
) -> Result<RefineResolution, RefineError> {
    let policy = RefinePolicy::LEGACY;
    if request.level > REFINE_MAX {
        return Err(RefineError::LevelOutOfRange {
            level: request.level,
        });
    }
    if !policy.allows(request.powder, request.level, request.item_index) {
        return Err(RefineError::NotAllowed {
            powder: request.powder,
            level: request.level,
            item_index: request.item_index,
        });
    }
    if request.roll > 99 {
        return Err(RefineError::RollOutOfRange { roll: request.roll });
    }
    let rate_bps = config
        .powder_rate_bps(request.powder, request.level, request.pity)
        .ok_or(RefineError::MissingRate {
            powder: request.powder,
            level: request.level,
        })?;
    let success = powder_roll_succeeds(rate_bps, request.roll);
    let to_level = if success {
        powder_next_level(request.level).ok_or(RefineError::LevelOutOfRange {
            level: request.level,
        })?
    } else {
        request.level
    };
    let failure = if success {
        None
    } else if request.sealed && (7..=9).contains(&request.level) {
        Some(RefineFailure::Destroyed)
    } else {
        Some(RefineFailure::Kept)
    };
    let next_pity = if success {
        0
    } else if pity_gain(request.powder, request.level, request.pity_roll) {
        request.pity.saturating_add(1).min(PITY_MAX)
    } else {
        request.pity
    };
    Ok(RefineResolution {
        success,
        rate_bps,
        from_level: request.level,
        to_level,
        next_pity,
        failure,
    })
}

fn loot_with(
    config: &GameConfig,
    template: &str,
    rolls: &[(i32, u32)],
    citizen: bool,
) -> Result<Vec<LootDrop>, LootError> {
    let table = config.drops_for(template);
    if table.is_empty() {
        return Err(LootError::NoTable {
            template: template.to_string(),
        });
    }
    // Bônus do canal de cidadania: `0` quando desligado ou quando quem matou não é cidadão
    // deste canal. Lido do snapshot **uma vez** por abate, como a tabela.
    let citizen_bonus = if citizen {
        config.citizen_bonus_bps()
    } else {
        Bps::ZERO
    };
    let mut out = Vec::new();
    for (item_index, rule) in table {
        let roll = rolls
            .iter()
            .find(|(item, _)| *item == item_index)
            .map(|(_, roll)| *roll)
            .ok_or(LootError::MissingRoll { item_index })?;
        let rate = effective_weight(config, item_index, rule, citizen_bonus)?;
        if drops(rate, roll).map_err(|_| LootError::InvalidRoll { item_index, roll })? {
            out.push(LootDrop {
                item_index,
                qty: rule.min_qty,
            });
        }
    }
    Ok(out)
}

/// Peso efetivo: multiplicador global de drop + bônus de cidadania, com teto da escala.
fn effective_weight(
    config: &GameConfig,
    item_index: i32,
    rule: DropRule,
    citizen_bonus: Bps,
) -> Result<u32, LootError> {
    if rule.weight <= 0 {
        return Err(LootError::InvalidWeight {
            item_index,
            weight: rule.weight,
        });
    }
    let base = u64::try_from(rule.weight).unwrap_or(u64::MAX);
    let scaled = config
        .apply_multiplier("drop", base)
        .min(u64::from(DROP_ROLL_MAX));
    let escalado = u32::try_from(scaled).unwrap_or(DROP_ROLL_MAX);
    // Bônus aplicado **depois** do multiplicador global e antes do sorteio (hipótese D-08 de
    // `docs/wyd/sistemas/drops.md` §1 para bônus de canal), com teto na escala do roll.
    Ok(effective_rate_with(escalado, &[citizen_bonus]))
}
