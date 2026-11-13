//! Sem sessão → 401; sem papel → 403 + `auth.denied` auditado.
//!
//! Espelha o `adminCall` do portal: nada chega ao repo sem staff, e tentativa
//! de conta comum deixa trilha (sem sessão não há autor para registrar).

use std::collections::HashMap;
use std::future::Future;
use std::sync::Mutex;
use std::task::{Context, Poll, Waker};

use wyd_admin::{
    find_account, request_grant, AdminError, AuditSink, GrantKind, GrantReceipt, GrantRequest,
    GrantService, MemoryGrantQueue, Role, Session, VecSink,
};
use wyd_db::{Account, Accounts, DbError};

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

fn block_on<F: Future>(f: F) -> F::Output {
    let waker: Waker = Waker::noop().clone();
    let mut cx = Context::from_waker(&waker);
    match Box::pin(f).as_mut().poll(&mut cx) {
        Poll::Ready(v) => v,
        Poll::Pending => panic!("futuro de teste completa de primeira"),
    }
}

fn player() -> Option<Session> {
    Some(Session {
        account_id: 5,
        role: Role::Player,
    })
}

fn grant_req() -> GrantRequest {
    GrantRequest {
        idempotency_key: "op-e2e".to_string(),
        character_id: 9,
        kind: GrantKind::Item { item_id: 7, qty: 1 },
    }
}

#[test]
fn no_session_is_401_without_touching_repo_or_audit() {
    let repo = FakeAccounts::new();
    let sink = VecSink::default();
    assert_eq!(
        block_on(find_account(None, "alvo", &repo, &sink)),
        Err(AdminError::Unauthorized)
    );
    assert!(sink.entries().is_empty(), "401 não tem autor: não audita");
}

#[test]
fn player_is_403_and_denied_is_audited() {
    let repo = FakeAccounts::new();
    let sink = VecSink::default();
    assert_eq!(
        block_on(find_account(player(), "alvo", &repo, &sink)),
        Err(AdminError::Forbidden {
            reason: "papel de staff necessário"
        })
    );
    let entries = sink.entries();
    assert_eq!(entries.len(), 1);
    assert_eq!(entries[0].action, "auth.denied:accounts.find");
    assert_eq!(entries[0].actor_id, 5);
}

#[test]
fn staff_grant_flow_is_audited_end_to_end() {
    let repo = FakeAccounts::new();
    let sink = VecSink::default();
    let service = GrantService::new();
    service.attach_queue(std::sync::Arc::new(MemoryGrantQueue::default()));
    let staff = Some(Session {
        account_id: 7,
        role: Role::Moderator,
    });
    let found = block_on(find_account(staff, "alvo", &repo, &sink)).expect("staff lê");
    assert_eq!(found.login, "alvo");
    let receipt: GrantReceipt =
        request_grant(staff, grant_req(), &service, &sink).expect("grant enfileirado");
    assert!(!receipt.replay);
    let replay = request_grant(staff, grant_req(), &service, &sink).expect("replay idempotente");
    assert!(replay.replay);
    assert_eq!(receipt.grant_id, replay.grant_id);
    let actions: Vec<_> = sink.entries().iter().map(|e| e.action.clone()).collect();
    assert_eq!(
        actions,
        ["accounts.find", "grants.request", "grants.replay"]
    );
    assert!(sink.entries().iter().all(|e| e.actor_id == 7));
}

#[test]
fn audit_sink_is_object_safe_and_shared() {
    fn accepts(_: &dyn AuditSink) {}
    accepts(&VecSink::default());
}
