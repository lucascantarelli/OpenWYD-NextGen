//! Contrato do transporte Cold Path (ADR-0008): rota, auth, envelope e limites.
//!
//! Prova o que o portal depende: `POST /admin/v1/{servico}/{metodo}`, token de
//! staff, `AdminResult` no corpo, `401`/`404`/`405`/`413` de transporte e
//! auditoria por chamada — tudo em loopback, sem `.proto`.

use std::collections::HashMap;
use std::future::Future;
use std::io::{Cursor, Read, Write};
use std::net::TcpStream;
use std::sync::Mutex;

use wyd_admin::transport::{http, json, serve_conn, Router, StaticTokens};
use wyd_admin::{DeliveryQueue, Role, Session, VecSink};
use wyd_db::{Account, Accounts, Character, Characters, DbError};

struct FakeAccounts {
    by_login: Mutex<HashMap<String, Account>>,
}

impl FakeAccounts {
    fn new() -> Self {
        Self {
            by_login: Mutex::new(HashMap::from([(
                "alvo".to_string(),
                Account {
                    id: 9,
                    login: "alvo".to_string(),
                    role: Role::Player,
                },
            )])),
        }
    }
}

impl Accounts for FakeAccounts {
    fn find_by_login(&self, login: &str) -> impl Future<Output = Result<Account, DbError>> + Send {
        let found = self
            .by_login
            .lock()
            .expect("fake acessível")
            .get(login)
            .cloned();
        async move {
            found.ok_or(DbError::NotFound {
                entity: "account",
                id: 0,
            })
        }
    }

    fn find_by_id(&self, id: i64) -> impl Future<Output = Result<Account, DbError>> + Send {
        let found = self
            .by_login
            .lock()
            .expect("fake acessível")
            .values()
            .find(|account| account.id == id)
            .cloned();
        async move {
            found.ok_or(DbError::NotFound {
                entity: "account",
                id,
            })
        }
    }

    fn create(
        &self,
        login: &str,
        role: Role,
    ) -> impl Future<Output = Result<Account, DbError>> + Send {
        let account = Account {
            id: 99,
            login: login.to_string(),
            role,
        };
        async move { Ok(account) }
    }
}

struct FakeCharacters {
    rows: Vec<Character>,
}

impl FakeCharacters {
    fn new() -> Self {
        let character = |id: i64, account_id: i64, slot: i64, name: &str, level: i64| Character {
            id,
            account_id,
            slot,
            name: name.to_string(),
            class: 1,
            level,
            exp: 0,
            hp: 100,
            mp: 100,
            pos_x: 0,
            pos_y: 0,
        };
        Self {
            rows: vec![
                character(1, 5, 0, "Baixo", 40),
                character(2, 5, 1, "Alto", 90),
                character(3, 9, 0, "Alvo", 70),
            ],
        }
    }
}

impl Characters for FakeCharacters {
    fn list_by_account(
        &self,
        account_id: i64,
    ) -> impl Future<Output = Result<Vec<Character>, DbError>> + Send {
        let rows: Vec<Character> = self
            .rows
            .iter()
            .filter(|c| c.account_id == account_id)
            .cloned()
            .collect();
        async move { Ok(rows) }
    }

    fn top_by_level(
        &self,
        limit: usize,
    ) -> impl Future<Output = Result<Vec<Character>, DbError>> + Send {
        let mut rows = self.rows.clone();
        rows.sort_by(|a, b| b.level.cmp(&a.level).then_with(|| a.name.cmp(&b.name)));
        rows.truncate(limit);
        async move { Ok(rows) }
    }

    fn create(
        &self,
        character: &Character,
    ) -> impl Future<Output = Result<Character, DbError>> + Send {
        let created = character.clone();
        async move { Ok(created) }
    }

    async fn soft_delete(&self, _id: i64) -> Result<(), DbError> {
        Ok(())
    }
}

fn tokens() -> StaticTokens {
    let mut tokens = StaticTokens::default();
    tokens.insert(
        "staff",
        Session {
            account_id: 7,
            role: Role::Moderator,
        },
    );
    tokens.insert(
        "admin",
        Session {
            account_id: 8,
            role: Role::Admin,
        },
    );
    tokens.insert(
        "player",
        Session {
            account_id: 5,
            role: Role::Player,
        },
    );
    tokens
}

fn router() -> Router<FakeAccounts, FakeCharacters, VecSink> {
    Router::new(
        FakeAccounts::new(),
        FakeCharacters::new(),
        VecSink::default(),
        tokens(),
    )
    .with_queue(DeliveryQueue::bounded(8).expect("fila válida"))
}

fn raw(method: &str, path: &str, token: Option<&str>, body: &str) -> String {
    let mut request = format!("{method} {path} HTTP/1.1\r\nhost: loopback\r\nx-trace-id: t-1\r\n");
    if let Some(token) = token {
        request.push_str(&format!("authorization: Bearer {token}\r\n"));
    }
    let length = body.len();
    request.push_str(&format!("content-length: {length}\r\n\r\n{body}"));
    request
}

fn dispatch(
    router: &Router<FakeAccounts, FakeCharacters, VecSink>,
    request: &str,
) -> (u16, json::Json) {
    let mut cursor = Cursor::new(request.as_bytes().to_vec());
    let parsed = http::read_request(&mut cursor, 1 << 20).expect("request válida");
    let reply = router.dispatch(&parsed);
    let body = json::Json::parse(&reply.body).expect("corpo é JSON");
    (reply.status, body)
}

