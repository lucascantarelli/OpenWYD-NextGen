//! Contadores de rede do transporte — atomics `Relaxed`, sem lock e sem alocação.
//!
//! Fronteira: `wyd-net` não depende de `wyd-core` (o DAG é protocolo ← net ← core binário),
//! então não compartilha o registro do motor. Cada crate publica seu **fragmento** de texto
//! Prometheus e o binário concatena os dois no `/metrics` (ver ADR-0018). O exporter HTTP do
//! core já aceita fragmento (`Metrics::render(extra)`).
//!
//! Instrumentação só nos pontos de decisão (consumo do balde, porte de handshake): 1 fetch_add
//! relaxed por operação, medido em `benches/limits.rs`.

use core::sync::atomic::{AtomicU64, Ordering};

/// Contadores do processo (único por processo, como o registro do motor).
///
/// Alinhado a uma linha de cache: os campos são escritos pelo mesmo thread no caminho
/// quente, mas o `static` não deve dividir linha com outros dados globais (false sharing).
#[derive(Debug, Default)]
#[repr(align(64))]
pub struct NetMetrics {
    packets: AtomicU64,
    rejected: AtomicU64,
    rejected_rate: AtomicU64,
    rejected_ip: AtomicU64,
    rejected_total: AtomicU64,
    rejected_timeout: AtomicU64,
    handshakes: AtomicU64,
    pending: AtomicU64,
    sessions: AtomicU64,
}

/// Instância global do processo.
pub static NET: NetMetrics = NetMetrics::new();

impl NetMetrics {
    /// Registro zerado (const para o `static`).
    #[must_use]
    pub const fn new() -> Self {
        Self {
            packets: AtomicU64::new(0),
            rejected: AtomicU64::new(0),
            rejected_rate: AtomicU64::new(0),
            rejected_ip: AtomicU64::new(0),
            rejected_total: AtomicU64::new(0),
            rejected_timeout: AtomicU64::new(0),
            handshakes: AtomicU64::new(0),
            pending: AtomicU64::new(0),
            sessions: AtomicU64::new(0),
        }
    }

    /// Pacote aceito pelo balde.
    #[inline]
    pub fn packet_accepted(&self) {
        #[cfg(feature = "metrics")]
        self.packets.fetch_add(1, Ordering::Relaxed);
    }

    /// Pacote recusado; `reason` espelha `Deny` (`rate`, `ip`, `total`, `timeout`).
    #[inline]
    pub fn packet_rejected(&self, reason: &str) {
        #[cfg(feature = "metrics")]
        {
            self.rejected.fetch_add(1, Ordering::Relaxed);
            match reason {
                "rate" => self.rejected_rate.fetch_add(1, Ordering::Relaxed),
                "ip" => self.rejected_ip.fetch_add(1, Ordering::Relaxed),
                "total" => self.rejected_total.fetch_add(1, Ordering::Relaxed),
                _ => self.rejected_timeout.fetch_add(1, Ordering::Relaxed),
            };
        }
        #[cfg(not(feature = "metrics"))]
        let _ = reason;
    }

    /// Handshake admitido.
    #[inline]
    pub fn handshake_opened(&self) {
        #[cfg(feature = "metrics")]
        self.handshakes.fetch_add(1, Ordering::Relaxed);
    }

    /// Handshake pendente entrou (+1) ou saiu (−1); satura em zero.
    #[inline]
    pub fn pending_delta(&self, delta: i64) {
        #[cfg(not(feature = "metrics"))]
        let _ = delta;
        #[cfg(feature = "metrics")]
        if delta >= 0 {
            self.pending
                .fetch_add(delta.unsigned_abs(), Ordering::Relaxed);
        } else {
            let _ = self
                .pending
                .try_update(Ordering::Relaxed, Ordering::Relaxed, |atual| {
                    Some(atual.saturating_sub(delta.unsigned_abs()))
                });
        }
    }

    /// Conexão aceita no total (+1) ou liberada (−1); satura em zero.
    #[inline]
    pub fn session_delta(&self, delta: i64) {
        #[cfg(not(feature = "metrics"))]
        let _ = delta;
        #[cfg(feature = "metrics")]
        if delta >= 0 {
            self.sessions
                .fetch_add(delta.unsigned_abs(), Ordering::Relaxed);
        } else {
            let _ = self
                .sessions
                .try_update(Ordering::Relaxed, Ordering::Relaxed, |atual| {
                    Some(atual.saturating_sub(delta.unsigned_abs()))
                });
        }
    }

    /// Pacotes contabilizados.
    #[must_use]
    pub fn packets(&self) -> u64 {
        self.packets.load(Ordering::Relaxed)
    }

    /// Recusas contabilizadas (todas as razões).
    #[must_use]
    pub fn rejected(&self) -> u64 {
        self.rejected.load(Ordering::Relaxed)
    }

    /// Fragmento Prometheus deste crate.
    #[must_use]
    pub fn render_prometheus(&self) -> String {
        let mut out = String::with_capacity(512);
        out.push_str("# HELP wyd_packets_total Pacotes aceitos pelo balde de tokens.\n");
        out.push_str("# TYPE wyd_packets_total counter\n");
        out.push_str(&format!("wyd_packets_total {}\n", self.packets()));
        out.push_str("# HELP wyd_packets_rejected_total Pacotes recusados, por motivo.\n");
        out.push_str("# TYPE wyd_packets_rejected_total counter\n");
        for (motivo, valor) in [
            ("rate", &self.rejected_rate),
            ("ip", &self.rejected_ip),
            ("total", &self.rejected_total),
            ("timeout", &self.rejected_timeout),
        ] {
            out.push_str(&format!(
                "wyd_packets_rejected_total{{reason=\"{motivo}\"}} {}\n",
                valor.load(Ordering::Relaxed)
            ));
        }
        out.push_str("# HELP wyd_handshakes_total Handshakes admitidos.\n");
        out.push_str("# TYPE wyd_handshakes_total counter\n");
        out.push_str(&format!(
            "wyd_handshakes_total {}\n",
            self.handshakes.load(Ordering::Relaxed)
        ));
        out.push_str("# HELP wyd_connections_pending Handshakes pendentes.\n");
        out.push_str("# TYPE wyd_connections_pending gauge\n");
        out.push_str(&format!(
            "wyd_connections_pending {}\n",
            self.pending.load(Ordering::Relaxed)
        ));
        out.push_str("# HELP wyd_sessions_open Conexoes abertas (gauge).\n");
        out.push_str("# TYPE wyd_sessions_open gauge\n");
        out.push_str(&format!(
            "wyd_sessions_open {}\n",
            self.sessions.load(Ordering::Relaxed)
        ));
        out
    }
}

#[cfg(all(test, feature = "metrics"))]
mod tests {
    use super::*;

    #[test]
    fn contadores_sobem_e_saturam_em_zero() {
        let m = NetMetrics::new();
        m.packet_accepted();
        m.packet_rejected("rate");
        m.packet_rejected("ip");
        assert_eq!(m.packets(), 1);
        assert_eq!(m.rejected(), 2);
        let texto = m.render_prometheus();
        assert!(texto.contains("wyd_packets_total 1"));
        assert!(texto.contains("wyd_packets_rejected_total{reason=\"rate\"} 1"));
        assert!(texto.contains("wyd_packets_rejected_total{reason=\"ip\"} 1"));
        m.session_delta(-5);
        assert!(texto.contains("wyd_sessions_open 0"));
    }
}
