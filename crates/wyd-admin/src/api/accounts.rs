//! Contas: leitura e criação via gateway (nunca estado vivo).
//!
//! Espelha o `adminCall` do portal: sessão de staff, `moderator_id` da sessão,
//! auditoria com autor. Troca de papel/bloqueio exigem métodos de escrita no
//! repo `pg` (04-3) + coluna de bloqueio — documentados como pendência, não
//! improvisados aqui.

use wyd_db::{Account, Accounts, Role};

use crate::audit::{log_action, log_denied, AuditSink};
use crate::auth::{may_change_role, Session, StaffSession};
use crate::error::AdminError;

/// Resumo devolvido ao BFF (sem hash, sem segredo).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccountSummary {
    pub id: i64,
    pub login: String,
    pub role: Role,
}

impl From<Account> for AccountSummary {
    fn from(a: Account) -> Self {
        Self {
            id: a.id,
            login: a.login,
            role: a.role,
        }
    }
}

pub(super) fn staff_or_deny(
    session: Option<Session>,
    sink: &impl AuditSink,
    action: &str,
) -> Result<StaffSession, AdminError> {
    match session {
        None => Err(AdminError::Unauthorized),
        Some(s) => match StaffSession::require(s) {
            Ok(staff) => Ok(staff),
            Err(e) => {
                log_denied(sink, s.account_id, action, "sem papel de staff");
                Err(e)
            }
        },
    }
}

/// Busca por login exato (staff; 401 sem sessão, 403 sem papel, 404 se não há).
pub async fn find_account(
    session: Option<Session>,
    login: &str,
    repo: &impl Accounts,
    sink: &impl AuditSink,
) -> Result<AccountSummary, AdminError> {
    let staff = staff_or_deny(session, sink, "accounts.find")?;
    let account = repo.find_by_login(login).await?;
    log_action(sink, &staff, "accounts.find", login, "leitura");
    Ok(account.into())
}

/// Cria conta: moderador só cria jogador; papel de staff exige admin.
pub async fn create_account(
    session: Option<Session>,
    login: &str,
    role: Role,
    repo: &impl Accounts,
    sink: &impl AuditSink,
) -> Result<AccountSummary, AdminError> {
    let staff = staff_or_deny(session, sink, "accounts.create")?;
    if role != Role::Player && !may_change_role(staff.role()) {
        log_denied(
            sink,
            staff.moderator_id(),
            "accounts.create",
            "papel além do alcance",
        );
        return Err(AdminError::Forbidden {
            reason: "só admin cria staff",
        });
    }
    Account::validate_login(login).map_err(AdminError::from)?;
    let account = repo.create_account(login, role).await?;
    log_action(
        sink,
        &staff,
        "accounts.create",
        login,
        &format!("papel {}", role.as_db()),
    );
    Ok(account.into())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;
    use std::sync::Mutex;

    pub struct FakeAccounts {
        next: Mutex<i64>,
        by_login: Mutex<HashMap<String, Account>>,
    }

    impl FakeAccounts {
        pub fn new() -> Self {
            Self {
                next: Mutex::new(1),
                by_login: Mutex::new(HashMap::new()),
            }
        }
    }

    impl Accounts for FakeAccounts {
        async fn find_by_login(&self, login: &str) -> Result<Account, wyd_db::DbError> {
            self.by_login
                .lock()
                .expect("fake acessível")
                .get(login)
                .cloned()
                .ok_or(wyd_db::DbError::NotFound {
                    entity: "account",
                    id: 0,
                })
        }

        async fn find_by_id(&self, id: i64) -> Result<Account, wyd_db::DbError> {
            self.by_login
                .lock()
                .expect("fake acessível")
                .values()
                .find(|account| account.id == id)
                .cloned()
                .ok_or(wyd_db::DbError::NotFound {
                    entity: "account",
                    id,
                })
        }

        async fn create_account(
            &self,
            login: &str,
            role: Role,
        ) -> Result<Account, wyd_db::DbError> {
            let mut by_login = self.by_login.lock().expect("fake acessível");
            if by_login.contains_key(login) {
                return Err(wyd_db::DbError::ConstraintViolation {
                    table: "accounts",
                    constraint: "accounts_login_unique",
                });
            }
            let mut next = self.next.lock().expect("fake acessível");
            let account = Account {
                id: *next,
                login: login.to_string(),
                role,
            };
            *next += 1;
            by_login.insert(login.to_string(), account.clone());
            Ok(account)
        }
    }

    fn staff(role: Role) -> Option<Session> {
        Some(Session {
            account_id: 7,
            role,
        })
    }

    use crate::block_on;

    #[test]
    fn moderator_creates_player_but_not_staff() {
        let repo = FakeAccounts::new();
        let sink = crate::audit::VecSink::default();
        let created = block_on(create_account(
            staff(Role::Moderator),
            "novo",
            Role::Player,
            &repo,
            &sink,
        ))
        .expect("moderador cria jogador");
        assert_eq!(created.login, "novo");
        let denied = block_on(create_account(
            staff(Role::Moderator),
            "outro",
            Role::Moderator,
            &repo,
            &sink,
        ));
        assert_eq!(
            denied,
            Err(AdminError::Forbidden {
                reason: "só admin cria staff"
            })
        );
        let entries = sink.entries();
        assert_eq!(entries.len(), 2);
        assert_eq!(entries[0].action, "accounts.create");
        assert_eq!(entries[1].action, "auth.denied:accounts.create");
    }

    #[test]
    fn duplicate_login_maps_to_invalid_input() {
        let repo = FakeAccounts::new();
        let sink = crate::audit::VecSink::default();
        block_on(create_account(
            staff(Role::Admin),
            "dup",
            Role::Player,
            &repo,
            &sink,
        ))
        .expect("primeira cria");
        let again = block_on(create_account(
            staff(Role::Admin),
            "dup",
            Role::Player,
            &repo,
            &sink,
        ));
        assert_eq!(
            again,
            Err(AdminError::InvalidInput {
                reason: "dados conflitam com as regras"
            })
        );
    }
}
