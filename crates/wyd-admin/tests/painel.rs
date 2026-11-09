//! Contrato do painel do jogador (FASE-07-2): escopo por conta, ranking e recuperação.
//!
//! Prova o que o BFF consome: `POST /admin/v1/panel/*` e `/accounts/recovery` no mesmo
//! roteador do ADR-0008, com envelope, auditoria sem login e recusa tipada. O `account_id`
//! chega do BFF (sessão assinada); aqui ele é o parâmetro que o portal injeta.

use std::collections::HashMap;
use std::future::Future;
use std::io::Cursor;
use std::sync::Mutex;

use wyd_admin::transport::{http, json, Router, StaticTokens};
use wyd_admin::{DeliveryQueue, Role, Session, VecSink};
use wyd_db::{Account, Accounts, Character, Characters, DbError};

struct FakeAccounts {
    rows: Mutex<HashMap<i64, Account>>,
}

impl FakeAccounts {
    fn new() -> Self {
        let mut rows = HashMap::new();
        rows.insert(
            5,
            Account {
                id: 5,
                login: "jogador".to_string(),
                role: Role::Player,
            },
        );
        rows.insert(
            9,
            Account {
                id: 9,
                login: "outro".to_string(),
                role: Role::Player,
            },
        );
        Self {
            rows: Mutex::new(rows),
        }
    }
}

impl Accounts for FakeAccounts {
    fn find_by_login(&self, login: &str) -> impl Future<Output = Result<Account, DbError>> + Send {
        let found = self
            .rows
            .lock()
            .expect("fake acessível")
            .values()
            .find(|a| a.login == login)
            .cloned();
        async move {
            found.ok_or(DbError::NotFound {
                entity: "account",
                id: 0,
            })
        }
    }

    fn find_by_id(&self, id: i64) -> impl Future<Output = Result<Account, DbError>> + Send {
        let found = self.rows.lock().expect("fake acessível").get(&id).cloned();
        async move {
            found.ok_or(DbError::NotFound {
                entity: "account",
                id,
            })
        }
    }

