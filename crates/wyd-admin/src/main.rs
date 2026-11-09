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
use wyd_db::{
    Account, AccountModeration, Accounts, AuditRow, AuditTrail, Characters, DbError, Economy,
    EconomySummary, NewAuditEntry, PgBackend, PoolConfig, Role, Sessions,
};

/// Porta padrão do contrato (ADR-0008) também no binário.
const DEFAULT_BIND: &str = "127.0.0.1:7600";
/// Endereço usado pelo `--healthcheck` dentro do próprio container.
const DEFAULT_HEALTHCHECK: &str = "127.0.0.1:7600";

fn main() -> ExitCode {
    let args: Vec<String> = std::env::args().skip(1).collect();
    let flag = |name: &str| args.iter().any(|arg| arg == name);
    if flag("--healthcheck") {
        return healthcheck();
    }
    // Operação/CI: aplica o schema e sai. Não existe job one-shot no compose
    // (ADR-0024): este mesmo caminho roda no boot do gateway.
    if flag("--migrate") {
        return migrate_only(flag("--reset"));
    }
    // `--seed-staff` continua existindo para uso manual/one-shot (CI, reparo de conta):
    // migra antes, porque num banco novo a tabela de contas ainda não existe.
    if flag("--seed-staff") {
        let migrated = migrate_only(false);
        return if migrated == ExitCode::SUCCESS {
            seed_staff()
        } else {
            migrated
        };
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

/// Portas de banco do roteador: o gateway é **síncrono** (ADR-0006), então cada leitura
/// roda no runtime que o binário já mantém e o futuro chega pronto ao `poll_once`.
///
/// Sem este adaptador o roteador polaria um futuro `sqlx` fora de contexto de runtime e o
/// processo morria com "this functionality requires a Tokio context" ao abrir qualquer página
/// do portal que lê conta/personagem (`panel/self`, `panel/characters`, `panel/rankings`).
struct Blocking {
    inner: PgBackend,
    runtime: std::sync::Arc<tokio::runtime::Runtime>,
}

impl Blocking {
    fn new(inner: PgBackend, runtime: std::sync::Arc<tokio::runtime::Runtime>) -> Self {
        Self { inner, runtime }
    }
}

/// Roda a operação no runtime e devolve o resultado já pronto.
fn run_on<T: Send>(
    runtime: &std::sync::Arc<tokio::runtime::Runtime>,
    future: impl std::future::Future<Output = T>,
) -> T {
    runtime.block_on(future)
}

impl Accounts for Blocking {
    fn find_by_login(
        &self,
        login: &str,
    ) -> impl std::future::Future<Output = Result<Account, DbError>> + Send {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        let login = login.to_string();
        async move { run_on(&rt, Accounts::find_by_login(&inner, &login)) }
    }

    fn find_by_id(
        &self,
        id: i64,
    ) -> impl std::future::Future<Output = Result<Account, DbError>> + Send {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        async move { run_on(&rt, Accounts::find_by_id(&inner, id)) }
    }

    fn pin_hash(
        &self,
        login: &str,
    ) -> impl std::future::Future<Output = Result<Option<String>, DbError>> + Send {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        let login = login.to_string();
        async move { run_on(&rt, Accounts::pin_hash(&inner, &login)) }
    }

    fn create_account(
        &self,
        login: &str,
        role: Role,
    ) -> impl std::future::Future<Output = Result<Account, DbError>> + Send {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        let login = login.to_string();
        async move { run_on(&rt, Accounts::create_account(&inner, &login, role)) }
    }

    fn moderation_state<'a>(
        &'a self,
        login: &'a str,
    ) -> std::pin::Pin<
        Box<dyn std::future::Future<Output = Result<AccountModeration, DbError>> + Send + 'a>,
    > {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        let login = login.to_string();
        Box::pin(async move { run_on(&rt, Accounts::moderation_state(&inner, &login)) })
    }

    fn set_role<'a>(
        &'a self,
        login: &'a str,
        role: Role,
    ) -> std::pin::Pin<
        Box<dyn std::future::Future<Output = Result<AccountModeration, DbError>> + Send + 'a>,
    > {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        let login = login.to_string();
        Box::pin(async move { run_on(&rt, Accounts::set_role(&inner, &login, role)) })
    }

    fn set_blocked<'a>(
        &'a self,
        login: &'a str,
        blocked: bool,
        reason: &'a str,
        by: i64,
    ) -> std::pin::Pin<
        Box<dyn std::future::Future<Output = Result<AccountModeration, DbError>> + Send + 'a>,
    > {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        let login = login.to_string();
        let reason = reason.to_string();
        Box::pin(async move {
            run_on(
                &rt,
                Accounts::set_blocked(&inner, &login, blocked, &reason, by),
            )
        })
    }
}

