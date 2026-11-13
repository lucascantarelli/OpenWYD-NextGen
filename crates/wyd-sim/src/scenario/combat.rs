//! Combate: um golpe, na ordem do tick.
//!
//! Fases do tick (fixas, nesta ordem — `docs/arquitetura/matematica-sim.md` §3):
//!
//! 1. `effects_tick` — durações descontam um tick e expiram;
//! 2. `regen` — HP/MP sobem pela taxa acumulada;
//! 3. `attacks` — golpes resolvidos em ordem crescente de `entity` do atacante;
//! 4. `threat` — ameaça do alvo é atualizada com o dano aplicado;
//! 5. `deaths` — HP zero vira morte (o core decide o *despawn*);
//! 6. `events` — o que aconteceu sai por evento, nunca por I/O no meio da fase.
//!
//! Dentro de um golpe: acerto → crítico → cadeia de dano. O roll de acerto é consumido
//! **sempre**; o de crítico só quando acertou. Essa assimetria é o que faz dois replays com a
//! mesma seed coincidirem — consumir roll em golpe perdido desalinha o fluxo inteiro.

use crate::formula::attributes::{self, AttrPoints, AttrWeight, GearBonus, GrowthProfile};
use crate::formula::damage::{damage_chain, DamageInput};
use crate::formula::hit::{crit_bps, crit_chance_bps, crits, hit_chance_bps, hits};
use crate::formula::resist::{Element, ResistProfile};
use crate::roll::Die;
use crate::units::{Bps, Tenths};

/// Fases do tick, na ordem executada.
pub const TICK_PHASES: [&str; 6] = [
    "effects_tick",
    "regen",
    "attacks",
    "threat",
    "deaths",
    "events",
];

/// Combatente com o que o combate precisa — derivados já calculados por `attributes`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Combatant {
    /// Índice da entidade (desempate determinístico).
    pub entity: u32,
    /// Nível.
    pub level: u16,
    /// Pontos de atributo.
    pub attrs: AttrPoints,
    /// Conversão de atributo em derivado.
    pub weight: AttrWeight,
    /// Crescimento por nível.
    pub growth: GrowthProfile,
    /// Soma de equipamento.
    pub gear: GearBonus,
    /// Absorção plana (gema de garnet, montaria).
    pub absorb: u32,
    /// Acurácia em bps (0 = neutro).
    pub accuracy_bps: Bps,
    /// Evasão em bps.
    pub evasion_bps: Bps,
    /// Crítico do item em décimos (`7.0` = `70`).
    pub crit_tenths: u32,
    /// Modificador de chance de crítico (passivas).
    pub crit_chance_mod_bps: Bps,
    /// Resistência a crítico (Concha Resistente).
    pub crit_resist_bps: Bps,
    /// Resistência elemental.
    pub resist: ResistProfile,
}

impl Combatant {
    /// Combatente neutro no nível pedido (todos os modificadores na identidade).
    #[must_use]
    pub fn of_level(entity: u32, level: u16) -> Self {
        Self {
            entity,
            level,
            attrs: AttrPoints::ZERO,
            weight: AttrWeight::ZERO,
            growth: GrowthProfile::default(),
            gear: GearBonus::default(),
            absorb: 0,
            accuracy_bps: Bps::ZERO,
            evasion_bps: Bps::ZERO,
            crit_tenths: 0,
            crit_chance_mod_bps: Bps::FULL,
            crit_resist_bps: Bps::ZERO,
            resist: ResistProfile::default(),
        }
    }

    /// Ataque já composto (nível + atributo + equipamento).
    #[must_use]
    pub fn attack_rating(self) -> u32 {
        attributes::derive(self.level, self.attrs, self.gear, self.growth).attack
    }

    /// Defesa já composta.
    #[must_use]
    pub fn defense_rating(self) -> u32 {
        attributes::derive(self.level, self.attrs, self.gear, self.growth).defense
    }

    /// Chance de crítico deste combatente contra o alvo.
    #[must_use]
    pub fn crit_chance(self, target: Self) -> Bps {
        crit_chance_bps(
            crit_bps(Tenths::new(self.crit_tenths)),
            self.crit_chance_mod_bps,
            target.crit_resist_bps,
        )
    }
}

/// Perfil do golpe desferido.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct AttackProfile {
    /// Dano base da arma/skill.
    pub base: u32,
    /// Poder da skill em bps.
    pub skill_bps: Bps,
    /// Buffs percentuais em bps.
    pub buff_bps: Bps,
    /// Multiplicador de crítico em bps (`20_000` = 2×).
    pub crit_multiplier_bps: Bps,
    /// Perfuração.
    pub pierce: u32,
    /// Elemento do golpe.
    pub element: Element,
}

