//! Composição de item — Aylin (+10), Anct, Odin (11 receitas + ramo +12+) e o passo +10 → +11.
//!
//! Porte de `origin/main`: `tmserver/internal/combine/{match,odin,combine}.go`,
//! `tmserver/internal/handler/{combine,combine_variants,combine_odin,refine}.go` e
//! `_MSG_CombineItem*.cpp` / `_MSG_UseItem.cpp`.
//!
//! Regras de fronteira, iguais às dos outros sistemas:
//!
//! - **Servidor é autoridade**: a função recebe slots e um `raw_roll`; nunca recebe "passou".
//! - **Recusa sem mutação**: toda validação (receita, gates, insumos, moeda, fila de eventos)
//!   acontece antes da primeira escrita.
//! - **Número não mora no handler**: taxas, bônus, penalidades e custo vêm de
//!   [`ComposeRules`] (tabela `combine_value`, seed `0028_seed_combine.sql`). Aqui ficam só
//!   ids de item (estrutura da receita) e o passo `GEM_LEVEL`.
//! - **Dois sorteios distintos**: [`roll_115`] (`Roll` do legado, achata ≥100 em −15) e
//!   [`roll_199`] (`RollOdin`, achata >100 em −99). Trocar um pelo outro muda a economia.
//!
//! Fontes e divergências: `docs/wyd/sistemas/refino-auditoria.md` §1–§3.

use crate::ecs::components::{
    gem_value, ComponentStore, Entity, Inventory, Item, BAG_SLOTS, GEM_EFFECT_NONE,
};
use crate::ecs::resources::DomainEvent;
use crate::error::CoreError;

/// Pedra do Sábio, catalisador do +10 (`MatchAilyn`).
pub const PEDRA_DO_SABIO: u32 = 1_774;
/// Primeira jóia (Diamante); as quatro são 2441..=2444.
pub const JEWEL_BASE: u32 = 2_441;
/// Última jóia (Garnet).
pub const JEWEL_LAST: u32 = 2_444;
/// Item bloqueado em toda receita de combine (`blockedItem747`).
pub const BLOCKED_ITEM: u32 = 747;
/// Poeira de Lactolerium — pó do passo +10 → +11.
pub const LAC_POWDER: u32 = 413;
/// `Lactolerium_100`: mesma grade do Lac, linha do Âmago (100%).
pub const LACTO_100: u32 = 4_141;
/// Primeira pedra Secreta (Água); Terra/Sol/Vento são +1/+2/+3.
pub const RUNE_LO: u32 = 5_334;
/// Última pedra Secreta (Vento).
pub const RUNE_HI: u32 = 5_337;
/// Refinação Abençoada: pedra neutra da receita +12+.
pub const NEUTRAL_RUNE: u32 = 3_338;
/// Emblema da Proteção: insumo alternativo de proteção no +12+.
pub const PROTECTION_EMBLEM: u32 = 4_043;
/// Item 769: nunca passa de +9, por nome (`_MSG_UseItem.cpp:802`).
pub const ITEM_NYERDES: u32 = 769;
/// Slot de equipamento da capa (Capa Celestial).
pub const CAPE_SLOT: usize = 15;

/// `EF_POS`: nPos do item no catálogo (`BASE_GetItemAbility`).
pub const EF_POS: u16 = 17;
/// `EF_ITEMLEVEL`: nível de item usado nos gates de Anct/Ehre.
pub const EF_ITEMLEVEL: u16 = 87;
/// `EF_MOBTYPE`: 3 marca item de montaria (recusado no Anct).
pub const EF_MOBTYPE: u16 = 112;

/// Passo especial do pó: +10 → +11 (`GEM_LEVEL`).
pub const GEM_LEVEL: u8 = 10;
/// Último refino do jogo.
pub const REFINE_MAX: u8 = 15;
/// Refino do item resultante de um Anct (`anctResultSanc`).
pub const ANCT_RESULT_REFINE: u8 = 7;
/// Lista de slots de uma mensagem de combine (`MAX_COMBINE`).
pub const COMBINE_SLOTS: usize = 8;
/// Slots que a receita do Aylin consome (7 posições do legado).
pub const AYLIN_SLOTS: usize = 7;
/// Teto de sacrifícios do Anct no legado (`MAX_COMBINE` − alvo − jóia).
pub const ANCT_MAX_SACRIFICES: usize = COMBINE_SLOTS - 2;
/// Nível de personagem que destrava o Lv40 celestial.
pub const DESTRAVE_LEVEL: u16 = 39;

/// Item resultante da receita "Pista de Runas".
pub const ODIN_PISTA_RESULT: u32 = 5_134;
/// Item resultante da receita "Pedra da Fúria".
pub const ODIN_FURIA_RESULT: u32 = 3_020;
/// Base das quatro pedras Secretas produzidas (Água = base).
pub const ODIN_SECRETA_BASE: u32 = 5_334;
/// Item resultante da receita "Semente de Cristal".
pub const ODIN_SEMENTE_RESULT: u32 = 4_032;

/// Topo do `rand() % 115` (0..=114).
pub const ROLL_115_MAX: u8 = 114;
/// Topo do `rand() % 199` (0..=198).
pub const ROLL_199_MAX: u8 = 198;

/// Fatos do catálogo que a composição lê (`item_catalog`).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct ItemFacts {
    /// `nPos` (0 = sem slot de equip).
    pub pos: u8,
    /// Grade do item (0..=8).
    pub grade: u8,
    /// `nUnique` (41..=49 identifica arma candidata a Anct).
    pub unique_id: u8,
    /// `Extra`: id do item resultante de um Anct ou do Sets do Odin.
    pub extra: u32,
}

/// Dados de conteúdo que a composição consulta.
///
/// Quem implementa é a borda (catálogo do banco). Nos testes, a implementação lê o próprio
/// seed `0028_seed_combine.sql`, então uma taxa trocada no dado muda o resultado — não há
/// número de taxa escondido neste módulo.
pub trait ComposeRules {
    /// Fatos do item no catálogo.
    fn facts(&self, item_index: u32) -> Option<ItemFacts>;
    /// `BASE_GetItemAbility`: soma do catálogo + instância para o efeito pedido.
    fn ability(&self, item_index: u32, effect: u16) -> i32;
    /// `combine_value(family, key)`: taxa (percentual) ou custo (moedas).
    fn value(&self, family: &str, key: &str) -> Option<i32>;
}

