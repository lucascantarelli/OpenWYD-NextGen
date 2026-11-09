//! `tmserver` — processo de UM canal (`wyd-server`, ADR-0022 §7).
//!
//! Um canal = um processo = um container (`wyd-canal-1`, `wyd-canal-2`), todos com o **mesmo**
//! Postgres: é o desenho do WYD (`docs/wyd/canais-e-cidadania.md` §5) e o que a imagem única
//! `wyd/server:${TAG}` torna barato. O canal muda por configuração (`WYD_SERVER_CHANNEL`), não
//! por build.
//!
//! Superfícies deste binário:
//!
//! - `:8281` CPSock (handshake + gate de versão) e a sonda UDP de status do canal;
//! - `:80` status (`/serv00.htm` e `/health`) — publicado no host só pelo canal de status;
//! - `:9898` `/metrics` (contrato da FASE-06-1).
//!
//! `--healthcheck` faz `GET /health` no próprio processo e só sai `0` com 200 (banco real).

use std::io::{BufRead, BufReader, Write};
use std::net::{TcpListener, TcpStream};
use std::process::ExitCode;
use std::sync::Arc;
use std::time::Duration;

use wyd_db::{Grant, Grants, PgBackend, PoolConfig};
use wyd_net::VersionGate;
use wyd_server::channels::World;
use wyd_server::metrics::ServiceMetrics;
use wyd_server::{cpsock, status};

const DEFAULT_CPSOCK: &str = "0.0.0.0:8281";
const DEFAULT_STATUS: &str = "0.0.0.0:8080";
const DEFAULT_UDP_STATUS: &str = "0.0.0.0:8281";
const DEFAULT_METRICS: &str = "0.0.0.0:9898";
const DEFAULT_HEALTHCHECK: &str = "127.0.0.1:8080";

fn main() -> ExitCode {
    if std::env::args().skip(1).any(|arg| arg == "--healthcheck") {
        return healthcheck();
    }
    match run() {
        Ok(()) => ExitCode::SUCCESS,
        Err(reason) => {
            log("error", "fatal", &format!("reason={reason}"));
            ExitCode::from(2)
        }
    }
}

