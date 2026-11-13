//! Acerto, evasão e crítico — duas chances independentes, dois rolls.
//!
//! **Fato:** evasão existe e é percentual (Clod Attack reduz 10% de evasão por 64 s;
//! `docs/wyd/classes.md` §2), crítico existe e item carrega grandeza decimal (`7.0` em
//! `docs/wyd/itens.md` l. 149), e há passivas que mexem na **chance** de levar crítico
//! (Concha Resistente) e no **dano** de crítico (Claw Skillful, +30%).
//!
//! **Hipótese (D-06):** o valor decimal do item é décimos de ponto percentual, então
//! `7.0` = 70 décimos = 700 bps = 7%. Base de crítico fora de item (nível, destreza) não é
//! publicada — fica como entrada do chamador, nunca constante escondida. O trace que
//! confirma é o log de golpe do cliente (`docs/cliente/depuracao.md`, T4).
//!
//! Metade aberta em cima: chance `10_000` sempre acerta, chance `0` nunca acerta — e o roll
//! do gerador é `< 10_000`, então não existe "acerto de fronteira" ambíguo.

use crate::units::{Bps, Tenths};

/// Quantos basis points vale um décimo de ponto percentual (`7,0%` → `700` bps).
pub const CRIT_BPS_PER_TENTH: u32 = 10;

/// Escala da chance de acerto em bps.
pub const HIT_SCALE: u32 = 10_000;

/// Converte o crítico do item (décimos) em chance percentual.
#[must_use]
pub fn crit_bps(crit: Tenths) -> Bps {
    crit.to_bps(CRIT_BPS_PER_TENTH)
}

/// Chance de acerto: `100% + acurácia − evasão`, limitada a `0..=100%`.
#[must_use]
pub fn hit_chance_bps(accuracy: Bps, evasion: Bps) -> Bps {
    let bruto = i64::from(HIT_SCALE) + i64::from(accuracy.raw()) - i64::from(evasion.raw());
    let limitado = bruto.clamp(0, i64::from(HIT_SCALE));
    Bps::new(u32::try_from(limitado).unwrap_or(0))
}

/// Chance de crítico: base do item × modificador de passiva − resistência a crítico.
///
/// `chance_mod` e `crit_resist` são entradas (passivas/buffs), não constantes: o módulo não
/// inventa quantos por cento vale uma passiva.
#[must_use]
pub fn crit_chance_bps(base: Bps, chance_mod: Bps, crit_resist: Bps) -> Bps {
    let ampliado = chance_mod.apply(base.raw());
    Bps::new(ampliado.saturating_sub(crit_resist.raw())).clamp(Bps::ZERO, Bps::FULL)
}

/// O golpe acerta? `roll` é metade aberta em cima (`0..10_000`).
#[must_use]
pub fn hits(roll: u32, chance: Bps) -> bool {
    roll < chance.raw()
}

/// O golpe é crítico? Mesma convenção de metade aberta de [`hits`].
#[must_use]
pub fn crits(roll: u32, chance: Bps) -> bool {
    roll < chance.raw()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn full_chance_always_hits_and_zero_never_does() {
        assert!(hits(0, Bps::FULL));
        assert!(hits(9_999, Bps::FULL));
        assert!(!hits(0, Bps::ZERO));
    }

    #[test]
    fn evasion_above_full_accuracy_saturates() {
        assert_eq!(hit_chance_bps(Bps::ZERO, Bps::from_percent(150)), Bps::ZERO);
        assert_eq!(
            hit_chance_bps(Bps::from_percent(20), Bps::from_percent(70)).raw(),
            5_000
        );
    }

    #[test]
    fn item_crit_is_tenths_of_a_percent() {
        assert_eq!(crit_bps(Tenths::new(70)).raw(), 700);
        assert_eq!(crit_bps(Tenths::new(35)).raw(), 350);
    }

    #[test]
    fn crit_modifier_and_resistance_compose() {
        let base = crit_bps(Tenths::new(70));
        let dobrado = crit_chance_bps(base, Bps::from_percent(200), Bps::ZERO);
        assert_eq!(dobrado.raw(), 1_400);
        let reduzido = crit_chance_bps(base, Bps::FULL, Bps::from_percent(3));
        assert_eq!(reduzido.raw(), 400);
    }

    #[test]
    fn crit_resistance_cannot_go_negative() {
        let chance = crit_chance_bps(Bps::from_percent(5), Bps::FULL, Bps::FULL);
        assert_eq!(chance, Bps::ZERO);
    }
}
