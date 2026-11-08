//! Regeneração de HP/MP — taxa por segundo distribuída em ticks sem perder fração.
//!
//! **Fato:** existe regeneração contínua e existe efeito que recupera por segundo
//! (`Another Change`: 2.000 HP/s; `docs/wyd/classes.md` §2). A taxa-base do personagem não é
//! publicada — por isso é entrada, não constante.
//!
//! O detalhe que evita divergência cliente×servidor: taxa por segundo **não** divide redondo
//! por tick. Guardar o resto em [`RegenAccumulator`] garante que 20 ticks de 2.000/s entregam
//! exatamente 2.000 HP, sem "1 HP perdido por tick" (o bug que só aparece em combate longo).

use crate::error::SimError;

/// Resto inteiro da divisão por tick — carregado entre ticks.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct RegenAccumulator {
    carry: u32,
}

impl RegenAccumulator {
    /// Resto acumulado (visível para auditoria/determinismo).
    #[must_use]
    pub const fn carry(self) -> u32 {
        self.carry
    }
}

/// Quantidade a aplicar neste tick, dada a taxa por segundo.
///
/// # Errors
/// [`SimError::ZeroTickRate`] quando a cadência é zero.
pub fn per_tick(
    rate_per_second: u32,
    tick_hz: u32,
    accumulator: &mut RegenAccumulator,
) -> Result<u32, SimError> {
    if tick_hz == 0 {
        return Err(SimError::ZeroTickRate);
    }
    let total = accumulator.carry.saturating_add(rate_per_second);
    accumulator.carry = total % tick_hz;
    Ok(total / tick_hz)
}

/// Soma regeneração respeitando o teto de HP/MP (nunca ultrapassa o máximo).
#[must_use]
pub fn apply_to(current: u32, max: u32, amount: u32) -> u32 {
    current.saturating_add(amount).min(max)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn no_fraction_is_lost_over_a_second() {
        let mut acc = RegenAccumulator::default();
        let total: u32 = (0..20)
            .map(|_| per_tick(2_000, 20, &mut acc).expect("cadência válida"))
            .sum();
        assert_eq!(total, 2_000);
        assert_eq!(acc.carry(), 0);
    }

    #[test]
    fn indivisible_rate_carries_instead_of_dropping() {
        let mut acc = RegenAccumulator::default();
        // 7 HP/s a 20 Hz = 0,35 por tick: 20 ticks têm de dar 7, não 0.
        let total: u32 = (0..20)
            .map(|_| per_tick(7, 20, &mut acc).expect("cadência válida"))
            .sum();
        assert_eq!(total, 7);
    }

    #[test]
    fn zero_tick_rate_is_an_error() {
        let mut acc = RegenAccumulator::default();
        assert_eq!(per_tick(10, 0, &mut acc), Err(SimError::ZeroTickRate));
    }

    #[test]
    fn regeneration_stops_at_max() {
        assert_eq!(apply_to(90, 100, 50), 100);
        assert_eq!(apply_to(10, 100, 5), 15);
        assert_eq!(apply_to(100, 100, 0), 100);
    }
}