/// Resultado de uma tentativa (Aylin/Anct/Odin/pó) para a borda e para o teste.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct ComposeOutcome {
    /// Família da receita (`Aylin`, `Anct`, `Odin`, `Powder`).
    pub family: &'static str,
    /// Id da receita (0 em pó e no Sets do Odin).
    pub recipe: u8,
    /// A tentativa passou?
    pub success: bool,
    /// Taxa efetiva acumulada, em pontos percentuais.
    pub rate_percent: i32,
    /// Valor sorteado (já achatado).
    pub roll: i32,
    /// Unidades retiradas da mochila (pilha inteira, como o legado).
    ///
    /// Conta também o item transformado (o alvo do `+12+` sai e volta regravado), por isso
    /// um passo com 2 pilhas de 10 + 4 runas reporta 25.
    pub consumed: u32,
    /// Moedas cobradas.
    pub coin_spent: u32,
    /// Emblemas da Proteção consumidos (ramo `+12+`).
    pub protect_items: u8,
    /// Item produzido, quando houve (a capa celestial é gravada no equipamento).
    pub produced: Option<Item>,
}

/// Receita da família Odin (`GetMatchCombineOdin`).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum OdinRecipe {
    /// Composição de Sets: fallback quando nenhuma receita numerada casa.
    Sets,
    /// Composição de armas/celestial (id 1).
    ItemCelestial,
    /// +11..+15 (id 2).
    Plus12,
    /// Pista de Runas (id 3).
    PistaDeRunas,
    /// Destrave Lv40 celestial (id 4).
    DestraveLv40,
    /// Pedra da Fúria (id 5).
    PedraDaFuria,
    /// Pedra Secreta da Água (id 6).
    SecretaAgua,
    /// Pedra Secreta da Terra (id 7).
    SecretaTerra,
    /// Pedra Secreta do Sol (id 8).
    SecretaSol,
    /// Pedra Secreta do Vento (id 9).
    SecretaVento,
    /// Semente de Cristal (id 10).
    SementeDeCristal,
    /// Capa Celestial (id 11).
    CapaCelestial,
}

impl OdinRecipe {
    /// Id de receita do legado (0 = Sets).
    #[must_use]
    pub const fn id(self) -> u8 {
        match self {
            Self::Sets => 0,
            Self::ItemCelestial => 1,
            Self::Plus12 => 2,
            Self::PistaDeRunas => 3,
            Self::DestraveLv40 => 4,
            Self::PedraDaFuria => 5,
            Self::SecretaAgua => 6,
            Self::SecretaTerra => 7,
            Self::SecretaSol => 8,
            Self::SecretaVento => 9,
            Self::SementeDeCristal => 10,
            Self::CapaCelestial => 11,
        }
    }

    /// Chave da taxa em `combine_value` (`family = 'Odin'`).
    #[must_use]
    pub const fn rate_key(self) -> &'static str {
        match self {
            Self::Sets => "SETS",
            Self::ItemCelestial => "CELESTIAL",
            Self::Plus12 => "PLUS12_BASE",
            Self::PistaDeRunas => "PISTA",
            Self::DestraveLv40 => "DESTRAVE_LV40",
            Self::PedraDaFuria => "PEDRA_DA_FURIA",
            Self::SecretaAgua => "SECRETA_AGUA",
            Self::SecretaTerra => "SECRETA_TERRA",
            Self::SecretaSol => "SECRETA_SOL",
            Self::SecretaVento => "SECRETA_VENTO",
            Self::SementeDeCristal => "SEMENTE_CRISTAL",
            Self::CapaCelestial => "CAPA_CELESTIAL",
        }
    }
}

/// Sorteio de combine do legado (`combine.Roll`): `rand()%115`, ≥100 vira −15.
///
/// O achatamento duplica a chance de 85..99 — distribuição de economia, não detalhe.
pub fn roll_115(raw: u8) -> Result<u8, CoreError> {
    if raw > ROLL_115_MAX {
        return Err(CoreError::RollOutOfRange {
            roll: u16::from(raw),
            max: u16::from(ROLL_115_MAX),
        });
    }
    let mut value = i32::from(raw);
    if value >= 100 {
        value -= 15;
    }
    Ok(value as u8)
}

/// Sorteio da família Odin (`combine.RollOdin`): `rand()%199`, >100 vira −99.
///
/// Deliberadamente **não** é [`roll_115`]: a dobra é outra e as duas não podem ser
/// confundidas (auditoria §4).
pub fn roll_199(raw: u8) -> Result<u8, CoreError> {
    if raw > ROLL_199_MAX {
        return Err(CoreError::RollOutOfRange {
            roll: u16::from(raw),
            max: u16::from(ROLL_199_MAX),
        });
    }
    let mut value = i32::from(raw);
    if value > 100 {
        value -= 99;
    }
    Ok(value as u8)
}

/// Código de gema da jóia do legado (2441..2444 → drop/perfuração/EXP/absorção).
#[must_use]
pub const fn jewel_effect(jewel: u32) -> u8 {
    match jewel {
        2_441 => crate::ecs::components::GEM_EFFECT_DROP,
        2_442 => crate::ecs::components::GEM_EFFECT_PIERCE,
        2_443 => crate::ecs::components::GEM_EFFECT_EXP,
        2_444 => crate::ecs::components::GEM_EFFECT_ABSORB,
        _ => GEM_EFFECT_NONE,
    }
}

/// O que acontece com o item quando um refino com pó falha (regra do legado).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PowderFailure {
    /// Item intocado; só o pó é gasto.
    Kept,
    /// Item destruído: **só** item selado em +7..+9.
    Destroyed,
}

/// Consequência de falha do caminho de pó para o item atual.
#[must_use]
pub fn powder_failure(sealed: bool, level: u8) -> PowderFailure {
    if sealed && (7..=9).contains(&level) {
        PowderFailure::Destroyed
    } else {
        PowderFailure::Kept
    }
}

