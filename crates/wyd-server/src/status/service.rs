//! `GET :80/serv00.htm` — a coluna Canal do cliente vem daqui (números), e a sonda UDP do
//! `:8281` diz ao cliente que o canal está acessível.
//!
//! O corpo é o CSV com **um número por canal** (`channels::render_status_body`), lido do
//! Postgres no boot. `GET /health` faz `SELECT 1` no pool (o healthcheck do container usa
//! esse caminho), então o container não fica "healthy" sem banco. Nenhum outro caminho
//! responde 200.

use std::io::BufReader;
use std::net::{TcpListener, TcpStream, UdpSocket};
use std::sync::Arc;
use std::time::Duration;

use wyd_admin::transport::http;
use wyd_db::{PgBackend, Sessions};

use crate::channels::{render_status_body, ChannelRules, World};
use crate::metrics::ServiceMetrics;
use crate::session::PolicyOverrides;

/// Caminho que o cliente busca (`docs/cliente/formatos.md` §4.1).
pub const STATUS_PATH: &str = "/serv00.htm";
/// Caminho de saúde (usado pelo `--healthcheck`).
pub const HEALTH_PATH: &str = "/health";

/// Tamanho da sonda UDP de status: o cliente 7662 manda **4 bytes** (`01 02 03 04`) para o
/// host do canal na `8281` e só considera o canal acessível quando o mesmo datagrama volta
/// (captura `target/teste-udp.json`). O legado respondia com eco.
pub const UDP_PROBE_LEN: usize = 4;

/// Dependências do servidor de status (`:80`): o `overrides` carrega a política efetiva
/// de manutenção para o `/health` (FASE-06-3).
pub struct StatusDeps {
    /// Socket do `:80` (publicado como `:80` só pelo canal de status).
    pub listener: TcpListener,
    /// Foto dos canais (o `serv00.htm` sai daqui).
    pub world: Arc<World>,
    /// Pool compartilhado com o handshake.
    pub backend: PgBackend,
    /// Runtime que executa as consultas.
    pub runtime: Arc<tokio::runtime::Runtime>,
    /// Contadores do processo.
    pub metrics: Arc<ServiceMetrics>,
    /// Grupo deste processo.
    pub group_id: i16,
    /// Canal deste processo.
    pub channel: i16,
    /// Sobrescritas de lotação/manutenção por processo (env).
    pub overrides: PolicyOverrides,
    /// Regras do papel do canal (FASE-03-10): o `/health` publica o papel vigente.
    pub rules: ChannelRules,
}

/// Aceita conexões do status HTTP para sempre (uma por vez: é uma leitura por cliente).
pub fn serve(deps: StatusDeps) {
    for aceito in deps.listener.incoming() {
        match aceito {
            Ok(stream) => handle(stream, &deps),
            Err(error) => {
                log("error", "accept", &format!("service=status reason={error}"));
                return;
            }
        }
    }
}