impl AttackProfile {
    /// Golpe físico neutro: só o dano base.
    #[must_use]
    pub fn new(base: u32) -> Self {
        Self {
            base,
            skill_bps: Bps::FULL,
            buff_bps: Bps::FULL,
            crit_multiplier_bps: Bps::from_percent(200),
            pierce: 0,
            element: Element::Physical,
        }
    }
}

/// Resultado de um golpe.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct AttackOutcome {
    /// Acertou?
    pub hit: bool,
    /// Foi crítico?
    pub crit: bool,
    /// Dano aplicado (0 quando errou).
    pub damage: u32,
}

/// Resolve um golpe consumindo exatamente dois rolls quando acerta e um quando erra.
pub fn resolve_attack(
    attacker: &Combatant,
    defender: &Combatant,
    attack: &AttackProfile,
    die: &mut impl Die,
) -> AttackOutcome {
    let chance_acerto = hit_chance_bps(attacker.accuracy_bps, defender.evasion_bps);
    if !hits(die.roll_bps(), chance_acerto) {
        return AttackOutcome {
            hit: false,
            crit: false,
            damage: 0,
        };
    }

    let critico = crits(die.roll_bps(), attacker.crit_chance(*defender));
    let entrada = DamageInput {
        base: attack.base,
        // `attack_rating` já soma nível, atributo e equipamento; a cadeia continua aditiva.
        attribute: attacker.attack_rating(),
        gear_attack: 0,
        skill_bps: attack.skill_bps,
        buff_bps: attack.buff_bps,
        crit_multiplier_bps: if critico {
            attack.crit_multiplier_bps
        } else {
            Bps::FULL
        },
        pierce: attack.pierce,
        defense: defender.defense_rating(),
        resist_bps: defender.resist.of(attack.element),
        absorb: defender.absorb,
    };

    AttackOutcome {
        hit: true,
        crit: critico,
        damage: damage_chain(&entrada),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::roll::XorShift64;

    #[test]
    fn tick_phases_are_the_documented_order() {
        assert_eq!(
            TICK_PHASES,
            [
                "effects_tick",
                "regen",
                "attacks",
                "threat",
                "deaths",
                "events",
            ]
        );
    }

    #[test]
    fn full_accuracy_and_no_crit_deals_the_full_chain() {
        let atacante = Combatant {
            gear: GearBonus {
                attack: 50,
                ..GearBonus::default()
            },
            ..Combatant::of_level(1, 1)
        };
        let alvo = Combatant {
            gear: GearBonus {
                defense: 20,
                ..GearBonus::default()
            },
            ..Combatant::of_level(2, 1)
        };
        let mut die = XorShift64::from_seed(1);
        let golpe = AttackProfile::new(100);
        let resultado = resolve_attack(&atacante, &alvo, &golpe, &mut die);
        assert!(resultado.hit);
        assert!(!resultado.crit, "crítico zero nunca sai");
        // (100 base + 50 ataque) − 20 defesa = 130.
        assert_eq!(resultado.damage, 130);
    }

    #[test]
    fn impossible_hit_consumes_one_roll_and_deals_nothing() {
        let atacante = Combatant::of_level(1, 1);
        let alvo = Combatant {
            evasion_bps: Bps::FULL,
            ..Combatant::of_level(2, 1)
        };
        let mut die = XorShift64::from_seed(9);
        let resultado = resolve_attack(&atacante, &alvo, &AttackProfile::new(100), &mut die);
        assert!(!resultado.hit);
        assert_eq!(resultado.damage, 0);
    }

    #[test]
    fn elemental_resist_applies_only_to_its_element() {
        let atacante = Combatant::of_level(1, 1);
        let alvo = Combatant {
            resist: ResistProfile {
                fire_bps: Bps::from_percent(50),
                frost_bps: Bps::ZERO,
            },
            ..Combatant::of_level(2, 1)
        };
        let mut die = XorShift64::from_seed(3);
        let fisico = resolve_attack(&atacante, &alvo, &AttackProfile::new(100), &mut die);
        let fogo = resolve_attack(
            &atacante,
            &alvo,
            &AttackProfile {
                element: Element::Fire,
                ..AttackProfile::new(100)
            },
            &mut die,
        );
        assert_eq!(fisico.damage, 100);
        assert_eq!(fogo.damage, 50);
    }
}