/// Consequência de falha do +12..+15 (Emblema rebaixa; Lac pode destruir).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Plus12Failure {
    /// Pacote de Lac: insumos destruídos e chance de destruir o item.
    Destroyed,
    /// Emblema da Proteção: item cai um nível.
    Downgraded,
}

/// Falha do +12..+15 conforme a proteção usada.
///
/// O legado **não** tem ramo de falha alcançável no +12+ (o `if` de falha está comentado no
/// código vivo e o sucesso é incondicional). A regra fica implementada e testada, mas o
/// handler não a aciona enquanto a norma for o legado — decisão registrada na auditoria §3.
#[must_use]
pub const fn plus12_failure(protection: bool) -> Plus12Failure {
    if protection {
        Plus12Failure::Downgraded
    } else {
        Plus12Failure::Destroyed
    }
}

/// Refino após o rebaixamento do Emblema (+11 → +10, +14 → +13), nunca abaixo de +7.
#[must_use]
pub fn downgrade_refine(refine: u8) -> u8 {
    if refine > ANCT_RESULT_REFINE {
        refine - 1
    } else {
        refine
    }
}

/// Posições de equipamento que entram na composição (`MatchAilyn` / `MatchTiny`).
#[must_use]
pub const fn is_compose_pos(pos: u8) -> bool {
    matches!(pos, 2 | 4 | 8 | 16 | 32 | 64 | 128 | 192)
}

fn ensure_alive(store: &ComponentStore, entity: Entity) -> Result<(), CoreError> {
    if store.is_alive(entity) {
        Ok(())
    } else {
        Err(CoreError::EntityNotAlive {
            index: entity.index,
            generation: entity.generation,
        })
    }
}

fn ensure_distinct(slots: &[usize]) -> Result<(), CoreError> {
    for (i, &slot) in slots.iter().enumerate() {
        if slots[..i].contains(&slot) {
            return Err(CoreError::ComposeSlotRepeated { slot: slot as u16 });
        }
    }
    Ok(())
}

fn ensure_bag_slot(slot: usize) -> Result<(), CoreError> {
    if slot >= BAG_SLOTS {
        return Err(CoreError::SlotOutOfRange {
            slot: slot as u16,
            max: (BAG_SLOTS - 1) as u16,
        });
    }
    Ok(())
}

fn read_slots<const N: usize>(
    store: &ComponentStore,
    entity: Entity,
    slots: &[usize; N],
) -> Result<[Item; N], CoreError> {
    let inv = store.inventory(entity).ok_or(CoreError::EntityNotAlive {
        index: entity.index,
        generation: entity.generation,
    })?;
    let mut items = [Item::default(); N];
    for (i, &slot) in slots.iter().enumerate() {
        ensure_bag_slot(slot)?;
        items[i] = inv.bag[slot];
    }
    Ok(items)
}

fn read_slots_dyn(
    store: &ComponentStore,
    entity: Entity,
    slots: &[usize],
) -> Result<[Item; COMBINE_SLOTS], CoreError> {
    if slots.len() > COMBINE_SLOTS {
        return Err(CoreError::ComposeTooManySlots {
            count: slots.len(),
            max: COMBINE_SLOTS,
        });
    }
    let inv = store.inventory(entity).ok_or(CoreError::EntityNotAlive {
        index: entity.index,
        generation: entity.generation,
    })?;
    let mut items = [Item::default(); COMBINE_SLOTS];
    for (i, &slot) in slots.iter().enumerate() {
        ensure_bag_slot(slot)?;
        items[i] = inv.bag[slot];
    }
    Ok(items)
}

fn mismatch(family: &'static str) -> CoreError {
    CoreError::ComposeRecipeMismatch { family }
}

fn value_or_zero<R: ComposeRules>(rules: &R, family: &str, key: &str) -> i32 {
    rules.value(family, key).unwrap_or(0)
}

fn consume_slots(inv: &mut Inventory, slots: &[usize], keep: &[usize]) -> u32 {
    let mut consumed: u32 = 0;
    for &slot in slots {
        if keep.contains(&slot) {
            continue;
        }
        let item = inv.bag[slot];
        if item.id != 0 {
            consumed = consumed.saturating_add(item.qty.max(1));
            inv.bag[slot] = Item::default();
        }
    }
    consumed
}

fn publish(store: &mut ComponentStore, event: DomainEvent) -> Result<(), CoreError> {
    store.publish(event)
}

