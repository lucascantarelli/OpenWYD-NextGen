//! Refino de acessório — sistema **posterior** ao 7662 (brinco, colar, cinto e afins).
//!
//! O original não refina acessório: a central Kersef diz que "acessórios não podem ser
//! combinados" no +0..+11, e o guia de refino de 2009 cobre arma/armadura. Quem publica
//! regra é o WYD Global (Bahamut), com receitas e falhas próprias:
//!
//! - brinco: +0..+9 a 40% por passo, +9..+11 a 5%, falha **mantém** o refino e dá 10% de
//!   Bahamut Tear; +11..+15 no NPC Torvald (20/15/10/5%);
//! - colar/cinto com 1 Poeira de Lac: +0..+6 60%, +6..+11 30%, falha **volta a +0**;
//! - colar/cinto com 100 Poeira de Lac: 75/50/25/10% e falha mantém.
//!
//! Nada aqui é constante de jogo: a tabela `accessory_refine_rule` é a verdade e o portal
//! a edita. Amuleto, ankh e pedra espiritual só existem quando houver fonte — o motor não
//! inventa passo nem taxa.

/// Último refino de acessório publicado (Torvald).
pub const ACCESSORY_MAX: u8 = 15;

/// Tipo de acessório. Só brinco/colar/cinto têm regra semeada hoje.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum AccessoryKind {
    Earring,
    Necklace,
    Belt,
    Amulet,
    Ankh,
    SpiritStone,
}

impl AccessoryKind {
    /// Código persistido em `accessory_refine_rule.kind`.
    #[must_use]
    pub const fn code(self) -> &'static str {
        match self {
            Self::Earring => "earring",
            Self::Necklace => "necklace",
            Self::Belt => "belt",
            Self::Amulet => "amulet",
            Self::Ankh => "ankh",
            Self::SpiritStone => "spirit_stone",
        }
    }
}

/// O que acontece com o acessório quando a tentativa falha.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FailMode {
    /// Mantém o refino atual (Bahamut: "não há chance de retorno").
    Keep,
    /// Volta a +0 (receita de 1 Lac do colar/cinto).
    Reset,
    /// Destrói o item (WYD International, patch 2025-02).
    Destroy,
}

/// Uma linha de `accessory_refine_rule`, já resolvida para o passo.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct AccessoryRule {
    pub step_from: u8,
    pub rate_bps: u16,
    pub fail_mode: FailMode,
    /// Chance (bps) do item de consolação na falha — Bahamut Tear a 10%.
    pub fail_bonus_bps: u16,
}

/// Resultado de uma tentativa.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Attempt {
    /// Subiu para `to`.
    Success { to: u8 },
    /// Falhou; `tear` diz se o item de consolação caiu.
    Failure { mode: FailMode, tear: bool },
    /// Não há receita para este passo (acessório sem regra ou receita esgotada).
    NoRule,
    /// Já está no teto de 15.
    Capped,
}

/// Resolve a tentativa de refino do passo `refine`.
///
/// `roll_bps` e `tear_roll_bps` são inteiros em 0..10000: a função é pura e determinística,
/// então o servidor grava os dois sorteios e a mesma entrada reproduz o resultado —
/// requisito de auditoria de refino (`refino.md` §8).
#[must_use]
pub fn attempt(rules: &[AccessoryRule], refine: u8, roll_bps: u16, tear_roll_bps: u16) -> Attempt {
    if refine >= ACCESSORY_MAX {
        return Attempt::Capped;
    }
    let Some(rule) = rules.iter().find(|rule| rule.step_from == refine) else {
        return Attempt::NoRule;
    };
    let roll = roll_bps.min(9_999);
    let rate = rule.rate_bps.min(10_000);
    if roll < rate {
        return Attempt::Success { to: refine + 1 };
    }
    Attempt::Failure {
        mode: rule.fail_mode,
        tear: tear_roll_bps.min(9_999) < rule.fail_bonus_bps.min(10_000),
    }
}

/// Refino resultante depois de uma falha.
#[must_use]
pub const fn refine_after_failure(refine: u8, mode: FailMode) -> Option<u8> {
    match mode {
        FailMode::Keep => Some(refine),
        FailMode::Reset => Some(0),
        FailMode::Destroy => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    const BAHAMUT_EARRING: [AccessoryRule; 3] = [
        AccessoryRule {
            step_from: 0,
            rate_bps: 4_000,
            fail_mode: FailMode::Keep,
            fail_bonus_bps: 1_000,
        },
        AccessoryRule {
            step_from: 9,
            rate_bps: 500,
            fail_mode: FailMode::Keep,
            fail_bonus_bps: 1_000,
        },
        AccessoryRule {
            step_from: 11,
            rate_bps: 2_000,
            fail_mode: FailMode::Keep,
            fail_bonus_bps: 1_000,
        },
    ];

    #[test]
    fn brinco_sobe_no_limite_da_taxa() {
        assert_eq!(
            attempt(&BAHAMUT_EARRING, 0, 3_999, 0),
            Attempt::Success { to: 1 }
        );
        assert_eq!(
            attempt(&BAHAMUT_EARRING, 0, 4_000, 9_999),
            Attempt::Failure {
                mode: FailMode::Keep,
                tear: false
            }
        );
    }

    #[test]
    fn falha_mantem_refino_e_sorteia_consolacao() {
        let falha = attempt(&BAHAMUT_EARRING, 9, 9_999, 999);
        assert_eq!(
            falha,
            Attempt::Failure {
                mode: FailMode::Keep,
                tear: true
            }
        );
        let kept = refine_after_failure(9, FailMode::Keep).expect("mantém");
        assert_eq!(kept, 9);
    }

    #[test]
    fn receita_de_um_lac_volta_a_zero() {
        let reset = [AccessoryRule {
            step_from: 3,
            rate_bps: 6_000,
            fail_mode: FailMode::Reset,
            fail_bonus_bps: 0,
        }];
        assert_eq!(
            refine_after_failure(3, FailMode::Reset),
            Some(0),
            "colar/cinto com 1 Lac voltam a +0"
        );
        assert_eq!(
            attempt(&reset, 3, 9_999, 0),
            Attempt::Failure {
                mode: FailMode::Reset,
                tear: false
            }
        );
    }

    #[test]
    fn passo_sem_receita_nao_inventa_taxa() {
        assert_eq!(attempt(&BAHAMUT_EARRING, 5, 0, 0), Attempt::NoRule);
        assert_eq!(attempt(&BAHAMUT_EARRING, 15, 0, 0), Attempt::Capped);
    }

    #[test]
    fn codigos_casam_com_a_tabela() {
        assert_eq!(AccessoryKind::Earring.code(), "earring");
        assert_eq!(AccessoryKind::SpiritStone.code(), "spirit_stone");
        assert_eq!(refine_after_failure(12, FailMode::Destroy), None);
    }
}
