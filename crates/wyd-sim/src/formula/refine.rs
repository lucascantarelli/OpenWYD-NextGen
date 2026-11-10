//! Refino — escada +0..+15, caminho de **pó** (legado) e **composição**.
//!
//! Duas famílias de regra, separadas de propósito, porque no jogo também são sistemas
//! diferentes:
//!
//! - **Pó** (`powder_*`): o caminho do legado (`_MSG_UseItem.cpp:802-976`,
//!   `SancRate.txt`, `RefineRate.txt`). A taxa vem de tabela por pó e **índice atual + 1**;
//!   a falha não destrói o item — acumula **piedade**, que soma `pity_weight` pontos
//!   percentuais por ponto. Tetos por pó vêm de [`RefinePolicy`].
//! - **Composição** (`blessing_*`, `anct_*`, `plus12_*`): taxa base + bônus de insumo
//!   (refinação abençoada, pedra secreta, sacrifício).
//!
//! Onde a fonte pública e o legado divergem, a divergência está registrada em
//! `docs/wyd/sistemas/refino-auditoria.md` e cada constante diz de qual fonte veio.
//! Percentual é **basis point** (`9000` = 90%): inteiro, determinístico, sem `f32` — a
//! conversão para exibição é problema da apresentação.

/// Último nível de refino do jogo (`REFINE_MAX` do legado).
pub const REFINE_MAX: u8 = 15;

/// Taxa de sucesso por passo (passo `1` = +0→+1), em basis points.
///
/// Passos 1–9: escada publicada do WYD International (90/85/…/50%).
/// Passo 10: composição do Aylin (30%, `docs/wyd/sistemas/refino.md` §3).
/// Passo 11: refino com Poeira de Lactolerium (3%; "Lac 100%" = 100%).
/// Passos 12–15: composição do Alquimista Odin (4/3/2/1%, doc §5).
pub const REFINE_RATE_BPS: [u16; 15] = [
    9_000, 8_500, 8_000, 7_500, 7_000, 6_500, 6_000, 5_500, 5_000, // +0..+8 → +9
    3_000, // +9 → +10 (composição, Aylin)
    300,   // +10 → +11 (pó de Lac)
    400,   // +11 → +12 (Odin)
    300,   // +12 → +13
    200,   // +13 → +14
    100,   // +14 → +15
];

/// Família do passo: pó (item + pó, falha não destrói) ou composição (insumos).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RefineKind {
    /// Refino com pó de Oriharukon/Lactolerium.
    Dust,
    /// Composição de NPC (Aylin, Odin) ou Anct.
    Compose,
}

impl RefineKind {
    /// Código persistido em `refine_rate.kind`.
    #[must_use]
    pub const fn code(self) -> &'static str {
        match self {
            Self::Dust => "dust",
            Self::Compose => "compose",
        }
    }
}

/// Família do passo `step` (+0→+1 é `1`), ou `None` fora de 1..=15.
#[must_use]
pub fn refine_step_kind(step: u8) -> Option<RefineKind> {
    match step {
        1..=9 | 11 => Some(RefineKind::Dust),
        10 | 12..=15 => Some(RefineKind::Compose),
        _ => None,
    }
}

/// Taxa de sucesso do passo `step` (+0→+1 é `1`), ou `None` fora de 1..=15.
#[must_use]
pub fn refine_success_bps(step: u8) -> Option<u16> {
    REFINE_RATE_BPS
        .get(usize::from(step).checked_sub(1)?)
        .copied()
}

// ---------------------------------------------------------------------------
// Refino com pó (legado)
// ---------------------------------------------------------------------------

/// Linha de bigorna do refino com pó (`SancRate.txt`).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Powder {
    /// Poeira de Oriharukon (`item_catalog` 412, `EF_VOLATILE` 4).
    Ori,
    /// Poeira de Lactolerium (413, `EF_VOLATILE` 5).
    Lac,
    /// Linha do Âmago, forçada pelo item `Lactolerium_100` (4141): sempre 100%.
    Amago,
}

impl Powder {
    /// Token persistido em `sanc_rate.powder`.
    #[must_use]
    pub const fn code(self) -> &'static str {
        match self {
            Self::Ori => "PO",
            Self::Lac => "PL",
            Self::Amago => "AMAGO",
        }
    }
}

