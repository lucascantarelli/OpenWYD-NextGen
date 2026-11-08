//! Transporte do Cold Path (ADR-0008): HTTP/1.1 + JSON em loopback.
//!
//! Roteia `POST /admin/v1/{servico}/{metodo}` para os endpoints de `api/*`,
//! autentica pelo token de staff, aplica o envelope `AdminResult` e audita toda
//! chamada autenticada. Nunca é importado pelo Hot Path (`wyd-core`/`wyd-net`).
//!
//! Enquanto a ADR-0006 mantém o runtime async diferido, os handlers usam futuros
//! prontos no primeiro poll (`poll_once`); um futuro pendente responde `500` em
//! vez de travar a conexão.

pub mod http;
pub mod json;

use std::future::Future;
use std::io::BufReader;
use std::net::{TcpListener, TcpStream};
use std::task::{Context, Poll, Waker};

use wyd_db::{Accounts, Characters, DeliveryQueue, Role};

use crate::api::{
    create_account, find_account, own_account, own_characters, request_grant, request_recovery,
    top_rankings, AccountSummary, GrantKind, GrantRequest, PanelAccount, PanelCharacter,
    RankingEntry,
};
use crate::audit::AuditSink;
use crate::auth::Session;
use crate::error::AdminError;
use crate::service::GrantService;

/// Porta do contrato no loopback (ADR-0008).
pub const ADMIN_PORT: u16 = 7600;

/// `AdminResult::OK` do portal (`admin-result.ts`).
const RESULT_OK: u8 = 1;
/// `AdminResult::FORBIDDEN`.
const RESULT_FORBIDDEN: u8 = 2;
/// `AdminResult::INVALID`.
const RESULT_INVALID: u8 = 3;
/// `AdminResult::NOT_FOUND`.
const RESULT_NOT_FOUND: u8 = 4;
/// `AdminResult::CONTENT_OWNED`.
const RESULT_CONTENT_OWNED: u8 = 5;

/// Verificação do token de sessão de staff (ADR-0008 §2).
pub trait Authenticator {
    /// Devolve a sessão para um token válido; `None` é `401`.
    fn authenticate(&self, token: &str) -> Option<Session>;
}

/// Registro explícito token → sessão (dev/testes).
///
/// A verificação HMAC do JWT emitido pelo portal entra na fiação do BFF; o
/// transporte não decide sozinho quem é staff.
#[derive(Debug, Default)]
pub struct StaticTokens {
    entries: Vec<(String, Session)>,
}

impl StaticTokens {
    /// Registra um token válido para a sessão indicada.
    pub fn insert(&mut self, token: impl Into<String>, session: Session) {
        self.entries.push((token.into(), session));
    }
}

impl Authenticator for StaticTokens {
    fn authenticate(&self, token: &str) -> Option<Session> {
        self.entries
            .iter()
            .find(|(known, _)| known == token)
            .map(|(_, session)| *session)
    }
}

/// Resposta pronta para o socket.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Reply {
    pub status: u16,
    pub body: String,
    pub trace_id: String,
}

impl Reply {
    fn ok(trace: &str, extras: &str) -> Self {
        let quoted = json::quote(trace);
        Self {
            status: 200,
            body: format!("{{\"result\":{RESULT_OK}{extras},\"trace_id\":{quoted}}}"),
            trace_id: trace.to_string(),
        }
    }

    fn business(trace: &str, result: u8, message: &str) -> Self {
        let quoted = json::quote(trace);
        let message = json::quote(message);
        Self {
            status: 200,
            body: format!("{{\"result\":{result},\"message\":{message},\"trace_id\":{quoted}}}"),
            trace_id: trace.to_string(),
        }
    }

    fn transport(status: u16, trace: &str, message: &str) -> Self {
        let quoted = json::quote(trace);
        let message = json::quote(message);
        Self {
            status,
            body: format!("{{\"message\":{message},\"trace_id\":{quoted}}}"),
            trace_id: trace.to_string(),
        }
    }

    fn from_error(trace: &str, error: &AdminError) -> Self {
        if matches!(error, AdminError::Unauthorized) {
            return Self::transport(401, trace, "sessão de staff ausente ou inválida");
        }
        Self::business(trace, admin_result(error), &error.to_string())
    }
}

const fn admin_result(error: &AdminError) -> u8 {
    match error {
        AdminError::Forbidden { .. } => RESULT_FORBIDDEN,
        AdminError::InvalidInput { .. } => RESULT_INVALID,
        AdminError::NotFound { .. } => RESULT_NOT_FOUND,
        AdminError::Conflict { .. } => RESULT_CONTENT_OWNED,
        AdminError::Unauthorized => 0,
    }
}