/// Aylin — composição que produz o primeiro `+10`.
///
/// Receita (`MatchAilyn`): dois itens **+9 iguais**, uma Pedra do Sábio, quatro **jóias
/// iguais** (todas as mesmas). Taxa `base + 4 × ChanceBase` (41% no legado), custo
/// `Ailyn/COST` (50.000.000) cobrado quando a receita é válida, **antes** do sorteio.
///
/// Falha: jóias, pedra e moeda perdidos; os dois +9 permanecem. Sucesso: o resultado fica no
/// primeiro slot, com `+10` e a gema da jóia; o segundo +9 é consumido.
///
/// Gate conservador: o `MatchAilyn` do legado não repete a checagem de `+9` (o cliente só
/// oferece +9), mas aceitá-la aqui impede produzir `+10` de um item `+0`. Item 769 é recusado
/// (teto de +9 por nome).
pub fn aylin<R: ComposeRules>(
    store: &mut ComponentStore,
    entity: Entity,
    slots: &[usize; AYLIN_SLOTS],
    rules: &R,
    raw_roll: u8,
) -> Result<ComposeOutcome, CoreError> {
    ensure_alive(store, entity)?;
    ensure_distinct(slots)?;
    let items = read_slots(store, entity, slots)?;

    let base = items[0];
    if base.id == 0 || items[1].id != base.id || base.refine < 9 || items[1].refine < 9 {
        return Err(mismatch("Aylin"));
    }
    if base.id == ITEM_NYERDES {
        return Err(CoreError::RefineCappedItem {
            item_index: base.id,
        });
    }
    let facts = rules.facts(base.id).ok_or(mismatch("Aylin"))?;
    if !is_compose_pos(facts.pos) {
        return Err(mismatch("Aylin"));
    }
    if items[2].id != PEDRA_DO_SABIO {
        return Err(mismatch("Aylin"));
    }
    let jewel = items[3].id;
    if !(JEWEL_BASE..=JEWEL_LAST).contains(&jewel) {
        return Err(mismatch("Aylin"));
    }
    if items[4..AYLIN_SLOTS].iter().any(|item| item.id != jewel) {
        return Err(mismatch("Aylin"));
    }
    if (5..=8).contains(&facts.grade) {
        let want = JEWEL_BASE + u32::from(facts.grade - 5);
        if jewel != want {
            return Err(mismatch("Aylin"));
        }
    }

    let cost = value_or_zero(rules, "Aylin", "COST").max(0) as u32;
    let available = store.coin(entity).unwrap_or(0);
    if available < cost {
        return Err(CoreError::NotEnoughCoin {
            needed: cost,
            available,
        });
    }

    let rate =
        value_or_zero(rules, "Aylin", "BASE") + 4 * value_or_zero(rules, "Aylin", "CHANCE_BASE");
    let roll = roll_115(raw_roll)?;
    let success = i32::from(roll) <= rate;

    let effect = jewel_effect(jewel);
    let produced = Item {
        id: base.id,
        qty: 1,
        refine: GEM_LEVEL,
        ancient: base.ancient,
        gem_effect: effect,
        gem_value: gem_value(effect, GEM_LEVEL).unwrap_or(0),
    };

    publish(
        store,
        DomainEvent::CombineFinished {
            entity,
            family: "Aylin",
            recipe: 0,
            success,
            rate_percent: rate,
            roll: i32::from(roll),
            produced_item: if success { produced.id } else { 0 },
        },
    )?;

    store.set_coin(entity, available - cost);
    let keep: &[usize] = if success { &[] } else { &slots[..2] };
    let inv = store.inventory_mut(entity).expect("entidade viva");
    let consumed = consume_slots(inv, slots, keep);
    if success {
        inv.bag[slots[0]] = produced;
    }

    Ok(ComposeOutcome {
        family: "Aylin",
        recipe: 0,
        success,
        rate_percent: rate,
        roll: i32::from(roll),
        consumed,
        coin_spent: cost,
        protect_items: 0,
        produced: success.then_some(produced),
    })
}

/// Compositor (Anct) — arma +9 vira arma Ancient +7 com o efeito da jóia.
///
/// Receita (`MatchAnct`): alvo com `nUnique` 41..=49, `Extra` > 0, `EF_MOBTYPE` ≠ 3 e `+9`;
/// uma jóia; 0..=6 sacrifícios em **+7/+8/+9** com `EF_POS` ≠ 0 e nível de item ≥ o do alvo.
/// Taxa `base + Σ bônus` (1% + 2/4/10 no legado).
///
/// O teto de sacrifícios é recusado por erro próprio (`ComposeSacrificeLimit`), não por
/// "receita inválida" — o jogador não passa dele por nenhum atalho. Sucesso e falha consomem
/// **todos** os insumos; falha não devolve nada.
pub fn anct<R: ComposeRules>(
    store: &mut ComponentStore,
    entity: Entity,
    slots: &[usize],
    rules: &R,
    raw_roll: u8,
) -> Result<ComposeOutcome, CoreError> {
    ensure_alive(store, entity)?;
    if slots.len() < 2 {
        return Err(mismatch("Anct"));
    }
    // Teto da régua antes de qualquer leitura: 8 posições − alvo − jóia = 6 sacrifícios.
    if slots.len() - 2 > ANCT_MAX_SACRIFICES {
        return Err(CoreError::ComposeSacrificeLimit {
            count: slots.len() - 2,
            max: ANCT_MAX_SACRIFICES,
        });
    }
    ensure_distinct(slots)?;
    let items = read_slots_dyn(store, entity, slots)?;

    let target = items[0];
    if target.id == 0 || target.refine != 9 || target.id == ITEM_NYERDES {
        return Err(if target.id == ITEM_NYERDES {
            CoreError::RefineCappedItem {
                item_index: target.id,
            }
        } else {
            mismatch("Anct")
        });
    }
    let facts = rules.facts(target.id).ok_or(mismatch("Anct"))?;
    if !(41..=49).contains(&facts.unique_id) || facts.extra == 0 {
        return Err(mismatch("Anct"));
    }
    if rules.ability(target.id, EF_MOBTYPE) == 3 {
        return Err(mismatch("Anct"));
    }
    let jewel = items[1].id;
    if !(JEWEL_BASE..=JEWEL_LAST).contains(&jewel) {
        return Err(mismatch("Anct"));
    }

    let sacrifices = &slots[2..];

    let target_level = rules.ability(target.id, EF_ITEMLEVEL);
    let mut rate = value_or_zero(rules, "Compositor", "BASE");
    for index in 0..sacrifices.len() {
        let sacrifice = items[index + 2];
        if sacrifice.id == 0 || rules.ability(sacrifice.id, EF_POS) == 0 {
            return Err(mismatch("Anct"));
        }
        if target_level > rules.ability(sacrifice.id, EF_ITEMLEVEL) {
            return Err(mismatch("Anct"));
        }
        let key = match sacrifice.refine {
            7 => "ITEM_+7",
            8 => "ITEM_+8",
            9 => "ITEM_+9",
            _ => return Err(mismatch("Anct")),
        };
        rate += value_or_zero(rules, "Compositor", key);
    }

    let roll = roll_115(raw_roll)?;
    let success = i32::from(roll) <= rate;
    let effect = jewel_effect(jewel);
    let produced = Item {
        id: facts.extra + (jewel - JEWEL_BASE),
        qty: 1,
        refine: ANCT_RESULT_REFINE,
        ancient: true,
        gem_effect: effect,
        gem_value: gem_value(effect, GEM_LEVEL).unwrap_or(0),
    };

    publish(
        store,
        DomainEvent::CombineFinished {
            entity,
            family: "Anct",
            recipe: 0,
            success,
            rate_percent: rate,
            roll: i32::from(roll),
            produced_item: if success { produced.id } else { 0 },
        },
    )?;

    let inv = store.inventory_mut(entity).expect("entidade viva");
    let consumed = consume_slots(inv, slots, &[]);
    if success {
        inv.bag[slots[0]] = produced;
    }

    Ok(ComposeOutcome {
        family: "Anct",
        recipe: 0,
        success,
        rate_percent: rate,
        roll: i32::from(roll),
        consumed,
        coin_spent: 0,
        protect_items: 0,
        produced: success.then_some(produced),
    })
}

