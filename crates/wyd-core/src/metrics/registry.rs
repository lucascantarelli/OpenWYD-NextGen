//! Registro de métricas por atomics — sem lock, sem alocação no caminho quente.
//!
//! Os limites do histograma são fixos (microssegundos) e cobrem de 0,5 ms a 1 s: o tick de
//! 20 Hz tem orçamento de 50 ms, então 0,5 ms–1 s separa "tick saudável" de "tick perdido".
//! P50/P95/P99 são calculados por bucket (limite superior do bucket que contém o quantil),
//! o que é suficiente para operação e é o mesmo número que o Prometheus publica via
//! `histogram_quantile` sobre os buckets exportados.

use core::sync::atomic::{AtomicU64, Ordering};

/// Orçamento default do tick (20 Hz). O `Engine` sobrescreve com `1e6 / WYD_TICK_HZ`.
pub const DEFAULT_TICK_BUDGET_US: u64 = 50_000;

/// Limites superiores do histograma de duração de tick, em microssegundos.
pub const TICK_BUCKETS_US: [u64; 10] = [
    500, 1_000, 2_500, 5_000, 10_000, 25_000, 50_000, 100_000, 250_000, 1_000_000,
];

/// Histograma cumulativo de valores inteiros (buckets fixos + soma + contagem).
#[derive(Debug)]
pub struct Histogram {
    buckets: [AtomicU64; TICK_BUCKETS_US.len()],
    count: AtomicU64,
    sum: AtomicU64,
}

impl Default for Histogram {
    fn default() -> Self {
        Self {
            buckets: core::array::from_fn(|_| AtomicU64::new(0)),
            count: AtomicU64::new(0),
            sum: AtomicU64::new(0),
        }
    }
}

impl Histogram {
    /// Registra um valor em µs nos buckets cumulativos (`le` do Prometheus).
    pub fn record(&self, value_us: u64) {
        for (index, limite) in TICK_BUCKETS_US.iter().enumerate() {
            if value_us <= *limite {
                self.buckets[index].fetch_add(1, Ordering::Relaxed);
            }
        }
        self.count.fetch_add(1, Ordering::Relaxed);
        self.sum.fetch_add(value_us, Ordering::Relaxed);
    }

    /// Leitura consistente para exportação (não bloqueia o escritor).
    #[must_use]
    pub fn snapshot(&self) -> HistogramSnapshot {
        let mut buckets = [0u64; TICK_BUCKETS_US.len()];
        for (destino, origem) in buckets.iter_mut().zip(self.buckets.iter()) {
            *destino = origem.load(Ordering::Relaxed);
        }
        HistogramSnapshot {
            buckets,
            count: self.count.load(Ordering::Relaxed),
            sum: self.sum.load(Ordering::Relaxed),
        }
    }
}

/// Cópia imutável do histograma no instante do scrape.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct HistogramSnapshot {
    /// Contagem cumulativa por bucket (`le`), na ordem de [`TICK_BUCKETS_US`].
    pub buckets: [u64; TICK_BUCKETS_US.len()],
    /// Total de amostras.
    pub count: u64,
    /// Soma dos valores (µs).
    pub sum: u64,
}

impl HistogramSnapshot {
    /// Média em µs (0 quando não há amostra).
    #[must_use]
    pub fn mean(&self) -> u64 {
        self.sum.checked_div(self.count).unwrap_or(0)
    }

    /// Quantil inteiro em µs: limite superior do primeiro bucket que o contém.
    #[must_use]
    pub fn quantile(&self, percent: u32) -> u64 {
        if self.count == 0 {
            return 0;
        }
        let alvo = u64::from(percent).saturating_mul(self.count).div_ceil(100);
        for (index, limite) in TICK_BUCKETS_US.iter().enumerate() {
            if self.buckets[index] >= alvo {
                return *limite;
            }
        }
        TICK_BUCKETS_US[TICK_BUCKETS_US.len() - 1]
    }
}