/// Roteador do contrato: `servico/metodo` → `api/*`.
#[derive(Debug)]
pub struct Router<A, C, S> {
    accounts: A,
    characters: C,
    audit: S,
    grants: GrantService,
    tokens: StaticTokens,
    max_body: usize,
}

impl<A: Accounts, C: Characters, S: AuditSink> Router<A, C, S> {
    /// Monta o roteador com as portas de persistência e auditoria.
    #[must_use]
    pub fn new(accounts: A, characters: C, audit: S, tokens: StaticTokens) -> Self {
        Self {
            accounts,
            characters,
            audit,
            grants: GrantService::new(),
            tokens,
            max_body: http::MAX_BODY,
        }
    }

    /// Limite de corpo (o teste usa valor pequeno para provar o `413`).
    #[must_use]
    pub fn with_max_body(mut self, max_body: usize) -> Self {
        self.max_body = max_body;
        self
    }

    /// Trilha de auditoria (testes leem/verificam o que foi registrado).
    #[must_use]
    pub fn audit(&self) -> &S {
        &self.audit
    }

    /// Conecta a fila de entrega dos grants (uma vez, na configuração).
    #[must_use]
    pub fn with_queue(self, queue: DeliveryQueue) -> Self {
        self.grants.attach_queue(queue);
        self
    }

    #[must_use]
    pub fn max_body(&self) -> usize {
        self.max_body
    }

    /// Resolve uma requisição já lida (auth → rota → handler → envelope).
    #[must_use]
    pub fn dispatch(&self, request: &http::Request) -> Reply {
        let trace = request.trace_id();
        if request.method != "POST" {
            return Reply::transport(405, trace, "método não permitido");
        }
        let Some(token) = bearer(request.header("authorization")) else {
            return Reply::transport(401, trace, "sessão de staff ausente");
        };
        let Some(session) = self.tokens.authenticate(token) else {
            return Reply::transport(401, trace, "sessão de staff inválida");
        };
        let Ok(body) = std::str::from_utf8(&request.body) else {
            return Reply::transport(400, trace, "corpo não é UTF-8");
        };
        let Ok(params) = json::Json::parse(body) else {
            return Reply::transport(400, trace, "JSON inválido");
        };
        let Some((service, method)) = route(&request.path) else {
            return Reply::transport(404, trace, "método desconhecido");
        };
        match (service, method) {
            ("accounts", "find") => self.accounts_find(session, &params, trace),
            ("accounts", "create") => self.accounts_create(session, &params, trace),
            ("accounts", "recovery") => self.accounts_recovery(session, &params, trace),
            ("grants", "request") => self.grants_request(session, &params, trace),
            ("panel", "self") => self.panel_self(session, &params, trace),
            ("panel", "characters") => self.panel_characters(session, &params, trace),
            ("panel", "rankings") => self.panel_rankings(session, &params, trace),
            _ => Reply::transport(404, trace, "método desconhecido"),
        }
    }

