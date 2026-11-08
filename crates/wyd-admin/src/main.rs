//! `wyd-gateway` — binário HTTP/JSON do Cold Path (ADR-0022).
//!
//! Responsabilidades, e nada além delas:
//!
//! - conectar o pool `sqlx` (`DATABASE_URL`) e provar `SELECT 1` no boot;
//! - carregar a foto do catálogo do Postgres (é o que `/api/health` do portal lê);
//! - servir `POST /admin/v1/{servico}/{metodo}` pelo roteador de `wyd-admin` (contrato único,
//!   sem reimplementar nada);
//! - servir `GET /health` (banco de verdade) e `GET /metrics` (texto Prometheus);
//! - auto-checar com `--healthcheck` (imagem distroless não tem `curl`/shell).
//!
//! Nenhum estado de personagem vivo passa por aqui: o gateway é BFF de leitura e de
//! intenções auditadas, como manda o invariante 7 do `AGENTS.md`.

use std::io::{BufRead, BufReader, Write};
use std::net::{TcpListener, TcpStream};
use std::process::ExitCode;
use std::time::Duration;

use wyd_admin::audit::{AuditEntry, AuditSink};
use wyd_admin::auth::Session;
use wyd_admin::metrics::GatewayMetrics;
use wyd_admin::transport::{http, Router, StaticTokens};
use wyd_db::{PgBackend, PoolConfig, Role};

/// Porta padrão do contrato (ADR-0008) também no binário.
const DEFAULT_BIND: &str = "127.0.0.1:7600";
/// Endereço usado pelo `--healthcheck` dentro do próprio container.
const DEFAULT_HEALTHCHECK: &str = "127.0.0.1:7600";

fn main() -> ExitCode {
    if std::env::args().skip(1).any(|arg| arg == "--healthcheck") {
        return healthcheck();
    }
    // `--seed-staff` continua existindo para uso manual/one-shot (CI, reparo de conta).
    if std::env::args().skip(1).any(|arg| arg == "--seed-staff") {
        return seed_staff();
    }
    // Com `WYD_SEED_STAFF=1` o seed roda no próprio boot do gateway: idempotente, sem
    // container one-shot sobrando parado no `docker compose up`
    // (`docs/rust/05-stack-docker.md` §5). Falha aqui é erro de configuração do ambiente.
    if std::env::var("WYD_SEED_STAFF").as_deref() == Ok("1") {
        let seeded = seed_staff();
        if seeded != ExitCode::SUCCESS {
            return seeded;
        }
    }
    match run() {
        Ok(()) => ExitCode::SUCCESS,
        Err(reason) => {
            log("error", "fatal", &format!("reason={reason}"));
            ExitCode::from(2)
        }
    }
}

/// Configuração por ambiente (ADR-0022 §2): nada de arquivo de config no container.
struct Config {
    bind: String,
    database_url: String,
    token: String,
}

impl Config {
    fn from_env() -> Result<Self, String> {
        let bind = std::env::var("WYD_GATEWAY_BIND").unwrap_or_else(|_| DEFAULT_BIND.to_string());
        let database_url = std::env::var("DATABASE_URL")
            .ok()
            .filter(|value| !value.trim().is_empty())
            .ok_or("DATABASE_URL ausente: o gateway não sobe sem banco")?;
        let token = std::env::var("WEB_API_TOKEN")
            .ok()
            .filter(|value| !value.trim().is_empty())
            .ok_or("WEB_API_TOKEN ausente: backend admin sem autenticação é fail-closed")?;
        Ok(Self {
            bind,
            database_url,
            token,
        })
    }
}

