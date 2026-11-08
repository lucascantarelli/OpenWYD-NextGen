//! `GET :80/serv00.htm` — a coluna Canal do cliente vem daqui.
//!
//! O corpo é o CSV de hosts dos canais (`channels::render_status_csv`), lido do Postgres no
//! boot. `GET /health` faz `SELECT 1` no pool (o healthcheck do container usa esse caminho),
//! então o container não fica "healthy" sem banco. Nenhum outro caminho responde 200.

use std::io::BufReader;
use std::net::{TcpListener, TcpStream};
use std::sync::Arc;
use std::time::Duration;

use wyd_admin::transport::http;
use wyd_db::PgBackend;

use crate::channels::{render_status_csv, World};
use crate::metrics::ServiceMetrics;

/// Caminho que o cliente busca (`docs/cliente/formatos.md` §4.1).
pub const STATUS_PATH: &str = "/serv00.htm";
/// Caminho de saúde (usado pelo `--healthcheck`).
pub const HEALTH_PATH: &str = "/health";

/// Aceita conexões do status HTTP para sempre (uma por vez: é uma leitura por cliente).
pub fn serve(
    listener: TcpListener,
    world: Arc<World>,
    backend: PgBackend,
    runtime: Arc<tokio::runtime::Runtime>,
    metrics: Arc<ServiceMetrics>,
) {
    for aceito in listener.incoming() {
        match aceito {
            Ok(stream) => handle(stream, &world, &backend, &runtime, &metrics),
            Err(error) => {
                log("error", "accept", &format!("service=status reason={error}"));
                return;
            }
        }
    }
}

fn handle(
    stream: TcpStream,
    world: &World,
    backend: &PgBackend,
    runtime: &tokio::runtime::Runtime,
    metrics: &ServiceMetrics,
) {
    let _ = stream.set_read_timeout(Some(Duration::from_secs(10)));
    let _ = stream.set_write_timeout(Some(Duration::from_secs(10)));
    // Diagnóstico: quem abriu TCP no :80 (é aqui que um cliente que disca a porta errada aparece).
    if let Ok(peer) = stream.peer_addr() {
        log("info", "tcp", &format!("service=status peer={peer}"));
    }
    let Ok(read_half) = stream.try_clone() else {
        return;
    };
    let mut reader = BufReader::new(read_half);
    let mut writer = stream;

    let request = match http::read_request_any(&mut reader, 64 * 1024) {
        Ok(request) => request,
        Err(error) => {
            log(
                "warn",
                "http",
                &format!("service=status result=invalido motivo={error:?}"),
            );
            let _ = http::write_text_response(
                &mut writer,
                error.status(),
                "text/plain; charset=utf-8",
                "requisicao invalida",
            );
            return;
        }
    };
    let path = request.path.split('?').next().unwrap_or("");

    match path {
        STATUS_PATH | "/" => {
            metrics.status_served();
            let corpo = render_status_csv(world);
            log(
                "info",
                "status",
                &format!("path={STATUS_PATH} bytes={}", corpo.len()),
            );
            let _ =
                http::write_text_response(&mut writer, 200, "text/plain; charset=utf-8", &corpo);
        }
        HEALTH_PATH => {
            let healthy = runtime.block_on(backend.ping()).is_ok();
            let (status, corpo) = if healthy {
                (
                    200,
                    format!(
                        "{{\"ok\":true,\"database\":\"up\",\"channels\":{},\"schema\":{}}}",
                        world.game_channel_count(),
                        wyd_db::SCHEMA_VERSION
                    ),
                )
            } else {
                (503, "{\"ok\":false,\"database\":\"down\"}".to_string())
            };
            metrics.health_checked(healthy);
            let _ = http::write_response(&mut writer, status, request.trace_id(), &corpo);
        }
        _ => {
            let _ = http::write_text_response(
                &mut writer,
                404,
                "text/plain; charset=utf-8",
                "nao encontrado",
            );
        }
    }
}

/// Log estruturado (uma linha, `key=value`).
fn log(level: &str, event: &str, fields: &str) {
    println!("level={level} event={event} {fields}");
}