fn in_plus12_window(refine: u8) -> bool {
    (GEM_LEVEL..REFINE_MAX).contains(&refine)
}

fn is_rune_or_neutral(item_index: u32) -> bool {
    (RUNE_LO..=RUNE_HI).contains(&item_index) || item_index == NEUTRAL_RUNE
}

/// As quatro pedras Secretas precisam estar todas presentes ou todas ausentes.
fn secreta_uneven(items: &[Item; COMBINE_SLOTS]) -> bool {
    let mut present = [false; 4];
    for item in items {
        if (RUNE_LO..=RUNE_HI).contains(&item.id) {
            present[(item.id - RUNE_LO) as usize] = true;
        }
    }
    let some = present.iter().any(|&p| p);
    let all = present.iter().all(|&p| p);
    some && !all
}

const PLUS12_REF_KEYS: [&str; 10] = [
    "PLUS12_REF_0",
    "PLUS12_REF_1",
    "PLUS12_REF_2",
    "PLUS12_REF_3",
    "PLUS12_REF_4",
    "PLUS12_REF_5",
    "PLUS12_REF_6",
    "PLUS12_REF_7",
    "PLUS12_REF_8",
    "PLUS12_REF_9",
];

const PLUS12_MINUS_KEYS: [&str; 4] = [
    "PLUS12_MINUS_12",
    "PLUS12_MINUS_13",
    "PLUS12_MINUS_14",
    "PLUS12_MINUS_15",
];

/// `GetMatchCombineOdin`: devolve a receita numerada, ou `None` (que o handler trata como
/// **Composição de Sets**, o fallback real do legado).
fn match_odin<R: ComposeRules>(items: &[Item; COMBINE_SLOTS], rules: &R) -> Option<OdinRecipe> {
    if items.iter().any(|item| item.id == BLOCKED_ITEM) {
        return None;
    }
    let idx = |i: usize| items[i].id;

    if idx(0) != 0
        && rules.facts(idx(1)).map(|f| f.extra) == Some(idx(0))
        && items[0].refine >= 9
        && items[1].refine == REFINE_MAX
        && (idx(2) == 542 || idx(2) == 772)
        && idx(3) == 5334
        && idx(4) == 5335
        && idx(5) == 5336
        && idx(6) == 5337
    {
        return Some(OdinRecipe::ItemCelestial);
    }

    let two_lac_packs =
        idx(0) == LAC_POWDER && items[0].qty >= 10 && idx(1) == LAC_POWDER && items[1].qty >= 10;
    let two_protections = idx(0) == PROTECTION_EMBLEM && idx(1) == PROTECTION_EMBLEM;
    if (two_lac_packs || two_protections)
        && in_plus12_window(items[2].refine)
        && is_rune_or_neutral(idx(3))
        && is_rune_or_neutral(idx(4))
        && is_rune_or_neutral(idx(5))
        && is_rune_or_neutral(idx(6))
        && rules.facts(idx(2)).is_some_and(|f| is_compose_pos(f.pos))
    {
        return Some(OdinRecipe::Plus12);
    }

    if (0..7).all(|i| idx(i) == LAC_POWDER) {
        return Some(OdinRecipe::PistaDeRunas);
    }

    const DESTRAVE: [u32; 7] = [4127, 4127, 5135, 5113, 5129, 5112, 5110];
    if (0..7).all(|i| idx(i) == DESTRAVE[i]) {
        return Some(OdinRecipe::DestraveLv40);
    }
    const FURIA: [u32; 7] = [5125, 5115, 5111, 5112, 5120, 5128, 5119];
    if (0..7).all(|i| idx(i) == FURIA[i]) {
        return Some(OdinRecipe::PedraDaFuria);
    }
    const AGUA: [u32; 7] = [5126, 5127, 5121, 5114, 5125, 5111, 5118];
    if (0..7).all(|i| idx(i) == AGUA[i]) {
        return Some(OdinRecipe::SecretaAgua);
    }
    const TERRA: [u32; 7] = [5131, 5113, 5115, 5116, 5125, 5112, 5114];
    if (0..7).all(|i| idx(i) == TERRA[i]) {
        return Some(OdinRecipe::SecretaTerra);
    }
    const SOL: [u32; 7] = [5110, 5124, 5117, 5129, 5114, 5125, 5128];
    if (0..7).all(|i| idx(i) == SOL[i]) {
        return Some(OdinRecipe::SecretaSol);
    }
    const VENTO: [u32; 7] = [5122, 5119, 5132, 5120, 5130, 5133, 5123];
    if (0..7).all(|i| idx(i) == VENTO[i]) {
        return Some(OdinRecipe::SecretaVento);
    }
    const SEMENTE: [u32; 7] = [421, 422, 423, 424, 425, 426, 427];
    if (0..7).all(|i| idx(i) == SEMENTE[i]) {
        return Some(OdinRecipe::SementeDeCristal);
    }
    const CAPA: [u32; 7] = [4127, 4127, 5135, 413, 413, 413, 413];
    if (0..7).all(|i| idx(i) == CAPA[i]) {
        return Some(OdinRecipe::CapaCelestial);
    }
    None
}