fn run() -> Result<(), String> {
    let config = Config::from_env()?;
    log(
        "info",
        "boot",
        &format!("version={}", env!("CARGO_PKG_VERSION")),
    );

    let runtime = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(2)
        .enable_all()
        .build()
        .map_err(|_| "runtime tokio indisponível".to_string())?;

    let backend = runtime
        .block_on(PgBackend::connect(
            &config.database_url,
            PoolConfig::local(),
        ))
        .map_err(|error| format!("banco recusou a conexão: {error}"))?;
    runtime
        .block_on(backend.ping())
        .map_err(|error| format!("banco sem SELECT 1: {error}"))?;
    log(
        "info",
        "db",
        &format!("ready=true schema={}", wyd_db::SCHEMA_VERSION),
    );

    let catalog = runtime
        .block_on(backend.catalog_snapshot())
        .map_err(|error| format!("catálogo indisponível (migrations aplicadas?): {error}"))?;
    log(
        "info",
        "catalog",
        &format!("items={} version={}", catalog.items.len(), catalog.version),
    );

    let mut tokens = StaticTokens::default();
    // Token processo→processo (nunca cookie de sessão); o autor é o próprio serviço.
    tokens.insert(
        config.token.clone(),
        Session {
            account_id: 0,
            role: Role::Admin,
        },
    );

    // O roteador não tem runtime (ADR-0006): o login recebe o pool por uma função
    // síncrona que resolve `credentials` com `block_on` e confere argon2id.
    let runtime = std::sync::Arc::new(runtime);
    let login_backend = backend.clone();
    let login_runtime = std::sync::Arc::clone(&runtime);
    let login: wyd_admin::transport::LoginCheck = std::sync::Arc::new(move |name, password| {
        let record = login_runtime
            .block_on(wyd_db::Accounts::credentials(&login_backend, name))
            .ok()?;
        wyd_admin::verify_password(password, &record.password_hash).then_some(
            wyd_admin::transport::LoginOk {
                account_id: record.account.id,
                login: record.account.login,
                role: record.account.role,
            },
        )
    });

    let router = Router::new(backend.clone(), backend.clone(), StdoutSink, tokens)
        .with_catalog(catalog)
        .with_login_check(login);
    let metrics = GatewayMetrics::new();

    let listener = TcpListener::bind(&config.bind)
        .map_err(|error| format!("bind {} falhou: {error}", config.bind))?;
    let addr = listener
        .local_addr()
        .map(|value| value.to_string())
        .unwrap_or_else(|_| config.bind.clone());
    log("info", "listen", &format!("addr={addr}"));

    loop {
        match listener.accept() {
            Ok((stream, _peer)) => handle(stream, &router, &metrics, &backend, &runtime),
            Err(error) => {
                log("error", "accept", &format!("reason={error}"));
                return Err("accept falhou".to_string());
            }
        }
    }
}

/// Atende uma conexão do serviço: health, métricas ou rota do contrato.
fn handle(
    stream: TcpStream,
    router: &Router<PgBackend, PgBackend, StdoutSink>,
    metrics: &GatewayMetrics,
    backend: &PgBackend,
    runtime: &tokio::runtime::Runtime,
) {
    let _ = stream.set_read_timeout(Some(Duration::from_secs(15)));
    let Ok(read_half) = stream.try_clone() else {
        return;
    };
    let mut reader = BufReader::new(read_half);
    let mut writer = stream;

    let request = match http::read_request_any(&mut reader, http::MAX_BODY) {
        Ok(request) => request,
        Err(error) => {
            metrics.record(error.status(), None);
            let _ = http::write_response(
                &mut writer,
                error.status(),
                "",
                "{\"message\":\"falha de transporte\"}",
            );
            return;
        }
    };
    let path = request.path.split('?').next().unwrap_or("");

    match (request.method.as_str(), path) {
        ("GET", "/health") => {
            let healthy = runtime.block_on(backend.ping()).is_ok();
            if healthy {
                metrics.record(200, Some(1));
                let body = format!(
                    "{{\"ok\":true,\"database\":\"up\",\"schema\":{}}}",
                    wyd_db::SCHEMA_VERSION
                );
                let _ = http::write_response(&mut writer, 200, request.trace_id(), &body);
            } else {
                metrics.record(503, None);
                let _ = http::write_response(
                    &mut writer,
                    503,
                    request.trace_id(),
                    "{\"ok\":false,\"database\":\"down\"}",
                );
            }
        }
        ("GET", "/metrics") => {
            metrics.record(200, Some(1));
            let body = metrics.render();
            let _ = http::write_text_response(
                &mut writer,
                200,
                "text/plain; version=0.0.4; charset=utf-8",
                &body,
            );
        }
        _ => {
            let reply = router.dispatch(&request);
            metrics.record(reply.status, result_of(&reply.body));
            let _ = http::write_response(&mut writer, reply.status, &reply.trace_id, &reply.body);
        }
    }
}