impl AuditTrail for Blocking {
    fn record<'a>(
        &'a self,
        entry: &'a NewAuditEntry,
    ) -> std::pin::Pin<Box<dyn std::future::Future<Output = Result<(), DbError>> + Send + 'a>> {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        Box::pin(async move { run_on(&rt, AuditTrail::record(&inner, entry)) })
    }

    fn recent(
        &self,
        limit: usize,
    ) -> std::pin::Pin<
        Box<dyn std::future::Future<Output = Result<Vec<AuditRow>, DbError>> + Send + '_>,
    > {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        Box::pin(async move { run_on(&rt, AuditTrail::recent(&inner, limit)) })
    }
}

impl wyd_admin::GrantTrail for Blocking {
    fn list_for_character(
        &self,
        character_id: i64,
        limit: usize,
    ) -> Result<Vec<wyd_db::QueuedGrantRow>, DbError> {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        run_on(
            &rt,
            wyd_db::Grants::list_for_character(&inner, character_id, limit),
        )
    }
}

impl wyd_admin::GrantQueue for Blocking {
    fn enqueue(&self, key: &str, by: i64, grant: &wyd_db::Grant) -> Result<(i64, bool), DbError> {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        let key = key.to_string();
        let grant = grant.clone();
        run_on(&rt, wyd_db::Grants::enqueue(&inner, &key, by, &grant))
    }
}

impl Economy for Blocking {
    fn summary(
        &self,
    ) -> std::pin::Pin<
        Box<dyn std::future::Future<Output = Result<EconomySummary, DbError>> + Send + '_>,
    > {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        Box::pin(async move { run_on(&rt, Economy::summary(&inner)) })
    }
}

impl Characters for Blocking {
    fn list_by_account(
        &self,
        account_id: i64,
    ) -> impl std::future::Future<Output = Result<Vec<wyd_db::Character>, DbError>> + Send {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        async move { run_on(&rt, Characters::list_by_account(&inner, account_id)) }
    }

    fn top_by_level(
        &self,
        limit: usize,
    ) -> impl std::future::Future<Output = Result<Vec<wyd_db::Character>, DbError>> + Send {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        async move { run_on(&rt, Characters::top_by_level(&inner, limit)) }
    }

    fn search_by_name(
        &self,
        term: &str,
        limit: usize,
    ) -> impl std::future::Future<Output = Result<Vec<wyd_db::Character>, DbError>> + Send {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        let term = term.to_string();
        async move { run_on(&rt, Characters::search_by_name(&inner, &term, limit)) }
    }

    fn create_character(
        &self,
        character: &wyd_db::Character,
    ) -> impl std::future::Future<Output = Result<wyd_db::Character, DbError>> + Send {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        let character = character.clone();
        async move { run_on(&rt, Characters::create_character(&inner, &character)) }
    }

