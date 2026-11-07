//! Laboratorio matematico (esqueleto).
//!
//! Simula interacoes de combate de forma deterministica para validar
//! balanceamento antes de tocar a engine.

/// Calculo placeholder deterministico: dano base menos absorcao, sem underflow.
#[must_use]
pub fn damage_after_absorb(base: u32, absorb: u32) -> u32 {
    base.saturating_sub(absorb)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn absorb_never_underflows() {
        assert_eq!(damage_after_absorb(100, 30), 70);
        assert_eq!(damage_after_absorb(10, 30), 0);
    }
}
