//! Dano e mitigação — a cadeia completa, com a ordem marcada como provisória.
//!
//! O único passo **provado** é o último: absorção nunca torna o dano negativo
//! ([`damage_after_absorb`], invariante do combate). Os demais passos são a hipótese de
//! menor complexidade compatível com o que as fontes confirmam (`docs/wyd/skills.md` §5:
//! físico/mágico separados, arma soma dano, maestria multiplica a skill, refino multiplica
//! até 1,9×, +10 dá absorção e perfuração) — decisões D-02/D-03/D-04 em
//! `docs/arquitetura/matematica-sim.md` §4, a confirmar por trace.
//!
//! Ordem executada (uma passada, sem laço, sem `f64`):
//! `base + atributo + equipamento` → `skill` → `buff` → `crítico` → `perfuração/defesa`
//! → `resistência` → `absorção` → saturação em zero.

use crate::units::{mul_bps, Bps};

/// Rótulos da ordem de resolução — o teste de estrutura prende esta lista.
pub const STAGE_ORDER: [&str; 7] = [
    "attack_power",
    "skill",
    "buff",
    "critical",
    "pierce_defense",
    "resist",
    "absorb",
];

/// Dano depois da absorção, com saturação em zero.
#[must_use]
pub fn damage_after_absorb(base: u32, absorb: u32) -> u32 {
    base.saturating_sub(absorb)
}

/// Entrada da cadeia de dano (uma atacante, um alvo, um golpe).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct DamageInput {
    /// Dano base da arma/skill.
    pub base: u32,
    /// Parcela de atributo (já convertida pelos pesos de conteúdo).
    pub attribute: u32,
    /// Parcela de equipamento.
    pub gear_attack: u32,
    /// Poder da skill em bps (`10_000` = neutro).
    pub skill_bps: Bps,
    /// Buffs percentuais somados em bps (`10_000` = neutro).
    pub buff_bps: Bps,
    /// Multiplicador de crítico em bps (`10_000` = golpe normal).
    pub crit_multiplier_bps: Bps,
    /// Perfuração (gema de esmeralda): reduz a defesa efetiva.
    pub pierce: u32,
    /// Defesa do alvo.
    pub defense: u32,
    /// Resistência elemental do alvo (0 = nenhuma).
    pub resist_bps: Bps,
    /// Absorção do alvo (gema de garnet, montaria).
    pub absorb: u32,
}

impl DamageInput {
    /// Entrada neutra: só o dano base, todo modificador na identidade.
    #[must_use]
    pub fn new(base: u32) -> Self {
        Self {
            base,
            attribute: 0,
            gear_attack: 0,
            skill_bps: Bps::FULL,
            buff_bps: Bps::FULL,
            crit_multiplier_bps: Bps::FULL,
            pierce: 0,
            defense: 0,
            resist_bps: Bps::ZERO,
            absorb: 0,
        }
    }
}

/// Poder de ataque antes dos percentuais: `base + atributo + equipamento`.
#[must_use]
pub fn attack_power(base: u32, attribute: u32, gear_attack: u32) -> u32 {
    base.saturating_add(attribute).saturating_add(gear_attack)
}

/// Defesa efetiva depois da perfuração (nunca negativa).
#[must_use]
pub fn effective_defense(defense: u32, pierce: u32) -> u32 {
    defense.saturating_sub(pierce)
}

/// Executa a cadeia inteira e devolve o dano final.
#[must_use]
pub fn damage_chain(input: &DamageInput) -> u32 {
    let bruto = attack_power(input.base, input.attribute, input.gear_attack);
    let com_skill = input.skill_bps.apply(bruto);
    let com_buff = input.buff_bps.apply(com_skill);
    let com_critico = input.crit_multiplier_bps.apply(com_buff);
    let apos_defesa = com_critico.saturating_sub(effective_defense(input.defense, input.pierce));
    let resist = input.resist_bps.clamp(Bps::ZERO, Bps::FULL);
    let apos_resistencia = apos_defesa.saturating_sub(mul_bps(apos_defesa, resist.raw()));
    damage_after_absorb(apos_resistencia, input.absorb)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn absorb_never_underflows() {
        assert_eq!(damage_after_absorb(100, 30), 70);
        assert_eq!(damage_after_absorb(10, 30), 0);
    }

    #[test]
    fn zero_absorb_keeps_base() {
        assert_eq!(damage_after_absorb(37, 0), 37);
    }

    #[test]
    fn neutral_input_keeps_base() {
        assert_eq!(damage_chain(&DamageInput::new(123)), 123);
    }

    #[test]
    fn stage_order_is_the_documented_one() {
        assert_eq!(
            STAGE_ORDER,
            [
                "attack_power",
                "skill",
                "buff",
                "critical",
                "pierce_defense",
                "resist",
                "absorb",
            ]
        );
    }

    #[test]
    fn chain_applies_percentages_before_defense() {
        let entrada = DamageInput {
            base: 100,
            skill_bps: Bps::from_percent(150),
            defense: 50,
            ..DamageInput::new(0)
        };
        // 100 × 1,5 = 150; 150 − 50 = 100.
        assert_eq!(damage_chain(&entrada), 100);
    }

    #[test]
    fn pierce_reduces_defense_not_damage() {
        let sem_perfuracao = DamageInput {
            base: 100,
            defense: 90,
            ..DamageInput::new(0)
        };
        assert_eq!(damage_chain(&sem_perfuracao), 10);

        let com_perfuracao = DamageInput {
            pierce: 40,
            ..sem_perfuracao
        };
        assert_eq!(
            damage_chain(&com_perfuracao),
            50,
            "perfuração consome defesa"
        );
        assert_eq!(effective_defense(90, 1_000), 0);
    }

    #[test]
    fn absorb_and_resist_cannot_go_below_zero() {
        let entrada = DamageInput {
            base: 50,
            resist_bps: Bps::FULL,
            absorb: 500,
            ..DamageInput::new(0)
        };
        assert_eq!(damage_chain(&entrada), 0);
    }

    #[test]
    fn crit_multiplier_is_a_percentage_of_the_whole() {
        let entrada = DamageInput {
            base: 100,
            crit_multiplier_bps: Bps::from_percent(200),
            ..DamageInput::new(0)
        };
        assert_eq!(damage_chain(&entrada), 200);
    }

    #[test]
    fn worked_example_matches_documented_chain() {
        let entrada = DamageInput {
            base: 100,
            attribute: 20,
            gear_attack: 10,
            skill_bps: Bps::from_percent(150),
            buff_bps: Bps::from_percent(110),
            crit_multiplier_bps: Bps::from_percent(200),
            pierce: 30,
            defense: 100,
            resist_bps: Bps::from_percent(25),
            absorb: 19,
        };
        // A conta completa está em docs/arquitetura/matematica-sim.md §3.2.1.
        assert_eq!(damage_chain(&entrada), 250);
    }
}