/// Item que o legado limita a +9 por nome, qualquer que seja o pó (`_MSG_UseItem.cpp:802`).
pub const ITEM_NYERDES: u32 = 769;

/// Item de cash que força a linha do Âmago (100% garantido) sem mudar a grade do Lac.
pub const ITEM_LACTO_100: u32 = 4_141;

/// Percentual que o legado usa quando a linha não existe na tabela (`Rate` devolve 0).
pub const SANC_MISSING_PERCENT: u8 = 0;

/// Taxa fixa do passo especial +10 → +11 fora da linha do Âmago (`SuccessRate`).
pub const SANC_GEM_LEVEL_BPS: u16 = 1_500;

/// Tetos do refino com pó (`RefinePolicy` do legado, `RefineRate.txt`).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct RefinePolicy {
    /// Nível mais alto que a poeira de Ori alcança (`PO_MAX`).
    pub ori_max: u8,
    /// Nível mais alto que a poeira de Lac alcança (`PL_MAX`).
    pub pl_max: u8,
    /// Passo especial: sucesso aqui vai para +11 mantendo a gema (`GEM_LEVEL`).
    pub gem_level: u8,
    /// Parede absoluta do caminho de pó (`HARD_CAP`): nele ou acima, recusa.
    pub hard_cap: u8,
}

impl RefinePolicy {
    /// Configuração do legado (`RefineRate.txt` / `_MSG_UseItem.cpp`).
    pub const LEGACY: Self = Self {
        ori_max: 6,
        pl_max: 9,
        gem_level: 10,
        hard_cap: 11,
    };

    /// O pó `powder` pode refinar um item no `level` atual?
    ///
    /// Reproduz a ordem de recusa de `_MSG_UseItem.cpp:802`: Ori para no seu teto; Lac
    /// recusa o próprio teto, o `HARD_CAP` e o item [`ITEM_NYERDES`] a partir de `PL_MAX`;
    /// o Âmago segue a mesma janela do Lac e só chega ao passo especial.
    #[must_use]
    pub const fn allows(self, powder: Powder, level: u8, item_index: u32) -> bool {
        match powder {
            Powder::Ori => level < self.ori_max,
            Powder::Lac | Powder::Amago => {
                level < self.hard_cap
                    && level != self.pl_max
                    && !(level >= self.pl_max && item_index == ITEM_NYERDES)
            }
        }
    }
}

/// Teto do contador de piedade (`maxPity` do legado).
pub const PITY_MAX: u8 = 20;

/// Módulo do sorteio de piedade: `rand() % 4` (`_MSG_UseItem.cpp:943`).
pub const PITY_ROLL_MAX: u8 = 3;

/// Sorteio até este valor acumula piedade (3 em 4 tentativas).
pub const PITY_ROLL_THRESHOLD: u8 = 2;

/// Acima deste nível o Ori não acumula mais piedade (`_MSG_UseItem.cpp:943`).
pub const PITY_ORI_MAX_LEVEL: u8 = 5;

/// Peso de cada ponto de piedade, indexado por `nível + 1` (`g_pSuccessRate`).
const PITY_WEIGHT: [u8; 11] = [5, 5, 5, 5, 4, 4, 3, 3, 2, 1, 0];

/// Quantos pontos percentuais cada ponto de piedade vale no nível dado.
///
/// O legado indexa a tabela em `nível + 1` (o slot 0 não é usado), então +9 vale 0 e
/// +0 vale 5. Fora de 0..=9 não há peso (o passo +10→+11 é fixo).
#[must_use]
pub fn pity_weight(level: u8) -> Option<u8> {
    PITY_WEIGHT.get(usize::from(level) + 1).copied()
}

/// A falha soma um ponto de piedade? `roll` é `rand() % 4`.
///
/// O legado só acumula em 3 de 4 falhas (roll 0..=2); o Lac sempre acumula, o Ori só até
/// +5.
#[must_use]
pub fn pity_gain(powder: Powder, level: u8, roll: u8) -> bool {
    if roll > PITY_ROLL_MAX || roll > PITY_ROLL_THRESHOLD {
        return false;
    }
    match powder {
        Powder::Ori => level <= PITY_ORI_MAX_LEVEL,
        Powder::Lac | Powder::Amago => true,
    }
}

