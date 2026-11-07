//! Tick em frequência fixa: acumulador com clamp, sem I/O, sem lock, sem alloc.
//!
//! Ordem canônica dos sistemas por tick (documentada aqui; a implementação de
//! cada sistema chega nas issues indicadas):
//!
//! 1. `Movement` (03-2) — integra posição, atualiza a grade espacial.
//! 2. `Combat` (03-5 corpo, FASE-10 fórmula) — aplica dano/aprovado, nunca inventado.
//! 3. `Visibility` (03-2/03-3) — publica delta de cena com `ESCENE_FIELD`.
//!
//! O relógio é lido pelo chamador via porta [`crate::ports::Clock`]; o
//! scheduler recebe só `elapsed_ms`, o que o mantém puro e determinístico.
//! `u64` de milissegundos não estoura na prática (>500 mil anos); o avanço é
//! saturado mesmo assim.

use crate::error::CoreError;

/// Frequências contratadas (Hz). Fora disso o boot recusa (`BadTickHz`).
pub const MIN_HZ: u32 = 20;
pub const MAX_HZ: u32 = 50;

/// Teto de passos por chamada: impede espiral da morte após stall (GC/pausa da VM).
pub const MAX_STEPS_PER_ADVANCE: u64 = 5;

/// Contador de ticks — newtype para não confundir com milissegundo.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Tick(u64);

impl Tick {
    #[must_use]
    pub fn get(self) -> u64 {
        self.0
    }
}

/// Configuração estática do loop (validada no boot, nunca default silencioso).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct SchedulerConfig {
    pub hz: u32,
}

impl SchedulerConfig {
    pub fn validated(hz: u32) -> Result<Self, CoreError> {
        if !(MIN_HZ..=MAX_HZ).contains(&hz) {
            return Err(CoreError::BadTickHz { hz });
        }
        Ok(Self { hz })
    }

    /// Intervalo inteiro em ms (trunca fração — determinístico, documentado).
    #[must_use]
    pub fn interval_ms(self) -> u64 {
        1000 / u64::from(self.hz)
    }
}

/// Acumulador de tempo real → passos de simulação em intervalo fixo.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Scheduler {
    tick: Tick,
    acc_ms: u64,
    interval_ms: u64,
}

impl Scheduler {
    pub fn new(config: SchedulerConfig) -> Self {
        Self {
            tick: Tick(0),
            acc_ms: 0,
            interval_ms: config.interval_ms(),
        }
    }

    #[must_use]
    pub fn tick(self) -> Tick {
        self.tick
    }

    /// Avança o acumulador; devolve quantos ticks fixos ficaram devidos
    /// (0..=MAX_STEPS_PER_ADVANCE). Excedente é descartado (clamp), não acumulado.
    pub fn advance(&mut self, elapsed_ms: u64) -> u64 {
        self.acc_ms = self.acc_ms.saturating_add(elapsed_ms);
        let mut steps = 0;
        while self.acc_ms >= self.interval_ms && steps < MAX_STEPS_PER_ADVANCE {
            self.acc_ms -= self.interval_ms;
            self.tick.0 += 1;
            steps += 1;
        }
        if self.acc_ms >= self.interval_ms {
            // Stall longo: descarta o atraso em vez de tentar recuperar.
            self.acc_ms = 0;
        }
        steps
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn config_rejects_out_of_contract_hz() {
        assert!(SchedulerConfig::validated(19).is_err());
        assert!(SchedulerConfig::validated(51).is_err());
        assert_eq!(
            SchedulerConfig::validated(19),
            Err(CoreError::BadTickHz { hz: 19 })
        );
        let cfg = SchedulerConfig::validated(20).expect("20 Hz válido");
        assert_eq!(cfg.interval_ms(), 50);
    }

    #[test]
    fn fixed_interval_accumulates_remainder() {
        let mut s = Scheduler::new(SchedulerConfig::validated(20).expect("válido"));
        assert_eq!(s.advance(30), 0);
        assert_eq!(s.advance(30), 1); // 60 ms = 1 tick + 10 ms de resto
        assert_eq!(s.advance(40), 1); // 10 + 40 = 50 ms = 1 tick exato
        assert_eq!(s.tick().get(), 2);
    }

    #[test]
    fn long_stall_is_clamped_not_repaid() {
        let mut s = Scheduler::new(SchedulerConfig::validated(20).expect("válido"));
        assert_eq!(s.advance(10_000), MAX_STEPS_PER_ADVANCE);
        assert_eq!(s.advance(0), 0); // atraso descartado, sem dívida
    }
}
