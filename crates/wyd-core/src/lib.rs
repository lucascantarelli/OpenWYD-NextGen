//! Engine Hot Path (esqueleto).
//!
//! Um unico processo por no de mundo, tick em frequencia fixa, sem GC e sem
//! mutex global no estado de jogo (detalhes em `pre-release-rust.md` FASE 3).

/// Ticks por segundo do game loop deterministico.
pub const TICK_HZ: u32 = 20;

/// Intervalo entre ticks em milissegundos.
#[must_use]
pub fn tick_interval_ms() -> u64 {
    1000 / u64::from(TICK_HZ)
}

/// Reexporta o marcador autoritativo de cena para o game loop.
pub use wyd_protocol::ESCENE_FIELD;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tick_interval_matches_20hz() {
        assert_eq!(TICK_HZ, 20);
        assert_eq!(tick_interval_ms(), 50);
    }
}