/// Composição Odin — 11 receitas numeradas + Sets e o ramo `+12+`.
///
/// `raw_roll` é o sorteio de **199** (não o de 115) e `jitter` é o `rand()%5` que a
/// composição de Sets/armas soma à taxa; os dois vêm do RNG da borda para o resultado ser
/// determinístico no teste.
///
/// O ramo `+12+` tem **sucesso incondicional** no código vivo do legado (o `if` de falha está
/// comentado na origem): a taxa é calculada e devolvida para auditoria, mas não barra o
/// passo. Todos os insumos são consumidos de qualquer forma.
pub fn odin<R: ComposeRules>(
    store: &mut ComponentStore,
    entity: Entity,
    slots: &[usize; COMBINE_SLOTS],
    rules: &R,
    raw_roll: u8,
    jitter: u8,
) -> Result<ComposeOutcome, CoreError> {
    ensure_alive(store, entity)?;
    ensure_distinct(slots)?;
    let items = read_slots(store, entity, slots)?;

    let recipe = match_odin(&items, rules).unwrap_or(OdinRecipe::Sets);

    if matches!(recipe, OdinRecipe::Sets | OdinRecipe::ItemCelestial) && items[1].id == 0 {
        return Err(mismatch("Odin"));
    }
    match recipe {
        OdinRecipe::Plus12 => {
            if secreta_uneven(&items) {
                return Err(mismatch("Odin"));
            }
            if items[2].refine >= REFINE_MAX {
                return Err(mismatch("Odin"));
            }
            if items[2].id == ITEM_NYERDES {
                return Err(CoreError::RefineCappedItem {
                    item_index: items[2].id,
                });
            }
            if rules.ability(items[2].id, EF_MOBTYPE) == 3 {
                return Err(mismatch("Odin"));
            }
        }
        OdinRecipe::DestraveLv40 => {
            let level = store.stats_of(entity).map_or(0, |stats| stats.level);
            if level != DESTRAVE_LEVEL
                || store.celestial_lv40(entity).unwrap_or(false)
                || !store.celestial(entity).unwrap_or(false)
            {
                return Err(mismatch("Odin"));
            }
        }
        OdinRecipe::CapaCelestial => {
            let cape = store
                .inventory(entity)
                .and_then(|inv| inv.equip.get(CAPE_SLOT))
                .copied()
                .unwrap_or_default();
            if cape.id == 0 || cape.refine >= 9 || !store.celestial(entity).unwrap_or(false) {
                return Err(mismatch("Odin"));
            }
        }
        _ => {}
    }

    let roll = roll_199(raw_roll)?;
    let mut rate = value_or_zero(rules, "Odin", recipe.rate_key());
    let mut protect_count = 0u8;
    if matches!(recipe, OdinRecipe::Sets | OdinRecipe::ItemCelestial) {
        rate += i32::from(jitter.min(4));
    }
    if recipe == OdinRecipe::Plus12 {
        for item in &items {
            if (RUNE_LO..=RUNE_HI).contains(&item.id) {
                rate += value_or_zero(rules, "Odin", "PLUS12_RUNE");
            } else if item.id == NEUTRAL_RUNE && item.refine <= 9 {
                rate += value_or_zero(rules, "Odin", PLUS12_REF_KEYS[usize::from(item.refine)]);
            } else if item.id == PROTECTION_EMBLEM {
                protect_count = protect_count.saturating_add(1);
            }
        }
        let level = items[2].refine;
        if (12..=REFINE_MAX).contains(&level) {
            rate -= value_or_zero(rules, "Odin", PLUS12_MINUS_KEYS[usize::from(level - 12)]);
        }
    }

    // O +12+ do legado vivo nunca falha depois dos gates.
    let success = recipe == OdinRecipe::Plus12 || i32::from(roll) <= rate;

    let mut produced: Option<Item> = None;
    let mut cape_out: Option<Item> = None;
    if success {
        produced = match recipe {
            OdinRecipe::Sets | OdinRecipe::ItemCelestial => {
                let mut result = items[1];
                if let Some(extra) = rules.facts(items[0].id).map(|facts| facts.extra) {
                    if extra > 0 {
                        result.id = extra;
                    }
                }
                result.refine = 0;
                result.gem_effect = GEM_EFFECT_NONE;
                result.gem_value = 0;
                Some(result)
            }
            OdinRecipe::Plus12 => {
                let mut result = items[2];
                result.refine = result.refine.saturating_add(1).min(REFINE_MAX);
                if result.gem_effect != GEM_EFFECT_NONE {
                    result.gem_value =
                        gem_value(result.gem_effect, result.refine).unwrap_or(result.gem_value);
                }
                Some(result)
            }
            OdinRecipe::PistaDeRunas => Some(Item {
                id: ODIN_PISTA_RESULT,
                qty: 1,
                ..Item::default()
            }),
            OdinRecipe::PedraDaFuria => Some(Item {
                id: ODIN_FURIA_RESULT,
                qty: 1,
                ..Item::default()
            }),
            OdinRecipe::SecretaAgua
            | OdinRecipe::SecretaTerra
            | OdinRecipe::SecretaSol
            | OdinRecipe::SecretaVento => Some(Item {
                id: ODIN_SECRETA_BASE + u32::from(recipe.id() - 6),
                qty: 1,
                ..Item::default()
            }),
            OdinRecipe::SementeDeCristal => Some(Item {
                id: ODIN_SEMENTE_RESULT,
                qty: 1,
                ..Item::default()
            }),
            OdinRecipe::CapaCelestial => {
                let mut cape = store
                    .inventory(entity)
                    .and_then(|inv| inv.equip.get(CAPE_SLOT))
                    .copied()
                    .unwrap_or_default();
                cape.refine = cape.refine.saturating_add(1).min(REFINE_MAX);
                if cape.gem_effect != GEM_EFFECT_NONE {
                    cape.gem_value =
                        gem_value(cape.gem_effect, cape.refine).unwrap_or(cape.gem_value);
                }
                cape_out = Some(cape);
                Some(cape)
            }
            OdinRecipe::DestraveLv40 => None,
        };
    }

    publish(
        store,
        DomainEvent::CombineFinished {
            entity,
            family: "Odin",
            recipe: recipe.id(),
            success,
            rate_percent: rate,
            roll: i32::from(roll),
            produced_item: produced.map_or(0, |item| item.id),
        },
    )?;

    // Sets/armas restauram o item-base na falha; o resto perde tudo.
    let keep: &[usize] =
        if !success && matches!(recipe, OdinRecipe::Sets | OdinRecipe::ItemCelestial) {
            &slots[1..2]
        } else {
            &[]
        };
    let inv = store.inventory_mut(entity).expect("entidade viva");
    let consumed = consume_slots(inv, slots, keep);
    match (recipe, success) {
        (OdinRecipe::Sets | OdinRecipe::ItemCelestial, true) => {
            if let Some(result) = produced {
                inv.bag[slots[1]] = result;
            }
        }
        (OdinRecipe::Plus12, true) => {
            if let Some(result) = produced {
                inv.bag[slots[2]] = result;
            }
        }
        (
            OdinRecipe::PistaDeRunas
            | OdinRecipe::PedraDaFuria
            | OdinRecipe::SecretaAgua
            | OdinRecipe::SecretaTerra
            | OdinRecipe::SecretaSol
            | OdinRecipe::SecretaVento
            | OdinRecipe::SementeDeCristal,
            true,
        ) => {
            if let Some(result) = produced {
                inv.bag[slots[0]] = result;
            }
        }
        _ => {}
    }
    if let Some(cape) = cape_out {
        inv.equip[CAPE_SLOT] = cape;
    }
    if recipe == OdinRecipe::DestraveLv40 && success {
        store.set_celestial_lv40(entity, true);
    }

    Ok(ComposeOutcome {
        family: "Odin",
        recipe: recipe.id(),
        success,
        rate_percent: rate,
        roll: i32::from(roll),
        consumed,
        coin_spent: 0,
        protect_items: protect_count,
        produced,
    })
}