fn run() -> Result<(), String> {
    let database_url = std::env::var("DATABASE_URL")
        .ok()
        .filter(|value| !value.trim().is_empty())
        .ok_or("DATABASE_URL ausente: o servidor de jogo não sobe sem banco")?;
    let channel: i64 = std::env::var("WYD_SERVER_CHANNEL")
        .unwrap_or_else(|_| "1".to_string())
        .parse()
        .map_err(|_| "WYD_SERVER_CHANNEL precisa ser um número")?;
    let status_enabled = std::env::var("WYD_SERVER_STATUS_ENABLED")
        .map(|value| value != "0")
        .unwrap_or(true);
    // Grupo (servidor) e tempos da cerca de sessão. `WYD_SERVER_GROUP_ID` precisa bater com
    // `server_channel.group_id`; o default é o grupo semeado pela migration 0029.
    let group_id: i16 = std::env::var("WYD_SERVER_GROUP_ID")
        .unwrap_or_else(|_| "1".to_string())
        .parse()
        .map_err(|_| "WYD_SERVER_GROUP_ID precisa ser um número")?;
    let session_idle = Duration::from_secs(
        std::env::var("WYD_SERVER_SESSION_IDLE_SECS")
            .unwrap_or_else(|_| "120".to_string())
            .parse()
            .map_err(|_| "WYD_SERVER_SESSION_IDLE_SECS precisa ser um número")?,
    );
    let session_stale = Duration::from_secs(
        std::env::var("WYD_SERVER_SESSION_STALE_SECS")
            .unwrap_or_else(|_| "180".to_string())
            .parse()
            .map_err(|_| "WYD_SERVER_SESSION_STALE_SECS precisa ser um número")?,
    );
    let canal_i16 = i16::try_from(channel).map_err(|_| "WYD_SERVER_CHANNEL fora de 1..10")?;

    log(
        "info",
        "boot",
        &format!("channel={channel} version={}", env!("CARGO_PKG_VERSION")),
    );

    let runtime = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(2)
        .enable_all()
        .build()
        .map_err(|_| "runtime tokio indisponível".to_string())?;

    // Schema no boot (ADR-0024): canal novo migra; restart é no-op. O advisory lock
    // de sessão serializa com o gateway/canal 2 subindo ao mesmo tempo.
    let report = runtime
        .block_on(wyd_db::migrate::run(&database_url))
        .map_err(|error| format!("schema: {error}"))?;
    log("info", "schema", &report.to_string());

    let backend = runtime
        .block_on(PgBackend::connect(&database_url, PoolConfig::local()))
        .map_err(|error| format!("banco recusou a conexão: {error}"))?;
    runtime
        .block_on(backend.ping())
        .map_err(|error| format!("banco sem SELECT 1: {error}"))?;

    let canais = runtime
        .block_on(backend.server_channels())
        .map_err(|error| format!("server_channel indisponível (migrations aplicadas?): {error}"))?;
    let world = Arc::new(World::new(canais)?);
    let metrics = Arc::new(ServiceMetrics::new(world.game_channel_count()));
    log(
        "info",
        "channels",
        &format!(
            "published={} status_url={}",
            world.game_channel_count(),
            wyd_server::channels::status_url(&world)
        ),
    );

    let runtime = Arc::new(runtime);

    // Dreno de grants duráveis (FASE-07-4): o portal/admin enfileira na
    // `delivery_queue`; este processo entrega ao personagem. Gold credita e baixa
    // na mesma instrução (`claim_gold`). Item e nível seguem pendentes porque o
    // inventário/nível do Hot Path ainda não consome a fila.
    {
        let drain_backend = backend.clone();
        let drain_runtime = Arc::clone(&runtime);
        std::thread::spawn(move || loop {
            std::thread::sleep(Duration::from_secs(1));
            match drain_runtime.block_on(Grants::pending(&drain_backend, 32)) {
                Ok(linhas) => {
                    for (id, grant) in linhas {
                        if !matches!(grant, Grant::Gold { .. }) {
                            continue;
                        }
                        match drain_runtime.block_on(Grants::claim_gold(&drain_backend, id)) {
                            Ok(Some((character_id, gold))) => log(
                                "info",
                                "grant",
                                &format!(
                                    "delivered kind=gold queue={id} character={character_id} gold={gold}"
                                ),
                            ),
                            Ok(None) => {}
                            Err(error) => {
                                log("error", "grant", &format!("queue={id} reason={error}"))
                            }
                        }
                    }
                }
                Err(error) => log("warn", "grant_drain", &format!("reason={error}")),
            }
        });
    }

    let cpsock_bind = bind("WYD_SERVER_CPSOCK_BIND", DEFAULT_CPSOCK)?;
    let cpsock_listener = TcpListener::bind(&cpsock_bind)
        .map_err(|error| format!("bind {cpsock_bind} falhou: {error}"))?;
    log(
        "info",
        "listen",
        &format!("service=cpsock addr={cpsock_bind}"),
    );

    let metrics_bind = bind("WYD_SERVER_METRICS_BIND", DEFAULT_METRICS)?;
    let metrics_listener = TcpListener::bind(&metrics_bind)
        .map_err(|error| format!("bind {metrics_bind} falhou: {error}"))?;
    log(
        "info",
        "listen",
        &format!("service=metrics addr={metrics_bind}"),
    );
    {
        let metrics = Arc::clone(&metrics);
        std::thread::spawn(move || wyd_server::metrics::serve(metrics_listener, metrics));
    }

    if status_enabled {
        // Sonda UDP na mesma porta do CPSock: TCP e UDP não colidem, e o cliente disca
        // sempre 8281 no host da célula do `serverlist.bin`.
        let udp_bind = bind("WYD_SERVER_UDP_STATUS_BIND", DEFAULT_UDP_STATUS)?;
        let udp = std::net::UdpSocket::bind(&udp_bind)
            .map_err(|error| format!("bind udp {udp_bind} falhou: {error}"))?;
        log(
            "info",
            "listen",
            &format!("service=status-udp addr={udp_bind}"),
        );
        std::thread::spawn(move || status::serve_udp(udp));

        let status_bind = bind("WYD_SERVER_STATUS_BIND", DEFAULT_STATUS)?;
        let status_listener = TcpListener::bind(&status_bind)
            .map_err(|error| format!("bind {status_bind} falhou: {error}"))?;
        log(
            "info",
            "listen",
            &format!("service=status addr={status_bind}"),
        );
        let backend = backend.clone();
        let world = Arc::clone(&world);
        let runtime = Arc::clone(&runtime);
        let metrics = Arc::clone(&metrics);
        std::thread::spawn(move || {
            status::serve(
                status_listener,
                world,
                backend,
                runtime,
                metrics,
                group_id,
                canal_i16,
            )
        });
    }

    let gate = VersionGate::from_protocol_version();
    log(
        "info",
        "gate",
        &format!("expected_version={}", gate.expected()),
    );
    let sessions = Arc::new(wyd_server::session::SessionGate::new(
        Arc::new(backend.clone()),
        Arc::clone(&runtime),
        group_id,
        canal_i16,
        session_idle,
        session_stale,
    ));
    // Seleção de personagem depois do login: mesmo pool e mesmo runtime do handshake.
    let selection = Arc::new(wyd_server::characters::SelectionStore::new(
        Arc::new(backend.clone()),
        Arc::clone(&runtime),
    ));
    log(
        "info",
        "session",
        &format!(
            "group={group_id} channel={canal_i16} idle={}s stale={}s",
            session_idle.as_secs(),
            session_stale.as_secs()
        ),
    );
    // Autenticação de conta: senha (Argon2id) + senha numérica, sobre o mesmo pool.
    let auth = Arc::new(wyd_server::auth::Authenticator::new(
        Arc::new(backend.clone()),
        Arc::clone(&runtime),
    ));
    cpsock::serve_listener(
        cpsock_listener,
        world,
        gate,
        Some(auth),
        Some(sessions),
        Some(selection),
    );
    Ok(())
}

fn bind(var: &str, default: &str) -> Result<String, String> {
    Ok(std::env::var(var).unwrap_or_else(|_| default.to_string()))
}

/// `--healthcheck`: `GET /health` no próprio processo; só 200 é saudável.
fn healthcheck() -> ExitCode {
    let addr = std::env::var("WYD_SERVER_HEALTHCHECK_ADDR")
        .unwrap_or_else(|_| DEFAULT_HEALTHCHECK.to_string());
    let Ok(mut stream) = TcpStream::connect(&addr) else {
        return ExitCode::FAILURE;
    };
    let _ = stream.set_read_timeout(Some(Duration::from_secs(5)));
    let request = "GET /health HTTP/1.1\r\nhost: localhost\r\nconnection: close\r\n\r\n";
    if stream.write_all(request.as_bytes()).is_err() {
        return ExitCode::FAILURE;
    }
    let mut reader = BufReader::new(stream);
    let mut status_line = String::new();
    if reader.read_line(&mut status_line).is_err() {
        return ExitCode::FAILURE;
    }
    if status_line.starts_with("HTTP/1.1 200") {
        ExitCode::SUCCESS
    } else {
        ExitCode::FAILURE
    }
}

fn log(level: &str, event: &str, fields: &str) {
    println!("level={level} event={event} {fields}");
}