    fn soft_delete(
        &self,
        id: i64,
    ) -> impl std::future::Future<Output = Result<(), DbError>> + Send {
        let rt = std::sync::Arc::clone(&self.runtime);
        let inner = self.inner.clone();
        async move { run_on(&rt, Characters::soft_delete(&inner, id)) }
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

    // Schema no boot (ADR-0024): banco novo é migrado aqui, restart é no-op.
    // O advisory lock do runner serializa com o canal subindo ao mesmo tempo.
    let report = runtime
        .block_on(wyd_db::migrate::run(&config.database_url))
        .map_err(|error| format!("schema: {error}"))?;
    log("info", "schema", &report.to_string());

    // Seed de staff depois do schema, no mesmo boot e idempotente (ADR-0024 §5).
    let seeded = seed_staff();
    if seeded != ExitCode::SUCCESS {
        return Err("seed do staff falhou (ver log staff_seed)".to_string());
    }

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
        // Senha primeiro; bloqueio depois. Assim o `blocked` só é revelado a quem
        // acertou a credencial — recusa de senha continua sem vazar existência.
        if !wyd_admin::verify_password(password, &record.password_hash) {
            return None;
        }
        Some(wyd_admin::transport::LoginOk {
            account_id: record.account.id,
            login: record.account.login,
            role: record.account.role,
            blocked: record.blocked,
        })
    });

    // Presença por canal: uma consulta por canal (nunca agregado do grupo), no
    // banco que o servidor de jogo mantém. O roteador só autentica e audita.
    let presence_backend = backend.clone();
    let presence_runtime = std::sync::Arc::clone(&runtime);
    let presence: wyd_admin::transport::PresenceCheck = std::sync::Arc::new(move || {
        let canais = presence_runtime
            .block_on(presence_backend.server_channels())
            .map_err(|_| "server_channel indisponível".to_string())?;
        let mut lista = Vec::new();
        for canal in canais.into_iter().filter(|c| c.enabled) {
            let (Ok(grupo), Ok(numero)) =
                (i16::try_from(canal.group_id), i16::try_from(canal.channel))
            else {
                continue;
            };
            // `channel = 0` é a linha de status do grupo, não um mundo.
            if numero <= 0 {
                continue;
            }
            let online = presence_runtime
                .block_on(presence_backend.live_sessions(grupo, numero))
                .map_err(|_| "character_session indisponível".to_string())?;
            lista.push(wyd_admin::ChannelPresence {
                group_id: canal.group_id,
                channel: canal.channel,
                host: canal.host,
                online,
            });
        }
        Ok(lista)
    });

    // Os dois ports usam o mesmo pool, mas cada futuro roda no runtime antes do
    // `poll_once` do roteador (ver `Blocking`).
    let accounts = Blocking::new(backend.clone(), std::sync::Arc::clone(&runtime));
    let characters = Blocking::new(backend.clone(), std::sync::Arc::clone(&runtime));
    // Trilha e economia usam o mesmo adaptador síncrono: o roteador não tem runtime.
    let trilha: std::sync::Arc<dyn AuditTrail + Send + Sync> = std::sync::Arc::new(Blocking::new(
        backend.clone(),
        std::sync::Arc::clone(&runtime),
    ));
    let economia: std::sync::Arc<dyn Economy + Send + Sync> = std::sync::Arc::new(Blocking::new(
        backend.clone(),
        std::sync::Arc::clone(&runtime),
    ));
    let sink = PersistentSink::new(backend.clone(), std::sync::Arc::clone(&runtime));
    let router = Router::new(accounts, characters, sink, tokens)
        .with_catalog(catalog)
        .with_login_check(login)
        .with_presence_check(presence)
        .with_audit_trail(trilha)
        .with_economy(economia)
        // Fila durável de grants (migration 0032): o portal enfileira aqui e o
        // servidor de jogo drena. Sem isto `grants/request` responde
        // "fila de entrega não conectada" — estado antigo da fase 07-4.
        .with_grant_queue(std::sync::Arc::new(Blocking::new(
            backend.clone(),
            std::sync::Arc::clone(&runtime),
        )))
        // Mesma conexão para a leitura da fila na tela de entregas.
        .with_grant_trail(std::sync::Arc::new(Blocking::new(
            backend.clone(),
            std::sync::Arc::clone(&runtime),
        )))
        // Troca de PIN do portal: mesma conexão CPSock do cliente (`0x0FDE` em duas
        // etapas). Sem `WYD_GAME_ADDR` o canal local do compose é o padrão.
        .with_game_address(
            std::env::var("WYD_GAME_ADDR")
                .ok()
                .filter(|value| !value.trim().is_empty())
                .unwrap_or_else(|| "127.0.0.1:8281".to_string()),
        );
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
fn handle<A, C, S>(
    stream: TcpStream,
    router: &Router<A, C, S>,
    metrics: &GatewayMetrics,
    backend: &PgBackend,
    runtime: &tokio::runtime::Runtime,
) where
    A: Accounts,
    C: Characters,
    S: AuditSink,
{
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

/// Decisão do seed: manter a credencial ou regravá-la.
#[derive(Debug, PartialEq, Eq)]
enum SeedPlan {
    Keep,
    Write,
}

/// Só mantém quando a conta é `admin` e o hash realmente autentica a senha.
fn plan_seed(existing: Option<(Role, &str)>, password: &str) -> SeedPlan {
    match existing {
        Some((Role::Admin, hash)) if wyd_admin::verify_password(password, hash) => SeedPlan::Keep,
        _ => SeedPlan::Write,
    }
}

/// Conteúdo do `.env` atual, quando o compose o monta em `WYD_ENV_FILE`.
///
/// Arquivo configurado e ilegível é erro: cair para o ambiente antigo do
/// container reintroduziria exatamente a senha defasada que este caminho evita.
fn env_file_text() -> Result<Option<String>, String> {
    let Some(path) = std::env::var("WYD_ENV_FILE")
        .ok()
        .map(|value| value.trim().to_string())
        .filter(|value| !value.is_empty())
    else {
        return Ok(None);
    };
    std::fs::read_to_string(path)
        .map(Some)
        .map_err(|_| "WYD_ENV_FILE definido mas ilegível".to_string())
}

/// Lê uma chave de um `.env` simples (`KEY=valor`, aspas opcionais).
///
/// Devolve `Some("")` quando a chave existe vazia e `None` quando está ausente;
/// o chamador distingue os dois casos para não voltar ao ambiente antigo.
fn parse_dotenv(text: &str, name: &str) -> Option<String> {
    for raw in text.lines() {
        let mut line = raw.trim();
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        if let Some(rest) = line.strip_prefix("export ") {
            line = rest.trim_start();
        }
        let Some((key, raw_value)) = line.split_once('=') else {
            continue;
        };
        if key.trim() != name {
            continue;
        }
        let value = raw_value.trim();
        let (value, quoted) = if value.len() >= 2
            && ((value.starts_with('"') && value.ends_with('"'))
                || (value.starts_with('\'') && value.ends_with('\'')))
        {
            (&value[1..value.len() - 1], true)
        } else {
            (value, false)
        };
        let value = if quoted {
            value
        } else {
            value.split(" #").next().unwrap_or(value).trim_end()
        };
        return Some(value.to_string());
    }
    None
}

/// Cria/atualiza a conta de staff a partir de `PORTAL_ADMIN_*`.
///
/// Roda em todo boot do gateway (e em `--seed-staff`). Quando `WYD_ENV_FILE`
/// aponta para o `.env` montado, lê a senha **atual** do arquivo em qualquer
/// reinício, inclusive `docker compose restart`; sem o mount, usa o ambiente.
/// Se a conta já existe com papel `admin` e a senha confere com o hash do banco,
/// não escreve nada. Caso contrário, gera argon2id e faz upsert, depois relê e
/// confirma o hash. Sem senha configurada e sem `WYD_SEED_STAFF=1`, não faz nada;
/// com o flag e senha ausente, falha — seed pedido e não aplicado é erro.
/// `--migrate`: aplica a série e sai; `--reset` recria o schema `public` (dev).
fn migrate_only(reset: bool) -> ExitCode {
    let Some(database_url) = std::env::var("DATABASE_URL")
        .ok()
        .filter(|value| !value.trim().is_empty())
    else {
        log("error", "schema", "reason=DATABASE_URL ausente");
        return ExitCode::from(2);
    };
    let runtime = match tokio::runtime::Builder::new_multi_thread()
        .worker_threads(1)
        .enable_all()
        .build()
    {
        Ok(runtime) => runtime,
        Err(_) => return ExitCode::from(2),
    };
    let dir = wyd_db::migrate::dir_from_env();
    match runtime.block_on(wyd_db::migrate::run_dir(&database_url, &dir, reset)) {
        Ok(report) => {
            log("info", "schema", &report.to_string());
            ExitCode::SUCCESS
        }
        Err(error) => {
            log("error", "schema", &format!("reason={error}"));
            ExitCode::from(2)
        }
    }
}

fn seed_staff() -> ExitCode {
    let arquivo = match env_file_text() {
        Ok(text) => text,
        Err(reason) => {
            log("error", "staff_seed", &format!("reason={reason}"));
            return ExitCode::from(2);
        }
    };
    // Com arquivo montado ele é a autoridade; sem mount, o ambiente do processo.
    let valor = |nome: &str| match arquivo.as_deref() {
        Some(texto) => parse_dotenv(texto, nome),
        None => std::env::var(nome).ok(),
    };
    let obrigatorio = valor("WYD_SEED_STAFF").as_deref() == Some("1");
    let user = valor("PORTAL_ADMIN_USER")
        .filter(|value| !value.trim().is_empty())
        .unwrap_or_else(|| "admin".to_string());
    let senha = valor("PORTAL_ADMIN_PASSWORD").filter(|value| !value.trim().is_empty());

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
    let atual = runtime.block_on(wyd_db::Accounts::credentials(&backend, &user));
    let plano = match &atual {
        Ok(record) => plan_seed(Some((record.account.role, &record.password_hash)), &senha),
        Err(DbError::NotFoundKey { .. }) => plan_seed(None, &senha),
        Err(error) => {
            log("error", "staff_seed", &format!("reason={error}"));
            return ExitCode::from(2);
        }
    };
    if plano == SeedPlan::Keep {
        log(
            "info",
            "staff_seed",
            &format!("login={user} changed=false verified=true"),
        );
        return ExitCode::SUCCESS;
    }

    let hash = match wyd_admin::hash_password(&senha) {
        Ok(hash) => hash,
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
            // Relê o banco: não basta o upsert ter sucesso, o hash gravado tem
            // que autenticar a senha do `.env` neste boot.
            let verificado = runtime
                .block_on(wyd_db::Accounts::credentials(&backend, &user))
                .map(|record| wyd_admin::verify_password(&senha, &record.password_hash))
                .unwrap_or(false);
            if !verificado {
                log(
                    "error",
                    "staff_seed",
                    "reason=hash_do_banco_nao_confere_com_PORTAL_ADMIN_PASSWORD",
                );
                return ExitCode::from(2);
            }
            log(
                "info",
                "staff_seed",
                &format!(
                    "login={} id={} role={} changed=true verified=true",
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

/// Destino de auditoria do serviço: `audit_log` no Postgres **e** stdout.
///
/// A trilha do banco é o que o painel lê (FASE-07-3); o stdout continua existindo
/// para `docker logs`/Loki. Falha ao gravar no banco não engole a ação — fica
/// registrada como erro no log do processo (append-only, sem retry inventado).
struct PersistentSink {
    backend: PgBackend,
    runtime: std::sync::Arc<tokio::runtime::Runtime>,
}

impl PersistentSink {
    fn new(backend: PgBackend, runtime: std::sync::Arc<tokio::runtime::Runtime>) -> Self {
        Self { backend, runtime }
    }
}

impl AuditSink for PersistentSink {
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
        let linha = NewAuditEntry {
            actor_id: entry.actor_id,
            actor_role: entry.actor_role,
            action: entry.action.clone(),
            target: entry.target.clone(),
            detail: entry.detail.clone(),
        };
        if let Err(error) = self
            .runtime
            .block_on(AuditTrail::record(&self.backend, &linha))
        {
            log(
                "error",
                "audit_write_failed",
                &format!("action={} reason={error}", entry.action),
            );
        }
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

    #[test]
    fn dotenv_lookup_handles_quotes_comments_and_empty_values() {
        let text = "# comentário\nexport PORTAL_ADMIN_USER='admin'\nPORTAL_ADMIN_PASSWORD=\"senha # 123\"\nWYD_SEED_STAFF=0 # dev\nVAZIO=\n";
        assert_eq!(
            parse_dotenv(text, "PORTAL_ADMIN_USER").as_deref(),
            Some("admin")
        );
        assert_eq!(
            parse_dotenv(text, "PORTAL_ADMIN_PASSWORD").as_deref(),
            Some("senha # 123")
        );
        assert_eq!(parse_dotenv(text, "WYD_SEED_STAFF").as_deref(), Some("0"));
        assert_eq!(parse_dotenv(text, "VAZIO").as_deref(), Some(""));
        assert_eq!(parse_dotenv(text, "AUSENTE"), None);
    }

    #[test]
    fn seed_keeps_only_matching_admin_hash() {
        let password = "senha-de-dev-123";
        let hash = wyd_admin::hash_password(password).expect("hash");
        assert_eq!(
            plan_seed(Some((Role::Admin, &hash)), password),
            SeedPlan::Keep
        );
        let outra = wyd_admin::hash_password("outra-senha-123").expect("hash");
        assert_eq!(
            plan_seed(Some((Role::Admin, &outra)), password),
            SeedPlan::Write
        );
        assert_eq!(
            plan_seed(Some((Role::Player, &hash)), password),
            SeedPlan::Write
        );
        assert_eq!(plan_seed(None, password), SeedPlan::Write);
    }
}