/// Resultado do passo de pó `+10 → +11`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PowderOutcome {
    /// A tentativa passou?
    pub success: bool,
    /// Taxa efetiva (15% no Lac, 100% no `Lactolerium_100`).
    pub rate_percent: i32,
    /// Sorteio de 0..=99.
    pub roll: u8,
    /// Refino antes.
    pub from_refine: u8,
    /// Refino depois (inalterado na falha).
    pub to_refine: u8,
}

/// Refino do passo especial: item `+10` + Poeira de Lactolerium → `+11`.
///
/// Só este passo está fiado aqui: ele é o único caminho legítimo de `+10` para cima e **não
/// usa piedade** (o legado curto-circuita a taxa em `GEM_LEVEL` e, na falha, não grava
/// contador). O caminho 0..9 continua no `wyd-sim` até a fiação de piedade no item.
///
/// Falha: o item fica intocado e o pó é gasto. Item 769 é recusado (teto de +9 por nome).
pub fn apply_powder<R: ComposeRules>(
    store: &mut ComponentStore,
    entity: Entity,
    target_slot: usize,
    powder_slot: usize,
    rules: &R,
    raw_roll: u8,
) -> Result<PowderOutcome, CoreError> {
    ensure_alive(store, entity)?;
    if target_slot >= BAG_SLOTS || powder_slot >= BAG_SLOTS {
        let slot = target_slot.max(powder_slot);
        return Err(CoreError::SlotOutOfRange {
            slot: slot as u16,
            max: (BAG_SLOTS - 1) as u16,
        });
    }
    if target_slot == powder_slot {
        return Err(CoreError::ComposeSlotRepeated {
            slot: target_slot as u16,
        });
    }
    let inv = store.inventory(entity).expect("entidade viva");
    let target = inv.bag[target_slot];
    let powder = inv.bag[powder_slot];
    if target.id == 0 {
        return Err(CoreError::EmptySlot {
            slot: target_slot as u16,
        });
    }
    if target.id == ITEM_NYERDES {
        return Err(CoreError::RefineCappedItem {
            item_index: target.id,
        });
    }
    if target.refine != GEM_LEVEL {
        return Err(CoreError::RefineNotAtGemLevel {
            refine: target.refine,
        });
    }
    let key = match powder.id {
        LAC_POWDER => "GEM_LEVEL",
        LACTO_100 => "AMAGO",
        outro => return Err(CoreError::WrongPowder { item_index: outro }),
    };
    if raw_roll > 99 {
        return Err(CoreError::RollOutOfRange {
            roll: u16::from(raw_roll),
            max: 99,
        });
    }
    let rate = value_or_zero(rules, "Powder", key);
    let success = rate != 0 && i32::from(raw_roll) <= rate;

    publish(
        store,
        DomainEvent::RefineFinished {
            entity,
            item_index: target.id,
            from_refine: target.refine,
            to_refine: if success {
                target.refine + 1
            } else {
                target.refine
            },
            rate_percent: rate,
            roll: raw_roll,
            success,
        },
    )?;

    let inv = store.inventory_mut(entity).expect("entidade viva");
    if inv.bag[powder_slot].qty <= 1 {
        inv.bag[powder_slot] = Item::default();
    } else {
        inv.bag[powder_slot].qty -= 1;
    }
    if success {
        let mut result = inv.bag[target_slot];
        result.refine = result.refine.saturating_add(1).min(REFINE_MAX);
        if result.gem_effect != GEM_EFFECT_NONE {
            result.gem_value =
                gem_value(result.gem_effect, result.refine).unwrap_or(result.gem_value);
        }
        inv.bag[target_slot] = result;
    }

    Ok(PowderOutcome {
        success,
        rate_percent: rate,
        roll: raw_roll,
        from_refine: target.refine,
        to_refine: if success {
            target.refine + 1
        } else {
            target.refine
        },
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ecs::components::{EntityKind, Inventory, Position, Stats};
    use std::collections::HashMap;

    /// Catálogo de teste: fatos/efeitos fixos e taxas carregadas à mão.
    struct Rules {
        facts: HashMap<u32, ItemFacts>,
        abilities: HashMap<(u32, u16), i32>,
        values: HashMap<(&'static str, &'static str), i32>,
    }

    impl Rules {
        fn novo() -> Self {
            Self {
                facts: HashMap::new(),
                abilities: HashMap::new(),
                values: HashMap::new(),
            }
        }

        fn fato(mut self, id: u32, facts: ItemFacts) -> Self {
            self.facts.insert(id, facts);
            self
        }
    }

    impl ComposeRules for Rules {
        fn facts(&self, item_index: u32) -> Option<ItemFacts> {
            self.facts.get(&item_index).copied()
        }

        fn ability(&self, item_index: u32, effect: u16) -> i32 {
            self.abilities
                .get(&(item_index, effect))
                .copied()
                .unwrap_or(0)
        }

        fn value(&self, family: &str, key: &str) -> Option<i32> {
            self.values
                .iter()
                .find(|((f, k), _)| *f == family && *k == key)
                .map(|(_, v)| *v)
        }
    }

    fn stats() -> Stats {
        Stats {
            level: 39,
            hp: 100,
            max_hp: 100,
            mp: 0,
            max_mp: 0,
            attack: 1,
            defense: 1,
        }
    }

    fn armazem(itens: &[(usize, Item)]) -> (ComponentStore, Entity) {
        let mut inv = Inventory::default();
        for (slot, item) in itens {
            inv.bag[*slot] = *item;
        }
        let mut store = ComponentStore::default();
        let e = store
            .spawn(EntityKind::Player, Position::default(), stats(), inv)
            .expect("spawn válido");
        (store, e)
    }

    /// As duas dobras são diferentes e não podem ser trocadas.
    #[test]
    fn roll_115_e_roll_199_dobram_diferente() {
        assert_eq!(roll_115(99), Ok(99));
        assert_eq!(roll_199(99), Ok(99));
        assert_eq!(roll_115(114), Ok(99), "115: ≥100 vira −15");
        assert_eq!(roll_199(114), Ok(15), "199: >100 vira −99");
        assert_eq!(roll_199(100), Ok(100));
        assert_eq!(roll_199(101), Ok(2));
        assert_eq!(roll_199(198), Ok(99));
        assert!(roll_115(115).is_err());
        assert!(roll_199(199).is_err());
        // Caso que distingue: taxa 40. O 115 dá 99 (falha); o 199 dá 15 (passa).
        assert!(i32::from(roll_115(114).unwrap()) > 40);
        assert!(i32::from(roll_199(114).unwrap()) <= 40);
    }

    #[test]
    fn regras_de_falha_do_p_o_e_do_emblema() {
        assert_eq!(powder_failure(false, 8), PowderFailure::Kept);
        assert_eq!(powder_failure(true, 8), PowderFailure::Destroyed);
        assert_eq!(powder_failure(true, 6), PowderFailure::Kept);
        assert_eq!(powder_failure(true, 10), PowderFailure::Kept);
        assert_eq!(plus12_failure(true), Plus12Failure::Downgraded);
        assert_eq!(plus12_failure(false), Plus12Failure::Destroyed);
        assert_eq!(downgrade_refine(11), 10);
        assert_eq!(downgrade_refine(14), 13);
        assert_eq!(downgrade_refine(7), 7);
    }

    #[test]
    fn joyas_mapeiam_para_o_efeito_da_gema() {
        assert_eq!(jewel_effect(2441), crate::ecs::components::GEM_EFFECT_DROP);
        assert_eq!(
            jewel_effect(2444),
            crate::ecs::components::GEM_EFFECT_ABSORB
        );
        assert_eq!(jewel_effect(1774), GEM_EFFECT_NONE);
    }

    #[test]
    fn match_odin_reconhece_o_ramo_mais_12() {
        let rules = Rules::novo().fato(
            1900,
            ItemFacts {
                pos: 64,
                ..ItemFacts::default()
            },
        );
        let mut items = [Item::default(); COMBINE_SLOTS];
        items[0] = Item {
            id: LAC_POWDER,
            qty: 10,
            ..Item::default()
        };
        items[1] = Item {
            id: LAC_POWDER,
            qty: 10,
            ..Item::default()
        };
        items[2] = Item {
            id: 1900,
            refine: 11,
            ..Item::default()
        };
        for item in items.iter_mut().take(7).skip(3) {
            *item = Item {
                id: RUNE_LO,
                ..Item::default()
            };
        }
        assert_eq!(match_odin(&items, &rules), Some(OdinRecipe::Plus12));
        // Uma pedra Secreta a menos é receita inválida, não "quase lá".
        items[3] = Item::default();
        assert_eq!(match_odin(&items, &rules), None);
    }

    #[test]
    fn match_odin_pista_e_semente_s_o_os_da_receita() {
        let rules = Rules::novo();
        let mut pista = [Item::default(); COMBINE_SLOTS];
        for item in pista.iter_mut().take(7) {
            *item = Item {
                id: LAC_POWDER,
                qty: 10,
                ..Item::default()
            };
        }
        assert_eq!(match_odin(&pista, &rules), Some(OdinRecipe::PistaDeRunas));
        let mut semente = [Item::default(); COMBINE_SLOTS];
        for (slot, id) in (421..=427).enumerate() {
            semente[slot] = Item {
                id,
                ..Item::default()
            };
        }
        assert_eq!(
            match_odin(&semente, &rules),
            Some(OdinRecipe::SementeDeCristal)
        );
    }

    #[test]
    fn aylin_recusa_item_769_e_o_passo_fora_do_gem_level() {
        let rules = Rules::novo().fato(
            769,
            ItemFacts {
                pos: 64,
                grade: 5,
                ..ItemFacts::default()
            },
        );
        let (mut store, e) = armazem(&[
            (
                0,
                Item {
                    id: 769,
                    qty: 1,
                    refine: 9,
                    ..Item::default()
                },
            ),
            (
                1,
                Item {
                    id: 769,
                    qty: 1,
                    refine: 9,
                    ..Item::default()
                },
            ),
            (
                2,
                Item {
                    id: PEDRA_DO_SABIO,
                    qty: 1,
                    ..Item::default()
                },
            ),
            (
                3,
                Item {
                    id: JEWEL_BASE,
                    qty: 1,
                    ..Item::default()
                },
            ),
            (
                4,
                Item {
                    id: JEWEL_BASE,
                    qty: 1,
                    ..Item::default()
                },
            ),
            (
                5,
                Item {
                    id: JEWEL_BASE,
                    qty: 1,
                    ..Item::default()
                },
            ),
            (
                6,
                Item {
                    id: JEWEL_BASE,
                    qty: 1,
                    ..Item::default()
                },
            ),
        ]);
        assert_eq!(
            aylin(&mut store, e, &[0, 1, 2, 3, 4, 5, 6], &rules, 0),
            Err(CoreError::RefineCappedItem { item_index: 769 })
        );
    }
}
