//! Progressão — pontos por nível, teto de maestria e degraus de skill.
//!
//! Tudo aqui é **CONFIRMADO** por fonte (`docs/wyd/progressao.md` §2,
//! `docs/wyd/skills.md` §2 e `docs/wyd/classes.md` §3), com duas exceções marcadas:
//!
//! - **Curva de EXP:** não existe para o build 7662 (`docs/wyd/progressao.md` §7). O módulo
//!   não embute curva nenhuma: recebe [`ExpTable`] pronta do conteúdo (ADR-0009). Curva
//!   inventada daria "progressão parecida", que é o erro que o manual proíbe.
//! - **Nível da 10ª skill:** a fonte só exige as anteriores + 4.000 de fama; usamos o nível
//!   da 9ª (201) como piso por consequência lógica — provisório, D-05.
//!
//! Unidade: EXP em `u64` (a 12ª/quests usam 1.950.000.000 e o excedente é retido), pontos em
//! `u32`, nível em `u16`. Sem arredondamento: só soma e divisão inteira exata.

use crate::error::SimError;

/// Nível máximo do Mortal (400); God/Arch/God2 são camadas, não outro teto numérico.
pub const MAX_LEVEL: u16 = 400;

/// Pontos de status por nível (6 em todas as faixas).
pub const STATUS_POINTS_PER_LEVEL: u32 = 6;
/// Pontos de aprendizado por nível na faixa 1–200.
pub const LEARNING_PER_LEVEL_1_200: u32 = 2;
/// Pontos de aprendizado por nível na faixa 201–355.
pub const LEARNING_PER_LEVEL_201_355: u32 = 3;
/// Pontos de aprendizado por nível na faixa 356–400 (o detalhe contraintuitivo).
pub const LEARNING_PER_LEVEL_356_400: u32 = 1;
/// Pontos de maestria por nível.
pub const MASTERY_PER_LEVEL: u32 = 2;
/// Teto de maestria por linhagem, base.
pub const MASTERY_CAP_BASE: u16 = 200;
/// Teto com a 8ª skill (ou item de maestria).
pub const MASTERY_CAP_EIGHTH: u16 = 255;
/// Teto com a 9ª skill.
pub const MASTERY_CAP_NINTH: u16 = 320;
/// Skills simultâneas por personagem.
pub const SKILL_SLOTS: usize = 24;

/// Curva de EXP: limiar **cumulativo** para alcançar cada nível (nível 1 = 0).
///
/// Fronteira de conteúdo: implementada por [`ExpCurve`] (memória) e por qualquer tabela
/// carregada do banco. A fórmula não conhece a curva, só a consulta.
pub trait ExpTable {
    /// EXP acumulada necessária para estar no `level` (1-based), se a tabela cobrir o nível.
    fn exp_for_level(&self, level: u16) -> Option<u64>;
    /// Último nível que a tabela descreve.
    fn max_level(&self) -> u16;
}

/// Curva em memória, validada na construção (crescente e não vazia).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ExpCurve {
    thresholds: Vec<u64>,
}

impl ExpCurve {
    /// Valida e guarda os limiares cumulativos (`thresholds[0]` = EXP do nível 1).
    ///
    /// # Errors
    /// [`SimError::EmptyExpTable`] sem limiares; [`SimError::NonMonotonicExp`] se a curva
    /// retroceder (curva de EXP que diminui é bug de conteúdo, não escolha).
    pub fn new(thresholds: Vec<u64>) -> Result<Self, SimError> {
        let Some(&primeiro) = thresholds.first() else {
            return Err(SimError::EmptyExpTable);
        };
        if primeiro != 0 {
            return Err(SimError::NonMonotonicExp { level: 1 });
        }
        for (indice, par) in thresholds.windows(2).enumerate() {
            if par[1] < par[0] {
                let nivel = u16::try_from(indice + 2).unwrap_or(MAX_LEVEL);
                return Err(SimError::NonMonotonicExp { level: nivel });
            }
        }
        Ok(Self { thresholds })
    }
}

impl ExpTable for ExpCurve {
    fn exp_for_level(&self, level: u16) -> Option<u64> {
        self.thresholds
            .get(usize::from(level).checked_sub(1)?)
            .copied()
    }

