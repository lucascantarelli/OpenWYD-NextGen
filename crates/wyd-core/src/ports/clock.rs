//! Tempo injetável: o tick lê `Clock`, o teste injeta determinismo.
//!
//! Produção usa [`SystemClock`] (relógio do SO); teste usa [`ManualClock`]
//! (tempo sob controle do script). O scheduler continua puro: recebe ms,
//! nunca o relógio.

use core::cell::Cell;

/// Fonte de tempo em milissegundos (trait pequeno, SOLID-I).
pub trait Clock {
    /// Milissegundos desde uma origem arbitrária porém monótona.
    fn now_ms(&self) -> u64;
}

/// Relógio do sistema (produção; saturado na origem para nunca panicar).
#[derive(Debug, Clone, Copy, Default)]
pub struct SystemClock;

impl Clock for SystemClock {
    fn now_ms(&self) -> u64 {
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map(|d| d.as_millis() as u64)
            .unwrap_or(0)
    }
}

/// Relógio manual (teste/determinismo): só anda via `advance`.
#[derive(Debug, Default)]
pub struct ManualClock {
    now: Cell<u64>,
}

impl ManualClock {
    #[must_use]
    pub fn at(ms: u64) -> Self {
        Self { now: Cell::new(ms) }
    }

    /// Avança o tempo manual; saturado (nunca volta, nunca estoura).
    pub fn advance(&self, ms: u64) {
        self.now.set(self.now.get().saturating_add(ms));
    }
}

impl Clock for ManualClock {
    fn now_ms(&self) -> u64 {
        self.now.get()
    }
}

/// Mede `elapsed` entre duas leituras, recusando volta no tempo.
///
/// Fora do scheduler: o tick nunca recebe `elapsed` negativo disfarçado.
pub fn elapsed_ms(clock: &impl Clock, before_ms: u64) -> Result<u64, crate::error::CoreError> {
    let now = clock.now_ms();
    now.checked_sub(before_ms)
        .ok_or(crate::error::CoreError::TimeWentBackwards {
            before_ms,
            now_ms: now,
        })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn manual_clock_only_moves_forward() {
        let c = ManualClock::at(1000);
        c.advance(50);
        assert_eq!(c.now_ms(), 1050);
        assert_eq!(elapsed_ms(&c, 1000).expect("para frente"), 50);
        assert_eq!(
            elapsed_ms(&c, 2000),
            Err(crate::error::CoreError::TimeWentBackwards {
                before_ms: 2000,
                now_ms: 1050
            })
        );
    }
}