/// Taxa efetiva do refino com pó, em basis points.
///
/// `table_percent` é a célula de `sanc_rate` para `(powder, nível + 1)` — a tabela do
/// legado é lida em `nível + 1`, e índice ausente vale [`SANC_MISSING_PERCENT`] (`0`,
/// sempre falha). Regras, na ordem do legado:
///
/// 1. linha do Âmago devolve **100%** para qualquer nível;
/// 2. no `gem_level` (+10) a taxa é fixa em [`SANC_GEM_LEVEL_BPS`] (15%) — a tabela não é
///    consultada;
/// 3. abaixo disso, `percentual + piedade × peso`, limitado a 100%.
///
/// `None` só acima de [`REFINE_MAX`] (fora do domínio do refino).
#[must_use]
pub fn powder_success_bps(
    policy: RefinePolicy,
    powder: Powder,
    level: u8,
    pity: u8,
    table_percent: u8,
) -> Option<u16> {
    if level > REFINE_MAX {
        return None;
    }
    if powder == Powder::Amago {
        return Some(10_000);
    }
    if level == policy.gem_level {
        return Some(SANC_GEM_LEVEL_BPS);
    }
    let base = u16::from(table_percent) * 100;
    // O peso é em pontos percentuais por ponto de piedade; a taxa final é em bps.
    let peso = u16::from(pity_weight(level)?) * 100;
    let bonus = u16::from(pity.min(PITY_MAX)) * peso;
    Some(base.saturating_add(bonus).min(10_000))
}

/// Próximo nível de um refino com pó bem-sucedido, ou `None` acima de [`REFINE_MAX`].
///
/// Todo sucesso sobe exatamente um nível: `g_pSancGrade` é 1 em todas as células, e
/// `SancRate.txt` não traz linha de grade para sobrescrever. No passo especial (+10→+11) o
/// índice da gema é preservado — isso é dado do item, não do nível.
#[must_use]
pub fn powder_next_level(level: u8) -> Option<u8> {
    (level < REFINE_MAX).then_some(level + 1)
}

/// O sorteio de pó (`rand()%100`) passou? `roll` precisa estar em `0..=99`.
///
/// A comparação do legado é **inclusiva** (`roll <= rate`) e taxa 0 nunca passa, mesmo com
/// roll 0.
#[must_use]
pub fn powder_roll_succeeds(rate_bps: u16, roll: u8) -> bool {
    rate_bps != 0 && u16::from(roll) * 100 <= rate_bps
}

/// Efeito que carrega nível + piedade/gema no `STRUCT_ITEM` (`ItemEffect.h:100`).
///
/// É o par que o cliente lê para desenhar o `+N`: o `SendItem` do refino o carrega
/// com o `cValue` de [`encode_sanc_cvalue`], nunca um texto.
pub const EF_SANC: u8 = 43;

/// Base do `cValue` de item +10 (`gemBase`, `Basedef.cpp:2294-2308`).
///
/// Cada nível acima de +10 ocupa 4 valores, um por índice de gema (0..=3).
pub const SANC_GEM_BASE: u8 = 230;

/// Empacota nível + auxiliar no `cValue` do par EF_SANC (`BASE_SetItemSanc`).
///
/// Abaixo de +10, `cValue = nível + 10×piedade` (piedade 0..=20); de +10 em diante,
/// `cValue = 230 + (nível−10)×4 + gema` (gema 0..=3). O caminho de pó zera a piedade
/// no sucesso e só a acumula na falha qualificada, então o `aux` aqui é piedade
/// (< +10) ou índice de gema (10+); o refino de pó nunca minta gema, sempre 0.
///
/// Fora de 0..=15 o nível satura nos extremos em vez de zerar como o legado (o `encode`
/// do legado com nível inválido apaga o item para +0 — aqui quem chama já validou contra
/// [`REFINE_MAX`], então saturar é só defesa, nunca comportamento).
#[must_use]
pub fn encode_sanc_cvalue(level: u8, aux: u8) -> u8 {
    let level = if level > REFINE_MAX {
        REFINE_MAX
    } else {
        level
    };
    if level >= 10 {
        let gem = if aux > 3 { 3 } else { aux };
        SANC_GEM_BASE + (level - 10) * 4 + gem
    } else {
        let pity = if aux > PITY_MAX { PITY_MAX } else { aux };
        level + pity * 10
    }
}

