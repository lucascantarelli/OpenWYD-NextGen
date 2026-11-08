//! Drop por item — um roll por linha, escala 1..1.000.000.
//!
//! **CONFIRMADO** (`docs/wyd/sistemas/drops.md` §1): cada item do monstro sorteia um número
//! de 1 a 1.000.000 e cai se `roll <= rate`. O erro clássico é fazer **um** roll e escolher
//! item ponderado — distribuição "mais justa" e errada. O módulo só implementa a regra certa:
//! é o chamador quem itera as linhas chamando [`drops`] uma vez por item.
//!
//! **Hipótese (D-08):** bônus de taxa (gema de diamante, +8%) entra como percentual sobre a
//! taxa e satura na escala do roll. A fonte diz que o bônus existe e é aplicado depois do
//! roll para bônus de canal, com teto — o teto aqui é `DROP_ROLL_MAX`, o topo da própria
//! escala, em vez de um número inventado.

use crate::error::SimError;
use crate::formula::refine::{GemEffect, GEM_DIAMOND};
use crate::units::Bps;

/// Topo da escala de drop do original (`1` = 0,0001%).
pub const DROP_ROLL_MAX: u32 = 1_000_000;

/// Bônus de drop da gema de diamante (`+8%` sobre a taxa — Kersef, *Refining +10*).
pub const GEM_DROP_BONUS: Bps = Bps::new(800);

/// Índice de catálogo da gema de diamante (reencontro entre catálogo e fórmula).
pub const DIAMOND_ITEM: u32 = GEM_DIAMOND;

/// Bônus da gema aplicável a uma taxa, se a gema for a de diamante.
#[must_use]
pub fn gem_bonus(effect: GemEffect) -> Bps {
    match effect {
        GemEffect::Drop => GEM_DROP_BONUS,
        GemEffect::Pierce | GemEffect::Exp | GemEffect::Absorb => Bps::ZERO,
    }
}

/// Taxa efetiva: `rate + rate × bônus`, limitada ao topo da escala.
#[must_use]
pub fn effective_rate(rate: u32, bonus: Bps) -> u32 {
    rate.saturating_add(bonus.apply(rate)).min(DROP_ROLL_MAX)
}

/// O item cai? `roll` precisa estar em `1..=DROP_ROLL_MAX`.
///
/// # Errors
/// [`SimError::RollOutOfRange`] fora da escala — roll zero silencioso esconderia bug de
/// gerador, então é erro tipado.
pub fn drops(rate: u32, roll: u32) -> Result<bool, SimError> {
    if !(1..=DROP_ROLL_MAX).contains(&roll) {
        return Err(SimError::RollOutOfRange {
            roll,
            max: DROP_ROLL_MAX,
        });
    }
    Ok(roll <= rate)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn roll_equal_to_rate_drops_and_one_above_does_not() {
        assert_eq!(drops(1_000, 1_000), Ok(true));
        assert_eq!(drops(1_000, 1_001), Ok(false));
        assert_eq!(drops(0, 1), Ok(false));
        assert_eq!(drops(DROP_ROLL_MAX, DROP_ROLL_MAX), Ok(true));
    }

    #[test]
    fn roll_outside_scale_is_an_error() {
        assert_eq!(
            drops(10, 0),
            Err(SimError::RollOutOfRange {
                roll: 0,
                max: DROP_ROLL_MAX
            })
        );
        assert_eq!(
            drops(10, DROP_ROLL_MAX + 1),
            Err(SimError::RollOutOfRange {
                roll: DROP_ROLL_MAX + 1,
                max: DROP_ROLL_MAX
            })
        );
    }

    #[test]
    fn diamond_gem_raises_rate_by_eight_percent_and_caps() {
        assert_eq!(effective_rate(10_000, GEM_DROP_BONUS), 10_800);
        assert_eq!(effective_rate(1, GEM_DROP_BONUS), 1, "8% de 1 trunca em 0");
        assert_eq!(effective_rate(DROP_ROLL_MAX, GEM_DROP_BONUS), DROP_ROLL_MAX);
    }

    #[test]
    fn only_diamond_carries_the_drop_bonus() {
        assert_eq!(gem_bonus(GemEffect::Drop).raw(), 800);
        assert_eq!(gem_bonus(GemEffect::Pierce), Bps::ZERO);
        assert_eq!(gem_bonus(GemEffect::Exp), Bps::ZERO);
        assert_eq!(gem_bonus(GemEffect::Absorb), Bps::ZERO);
        assert_eq!(DIAMOND_ITEM, 3386);
    }
}