/// `--seed-staff`: cria/atualiza a conta de staff a partir de `PORTAL_ADMIN_*`.
///
/// Lê `PORTAL_ADMIN_USER` (padrão `admin`) e `PORTAL_ADMIN_PASSWORD`, calcula o
/// hash argon2id e faz upsert com papel `admin`. Sem senha configurada e sem
/// `WYD_SEED_STAFF=1`, não faz nada (stack de produção que não quer seed some).
/// Com `WYD_SEED_STAFF=1` e senha ausente, falha: seed pedido e não aplicado é
/// erro de configuração, não sucesso silencioso.
fn seed_staff() -> ExitCode {
    let obrigatorio = std::env::var("WYD_SEED_STAFF").as_deref() == Ok("1");
    let user = std::env::var("PORTAL_ADMIN_USER")
        .ok()
        .filter(|value| !value.trim().is_empty())
        .unwrap_or_else(|| "admin".to_string());
    let senha = std::env::var("PORTAL_ADMIN_PASSWORD")
        .ok()
        .filter(|value| !value.trim().is_empty());

    let Some(senha) = senha else {
        if obrigatorio {
            log(
                "error",
                "staff_seed",
                "reason=PORTAL_ADMIN_PASSWORD ausente com WYD_SEED_STAFF=1",
            );
            return ExitCode::from(2);
        }
        log(
            "info",
            "staff_seed",
            "skip=true motivo=sem_credencial_configurada",
        );
        return ExitCode::SUCCESS;
    };

    let Some(database_url) = std::env::var("DATABASE_URL")
        .ok()
        .filter(|value| !value.trim().is_empty())
    else {
        log("error", "staff_seed", "reason=DATABASE_URL ausente");
        return ExitCode::from(2);
    };

    let hash = match wyd_admin::hash_password(&senha) {
        Ok(hash) => hash,
        Err(error) => {
            log("error", "staff_seed", &format!("reason={error}"));
            return ExitCode::from(2);
        }
    };

    let runtime = match tokio::runtime::Builder::new_multi_thread()
        .worker_threads(1)
        .enable_all()
        .build()
    {
        Ok(runtime) => runtime,
        Err(_) => return ExitCode::from(2),
    };
    let backend = match runtime.block_on(wyd_db::PgBackend::connect(
        &database_url,
        wyd_db::PoolConfig::local(),
    )) {
        Ok(backend) => backend,
        Err(error) => {
            log("error", "staff_seed", &format!("reason={error}"));
            return ExitCode::from(2);
        }
    };
    match runtime.block_on(wyd_db::Accounts::set_credentials(
        &backend,
        &user,
        &hash,
        Role::Admin,
    )) {
        Ok(account) => {
            log(
                "info",
                "staff_seed",
                &format!(
                    "login={} id={} role={}",
                    account.login,
                    account.id,
                    account.role.as_db()
                ),
            );
            ExitCode::SUCCESS
        }
        Err(error) => {
            log("error", "staff_seed", &format!("reason={error}"));
            ExitCode::from(2)
        }
    }
}

/// `--healthcheck`: sai `0` só com `200` do próprio `/health` (que consulta o banco).
fn healthcheck() -> ExitCode {
    let addr = std::env::var("WYD_GATEWAY_HEALTHCHECK_ADDR")
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

/// `result` do envelope `AdminResult` para classificar a métrica (sem parse de JSON inteiro).
fn result_of(body: &str) -> Option<u8> {
    let rest = body.split("\"result\":").nth(1)?;
    let digits: String = rest.chars().take_while(char::is_ascii_digit).collect();
    digits.parse().ok()
}

/// Log estruturado em uma linha (`key=value`), sem dependência de crate de log.
fn log(level: &str, event: &str, fields: &str) {
    println!("level={level} event={event} {fields}");
}

/// Destino de auditoria do serviço: stdout estruturado do container (`docker logs`).
///
/// **Lacuna declarada (ADR-0022):** a tabela `audit_log` ainda não tem porta de escrita em
/// `wyd-admin`; até ela existir o registro fica no log do container, nunca descartado.
#[derive(Debug, Default)]
struct StdoutSink;

impl AuditSink for StdoutSink {
    fn record(&self, entry: AuditEntry) {
        log(
            "info",
            "audit",
            &format!(
                "actor={} role={} action={} target={} detail={:?}",
                entry.actor_id,
                entry.actor_role.as_db(),
                entry.action,
                entry.target,
                entry.detail
            ),
        );
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn result_is_read_from_envelope() {
        assert_eq!(result_of("{\"result\":1,\"items\":[]}"), Some(1));
        assert_eq!(result_of("{\"result\":4,\"message\":\"x\"}"), Some(4));
        assert_eq!(result_of("{\"message\":\"sem resultado\"}"), None);
    }

    #[test]
    fn missing_db_or_token_is_fail_closed() {
        // Sem ambiente, a config recusa subir (não existe default inseguro).
        std::env::remove_var("DATABASE_URL");
        std::env::remove_var("WEB_API_TOKEN");
        assert!(Config::from_env().is_err());
    }
}
