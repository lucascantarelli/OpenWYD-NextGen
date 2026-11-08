//! Métricas do motor: contadores atômicos, histograma de tick e exporter Prometheus.
//!
//! Sem dependência de crate de métricas: o contrato que importa é o **formato de texto
//! Prometheus**, e o registro cabe em atomics `Relaxed` (sem lock, sem alocação no quente).
//! Decisão e custo em `docs/arquitetura/adr/0018-telemetria-tracing.md`.
//!
//! Só declara e re-exporta; registro, histograma, memória e servidor moram cada um no seu arquivo.

pub mod exporter;
pub mod mem;
pub mod registry;

pub use exporter::{MetricsServer, MetricsSource};
pub use registry::{HistogramSnapshot, Metrics, TickSnapshot, TICK_BUCKETS_US};
