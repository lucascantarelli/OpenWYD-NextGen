//! Login do portal (`account_web/verify_credentials`) com argon2id de verdade.
//!
//! Prova o contrato que o BFF consome: `ok`/`account_id`/`role`, recusa igual para
//! senha errada e conta inexistente, e conta sem credencial (`NO_CREDENTIAL`) que
//! nunca autentica. O hash não sai na resposta.

use std::future::Future;
use std::io::Cursor;

use wyd_admin::transport::{http, json, Router, StaticTokens};
use wyd_admin::{hash_password, Role, Session, VecSink};
use wyd_db::{Account, Accounts, Character, Characters, CredentialRecord, DbError, NO_CREDENTIAL};

struct FakeAccounts {
    login: &'static str,
    hash: String,
}

impl Accounts for FakeAccounts {
    fn find_by_login(&self, login: &str) -> impl Future<Output = Result<Account, DbError>> + Send {
        let found = (login == self.login).then(|| Account {
            id: 42,
            login: login.to_string(),
            role: Role::Admin,
        });
        async move { found.ok_or(DbError::NotFoundKey { entity: "account" }) }
    }

    fn find_by_id(&self, id: i64) -> impl Future<Output = Result<Account, DbError>> + Send {
        let found = (id == 42).then(|| Account {
            id: 42,
            login: "admin".to_string(),
            role: Role::Admin,
        });
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
            id: 43,
            login: login.to_string(),
            role,
        };
        async move { Ok(account) }
    }

    fn credentials(
        &self,
        login: &str,
    ) -> impl Future<Output = Result<CredentialRecord, DbError>> + Send {
        let found = (login == self.login).then(|| CredentialRecord {
            account: Account {
                id: 42,
                login: self.login.to_string(),
                role: Role::Admin,
            },
            password_hash: self.hash.clone(),
            blocked: false,
        });
        async move { found.ok_or(DbError::NotFoundKey { entity: "account" }) }
    }
}

struct NoCharacters;

impl Characters for NoCharacters {
    async fn list_by_account(&self, _account_id: i64) -> Result<Vec<Character>, DbError> {
        Ok(Vec::new())
    }

    async fn top_by_level(&self, _limit: usize) -> Result<Vec<Character>, DbError> {
        Ok(Vec::new())
    }

    async fn search_by_name(&self, _term: &str, _limit: usize) -> Result<Vec<Character>, DbError> {
        Ok(Vec::new())
    }

    async fn create_character(&self, _character: &Character) -> Result<Character, DbError> {
        Err(DbError::NotFound {
            entity: "character",
            id: 0,
        })
    }

    async fn soft_delete(&self, _id: i64) -> Result<(), DbError> {
        Ok(())
    }
}

fn router(hash: &str) -> Router<FakeAccounts, NoCharacters, VecSink> {
    let mut tokens = StaticTokens::default();
    tokens.insert(
        "token-do-portal",
        Session {
            account_id: 0,
            role: Role::Admin,
        },
    );
    // O gateway monta esta função com o pool real (`block_on`); no teste ela fecha
    // sobre o hash argon2id direto, sem banco.
    let esperado = hash.to_string();
    let check: wyd_admin::LoginCheck = std::sync::Arc::new(move |name: &str, password: &str| {
        (name == "admin" && wyd_admin::verify_password(password, &esperado)).then(|| {
            wyd_admin::LoginOk {
                account_id: 42,
                login: "admin".to_string(),
                role: Role::Admin,
                blocked: false,
            }
        })
    });
    Router::new(
        FakeAccounts {
            login: "admin",
            hash: hash.to_string(),
        },
        NoCharacters,
        VecSink::default(),
        tokens,
    )
    .with_login_check(check)
}

fn login(router: &Router<FakeAccounts, NoCharacters, VecSink>, body: &str) -> json::Json {
    let request = format!(
        "POST /admin/v1/account_web/verify_credentials HTTP/1.1\r\n\
         host: loopback\r\n\
         x-trace-id: login-1\r\n\
         authorization: Bearer token-do-portal\r\n\
         content-length: {}\r\n\r\n{body}",
        body.len()
    );
    let mut cursor = Cursor::new(request.into_bytes());
    let parsed = http::read_request(&mut cursor, 1 << 20).expect("request válida");
    let reply = router.dispatch(&parsed);
    assert_eq!(reply.status, 200, "corpo: {}", reply.body);
    json::Json::parse(&reply.body).expect("corpo é JSON")
}

fn field<'a>(body: &'a json::Json, key: &str) -> Option<&'a json::Json> {
    body.get(key)
}

#[test]
fn correct_password_logs_in_with_role_and_id() {
    let hash = hash_password("senha-do-admin-123").expect("hash");
    let router = router(&hash);
    let body = login(
        &router,
        r#"{"name":"admin","password":"senha-do-admin-123"}"#,
    );
    assert_eq!(field(&body, "result").and_then(json::Json::as_i64), Some(1));
    assert_eq!(field(&body, "ok").and_then(json::Json::as_bool), Some(true));
    assert_eq!(
        field(&body, "account_id").and_then(json::Json::as_i64),
        Some(42)
    );
    assert_eq!(
        field(&body, "role").and_then(json::Json::as_str),
        Some("admin")
    );
    assert!(
        !body.render().contains("$argon2"),
        "o hash nunca volta ao BFF"
    );
    assert!(
        !body.render().contains("senha-do-admin-123"),
        "a senha nunca volta ao BFF"
    );
}

#[test]
fn wrong_password_and_unknown_login_share_the_same_refusal() {
    let hash = hash_password("senha-do-admin-123").expect("hash");
    let router = router(&hash);
    let errada = login(&router, r#"{"name":"admin","password":"outra-senha-123"}"#);
    let inexistente = login(
        &router,
        r#"{"name":"ninguem","password":"senha-do-admin-123"}"#,
    );
    for body in [&errada, &inexistente] {
        assert_eq!(field(body, "ok").and_then(json::Json::as_bool), Some(false));
        assert_eq!(
            field(body, "account_id").and_then(json::Json::as_i64),
            Some(0)
        );
        assert_eq!(field(body, "role").and_then(json::Json::as_str), Some(""));
    }
    assert_eq!(
        errada.render().replace("\"admin\"", ""),
        inexistente.render().replace("\"ninguem\"", "")
    );
}

#[test]
fn account_without_credential_never_authenticates() {
    let router = router(NO_CREDENTIAL);
    let body = login(
        &router,
        r#"{"name":"admin","password":"qualquer-senha-123"}"#,
    );
    assert_eq!(
        field(&body, "ok").and_then(json::Json::as_bool),
        Some(false)
    );
}

#[test]
fn missing_fields_are_invalid_input_not_login_failure() {
    let hash = hash_password("senha-do-admin-123").expect("hash");
    let router = router(&hash);
    let body = login(&router, r#"{"name":"admin"}"#);
    assert_eq!(field(&body, "result").and_then(json::Json::as_i64), Some(3));
}