/// Resumo do tick para log periódico e teste.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct TickSnapshot {
    /// Ticks medidos desde o boot do processo.
    pub count: u64,
    /// P50 (µs).
    pub p50: u64,
    /// P95 (µs).
    pub p95: u64,
    /// P99 (µs).
    pub p99: u64,
    /// Média (µs).
    pub mean: u64,
}

/// Registro central do motor — único por processo, compartilhado por referência.
#[derive(Debug, Default)]
pub struct Metrics {
    ticks: AtomicU64,
    tick_us: Histogram,
    tick_budget_us: AtomicU64,
    entities: AtomicU64,
    heap_bytes: AtomicU64,
    heap_samples: AtomicU64,
}

impl Metrics {
    /// Registro vazio, com o orçamento default de 20 Hz até o `Engine` informar o real.
    #[must_use]
    pub fn new() -> Self {
        let metrics = Self::default();
        metrics.set_tick_budget_us(DEFAULT_TICK_BUDGET_US);
        metrics
    }

    /// Orçamento do tick em µs (`1e6 / hz`), para painel e alerta compararem com o P99.
    pub fn set_tick_budget_us(&self, micros: u64) {
        self.tick_budget_us.store(micros.max(1), Ordering::Relaxed);
    }

    /// Orçamento do tick em µs.
    #[must_use]
    pub fn tick_budget_us(&self) -> u64 {
        self.tick_budget_us.load(Ordering::Relaxed)
    }

    /// Contabiliza um tick concluído. `micros` é o tempo do tick inteiro (não só a IA).
    pub fn record_tick(&self, micros: u64, entities: u64) {
        self.ticks.fetch_add(1, Ordering::Relaxed);
        self.tick_us.record(micros);
        self.entities.store(entities, Ordering::Relaxed);
    }

    /// Atualiza a gauge de entidades vivas (chamada no fim do tick).
    pub fn set_entities(&self, entities: u64) {
        self.entities.store(entities, Ordering::Relaxed);
    }

    /// Amostra de memória residente (fora do quente, período maior que o tick).
    pub fn set_heap_bytes(&self, bytes: u64) {
        self.heap_bytes.store(bytes, Ordering::Relaxed);
        self.heap_samples.fetch_add(1, Ordering::Relaxed);
    }

    /// Ticks medidos.
    #[must_use]
    pub fn ticks(&self) -> u64 {
        self.ticks.load(Ordering::Relaxed)
    }

    /// Memória residente amostrada (0 = nunca amostrada/não disponível nesta plataforma).
    #[must_use]
    pub fn heap_bytes(&self) -> u64 {
        self.heap_bytes.load(Ordering::Relaxed)
    }

    /// Amostras de memória já feitas.
    #[must_use]
    pub fn heap_samples(&self) -> u64 {
        self.heap_samples.load(Ordering::Relaxed)
    }

    /// Resumo P50/P95/P99 do tick.
    #[must_use]
    pub fn tick_snapshot(&self) -> TickSnapshot {
        let h = self.tick_us.snapshot();
        TickSnapshot {
            count: h.count,
            p50: h.quantile(50),
            p95: h.quantile(95),
            p99: h.quantile(99),
            mean: h.mean(),
        }
    }

    /// Histograma bruto (para o exporter e para teste).
    #[must_use]
    pub fn histogram(&self) -> HistogramSnapshot {
        self.tick_us.snapshot()
    }

