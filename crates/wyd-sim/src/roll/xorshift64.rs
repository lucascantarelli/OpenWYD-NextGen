//! xorshift64\* — gerador mínimo, determinístico e auditável (sem dependência externa).
//!
//! Escolha deliberada (ADR-0011): o laboratório precisa de reprodutibilidade bit-a-bit, não
//! de qualidade criptográfica. Doze linhas de estado visível valem mais que uma crate nova
//! quando o requisito é "mesma seed → mesma sequência".

use crate::error::SimError;
use crate::roll::Die;

/// Estado inicial usado quando a seed pedida é `0` (xorshift tem ponto fixo em zero).
pub const DEFAULT_SEED: u64 = 0x9E37_79B9_7F4A_7C15;

/// Multiplicador final do xorshift64\*.
const MULTIPLIER: u64 = 0x2545_F491_4F6C_DD1D;

/// Gerador com estado explícito: `from_seed(s).state()` é a assinatura do replay.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct XorShift64 {
    state: u64,
}

impl XorShift64 {
    /// Cria o gerador; `seed = 0` é trocada por [`DEFAULT_SEED`] para não travar em zero.
    #[must_use]
    pub const fn from_seed(seed: u64) -> Self {
        Self {
            state: if seed == 0 { DEFAULT_SEED } else { seed },
        }
    }

    /// Estado atual — gravar isto (ou a seed) é o que torna o replay auditável.
    #[must_use]
    pub const fn state(&self) -> u64 {
        self.state
    }

    /// Próximo `u64` da sequência (xorshift64\*).
    pub fn next_u64(&mut self) -> u64 {
        let mut x = self.state;
        x ^= x << 13;
        x ^= x >> 7;
        x ^= x << 17;
        self.state = x;
        x.wrapping_mul(MULTIPLIER)
    }
}

impl Die for XorShift64 {
    fn roll_1_to(&mut self, max: u32) -> Result<u32, SimError> {
        if max == 0 {
            return Err(SimError::RollOutOfRange { roll: 0, max });
        }
        let limite = u64::from(max);
        let sorteado = self.next_u64() % limite;
        Ok(u32::try_from(sorteado + 1).unwrap_or(max))
    }

    fn roll_bps(&mut self) -> u32 {
        // Viés de módulo em 2^64 é desprezível para o laboratório e, sobretudo,
        // é determinístico: 10_000 não divide 2^64, e o desvio é < 1e-15.
        let sorteado = self.next_u64() % u64::from(super::CHANCE_SCALE);
        u32::try_from(sorteado).unwrap_or(super::CHANCE_SCALE - 1)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn zero_seed_falls_back_to_default() {
        assert_eq!(
            XorShift64::from_seed(0).state(),
            XorShift64::from_seed(DEFAULT_SEED).state()
        );
    }

    #[test]
    fn same_seed_same_sequence() {
        let mut a = XorShift64::from_seed(42);
        let mut b = XorShift64::from_seed(42);
        let seq_a: Vec<u64> = (0..32).map(|_| a.next_u64()).collect();
        let seq_b: Vec<u64> = (0..32).map(|_| b.next_u64()).collect();
        assert_eq!(seq_a, seq_b);
        assert_eq!(a.state(), b.state());
    }

    #[test]
    fn different_seeds_diverge() {
        assert_ne!(
            XorShift64::from_seed(1).next_u64(),
            XorShift64::from_seed(2).next_u64()
        );
    }

    #[test]
    fn rolls_stay_inside_their_scale() {
        let mut die = XorShift64::from_seed(7);
        for _ in 0..1_000 {
            let bps = die.roll_bps();
            assert!(bps < 10_000, "bps {bps}");
            let drop = die.roll_1_to(1_000_000).expect("max não é zero");
            assert!((1..=1_000_000).contains(&drop), "drop {drop}");
        }
    }

    #[test]
    fn zero_max_is_an_error_not_a_panic() {
        let mut die = XorShift64::from_seed(1);
        assert_eq!(
            die.roll_1_to(0),
            Err(SimError::RollOutOfRange { roll: 0, max: 0 })
        );
    }
}