    fn max_level(&self) -> u16 {
        let len = self.thresholds.len();
        u16::try_from(len).unwrap_or(MAX_LEVEL).min(MAX_LEVEL)
    }
}

/// Estado de nível + EXP dentro do nível.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct LevelState {
    /// Nível atual (1-based).
    pub level: u16,
    /// EXP acumulada **dentro** do nível (já descontados os limiares anteriores).
    pub exp_into_level: u64,
}

/// Aplica EXP e devolve o novo estado.
///
/// No teto (nível máximo do Mortal ou da tabela) o excedente é **retido** em
/// `exp_into_level`, nunca descartado nem convertido em nível inexistente —
/// `docs/wyd/progressao.md` §6, regra 3.
pub fn grant_exp(state: LevelState, amount: u64, table: &impl ExpTable) -> LevelState {
    let teto = table.max_level().clamp(1, MAX_LEVEL);
    let mut nivel = state.level.clamp(1, teto);
    let mut restante = state.exp_into_level.saturating_add(amount);

    while nivel < teto {
        let proximo = nivel.saturating_add(1);
        let Some(limiar) = table.exp_for_level(proximo) else {
            break;
        };
        let Some(atual) = table.exp_for_level(nivel) else {
            break;
        };
        let custo = limiar.saturating_sub(atual);
        if restante < custo {
            break;
        }
        restante -= custo;
        nivel = proximo;
    }

    LevelState {
        level: nivel,
        exp_into_level: restante,
    }
}

/// EXP que ainda falta para o próximo nível, ou `None` no teto.
#[must_use]
pub fn exp_to_next(state: LevelState, table: &impl ExpTable) -> Option<u64> {
    let proximo = state.level.saturating_add(1);
    let limiar = table.exp_for_level(proximo)?;
    let atual = table.exp_for_level(state.level)?;
    Some(
        limiar
            .saturating_sub(atual)
            .saturating_sub(state.exp_into_level),
    )
}

/// Pontos de status acumulados até `level` (6 por nível; nível 1 = 0).
#[must_use]
pub fn status_points(level: u16) -> u32 {
    u32::from(level.min(MAX_LEVEL))
        .saturating_sub(1)
        .saturating_mul(STATUS_POINTS_PER_LEVEL)
}

/// Pontos de aprendizado acumulados até `level` (faixas 2 / 3 / 1).
#[must_use]
pub fn learning_points(level: u16) -> u32 {
    let ganhos = u32::from(level.min(MAX_LEVEL)).saturating_sub(1);
    if ganhos == 0 {
        return 0;
    }
    let faixa_inicial = ganhos.min(199);
    let faixa_media = ganhos.saturating_sub(199).min(155);
    let faixa_final = ganhos.saturating_sub(354);
    faixa_inicial
        .saturating_mul(LEARNING_PER_LEVEL_1_200)
        .saturating_add(faixa_media.saturating_mul(LEARNING_PER_LEVEL_201_355))
        .saturating_add(faixa_final.saturating_mul(LEARNING_PER_LEVEL_356_400))
}

/// Pontos de maestria acumulados até `level` (2 por nível).
#[must_use]
pub fn mastery_points(level: u16) -> u32 {
    u32::from(level.min(MAX_LEVEL))
        .saturating_sub(1)
        .saturating_mul(MASTERY_PER_LEVEL)
}

/// Limite de maestria **por nível** (`nível × 1,5`, piso inteiro).
#[must_use]
pub fn mastery_limit(level: u16) -> u32 {
    u32::from(level.min(MAX_LEVEL)).saturating_mul(3) / 2
}

/// Teto de maestria vigente por linhagem (320 com 9ª, 255 com 8ª, 200 base).
#[must_use]
pub fn mastery_cap(has_eighth: bool, has_ninth: bool) -> u16 {
    if has_ninth {
        MASTERY_CAP_NINTH
    } else if has_eighth {
        MASTERY_CAP_EIGHTH
    } else {
        MASTERY_CAP_BASE
    }
}