    /// Corpo Prometheus do registro do motor, opcionalmente com fragmentos de outras crates.
    #[must_use]
    pub fn render(&self, extra: &str) -> String {
        let h = self.tick_us.snapshot();
        let mut out = String::with_capacity(1024);
        out.push_str("# HELP wyd_ticks_total Ticks processados desde o boot.\n");
        out.push_str("# TYPE wyd_ticks_total counter\n");
        out.push_str(&format!("wyd_ticks_total {}\n", self.ticks()));
        out.push_str("# HELP wyd_tick_duration_microseconds Duracao do tick inteiro.\n");
        out.push_str("# TYPE wyd_tick_duration_microseconds histogram\n");
        for (index, limite) in TICK_BUCKETS_US.iter().enumerate() {
            out.push_str(&format!(
                "wyd_tick_duration_microseconds_bucket{{le=\"{limite}\"}} {}\n",
                h.buckets[index]
            ));
        }
        out.push_str(&format!(
            "wyd_tick_duration_microseconds_bucket{{le=\"+Inf\"}} {}\n",
            h.count
        ));
        out.push_str(&format!(
            "wyd_tick_duration_microseconds_sum {}\nwyd_tick_duration_microseconds_count {}\n",
            h.sum, h.count
        ));
        out.push_str("# HELP wyd_tick_budget_microseconds Orcamento do tick em microssegundos (1e6 / WYD_TICK_HZ).\n");
        out.push_str("# TYPE wyd_tick_budget_microseconds gauge\n");
        out.push_str(&format!(
            "wyd_tick_budget_microseconds {}\n",
            self.tick_budget_us()
        ));
        out.push_str("# HELP wyd_entities_active Entidades vivas no mundo.\n");
        out.push_str("# TYPE wyd_entities_active gauge\n");
        out.push_str(&format!(
            "wyd_entities_active {}\n",
            self.entities.load(Ordering::Relaxed)
        ));
        out.push_str("# HELP process_resident_memory_bytes Memoria residente amostrada.\n");
        out.push_str("# TYPE process_resident_memory_bytes gauge\n");
        out.push_str(&format!(
            "process_resident_memory_bytes {}\n",
            self.heap_bytes()
        ));
        out.push_str("# HELP wyd_memory_samples_total Amostras de memoria coletadas.\n");
        out.push_str("# TYPE wyd_memory_samples_total counter\n");
        out.push_str(&format!(
            "wyd_memory_samples_total {}\n",
            self.heap_samples()
        ));
        if !extra.is_empty() {
            out.push_str(extra);
            if !extra.ends_with('\n') {
                out.push('\n');
            }
        }
        out
    }
}

impl crate::metrics::MetricsSource for Metrics {
    fn scrape(&self) -> String {
        self.render("")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn histograma_cumulativo_e_quantis() {
        let h = Histogram::default();
        for valor in [100, 200, 900, 1_200, 4_000] {
            h.record(valor);
        }
        let s = h.snapshot();
        assert_eq!(s.count, 5);
        assert_eq!(s.buckets[0], 2, "100/200 ≤ 500");
        assert_eq!(s.buckets[1], 3, "900 ≤ 1000");
        assert_eq!(
            *s.buckets.last().expect("bucket"),
            5,
            "cumulativo até o topo"
        );
        assert_eq!(s.quantile(50), 1_000, "mediana cai no bucket de 1 ms");
        assert_eq!(s.quantile(99), 5_000, "4000 µs cai no bucket de 5 ms");
        assert_eq!(s.mean(), 1_280);
    }

    #[test]
    fn registro_conta_ticks_e_entidades() {
        let m = Metrics::new();
        m.record_tick(1_500, 42);
        m.record_tick(2_500, 43);
        assert_eq!(m.ticks(), 2);
        let t = m.tick_snapshot();
        assert_eq!(t.count, 2);
        assert_eq!(t.p50, 2_500);
        assert_eq!(t.mean, 2_000);
        assert!(m.render("").contains("wyd_entities_active 43"));
        assert!(m
            .render("")
            .contains("wyd_tick_duration_microseconds_count 2"));
        assert_eq!(m.tick_budget_us(), DEFAULT_TICK_BUDGET_US);
        m.set_tick_budget_us(20_000);
        assert!(m.render("").contains("wyd_tick_budget_microseconds 20000"));
    }

    #[test]
    fn render_anexa_fragmento_de_outra_crate_sem_quebrar_linha() {
        let m = Metrics::new();
        let corpo = m.render("wyd_packets_total 7");
        assert!(corpo.contains("wyd_packets_total 7\n"));
        assert!(corpo.ends_with('\n'));
    }
}