/// O que acontece com o item quando o refino com pó falha.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PowderFailure {
    /// Item intocado; só o pó é gasto e a piedade cresce (caminho de equipamento comum).
    Kept,
    /// Item destruído (item **selado** em +7..+9).
    Destroyed,
}

/// Consequência de falha do caminho de pó.
///
/// O caminho de equipamento comum do legado **nunca** destrói (`refineFail`): o item fica
/// como está. A destruição em +7..+9 é a regra publicada para item **selado**, um dos
/// quatro ramos de pó que o legado adiou (`_MSG_UseItem.cpp:224`).
#[must_use]
pub fn powder_failure(sealed: bool, level: u8) -> PowderFailure {
    if sealed && (7..=9).contains(&level) {
        PowderFailure::Destroyed
    } else {
        PowderFailure::Kept
    }
}

// ---------------------------------------------------------------------------
// Composição (+10, Anct, +12..+15)
// ---------------------------------------------------------------------------

/// Bônus de sucesso de uma **refinação abençoada**, por nível da própria pedra, em bps.
///
/// Tabela do WYD International (doc §2): 1/1/2/3/4/5/6/7/8/10%. Quatro pedras +9 somam
/// 40%. A tabela da central Kersef é outra (2…15%) e está registrada na auditoria; o valor
/// adotado é o desta constante.
pub const BLESSING_BONUS_BPS: [u16; 10] = [100, 100, 200, 300, 400, 500, 600, 700, 800, 1_000];

/// Quantas refinações abençoadas/insumos a receita aceita.
pub const BLESSING_MAX_ITEMS: u8 = 4;

/// Bônus de uma refinação abençoada no refino dado, ou `None` fora de 0..=9.
#[must_use]
pub fn blessing_bonus_bps(refine: u8) -> Option<u16> {
    BLESSING_BONUS_BPS.get(usize::from(refine)).copied()
}

/// Soma dos bônus de uma lista de pedras (o chamador garante o limite de slots).
#[must_use]
pub fn blessing_total_bps(blessings: &[u8]) -> Option<u16> {
    let mut total = 0u16;
    for &refine in blessings {
        total = total.saturating_add(blessing_bonus_bps(refine)?);
    }
    Some(total)
}

/// Regra de taxa do Anct: base + bônus por item de sacrifício (+7..+10).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct AnctRules {
    /// Taxa base da receita, em bps.
    pub base_bps: u16,
    /// Bônus por sacrifício em +7, +8, +9 e +10 (0 = nível não aceito).
    pub bonus_bps: [u16; 4],
    /// Quantos itens de sacrifício a receita aceita.
    pub max_sacrifices: usize,
}

impl AnctRules {
    /// Regra do legado (`g_pAnctChance` 2/4/10, base 1%, +10 recusado).
    pub const LEGACY: Self = Self {
        base_bps: 100,
        bonus_bps: [200, 400, 1_000, 0],
        // MAX_COMBINE é 8 e os slots 0 e 1 são alvo e joia: sobram 6 sacrifícios.
        max_sacrifices: 6,
    };

    /// Regra publicada (base 5%, +7/+8/+9/+10 = 2/4/10/12%), doc §4.
    pub const PUBLISHED: Self = Self {
        base_bps: 500,
        bonus_bps: [200, 400, 1_000, 1_200],
        max_sacrifices: 4,
    };
}

/// Nível do item resultante de um Anct bem-sucedido (`anctResultSanc`).
pub const ANCT_RESULT_REFINE: u8 = 7;

/// Taxa da composição Anct: base + bônus de cada sacrifício.
///
/// `None` quando há mais itens que `rules.max_sacrifices` ou algum sacrifício em nível que
/// a régua não aceita (o legado trata como receita inválida, não como taxa 0).
#[must_use]
pub fn anct_rate_bps(rules: AnctRules, sacrifices: &[u8]) -> Option<u16> {
    if sacrifices.len() > rules.max_sacrifices {
        return None;
    }
    let mut total = rules.base_bps;
    for &level in sacrifices {
        let index = usize::from(level.checked_sub(7)?);
        let bonus = *rules.bonus_bps.get(index)?;
        if bonus == 0 {
            return None;
        }
        total = total.saturating_add(bonus);
    }
    Some(total.min(10_000))
}