    fn create_account(
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
        let mk = |id: i64, account_id: i64, slot: i64, name: &str, level: i64| Character {
            id,
            account_id,
            slot,
            name: name.to_string(),
            class: 2,
            level,
            exp: 0,
            hp: 100,
            mp: 100,
            pos_x: 0,
            pos_y: 0,
            ..Character::default()
        };
        Self {
            rows: vec![
                mk(1, 5, 0, "Baixo", 40),
                mk(2, 5, 1, "Alto", 90),
                mk(3, 9, 0, "Alvo", 70),
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

    fn create_character(
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

fn dispatch(
    router: &Router<FakeAccounts, FakeCharacters, VecSink>,
    path: &str,
    token: Option<&str>,
    body: &str,
) -> (u16, String, json::Json) {
    let mut request = format!(
        "POST {path} HTTP/1.1\r\nhost: loopback\r\nx-trace-id: t-9\r\ncontent-length: {}\r\n",
        body.len()
    );
    if let Some(token) = token {
        request.push_str(&format!("authorization: Bearer {token}\r\n"));
    }
    request.push_str("\r\n");
    request.push_str(body);
    let mut cursor = Cursor::new(request.into_bytes());
    let parsed = http::read_request(&mut cursor, 1 << 20).expect("request válida");
    let reply = router.dispatch(&parsed);
    let parsed = json::Json::parse(&reply.body).expect("corpo é JSON");
    (reply.status, reply.body, parsed)
}

fn result_of(body: &json::Json) -> i64 {
    body.get("result")
        .and_then(json::Json::as_i64)
        .expect("result")
}

#[test]
fn sem_token_e_401_sem_auditoria() {
    let router = router();
    let (status, _, _) = dispatch(&router, "/admin/v1/panel/self", None, r#"{"account_id":5}"#);
    assert_eq!(status, 401);
    assert!(router.audit().entries().is_empty());
}

#[test]
fn jogador_le_a_propria_conta_no_painel() {
    let router = router();
    let (status, _, body) = dispatch(
        &router,
        "/admin/v1/panel/self",
        Some("player"),
        r#"{"account_id":5}"#,
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    assert_eq!(
        body.get("account")
            .and_then(|a| a.get("login"))
            .and_then(json::Json::as_str),
        Some("jogador")
    );
    let entries = router.audit().entries();
    assert_eq!(entries.len(), 1);
    assert_eq!(entries[0].action, "panel.account");
    assert_eq!(entries[0].target, "account:5");
}

#[test]
fn personagens_ficam_escopados_na_conta_pedida() {
    let router = router();
    let (_, _, body) = dispatch(
        &router,
        "/admin/v1/panel/characters",
        Some("player"),
        r#"{"account_id":5}"#,
    );
    let Some(json::Json::Arr(items)) = body.get("characters") else {
        panic!("characters é array");
    };
    assert_eq!(items.len(), 2, "conta 5 tem dois personagens");
    let (_, _, outro) = dispatch(
        &router,
        "/admin/v1/panel/characters",
        Some("player"),
        r#"{"account_id":9}"#,
    );
    let Some(json::Json::Arr(outros)) = outro.get("characters") else {
        panic!("characters é array");
    };
    assert_eq!(outros.len(), 1, "conta 9 tem um");
    assert!(
        !outro.contains_key_any(&["account_id"]),
        "resposta não devolve account_id"
    );
}

#[test]
fn ranking_vem_ordenado_e_sem_identidade_de_conta() {
    let router = router();
    let (status, raw, body) = dispatch(
        &router,
        "/admin/v1/panel/rankings",
        Some("player"),
        r#"{"limit":2}"#,
    );
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 1);
    let Some(json::Json::Arr(items)) = body.get("ranking") else {
        panic!("ranking é array");
    };
    assert_eq!(items.len(), 2);
    let primeiro = items[0]
        .get("name")
        .and_then(json::Json::as_str)
        .expect("nome");
    assert_eq!(primeiro, "Alto", "maior nível primeiro");
    assert_eq!(items[0].get("rank").and_then(json::Json::as_i64), Some(1));
    assert!(!raw.contains("account_id"), "ranking não expõe conta");
}

#[test]
fn recuperacao_responde_igual_exista_ou_nao_e_nao_grava_login() {
    let router = router();
    let (_, _, existe) = dispatch(
        &router,
        "/admin/v1/accounts/recovery",
        Some("player"),
        r#"{"login":"jogador"}"#,
    );
    let (_, _, inexistente) = dispatch(
        &router,
        "/admin/v1/accounts/recovery",
        Some("player"),
        r#"{"login":"fantasma"}"#,
    );
    assert_eq!(result_of(&existe), 1);
    assert_eq!(result_of(&inexistente), 1);
    let entries = router.audit().entries();
    assert_eq!(entries.len(), 2);
    for entry in &entries {
        assert_eq!(entry.action, "accounts.recovery");
        assert_eq!(entry.target, "login:*");
        assert!(!entry.detail.contains("jogador"));
        assert!(!entry.detail.contains("fantasma"));
    }
}

#[test]
fn parametros_invalidos_sao_recusa_de_negocio() {
    let router = router();
    let (status, _, body) = dispatch(&router, "/admin/v1/panel/self", Some("player"), "{}");
    assert_eq!(status, 200);
    assert_eq!(result_of(&body), 3, "account_id obrigatório");

    let (_, _, body) = dispatch(
        &router,
        "/admin/v1/panel/rankings",
        Some("player"),
        r#"{"limit":0}"#,
    );
    assert_eq!(result_of(&body), 3, "limite fora da faixa");

    let (status, _, _) = dispatch(&router, "/admin/v1/panel/nada", Some("player"), "{}");
    assert_eq!(status, 404);
    assert!(
        router.audit().entries().is_empty(),
        "recusa de validação não audita leitura"
    );
}

trait JsonContains {
    fn contains_key_any(&self, keys: &[&str]) -> bool;
}

impl JsonContains for json::Json {
    fn contains_key_any(&self, keys: &[&str]) -> bool {
        match self {
            json::Json::Obj(fields) => fields
                .iter()
                .any(|(name, value)| keys.contains(&name.as_str()) || value.contains_key_any(keys)),
            json::Json::Arr(items) => items.iter().any(|item| item.contains_key_any(keys)),
            _ => false,
        }
    }
}