    fn accounts_recovery(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(login) = params.get("login").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo login obrigatório");
        };
        match request_recovery(Some(session), login, &self.audit) {
            Ok(()) => Reply::ok(trace, ",\"requested\":true"),
            Err(error) => Reply::from_error(trace, &error),
        }
    }

    fn panel_self(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(account_id) = params.get("account_id").and_then(json::Json::as_i64) else {
            return Reply::business(trace, RESULT_INVALID, "campo account_id obrigatório");
        };
        match poll_once(own_account(
            Some(session),
            account_id,
            &self.accounts,
            &self.audit,
        )) {
            Some(Ok(account)) => Reply::ok(
                trace,
                &format!(",\"account\":{}", panel_account_json(&account)),
            ),
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    fn panel_characters(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(account_id) = params.get("account_id").and_then(json::Json::as_i64) else {
            return Reply::business(trace, RESULT_INVALID, "campo account_id obrigatório");
        };
        match poll_once(own_characters(
            Some(session),
            account_id,
            &self.characters,
            &self.audit,
        )) {
            Some(Ok(characters)) => {
                let lista = characters_json(&characters);
                Reply::ok(trace, &format!(",\"characters\":{lista}"))
            }
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    fn panel_rankings(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let limit = params
            .get("limit")
            .and_then(json::Json::as_i64)
            .unwrap_or(10);
        let limit = usize::try_from(limit).unwrap_or(0);
        match poll_once(top_rankings(
            Some(session),
            limit,
            &self.characters,
            &self.audit,
        )) {
            Some(Ok(entries)) => {
                let lista = rankings_json(&entries);
                Reply::ok(trace, &format!(",\"ranking\":{lista}"))
            }
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    fn accounts_find(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(login) = params.get("login").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo login obrigatório");
        };
        match poll_once(find_account(
            Some(session),
            login,
            &self.accounts,
            &self.audit,
        )) {
            Some(Ok(account)) => {
                let account = account_json(&account);
                Reply::ok(trace, &format!(",\"account\":{account}"))
            }
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    fn accounts_create(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(login) = params.get("login").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo login obrigatório");
        };
        let Some(role) = params.get("role").and_then(json::Json::as_str) else {
            return Reply::business(trace, RESULT_INVALID, "campo role obrigatório");
        };
        let Some(role) = role_from_str(role) else {
            return Reply::business(trace, RESULT_INVALID, "role inválido");
        };
        match poll_once(create_account(
            Some(session),
            login,
            role,
            &self.accounts,
            &self.audit,
        )) {
            Some(Ok(account)) => {
                let account = account_json(&account);
                Reply::ok(trace, &format!(",\"account\":{account}"))
            }
            Some(Err(error)) => Reply::from_error(trace, &error),
            None => Reply::transport(500, trace, "handler pendente de runtime async"),
        }
    }

    fn grants_request(&self, session: Session, params: &json::Json, trace: &str) -> Reply {
        let Some(idempotency_key) = params.get("idempotency_key").and_then(json::Json::as_str)
        else {
            return Reply::business(trace, RESULT_INVALID, "campo idempotency_key obrigatório");
        };
        let Some(character_id) = params.get("character_id").and_then(json::Json::as_i64) else {
            return Reply::business(trace, RESULT_INVALID, "campo character_id obrigatório");
        };
        let Some(kind) = params.get("kind").and_then(grant_kind) else {
            return Reply::business(trace, RESULT_INVALID, "campo kind inválido");
        };
        let request = GrantRequest {
            idempotency_key: idempotency_key.to_string(),
            character_id,
            kind,
        };
        match request_grant(Some(session), request, &self.grants, &self.audit) {
            Ok(receipt) => {
                let key = json::quote(&receipt.idempotency_key);
                let grant_id = receipt.grant_id;
                let replay = receipt.replay;
                let extras = format!(
                    ",\"idempotency_key\":{key},\"grant_id\":{grant_id},\"replay\":{replay}"
                );
                Reply::ok(trace, &extras)
            }
            Err(error) => Reply::from_error(trace, &error),
        }
    }
}

/// Fila de conexões do Cold Path (sequencial: um cliente local, tráfego de admin).
pub fn serve<A: Accounts, C: Characters, S: AuditSink>(
    listener: &TcpListener,
    router: &Router<A, C, S>,
) -> std::io::Result<()> {
    loop {
        match listener.accept() {
            Ok((stream, _)) => serve_conn(stream, router),
            Err(error) => return Err(error),
        }
    }
}

/// Atende uma conexão: lê uma requisição, responde e fecha (`Connection: close`).
pub fn serve_conn<A: Accounts, C: Characters, S: AuditSink>(
    stream: TcpStream,
    router: &Router<A, C, S>,
) {
    let Ok(peer) = stream.peer_addr() else {
        return;
    };
    if !peer.ip().is_loopback() {
        return;
    }
    let Ok(read_half) = stream.try_clone() else {
        return;
    };
    let mut reader = BufReader::new(read_half);
    let mut writer = stream;
    let reply = match http::read_request(&mut reader, router.max_body()) {
        Ok(request) => router.dispatch(&request),
        Err(error) => Reply::transport(error.status(), "", "falha de transporte"),
    };
    let _ = http::write_response(&mut writer, reply.status, &reply.trace_id, &reply.body);
}

/// Executa um futuro que deve estar pronto no primeiro poll (ADR-0006).
fn poll_once<F: Future>(future: F) -> Option<F::Output> {
    let waker: Waker = Waker::noop().clone();
    let mut context = Context::from_waker(&waker);
    let mut pinned = Box::pin(future);
    match pinned.as_mut().poll(&mut context) {
        Poll::Ready(value) => Some(value),
        Poll::Pending => None,
    }
}

fn route(path: &str) -> Option<(&str, &str)> {
    let rest = path.strip_prefix("/admin/v1/")?;
    let (service, method) = rest.split_once('/')?;
    let valid = |segment: &str| {
        !segment.is_empty()
            && segment
                .bytes()
                .all(|b| b.is_ascii_lowercase() || b.is_ascii_digit() || b == b'_')
    };
    if valid(service) && valid(method) {
        Some((service, method))
    } else {
        None
    }
}

fn bearer(header: Option<&str>) -> Option<&str> {
    let (scheme, token) = header?.split_once(' ')?;
    if !scheme.eq_ignore_ascii_case("bearer") {
        return None;
    }
    let token = token.trim();
    if token.is_empty() {
        None
    } else {
        Some(token)
    }
}

fn role_from_str(role: &str) -> Option<Role> {
    match role {
        "player" => Some(Role::Player),
        "moderator" => Some(Role::Moderator),
        "admin" => Some(Role::Admin),
        _ => None,
    }
}

fn grant_kind(value: &json::Json) -> Option<GrantKind> {
    match value.get("type").and_then(json::Json::as_str)? {
        "item" => Some(GrantKind::Item {
            item_id: value.get("item_id").and_then(json::Json::as_i64)?,
            qty: value.get("qty").and_then(json::Json::as_i64)?,
        }),
        "gold" => Some(GrantKind::Gold {
            amount: value.get("amount").and_then(json::Json::as_i64)?,
        }),
        "level" => Some(GrantKind::Level {
            delta: value.get("delta").and_then(json::Json::as_i64)?,
        }),
        _ => None,
    }
}

fn panel_account_json(account: &PanelAccount) -> String {
    json::Json::Obj(vec![
        ("id".to_string(), json::Json::Num(account.id as f64)),
        ("login".to_string(), json::Json::Str(account.login.clone())),
        (
            "role".to_string(),
            json::Json::Str(account.role.as_db().to_string()),
        ),
    ])
    .render()
}

fn characters_json(characters: &[PanelCharacter]) -> String {
    let items = characters
        .iter()
        .map(|c| {
            json::Json::Obj(vec![
                ("id".to_string(), json::Json::Num(c.id as f64)),
                ("slot".to_string(), json::Json::Num(c.slot as f64)),
                ("name".to_string(), json::Json::Str(c.name.clone())),
                ("class".to_string(), json::Json::Num(c.class as f64)),
                ("level".to_string(), json::Json::Num(c.level as f64)),
            ])
        })
        .collect();
    json::Json::Arr(items).render()
}

fn rankings_json(entries: &[RankingEntry]) -> String {
    let items = entries
        .iter()
        .map(|e| {
            json::Json::Obj(vec![
                ("rank".to_string(), json::Json::Num(e.rank as f64)),
                ("name".to_string(), json::Json::Str(e.name.clone())),
                ("class".to_string(), json::Json::Num(e.class as f64)),
                ("level".to_string(), json::Json::Num(e.level as f64)),
            ])
        })
        .collect();
    json::Json::Arr(items).render()
}

fn account_json(account: &AccountSummary) -> String {
    json::Json::Obj(vec![
        ("id".to_string(), json::Json::Num(account.id as f64)),
        ("login".to_string(), json::Json::Str(account.login.clone())),
        (
            "role".to_string(),
            json::Json::Str(account.role.as_db().to_string()),
        ),
    ])
    .render()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn route_accepts_only_contract_shape() {
        assert_eq!(route("/admin/v1/accounts/find"), Some(("accounts", "find")));
        assert_eq!(route("/admin/v1/accounts"), None);
        assert_eq!(route("/admin/v1/accounts/find/extra"), None);
        assert_eq!(route("/admin/v1/Accounts/find"), None);
        assert_eq!(route("/outro/accounts/find"), None);
    }

    #[test]
    fn bearer_is_case_insensitive_and_rejects_empty() {
        assert_eq!(bearer(Some("Bearer abc")), Some("abc"));
        assert_eq!(bearer(Some("bearer abc")), Some("abc"));
        assert_eq!(bearer(Some("Bearer   ")), None);
        assert_eq!(bearer(Some("Basic abc")), None);
        assert_eq!(bearer(None), None);
    }

    #[test]
    fn static_tokens_map_token_to_session() {
        let mut tokens = StaticTokens::default();
        tokens.insert(
            "t",
            Session {
                account_id: 7,
                role: Role::Admin,
            },
        );
        assert_eq!(
            tokens.authenticate("t"),
            Some(Session {
                account_id: 7,
                role: Role::Admin
            })
        );
        assert_eq!(tokens.authenticate("outro"), None);
    }
}