/// Taxa base da composição +11→+12 … +14→+15, em bps (doc §5: 4/3/2/1%).
pub const PLUS12_BASE_BPS: [u16; 4] = [400, 300, 200, 100];

/// Bônus de cada pedra secreta na composição +12..+15 (+1%).
pub const SECRET_STONE_BPS: u16 = 100;

/// Taxa da composição +12..+15: base do passo + pedras secretas + bônus abençoados.
///
/// `None` fora de +11..+14 (não há passo a subir) ou com pedra abençoada fora de 0..=9.
#[must_use]
pub fn plus12_rate_bps(level: u8, secret_stones: u8, blessings: &[u8]) -> Option<u16> {
    let index = usize::from(level.checked_sub(11)?);
    let base = *PLUS12_BASE_BPS.get(index)?;
    let stones = u16::from(secret_stones).saturating_mul(SECRET_STONE_BPS);
    let abencoadas = blessing_total_bps(blessings)?;
    Some(
        base.saturating_add(stones)
            .saturating_add(abencoadas)
            .min(10_000),
    )
}

/// Consequência de falha na composição +12..+15 (doc §5).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Plus12Failure {
    /// Pacote de Lac: insumos destruídos e o item também (chance pequena, sem número).
    Destroyed,
    /// Emblema da Proteção: insumos destruídos e o item **cai um nível**.
    Downgraded,
}

/// Resultado do item conforme o insumo de proteção usado.
#[must_use]
pub const fn plus12_failure(protection: bool) -> Plus12Failure {
    if protection {
        Plus12Failure::Downgraded
    } else {
        Plus12Failure::Destroyed
    }
}

/// Nível seguinte da escada +12..+15 (`None` fora de +11..+14).
#[must_use]
pub fn plus12_next_level(level: u8) -> Option<u8> {
    (11..=14).contains(&level).then_some(level + 1)
}

// ---------------------------------------------------------------------------
// Gemas de efeito (item Anct ou +10..+15)
// ---------------------------------------------------------------------------

/// `item_catalog.item_index` das gemas de efeito.
pub const GEM_DIAMOND: u32 = 3386;
/// Gema de Esmeralda (perfuração).
pub const GEM_EMERALD: u32 = 3387;
/// Gema de Coral (+2% EXP).
pub const GEM_CORAL: u32 = 3388;
/// Gema de Garnet (absorção).
pub const GEM_GARNET: u32 = 3389;

/// Refino mínimo para gema em item comum.
pub const GEM_MIN_REFINE: u8 = 10;
/// Refino máximo conhecido.
pub const GEM_MAX_REFINE: u8 = 15;
/// Perfuração/absorção por nível acima de +9.
pub const GEM_PER_LEVEL: u32 = 40;
/// Drop fixo, em pontos percentuais.
pub const GEM_DROP_PERCENT: u32 = 8;
/// EXP fixa, em pontos percentuais.
pub const GEM_EXP_PERCENT: u32 = 2;

/// Efeito extra aplicado pela gema.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum GemEffect {
    /// +8% de drop (fixo).
    Drop,
    /// Perfuração de dano (40 por nível acima de +9).
    Pierce,
    /// +2% de EXP (fixo).
    Exp,
    /// Absorção de dano (40 por nível acima de +9).
    Absorb,
}

impl GemEffect {
    /// Código persistido em `refine_gem_effect.effect_code`.
    #[must_use]
    pub const fn code(self) -> &'static str {
        match self {
            Self::Drop => "drop",
            Self::Pierce => "pierce",
            Self::Exp => "exp",
            Self::Absorb => "absorb",
        }
    }

    /// Traduz o `item_index` da gema no efeito correspondente.
    #[must_use]
    pub const fn from_item(item_index: u32) -> Option<Self> {
        match item_index {
            GEM_DIAMOND => Some(Self::Drop),
            GEM_EMERALD => Some(Self::Pierce),
            GEM_CORAL => Some(Self::Exp),
            GEM_GARNET => Some(Self::Absorb),
            _ => None,
        }
    }
}

/// Motivo de recusa da gema.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum GemError {
    /// Não é uma das quatro gemas.
    UnknownGem { item_index: u32 },
    /// Item comum abaixo de +10.
    RefineTooLow { refine: u8 },
    /// Refino acima de +15.
    RefineTooHigh { refine: u8 },
}

