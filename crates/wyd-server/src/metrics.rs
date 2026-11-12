//! `GET :9898/metrics` — contrato da QA-04, sem nome inventado.
//!
//! O fragmento de rede vem de `wyd_net::NetMetrics::render_prometheus`
//! (`wyd_packets_total`, `wyd_packets_rejected_total{reason}`, `wyd_handshakes_total`,
//! `wyd_connections_pending`, `wyd_sessions_open`, `wyd_client_version_*`) — os mesmos nomes
//! que o Prometheus já consome. Este arquivo só acrescenta os gauges do serviço, declarados
//! na ADR-0022 §7: `wyd_server_uptime_seconds`, `wyd_server_channels_published`,
//! `wyd_status_requests_total`, `wyd_status_health_failures_total`.
//!
//! `wyd_ticks_total`/`wyd_tick_duration_microseconds`/`wyd_entities_active` vêm do registro do
//! motor (`wyd_core::Metrics`) da thread de tick (GAME-07): são número medido, não gauge
//! zerado. Antes do loop existir eles **não** eram publicados — publicá-los em zero seria
//! mentira (ADR-0022 §7).

use std::io::BufReader;
use std::net::TcpListener;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::{Arc, OnceLock};
use std::time::{Duration, Instant};

use wyd_admin::transport::http;

use crate::channels::World;
use crate::config::LiveConfig;

/// Contadores do processo de jogo.
#[derive(Debug)]
pub struct ServiceMetrics {
    started: Instant,
    status_requests: AtomicU64,
    health_failures: AtomicU64,
    channels: u64,
    engine: OnceLock<Arc<wyd_core::Metrics>>,
    config: OnceLock<Arc<LiveConfig>>,
}

impl ServiceMetrics {
    #[must_use]
    pub fn new(channels: usize) -> Self {
        Self {
            started: Instant::now(),
            status_requests: AtomicU64::new(0),
            health_failures: AtomicU64::new(0),
            channels: channels as u64,
            engine: OnceLock::new(),
            config: OnceLock::new(),
        }
    }

    /// Anexa o registro do motor da thread de tick (idempotente: a primeira vez vale).
    pub fn attach_engine(&self, engine: Arc<wyd_core::Metrics>) {
        let _ = self.engine.set(engine);
    }

    /// Anexa o snapshot de configuração viva (idempotente).
    pub fn attach_config(&self, config: Arc<LiveConfig>) {
        let _ = self.config.set(config);
    }

    pub fn status_served(&self) {
        self.status_requests.fetch_add(1, Ordering::Relaxed);
    }

    pub fn health_checked(&self, healthy: bool) {
        if !healthy {
            self.health_failures.fetch_add(1, Ordering::Relaxed);
        }
    }

    /// Texto Prometheus: fragmento de rede + gauges do serviço.
    #[must_use]
    pub fn render(&self) -> String {
        let mut out = wyd_net::NET.render_prometheus();
        if let Some(engine) = self.engine.get() {
            out.push_str(&engine.render(""));
        }
        if let Some(config) = self.config.get() {
            out.push_str(&config.render_metrics());
        }
        out.push_str(
            "# HELP wyd_server_uptime_seconds Segundos desde o start do servidor de jogo.\n",
        );
        out.push_str("# TYPE wyd_server_uptime_seconds gauge\n");
        out.push_str(&format!(
            "wyd_server_uptime_seconds {}\n",
            self.started.elapsed().as_secs()
        ));
        out.push_str(
            "# HELP wyd_server_channels_published Canais de jogo publicados no serv00.htm.\n",
        );
        out.push_str("# TYPE wyd_server_channels_published gauge\n");
        out.push_str(&format!(
            "wyd_server_channels_published {}\n",
            self.channels
        ));
        out.push_str("# HELP wyd_status_requests_total Leituras de serv00.htm atendidas.\n");
        out.push_str("# TYPE wyd_status_requests_total counter\n");
        out.push_str(&format!(
            "wyd_status_requests_total {}\n",
            self.status_requests.load(Ordering::Relaxed)
        ));
        out.push_str("# HELP wyd_status_health_failures_total Checks de /health que falharam.\n");
        out.push_str("# TYPE wyd_status_health_failures_total counter\n");
        out.push_str(&format!(
            "wyd_status_health_failures_total {}\n",
            self.health_failures.load(Ordering::Relaxed)
        ));
        out.push_str("# HELP wyd_ip_limit_refusals_total Conexões recusadas por limite de IP.\n");
        out.push_str("# TYPE wyd_ip_limit_refusals_total counter\n");
        out.push_str(&format!(
            "wyd_ip_limit_refusals_total {}\n",
            ip_limit_refusals()
        ));
        out
    }
}

/// Recusas por limite de IP desde o start (contador do processo de jogo).
static IP_LIMIT_REFUSALS: AtomicU64 = AtomicU64::new(0);

/// Conta uma recusa por limite de IP e devolve o total (para o log agregado).
pub fn ip_limit_refused() -> u64 {
    IP_LIMIT_REFUSALS.fetch_add(1, Ordering::Relaxed) + 1
}

/// Total de recusas por limite de IP (também no `/metrics`).
#[must_use]
pub fn ip_limit_refusals() -> u64 {
    IP_LIMIT_REFUSALS.load(Ordering::Relaxed)
}

/// Serve `/metrics` para sempre no listener de métricas.
pub fn serve(listener: TcpListener, metrics: Arc<ServiceMetrics>) {
    for aceito in listener.incoming() {
        match aceito {
            Ok(stream) => handle(stream, &metrics),
            Err(error) => {
                log(
                    "error",
                    "accept",
                    &format!("service=metrics reason={error}"),
                );
                return;
            }
        }
    }
}

fn handle(stream: std::net::TcpStream, metrics: &ServiceMetrics) {
    let _ = stream.set_read_timeout(Some(Duration::from_secs(10)));
    let _ = stream.set_write_timeout(Some(Duration::from_secs(10)));
    let Ok(read_half) = stream.try_clone() else {
        return;
    };
    let mut reader = BufReader::new(read_half);
    let mut writer = stream;
    let Ok(request) = http::read_request_any(&mut reader, 16 * 1024) else {
        return;
    };
    let path = request.path.split('?').next().unwrap_or("");
    if path == "/metrics" {
        let corpo = metrics.render();
        let _ = http::write_text_response(
            &mut writer,
            200,
            "text/plain; version=0.0.4; charset=utf-8",
            &corpo,
        );
    } else {
        let _ = http::write_text_response(
            &mut writer,
            404,
            "text/plain; charset=utf-8",
            "nao encontrado",
        );
    }
}

/// Quantos canais a foto publica (para o log de boot e o gauge).
#[must_use]
pub fn channel_gauge(world: &World) -> u64 {
    world.game_channel_count() as u64
}

fn log(level: &str, event: &str, fields: &str) {
    println!("level={level} event={event} {fields}");
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn gauges_do_servico_aparecem_no_texto() {
        let metrics = ServiceMetrics::new(2);
        metrics.status_served();
        metrics.health_checked(false);
        let texto = metrics.render();
        assert!(texto.contains("wyd_server_channels_published 2"));
        assert!(texto.contains("wyd_status_requests_total 1"));
        assert!(texto.contains("wyd_status_health_failures_total 1"));
        assert!(texto.contains("# TYPE wyd_server_uptime_seconds gauge"));
        // O fragmento de rede continua sendo a fonte dos nomes da QA-04.
        assert!(texto.contains("wyd_handshakes_total"));
    }
}
