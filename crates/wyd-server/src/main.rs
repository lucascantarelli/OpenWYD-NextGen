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
//! - `:9898` `/metrics` (contrato da FASE-06-1), agora com as métricas **reais** do motor;
//! - thread `wyd-tick` (FASE-03-7): `Engine` do `wyd-core` a `WYD_TICK_HZ`, dona do mundo.
//!
//! `--healthcheck` faz `GET /health` no próprio processo e só sai `0` com 200 (banco real).

use std::io::{BufRead, BufReader, Write};
use std::net::{TcpListener, TcpStream};
use std::process::ExitCode;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::Duration;

use wyd_db::{Grant, Grants, PgBackend, PoolConfig};
use wyd_net::VersionGate;
use wyd_server::channels::World;
use wyd_server::metrics::ServiceMetrics;
use wyd_server::world::SaveSink;
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
    let tick_hz = wyd_server::world::tick_hz_from_env()?;
    // Teto de cena por varredura (FASE-03-9): validado no boot, nunca corrigido em silêncio.
    let scene_budget = wyd_server::world::scene_budget_from_env()?;

    log(
        "info",
        "boot",
        &format!("channel={channel} version={}", env!("CARGO_PKG_VERSION")),
    );
    // Lotação/manutenção (FASE-06-3): o banco é a fonte, o env sobrescreve por processo.
    let capacity_overrides = wyd_server::channels::policy_overrides_from_env();
    // Teto baixo por engano trava o canal: o aviso é no boot, antes de recusar ninguém.
    let nivel = if capacity_overrides.max_players.is_some_and(|teto| teto < 10) {
        "warn"
    } else {
        "info"
    };
    log(
        nivel,
        "session",
        &format!(
            "capacity max_players={} maintenance={} (env sobrescreve banco)",
            capacity_overrides
                .max_players
                .map_or("banco".to_string(), |teto| teto.to_string()),
            capacity_overrides
                .maintenance
                .map_or("banco".to_string(), |ativo| ativo.to_string()),
        ),
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
    // Um runtime só para todo o processo (borda, config, status e saves).
    let runtime = Arc::new(runtime);

    let canais = runtime
        .block_on(backend.server_channels())
        .map_err(|error| format!("server_channel indisponível (migrations aplicadas?): {error}"))?;
    let world = Arc::new(World::new(canais)?);
    // Papel do canal (FASE-03-10): o banco é a fonte, o env sobrescreve. Canal sem linha =
    // `normal`, então nenhum comportamento muda por omissão.
    let role_env = wyd_server::channels::role_overrides_from_env();
    let role_db = runtime
        .block_on(wyd_db::Sessions::channel_role(
            &backend, group_id, canal_i16,
        ))
        .unwrap_or(None);
    let rules = wyd_server::channels::ChannelRules::new(wyd_server::channels::effective_role(
        role_db, role_env,
    ));
    let origem_papel = if role_env.is_some() { "env" } else { "banco" };
    log(
        "info",
        "channel",
        &format!(
            "role={} pvp={} shop={} source={origem_papel} group={group_id} channel={canal_i16}",
            rules.as_str(),
            rules.pvp_allowed(true),
            rules.personal_shop_allowed()
        ),
    );
    let metrics = Arc::new(ServiceMetrics::new(world.game_channel_count()));
    // Registro do motor anexado ao `/metrics` **antes** do primeiro scrape: o HTTP não
    // publica tick zerado (ADR-0022 §7).
    let engine_metrics = Arc::new(wyd_core::Metrics::new());
    metrics.attach_engine(Arc::clone(&engine_metrics));

    // Configuração viva: o boot exige uma foto válida (sem default silencioso); a partir
    // daqui a thread `wyd-config` observa `config_revision` e troca o `Arc` sem tocar no tick.
    let config_source = Arc::new(wyd_server::config::PgConfigSource::new(
        backend.clone(),
        Arc::clone(&runtime),
    ));
    let config_inicial = config_source
        .load_current()
        .map_err(|error| format!("config inicial: {error}"))?;
    log(
        "info",
        "config",
        &format!(
            "revision={} refine={} sanc={} combine={} drops={} npcs={} routes={}",
            config_inicial.revision(),
            config_inicial.refine_steps(),
            config_inicial.sanc_cells(),
            config_inicial.combine_values(),
            config_inicial.drop_overrides(),
            config_inicial.npc_count(),
            config_inicial.route_count()
        ),
    );
    let live_config = Arc::new(wyd_server::config::LiveConfig::new(config_inicial));
    metrics.attach_config(Arc::clone(&live_config));
    let config_poll = wyd_server::config::poll_interval_from_env()?;
    let config_stop = Arc::new(AtomicBool::new(false));
    let config_thread = wyd_server::config::spawn_reloader(
        Arc::clone(&live_config),
        config_source,
        config_poll,
        Arc::clone(&config_stop),
    )?;
    log(
        "info",
        "config",
        &format!("reloader=on poll_ms={}", config_poll.as_millis()),
    );
    let mut engine = wyd_server::world::Engine::new(tick_hz, Arc::clone(&engine_metrics))
        .map_err(|error| format!("tick: {error}"))?;
    // PvP é decisão do canal × mapa: a penalidade de CP só existe quando o papel permite.
    engine.set_pvp(rules.pvp_allowed(true));
    let tick_interval = engine.interval_ms();
    log(
        "info",
        "channels",
        &format!(
            "published={} status_url={}",
            world.game_channel_count(),
            wyd_server::channels::status_url(&world)
        ),
    );

    // Fila de save do mundo (FASE-03-8): `enqueue` é `try_send` na borda; o dreno grava
    // fora do tick, coalescendo por personagem. Sem banco no laço de simulação.
    let (save_queue, mut save_drain) = wyd_db::SaveQueue::bounded(wyd_db::SavePolicy::local())
        .map_err(|error| format!("fila de save: {error}"))?;
    let saves = Arc::new(wyd_server::world::SaveQueueSink::new(Arc::new(save_queue)));
    let save_stop = Arc::new(AtomicBool::new(false));
    let save_thread = {
        let stop = Arc::clone(&save_stop);
        let drain_runtime = Arc::clone(&runtime);
        let drain_backend = backend.clone();
        std::thread::spawn(move || {
            save_drain_loop(&mut save_drain, &drain_runtime, &drain_backend, &stop);
        })
    };

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
            status::serve(status::StatusDeps {
                listener: status_listener,
                world,
                backend,
                runtime,
                metrics,
                group_id,
                channel: canal_i16,
                overrides: capacity_overrides,
                rules,
            })
        });
    }

    // Tick do mundo: um por processo, sem mutex global e sem I/O no laço.
    let (mut tick_thread, engine_handle) =
        wyd_server::world::EngineThread::spawn(engine, tick_interval)?;
    log(
        "info",
        "tick",
        &format!(
            "hz={tick_hz} interval_ms={tick_interval} scene_budget={scene_budget} thread=wyd-tick"
        ),
    );

    // NPCs do snapshot: materializa `npc_definition` do mapa do canal e reconcilia a cada
    // revisão nova — a edição no portal muda o mundo sem reiniciar o processo. O mapa vem do
    // ambiente porque `character` ainda não persiste mapa (FASE-03-9).
    let map_id: i16 = std::env::var("WYD_SERVER_MAP_ID")
        .unwrap_or_else(|_| "0".to_string())
        .parse()
        .map_err(|_| "WYD_SERVER_MAP_ID precisa ser um número".to_string())?;
    let npc_inicial = engine_handle.sync_npcs(wyd_server::world::npc_specs(
        &live_config.snapshot(),
        map_id,
    ));
    log(
        "info",
        "npc",
        &format!(
            "result=spawn map={map_id} spawned={} kept={} removed={} rejected={}",
            npc_inicial.spawned, npc_inicial.kept, npc_inicial.removed, npc_inicial.rejected
        ),
    );
    {
        let handle = engine_handle.clone();
        let live = Arc::clone(&live_config);
        std::thread::spawn(move || {
            let mut aplicada = live.revision();
            loop {
                std::thread::sleep(Duration::from_millis(200));
                let revisao = live.revision();
                if revisao == aplicada {
                    continue;
                }
                let report =
                    handle.sync_npcs(wyd_server::world::npc_specs(&live.snapshot(), map_id));
                log(
                    "info",
                    "npc",
                    &format!(
                        "result=reload revision={revisao} spawned={} kept={} removed={} rejected={}",
                        report.spawned, report.kept, report.removed, report.rejected
                    ),
                );
                aplicada = revisao;
            }
        });
    }

    let gate = VersionGate::from_protocol_version();
    log(
        "info",
        "gate",
        &format!("expected_version={}", gate.expected()),
    );
    let sessions = Arc::new(
        wyd_server::session::SessionGate::new(
            Arc::new(backend.clone()),
            Arc::clone(&runtime),
            group_id,
            canal_i16,
            session_idle,
            session_stale,
        )
        .with_overrides(capacity_overrides),
    );
    // Reaper de sessões: libera no boot e a cada 15 s as linhas cujo `renewed_at` venceu
    // (cliente morto sem FIN/leave), para o `/health` não contar sessão fantasma. TTL =
    // `WYD_SERVER_SESSION_STALE_SECS` (default 180 s), o mesmo do lease em memória.
    {
        let backend = Arc::new(backend.clone());
        let runtime = Arc::clone(&runtime);
        let ttl = i64::try_from(session_stale.as_secs()).unwrap_or(180);
        std::thread::spawn(move || {
            log(
                "info",
                "reaper",
                &format!("result=started ttl_s={ttl} interval_s=15"),
            );
            loop {
                match runtime.block_on(wyd_db::Sessions::release_stale_sessions(
                    backend.as_ref(),
                    ttl,
                )) {
                    Ok(liberadas) if liberadas > 0 => log(
                        "info",
                        "session",
                        &format!("reaper result=released count={liberadas} ttl_s={ttl}"),
                    ),
                    Ok(_) => {}
                    Err(error) => log(
                        "warn",
                        "session",
                        &format!("reaper result=failed reason={error}"),
                    ),
                }
                std::thread::sleep(Duration::from_secs(15));
            }
        });
    }

    // Seleção de personagem depois do login: mesmo pool e mesmo runtime do handshake.
    let selection = Arc::new(
        wyd_server::characters::SelectionStore::new(
            Arc::new(backend.clone()),
            Arc::clone(&runtime),
        )
        .with_world(engine_handle.clone(), saves.clone())
        .with_config(Arc::clone(&live_config)),
    );

    // Save periódico: foto do mundo a cada 5 s, sem tocar no tick.
    {
        let handle = engine_handle.clone();
        let saves = Arc::clone(&saves);
        std::thread::spawn(move || loop {
            std::thread::sleep(Duration::from_secs(5));
            let estados = handle.snapshots();
            for estado in &estados {
                saves.save(*estado);
            }
            if !estados.is_empty() {
                log(
                    "info",
                    "save",
                    &format!("periodic players={}", estados.len()),
                );
            }
        });
    }
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
    // Encerramento ordenado: foto final → dreno (flush da última fila) → tick.
    for estado in engine_handle.snapshots() {
        saves.save(estado);
    }
    save_stop.store(true, Ordering::Relaxed);
    let _ = save_thread.join();
    tick_thread.stop_and_join();
    config_stop.store(true, Ordering::Relaxed);
    let _ = config_thread.join();
    Ok(())
}

/// Dreno de save: flush a cada 1 s e uma última passagem depois do sinal de parada.
fn save_drain_loop(
    drain: &mut wyd_db::SaveDrain,
    runtime: &tokio::runtime::Runtime,
    backend: &PgBackend,
    stop: &AtomicBool,
) {
    loop {
        match runtime.block_on(drain.flush(backend)) {
            Ok(report) if report.written > 0 || report.missing > 0 => log(
                "info",
                "save",
                &format!(
                    "written={} coalesced={} missing={}",
                    report.written, report.coalesced, report.missing
                ),
            ),
            Ok(_) => {}
            Err(error) => log(
                "warn",
                "save",
                &format!("result=flush_failed reason={error}"),
            ),
        }
        if stop.load(Ordering::Relaxed) {
            break;
        }
        std::thread::sleep(Duration::from_secs(1));
    }
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