/// Resultado da aplicação.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct AppliedGem {
    pub effect: GemEffect,
    pub value: u32,
    pub refine: u8,
}

/// Valor do efeito no refino dado, ou `None` fora de 10..=15.
#[must_use]
pub fn gem_effect_value(effect: GemEffect, refine: u8) -> Option<u32> {
    if !(GEM_MIN_REFINE..=GEM_MAX_REFINE).contains(&refine) {
        return None;
    }
    let steps = u32::from(refine - (GEM_MIN_REFINE - 1));
    match effect {
        GemEffect::Drop => Some(GEM_DROP_PERCENT),
        GemEffect::Exp => Some(GEM_EXP_PERCENT),
        GemEffect::Pierce | GemEffect::Absorb => Some(GEM_PER_LEVEL * steps),
    }
}

/// Valida e aplica a gema. Anct abaixo de +10 recebe o valor base, nunca negativo.
pub fn apply_gem(gem_item: u32, refine: u8, ancient: bool) -> Result<AppliedGem, GemError> {
    let effect = GemEffect::from_item(gem_item).ok_or(GemError::UnknownGem {
        item_index: gem_item,
    })?;
    if refine > GEM_MAX_REFINE {
        return Err(GemError::RefineTooHigh { refine });
    }
    if !ancient && refine < GEM_MIN_REFINE {
        return Err(GemError::RefineTooLow { refine });
    }
    let effective = refine.max(GEM_MIN_REFINE);
    Ok(AppliedGem {
        effect,
        value: gem_effect_value(effect, effective).expect("faixa validada"),
        refine: effective,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ladder_covers_all_fifteen_steps() {
        assert_eq!(refine_success_bps(1), Some(9_000));
        assert_eq!(refine_success_bps(5), Some(7_000));
        assert_eq!(refine_success_bps(9), Some(5_000));
        assert_eq!(refine_success_bps(10), Some(3_000));
        assert_eq!(refine_success_bps(11), Some(300));
        assert_eq!(refine_success_bps(15), Some(100));
        assert_eq!(refine_success_bps(0), None);
        assert_eq!(refine_success_bps(16), None);
    }

    #[test]
    fn step_kind_splits_dust_from_composition() {
        assert_eq!(refine_step_kind(1), Some(RefineKind::Dust));
        assert_eq!(refine_step_kind(9), Some(RefineKind::Dust));
        assert_eq!(refine_step_kind(10), Some(RefineKind::Compose));
        assert_eq!(refine_step_kind(11), Some(RefineKind::Dust));
        assert_eq!(refine_step_kind(12), Some(RefineKind::Compose));
        assert_eq!(refine_step_kind(15), Some(RefineKind::Compose));
        assert_eq!(refine_step_kind(16), None);
        assert_eq!(RefineKind::Dust.code(), "dust");
        assert_eq!(RefineKind::Compose.code(), "compose");
    }

    #[test]
    fn policy_caps_match_the_legacy_file() {
        let p = RefinePolicy::LEGACY;
        assert!(p.allows(Powder::Ori, 5, 100));
        assert!(!p.allows(Powder::Ori, 6, 100), "Ori para em +6");
        assert!(p.allows(Powder::Lac, 8, 100));
        assert!(!p.allows(Powder::Lac, 9, 100), "Lac recusa o próprio teto");
        assert!(p.allows(Powder::Lac, 10, 100), "o passo +10 → +11 é do Lac");
        assert!(!p.allows(Powder::Lac, 11, 100), "HARD_CAP 11");
        assert!(!p.allows(Powder::Amago, 9, 100));
        assert!(p.allows(Powder::Amago, 10, 100));
        assert!(
            !p.allows(Powder::Lac, 10, ITEM_NYERDES),
            "item 769 não passa de +9"
        );
    }

    #[test]
    fn powder_rate_adds_pity_weight() {
        let p = RefinePolicy::LEGACY;
        assert_eq!(powder_success_bps(p, Powder::Ori, 0, 0, 100), Some(10_000));
        assert_eq!(powder_success_bps(p, Powder::Ori, 4, 0, 40), Some(4_000));
        // +4→+5: 40% + 4 pontos × peso 4 = 56%.
        assert_eq!(powder_success_bps(p, Powder::Ori, 4, 4, 40), Some(5_600));
        assert_eq!(
            powder_success_bps(p, Powder::Ori, 5, 20, SANC_MISSING_PERCENT),
            Some(6_000),
            "+5→+6 depende só da piedade (tabela vale 0)"
        );
        assert_eq!(
            powder_success_bps(p, Powder::Ori, 9, 20, 10),
            Some(1_000),
            "peso em +9 é zero"
        );
    }

    #[test]
    fn powder_special_steps_are_flat() {
        let p = RefinePolicy::LEGACY;
        assert_eq!(powder_success_bps(p, Powder::Lac, 10, 7, 10), Some(1_500));
        assert_eq!(powder_success_bps(p, Powder::Amago, 10, 0, 0), Some(10_000));
        assert_eq!(powder_success_bps(p, Powder::Amago, 0, 0, 0), Some(10_000));
        assert_eq!(powder_success_bps(p, Powder::Lac, 16, 0, 0), None);
    }

    #[test]
    fn pity_only_grows_on_three_of_four_rolls() {
        assert!(pity_gain(Powder::Lac, 8, 0));
        assert!(pity_gain(Powder::Lac, 8, 2));
        assert!(!pity_gain(Powder::Lac, 8, 3));
        assert!(pity_gain(Powder::Ori, 5, 1));
        assert!(!pity_gain(Powder::Ori, 6, 1), "Ori só acumula até +5");
        assert!(pity_weight(0) == Some(5));
        assert_eq!(pity_weight(9), Some(0));
        assert_eq!(pity_weight(10), None);
    }

    #[test]
    fn sanc_cvalue_packs_level_plus_ten_times_pity() {
        // Abaixo de +10: cValue = nível + 10×piedade (golden do empacotamento).
        assert_eq!(encode_sanc_cvalue(0, 0), 0);
        assert_eq!(encode_sanc_cvalue(5, 0), 5);
        assert_eq!(encode_sanc_cvalue(5, 3), 35);
        assert_eq!(encode_sanc_cvalue(9, 20), 209);
        assert_eq!(encode_sanc_cvalue(0, 20), 200);
        // Piedade acima do teto satura, nunca estoura o byte.
        assert_eq!(encode_sanc_cvalue(9, 255), 209);
    }

    #[test]
    fn sanc_cvalue_gem_form_above_nine() {
        // +10 em diante: 230 + (nível−10)×4 + gema (gema sempre 0 no pó).
        assert_eq!(encode_sanc_cvalue(10, 0), 230);
        assert_eq!(encode_sanc_cvalue(10, 2), 232);
        assert_eq!(encode_sanc_cvalue(11, 0), 234);
        assert_eq!(encode_sanc_cvalue(15, 0), 250);
        assert_eq!(encode_sanc_cvalue(15, 3), 253);
        assert_eq!(encode_sanc_cvalue(12, 9), 241, "gema satura em 3");
    }

    #[test]
    fn dust_failure_only_destroys_sealed_items() {
        assert_eq!(powder_failure(false, 8), PowderFailure::Kept);
        assert_eq!(powder_failure(true, 8), PowderFailure::Destroyed);
        assert_eq!(powder_failure(true, 6), PowderFailure::Kept);
        assert_eq!(powder_failure(true, 10), PowderFailure::Kept);
    }

    #[test]
    fn powder_roll_is_inclusive_and_zero_never_passes() {
        assert!(powder_roll_succeeds(300, 2));
        assert!(
            powder_roll_succeeds(300, 3),
            "3% inclui o roll 3: 0..=3 no sorteio de 100"
        );
        assert!(!powder_roll_succeeds(300, 4));
        assert!(!powder_roll_succeeds(0, 0), "taxa 0 nunca passa");
        assert!(powder_roll_succeeds(10_000, 99));
    }

    #[test]
    fn powder_success_always_moves_one_level() {
        assert_eq!(powder_next_level(0), Some(1));
        assert_eq!(powder_next_level(10), Some(11));
        assert_eq!(powder_next_level(14), Some(15));
        assert_eq!(powder_next_level(15), None);
    }

    #[test]
    fn blessing_table_matches_the_documented_tiers() {
        assert_eq!(blessing_bonus_bps(0), Some(100));
        assert_eq!(blessing_bonus_bps(1), Some(100));
        assert_eq!(blessing_bonus_bps(2), Some(200));
        assert_eq!(blessing_bonus_bps(9), Some(1_000));
        assert_eq!(blessing_bonus_bps(10), None);
        assert_eq!(
            blessing_total_bps(&[9, 9, 9, 9]),
            Some(4_000),
            "4 pedras +9 = 40%"
        );
    }

    #[test]
    fn anct_rules_reproduce_both_sources() {
        // Legado: base 1% + 2/4/10 por +7/+8/+9.
        assert_eq!(
            anct_rate_bps(AnctRules::LEGACY, &[7, 8, 9]),
            Some(100 + 200 + 400 + 1_000)
        );
        assert_eq!(
            anct_rate_bps(AnctRules::LEGACY, &[10]),
            None,
            "legado recusa +10 como sacrifício"
        );
        // Publicado: base 5% e +12% em +10; 4 reforços +10 = 53%.
        assert_eq!(
            anct_rate_bps(AnctRules::PUBLISHED, &[10, 10, 10, 10]),
            Some(5_300)
        );
        assert_eq!(anct_rate_bps(AnctRules::PUBLISHED, &[6]), None);
        assert_eq!(
            anct_rate_bps(AnctRules::PUBLISHED, &[7, 7, 7, 7, 7]),
            None,
            "publicado: no máximo 4 sacrifícios"
        );
        assert_eq!(
            anct_rate_bps(AnctRules::LEGACY, &[7, 7, 7, 7, 7]),
            Some(100 + 5 * 200),
            "legado: MAX_COMBINE 8 deixa 6 slots de sacrifício"
        );
        assert_eq!(ANCT_RESULT_REFINE, 7);
    }

    #[test]
    fn plus12_rate_adds_stones_and_blessings() {
        assert_eq!(plus12_rate_bps(11, 0, &[]), Some(400));
        assert_eq!(
            plus12_rate_bps(11, 4, &[]),
            Some(800),
            "+11→+12 com 4 pedras secretas = 8%"
        );
        assert_eq!(plus12_rate_bps(14, 0, &[]), Some(100));
        assert_eq!(
            plus12_rate_bps(11, 0, &[9]),
            Some(1_400),
            "bênção +9 soma 10%"
        );
        assert_eq!(plus12_rate_bps(10, 0, &[]), None);
        assert_eq!(plus12_rate_bps(15, 0, &[]), None);
        assert_eq!(plus12_rate_bps(11, 0, &[10]), None);
        assert_eq!(plus12_next_level(11), Some(12));
        assert_eq!(plus12_next_level(14), Some(15));
        assert_eq!(plus12_next_level(15), None);
        assert_eq!(plus12_failure(true), Plus12Failure::Downgraded);
        assert_eq!(plus12_failure(false), Plus12Failure::Destroyed);
    }

    #[test]
    fn gem_scale_matches_kersef() {
        for (refine, value) in [
            (10, 40),
            (11, 80),
            (12, 120),
            (13, 160),
            (14, 200),
            (15, 240),
        ] {
            assert_eq!(gem_effect_value(GemEffect::Pierce, refine), Some(value));
            assert_eq!(gem_effect_value(GemEffect::Absorb, refine), Some(value));
        }
    }

    #[test]
    fn flat_effects_do_not_scale() {
        assert_eq!(gem_effect_value(GemEffect::Drop, 10), Some(8));
        assert_eq!(gem_effect_value(GemEffect::Drop, 15), Some(8));
        assert_eq!(gem_effect_value(GemEffect::Exp, 12), Some(2));
    }

    #[test]
    fn rejects_unknown_gem_and_out_of_range() {
        assert_eq!(
            apply_gem(2441, 12, false),
            Err(GemError::UnknownGem { item_index: 2441 })
        );
        assert_eq!(
            apply_gem(GEM_DIAMOND, 9, false),
            Err(GemError::RefineTooLow { refine: 9 })
        );
        assert_eq!(
            apply_gem(GEM_DIAMOND, 16, true),
            Err(GemError::RefineTooHigh { refine: 16 })
        );
    }

    #[test]
    fn ancient_below_ten_uses_base_value() {
        assert_eq!(
            apply_gem(GEM_GARNET, 9, true),
            Ok(AppliedGem {
                effect: GemEffect::Absorb,
                value: 40,
                refine: 10
            })
        );
    }
}