/// Degrau de skill de topo.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum SkillTier {
    /// 9ª — Armia, 255 pontos, nível 201, mesma linhagem da 8ª.
    Ninth,
    /// 10ª — 4.000 de fama.
    Tenth,
    /// 11ª — nível 281 e 1.000 de fama.
    Eleventh,
    /// 12ª — Celestial acima de 320 e 12.000 de fama.
    Twelfth,
}

/// Requisito de aquisição de um degrau de skill.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SkillRequirement {
    /// Nível mínimo.
    pub min_level: u16,
    /// Fama exigida.
    pub fame: u32,
    /// Pontos de skill exigidos.
    pub skill_points: u32,
}

/// Requisito publicado de cada degrau (`docs/wyd/skills.md` §1).
#[must_use]
pub fn skill_requirement(tier: SkillTier) -> SkillRequirement {
    match tier {
        SkillTier::Ninth => SkillRequirement {
            min_level: 201,
            fame: 0,
            skill_points: 255,
        },
        // Piso lógico: a 10ª pressupõe a 9ª — provisório (D-05).
        SkillTier::Tenth => SkillRequirement {
            min_level: 201,
            fame: 4_000,
            skill_points: 0,
        },
        SkillTier::Eleventh => SkillRequirement {
            min_level: 281,
            fame: 1_000,
            skill_points: 0,
        },
        // "Celestial > 320" na fonte: 321 é o primeiro nível que satisfaz.
        SkillTier::Twelfth => SkillRequirement {
            min_level: 321,
            fame: 12_000,
            skill_points: 0,
        },
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn curva() -> ExpCurve {
        // Curva sintética só para exercitar a máquina — não é a curva do jogo.
        ExpCurve::new(vec![0, 100, 300, 600]).expect("curva válida")
    }

    #[test]
    fn status_points_are_six_per_level() {
        assert_eq!(status_points(1), 0);
        assert_eq!(status_points(2), 6);
        assert_eq!(status_points(400), 2_394);
    }

    #[test]
    fn learning_points_drop_from_three_to_one() {
        assert_eq!(learning_points(200), 398);
        assert_eq!(learning_points(201), 401);
        assert_eq!(learning_points(355), 863);
        // A queda contraintuitiva: 356 soma só 1 em relação a 355.
        assert_eq!(learning_points(356), 864);
        assert_eq!(learning_points(400), 908);
    }

    #[test]
    fn mastery_caps_follow_skills_owned() {
        assert_eq!(mastery_cap(false, false), MASTERY_CAP_BASE);
        assert_eq!(mastery_cap(true, false), MASTERY_CAP_EIGHTH);
        assert_eq!(mastery_cap(true, true), MASTERY_CAP_NINTH);
    }

    #[test]
    fn mastery_limit_is_level_times_one_and_a_half() {
        assert_eq!(mastery_limit(200), 300);
        assert_eq!(mastery_limit(201), 301);
        assert_eq!(mastery_points(400), 798);
    }

    #[test]
    fn grant_exp_levels_up_by_cumulative_threshold() {
        let tabela = curva();
        let estado = LevelState {
            level: 1,
            exp_into_level: 0,
        };
        assert_eq!(
            grant_exp(estado, 100, &tabela),
            LevelState {
                level: 2,
                exp_into_level: 0
            }
        );
        assert_eq!(
            grant_exp(estado, 250, &tabela),
            LevelState {
                level: 2,
                exp_into_level: 150
            }
        );
    }

    #[test]
    fn exp_at_cap_is_retained_not_discarded() {
        let tabela = curva();
        let topo = LevelState {
            level: tabela.max_level(),
            exp_into_level: 10,
        };
        let depois = grant_exp(topo, 999, &tabela);
        assert_eq!(depois.level, tabela.max_level());
        assert_eq!(depois.exp_into_level, 1_009, "excedente retido");
        assert_eq!(exp_to_next(depois, &tabela), None);
    }

    #[test]
    fn invalid_curves_are_rejected() {
        assert_eq!(ExpCurve::new(vec![]), Err(SimError::EmptyExpTable));
        assert_eq!(
            ExpCurve::new(vec![0, 100, 50]),
            Err(SimError::NonMonotonicExp { level: 3 })
        );
        assert_eq!(
            ExpCurve::new(vec![7, 100]),
            Err(SimError::NonMonotonicExp { level: 1 })
        );
    }
}