fn handle(stream: TcpStream, deps: &StatusDeps) {
    let world = &deps.world;
    let backend = &deps.backend;
    let runtime = &deps.runtime;
    let metrics = &deps.metrics;
    let group_id = deps.group_id;
    let channel = deps.channel;
    let overrides = deps.overrides;
    let rules = deps.rules;
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
            let online = online_por_canal(world, backend, runtime);
            let corpo = render_status_body(world, &online);
            log(
                "info",
                "status",
                &format!(
                    "path={STATUS_PATH} bytes={} online={}",
                    corpo.len(),
                    online
                        .iter()
                        .map(u64::to_string)
                        .collect::<Vec<_>>()
                        .join("/")
                ),
            );
            let _ =
                http::write_text_response(&mut writer, 200, "text/plain; charset=utf-8", &corpo);
        }
        HEALTH_PATH => {
            let healthy = runtime.block_on(backend.ping()).is_ok();
            let (status, corpo) = if healthy {
                // `online` é a presença **deste** canal: quem consulta o `/health` do canal 1
                // não recebe número de sessão do canal 2.
                let online = runtime
                    .block_on(backend.live_sessions(group_id, channel))
                    .unwrap_or(0);
                // Manutenção efetiva (FASE-06-3): env sobrescreve o banco; banco fora
                // do ar aqui é impossível (o `ping` acima passou), então linha ausente
                // ou erro de leitura = sem manutenção.
                let manutencao = overrides.maintenance.or_else(|| {
                    runtime
                        .block_on(backend.channel_capacity(group_id, channel))
                        .ok()
                        .map(|linha| linha.maintenance)
                });
                let manutencao = manutencao.unwrap_or(false);
                (
                    200,
                    health_body(&HealthFacts {
                        channels: world.game_channel_count(),
                        group_id,
                        channel,
                        rules,
                        online,
                        maintenance: manutencao,
                        schema: wyd_db::SCHEMA_VERSION,
                    }),
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

/// O que o `/health` publica além do `SELECT 1`.
///
/// `rules` entra como **teto** do papel: sem mapa em mãos, `pvp` é `true` só quando o canal
/// não restringe (canal normal); no verde é `false` sempre.
#[derive(Debug, Clone, Copy)]
struct HealthFacts {
    channels: usize,
    group_id: i16,
    channel: i16,
    rules: ChannelRules,
    online: u64,
    maintenance: bool,
    schema: u32,
}

/// Corpo JSON do `/health` (uma linha, sem dependência de serializer).
fn health_body(fatos: &HealthFacts) -> String {
    format!(
        "{{\"ok\":true,\"database\":\"up\",\"channels\":{},\"group\":{},\"channel\":{},\"role\":\"{}\",\"pvp\":{},\"shop\":{},\"online\":{},\"maintenance\":{},\"schema\":{}}}",
        fatos.channels,
        fatos.group_id,
        fatos.channel,
        fatos.rules.as_str(),
        fatos.rules.pvp_allowed(true),
        fatos.rules.personal_shop_allowed(),
        fatos.online,
        fatos.maintenance,
        fatos.schema
    )
}

/// Sessões vivas por canal de jogo, na ordem de [`World::game_channels`].
///
/// Cada número vem de `character_session` **filtrada por grupo e canal**: o canal publica a
/// própria presença e nenhuma entidade de outro canal (isolamento de mundo; ver
/// `docs/wyd/canais-e-cidadania.md` §5.2). Consulta que ignorasse o canal viraria agregado do
/// grupo e o teste `contagem_de_um_canal_nao_aparece_no_outro` não pegaria — o do banco pega.
fn online_por_canal(
    world: &World,
    backend: &PgBackend,
    runtime: &tokio::runtime::Runtime,
) -> Vec<u64> {
    world
        .game_channels()
        .map(|canal| {
            let grupo = i16::try_from(canal.group_id).unwrap_or(0);
            let indice = i16::try_from(canal.channel).unwrap_or(0);
            runtime
                .block_on(backend.live_sessions(grupo, indice))
                .unwrap_or(0)
        })
        .collect()
}

/// Responde a sonda UDP de status do canal para sempre.
///
/// Datagrama com tamanho diferente de [`UDP_PROBE_LEN`] é logado e ignorado: o canal não
/// inventa resposta e a thread nunca cai por pacote inesperado (entrada de rede não gera
/// `panic!`).
pub fn serve_udp(socket: UdpSocket) {
    let mut buf = [0u8; 64];
    loop {
        match socket.recv_from(&mut buf) {
            Ok((len, origem)) if len == UDP_PROBE_LEN => {
                match socket.send_to(&buf[..len], origem) {
                    Ok(enviado) => log(
                        "info",
                        "udp",
                        &format!("service=status probe=answered peer={origem} bytes={enviado}"),
                    ),
                    Err(error) => log(
                        "warn",
                        "udp",
                        &format!("service=status probe=falhou peer={origem} reason={error}"),
                    ),
                }
            }
            Ok((len, origem)) => log(
                "warn",
                "udp",
                &format!("service=status probe=ignorada peer={origem} bytes={len}"),
            ),
            Err(error) => log("error", "udp", &format!("service=status reason={error}")),
        }
    }
}

/// Log estruturado (uma linha, `key=value`).
fn log(level: &str, event: &str, fields: &str) {
    println!("level={level} event={event} {fields}");
}

#[cfg(test)]
mod tests {
    use super::*;
    use wyd_db::ChannelRole;

    fn fatos(role: ChannelRole) -> HealthFacts {
        HealthFacts {
            channels: 2,
            group_id: 1,
            channel: 2,
            rules: crate::channels::ChannelRules::new(role),
            online: 7,
            maintenance: false,
            schema: 41,
        }
    }

    #[test]
    fn health_publica_papel_do_canal() {
        let verde = health_body(&fatos(ChannelRole::Verde));
        assert!(
            verde.contains("\"role\":\"verde\""),
            "papel verde no corpo: {verde}"
        );
        assert!(verde.contains("\"pvp\":false"), "verde sem PvP: {verde}");
        assert!(verde.contains("\"shop\":true"), "loja aberta: {verde}");
        assert!(verde.contains("\"online\":7"), "presença do canal: {verde}");
        assert!(verde.contains("\"schema\":41"), "schema corrente: {verde}");

        let normal = health_body(&fatos(ChannelRole::Normal));
        assert!(normal.contains("\"role\":\"normal\""));
        assert!(
            normal.contains("\"pvp\":true"),
            "canal normal não restringe: {normal}"
        );
    }
}
