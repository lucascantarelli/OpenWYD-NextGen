//! Métricas do gateway (ADR-0022): contadores atômicos, sem dependência do Hot Path.
//!
//! O Cold Path não importa `wyd-core` (ADR-0001): repetir aqui um renderizador de 30 linhas é
//! mais barato — e mais honesto — que arrastar o motor para dentro do processo administrativo.
//! Os nomes são os declarados na ADR-0022 §4 e consumidos pelas regras da FASE-12-1.

use std::sync::atomic::{AtomicU64, Ordering};
use std::time::Instant;

/// Contadores do processo gateway, em `Relaxed` (nenhuma ordenação é observável aqui).
#[derive(Debug)]
pub struct GatewayMetrics {
    ok: AtomicU64,
    business: AtomicU64,
    transport: AtomicU64,
    unauthorized: AtomicU64,
    started: Instant,
}

impl Default for GatewayMetrics {
    fn default() -> Self {
        Self::new()
    }
}

impl GatewayMetrics {
    #[must_use]
    pub fn new() -> Self {
        Self {
            ok: AtomicU64::new(0),
            business: AtomicU64::new(0),
            transport: AtomicU64::new(0),
            unauthorized: AtomicU64::new(0),
            started: Instant::now(),
        }
    }

    /// Classifica a resposta: `200` com `result == 1` é `ok`; `200` com outro `result` é
    /// `business`; `401` é `unauthorized`; todo o resto é `transport`.
    pub fn record(&self, status: u16, result: Option<u8>) {
        let counter = match (status, result) {
            (200, Some(1)) => &self.ok,
            (200, _) => &self.business,
            (401, _) => &self.unauthorized,
            _ => &self.transport,
        };
        counter.fetch_add(1, Ordering::Relaxed);
    }

    /// Texto Prometheus 0.0.4 (mesmo contrato do exporter da FASE-06-1).
    #[must_use]
    pub fn render(&self) -> String {
        let mut out = String::with_capacity(512);
        out.push_str("# HELP wyd_gateway_requests_total Requisicoes atendidas pelo gateway, por resultado.\n");
        out.push_str("# TYPE wyd_gateway_requests_total counter\n");
        for (rotulo, valor) in [
            ("ok", self.ok.load(Ordering::Relaxed)),
            ("business", self.business.load(Ordering::Relaxed)),
            ("transport", self.transport.load(Ordering::Relaxed)),
            ("unauthorized", self.unauthorized.load(Ordering::Relaxed)),
        ] {
            out.push_str(&format!(
                "wyd_gateway_requests_total{{result=\"{rotulo}\"}} {valor}\n"
            ));
        }
        out.push_str("# HELP wyd_gateway_uptime_seconds Segundos desde o start do gateway.\n");
        out.push_str("# TYPE wyd_gateway_uptime_seconds gauge\n");
        out.push_str(&format!(
            "wyd_gateway_uptime_seconds {}\n",
            self.started.elapsed().as_secs()
        ));
        out
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn classifica_por_status_e_result() {
        let m = GatewayMetrics::new();
        m.record(200, Some(1));
        m.record(200, Some(3));
        m.record(401, None);
        m.record(500, None);
        let texto = m.render();
        assert!(texto.contains("wyd_gateway_requests_total{result=\"ok\"} 1"));
        assert!(texto.contains("wyd_gateway_requests_total{result=\"business\"} 1"));
        assert!(texto.contains("wyd_gateway_requests_total{result=\"unauthorized\"} 1"));
        assert!(texto.contains("wyd_gateway_requests_total{result=\"transport\"} 1"));
        assert!(texto.contains("# TYPE wyd_gateway_uptime_seconds gauge"));
    }
}
