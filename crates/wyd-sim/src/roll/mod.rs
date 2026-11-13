//! Sorteio determinístico — a única fonte de aleatoriedade do laboratório.
//!
//! Fórmula **não** sorteia (princípio de `docs/rust/06-matematica.md` §1): ela recebe o roll
//! pronto. Quem sorteia é esta fronteira, com seed explícita e estado inspecionável — sem
//! relógio, sem `/dev/urandom`, sem `rand` de dependência externa.
//!
//! Replay usa o mesmo contrato: trocar [`XorShift64`] por um [`Die`] que devolve rolls
//! gravados reproduz a partida sem tocar nas fórmulas.

pub mod xorshift64;

pub use xorshift64::XorShift64;

use crate::error::SimError;

/// Escala das chances em basis points (`0..10_000`).
pub const CHANCE_SCALE: u32 = 10_000;

/// Fonte de números pseudoaleatórios determinística.
pub trait Die {
    /// Inteiro em `1..=max` (inclusivo). `max` zero é erro, não `panic!`.
    ///
    /// # Errors
    /// [`SimError::RollOutOfRange`] quando `max == 0`.
    fn roll_1_to(&mut self, max: u32) -> Result<u32, SimError>;

    /// Basis point em `0..10_000` — metade aberta em cima, então `10_000` nunca sai.
    fn roll_bps(&mut self) -> u32;
}