fn result_of(body: &json::Json) -> i64 {
    body.get("result")
        .and_then(json::Json::as_i64)
        .expect("result")
}

#[test]
fn no_token_is_401_and_does_not_audit() {
    let router = router();
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/accounts/find",
            None,
            r#"{"login":"alvo"}"#,
        ),
    );
    assert_eq!(status, 401);
    assert!(body.get("message").is_some());
    assert!(router.audit().entries().is_empty(), "401 não tem autor");
}

#[test]
fn player_is_business_forbidden_and_denied_is_audited() {
    let router = router();
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/accounts/find",
            Some("player"),
            r#"{"login":"alvo"}"#,
        ),
    );
    assert_eq!(status, 200, "recusa de negócio viaja no corpo");
    assert_eq!(result_of(&body), 2);
    let entries = router.audit().entries();
    assert_eq!(entries.len(), 1);
    assert_eq!(entries[0].action, "auth.denied:accounts.find");
    assert_eq!(entries[0].actor_id, 5);
}

#[test]
fn staff_finds_account_with_envelope_and_trace() {
    let router = router();
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/accounts/find",
            Some("staff"),
            r#"{"login":"alvo"}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    assert_eq!(
        body.get("account")
            .and_then(|a| a.get("login"))
            .and_then(json::Json::as_str),
        Some("alvo")
    );
    assert_eq!(
        body.get("trace_id").and_then(json::Json::as_str),
        Some("t-1")
    );
    assert_eq!(router.audit().entries()[0].action, "accounts.find");
}

#[test]
fn moderator_cannot_create_staff() {
    let router = router();
    let (status, body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/accounts/create",
            Some("staff"),
            r#"{"login":"novo","role":"admin"}"#,
        ),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 2);
    let (_, ok_body) = dispatch(
        &router,
        &raw(
            "POST",
            "/admin/v1/accounts/create",
            Some("admin"),
            r#"{"login":"novo","role":"player"}"#,
        ),
    );
    assert_eq!(result_of(&ok_body), 1);
}

#[test]
fn grant_is_idempotent_across_calls() {
    let router = router();
    let body = r#"{"idempotency_key":"op-1","character_id":9,"kind":{"type":"gold","amount":100}}"#;
    let (status, first) = dispatch(
        &router,
        &raw("POST", "/admin/v1/grants/request", Some("staff"), body),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&first), 1);
    assert_eq!(
        first.get("replay").and_then(json::Json::as_bool),
        Some(false)
    );
    let (_, second) = dispatch(
        &router,
        &raw("POST", "/admin/v1/grants/request", Some("staff"), body),
    );
    assert_eq!(
        second.get("replay").and_then(json::Json::as_bool),
        Some(true)
    );
    assert_eq!(
        second.get("grant_id").and_then(json::Json::as_i64),
        first.get("grant_id").and_then(json::Json::as_i64)
    );
}

#[test]
fn invalid_params_are_business_invalid_without_touching_repo() {
    let router = router();
    let (status, body) = dispatch(
        &router,
        &raw("POST", "/admin/v1/grants/request", Some("staff"), r#"{}"#),
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 3);
    assert!(
        router.audit().entries().is_empty(),
        "validação falha antes de tocar repo/auditoria"
    );
}

#[test]
fn unknown_method_is_404_and_broken_json_is_400() {
    let router = router();
    let (status, _) = dispatch(
        &router,
        &raw("POST", "/admin/v1/nao/existe", Some("staff"), "{}"),
    );
    assert_eq!(status, 404);
    let (status, _) = dispatch(
        &router,
        &raw("POST", "/admin/v1/accounts/find", Some("staff"), "{"),
    );
    assert_eq!(status, 400);
}

#[test]
fn get_is_405_and_big_body_is_413() {
    let mut cursor = Cursor::new(raw("GET", "/admin/v1/accounts/find", None, "").into_bytes());
    assert_eq!(
        http::read_request(&mut cursor, 1024)
            .expect_err("GET não é aceito")
            .status(),
        405
    );

    let big = raw("POST", "/admin/v1/accounts/find", None, &"x".repeat(64));
    let mut cursor = Cursor::new(big.into_bytes());
    assert_eq!(
        http::read_request(&mut cursor, 16)
            .expect_err("corpo grande não é lido")
            .status(),
        413
    );
}

#[test]
fn loopback_socket_roundtrip_is_no_store() {
    let router = router();
    let listener = http::bind_loopback(0).expect("porta efêmera");
    let addr = listener.local_addr().expect("endereço local");
    std::thread::scope(|scope| {
        scope.spawn(|| {
            let (stream, _) = listener.accept().expect("aceita conexão");
            serve_conn(stream, &router);
        });
        let mut client = TcpStream::connect(addr).expect("conecta");
        client
            .write_all(
                raw(
                    "POST",
                    "/admin/v1/accounts/find",
                    Some("staff"),
                    r#"{"login":"alvo"}"#,
                )
                .as_bytes(),
            )
            .expect("envia");
        let mut response = Vec::new();
        client.read_to_end(&mut response).expect("lê até EOF");
        let text = String::from_utf8(response).expect("utf-8");
        assert!(text.starts_with("HTTP/1.1 200 OK"), "{text}");
        assert!(text.contains("cache-control: no-store"));
        assert!(text.contains("x-trace-id: t-1"));
        assert!(text.contains("\"result\":1"));
    });
}
