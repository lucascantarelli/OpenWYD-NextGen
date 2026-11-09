//! Moderação de contas (FASE-07-3): papel, bloqueio e trilha de auditoria.
//!
//! Toda função exige sessão de staff, decide o alcance pelo papel **da sessão**
//! (nunca do corpo) e registra a ação com autor/alvo. Duas travas de segurança
//! que o domínio precisa ter:
//!
//! - ninguém altera o próprio papel nem bloqueia a própria conta (evita
//!   auto-lockout do único admin);
//! - moderador não bloqueia staff — papel de staff só admin mexe.
//!
//! O bloqueio é decisão com motivo obrigatório: sem motivo não há trilha
//! explicável, e auditoria sem explicação não serve para revisão.

use wyd_db::{AccountModeration, Accounts, AuditRow, AuditTrail, Role};

use crate::audit::{log_action, log_denied, AuditSink};
use crate::auth::{may_change_role, Session, StaffSession};
use crate::error::AdminError;

use super::accounts::staff_or_deny;

/// Teto do motivo do bloqueio (texto de staff, sem PII além do próprio motivo).
pub const MAX_REASON: usize = 200;

fn alvo(estado: &AccountModeration) -> String {
    format!("account:{}", estado.account.id)
}

fn so_admin(
    staff: &StaffSession,
    sink: &impl AuditSink,
    action: &str,
    reason: &'static str,
) -> Result<(), AdminError> {
    if may_change_role(staff.role()) {
        return Ok(());
    }
    log_denied(
        sink,
        staff.moderator_id(),
        staff.role(),
        action,
        "papel além do alcance",
    );
    Err(AdminError::Forbidden { reason })
}

/// Leitura do estado de moderação de uma conta (`accounts.moderation`).
pub async fn moderation(
    session: Option<Session>,
    login: &str,
    repo: &impl Accounts,
    sink: &impl AuditSink,
) -> Result<AccountModeration, AdminError> {
    let staff = staff_or_deny(session, sink, "accounts.moderation")?;
    let estado = repo.moderation_state(login).await?;
    log_action(
        sink,
        &staff,
        "accounts.moderation",
        &alvo(&estado),
        "leitura",
    );
    Ok(estado)
}

/// Troca o papel da conta (`accounts.set_role`) — exclusivo de admin.
pub async fn set_role(
    session: Option<Session>,
    login: &str,
    role: Role,
    repo: &impl Accounts,
    sink: &impl AuditSink,
) -> Result<AccountModeration, AdminError> {
    let staff = staff_or_deny(session, sink, "accounts.set_role")?;
    so_admin(&staff, sink, "accounts.set_role", "só admin troca papel")?;
    let atual = repo.moderation_state(login).await?;
    if atual.account.id == staff.moderator_id() {
        return Err(AdminError::Conflict {
            reason: "não altere o próprio papel",
        });
    }
    let novo = repo.set_role(login, role).await?;
    log_action(
        sink,
        &staff,
        "accounts.set_role",
        &alvo(&novo),
        &format!(
            "{} -> {}",
            atual.account.role.as_db(),
            novo.account.role.as_db()
        ),
    );
    Ok(novo)
}

/// Bloqueia/libera a conta (`accounts.block`/`accounts.unblock`).
///
/// Bloqueio exige motivo; liberação não. Moderador só alcança jogador.
pub async fn set_blocked(
    session: Option<Session>,
    login: &str,
    blocked: bool,
    reason: &str,
    repo: &impl Accounts,
    sink: &impl AuditSink,
) -> Result<AccountModeration, AdminError> {
    let acao = if blocked {
        "accounts.block"
    } else {
        "accounts.unblock"
    };
    let staff = staff_or_deny(session, sink, acao)?;
    let atual = repo.moderation_state(login).await?;
    if atual.account.id == staff.moderator_id() {
        return Err(AdminError::Conflict {
            reason: "não bloqueie a própria conta",
        });
    }
    if blocked && atual.account.role != Role::Player {
        so_admin(&staff, sink, acao, "só admin bloqueia staff")?;
    }
    let motivo = reason.trim();
    if blocked && motivo.is_empty() {
        return Err(AdminError::InvalidInput {
            reason: "motivo do bloqueio obrigatório",
        });
    }
    if motivo.chars().count() > MAX_REASON {
        return Err(AdminError::InvalidInput {
            reason: "motivo do bloqueio longo demais",
        });
    }
    let novo = repo
        .set_blocked(login, blocked, motivo, staff.moderator_id())
        .await?;
    let detalhe = if blocked {
        format!("bloqueio: {motivo}")
    } else {
        "liberação".to_string()
    };
    log_action(sink, &staff, acao, &alvo(&novo), &detalhe);
    Ok(novo)
}

/// Últimas linhas da trilha (`audit.list`) — leitura exclusiva de admin.
pub async fn recent_audit(
    session: Option<Session>,
    limit: usize,
    trail: &(dyn AuditTrail + Send + Sync),
    sink: &impl AuditSink,
) -> Result<Vec<AuditRow>, AdminError> {
    let staff = staff_or_deny(session, sink, "audit.list")?;
    so_admin(&staff, sink, "audit.list", "só admin lê a trilha")?;
    if limit == 0 || limit > 200 {
        return Err(AdminError::InvalidInput {
            reason: "limite da trilha fora da faixa",
        });
    }
    let linhas = trail.recent(limit).await?;
    log_action(
        sink,
        &staff,
        "audit.list",
        "audit_log",
        &format!("{} linhas", linhas.len()),
    );
    Ok(linhas)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::audit::VecSink;
    use crate::block_on;
    use wyd_db::{Account, DbError};

    #[derive(Default)]
    struct FakeModeration {
        estado: Option<(Account, bool, String)>,
        escritas: std::sync::Mutex<usize>,
    }

    impl FakeModeration {
        fn com(role: Role) -> Self {
            Self {
                estado: Some((
                    Account {
                        id: 9,
                        login: "alvo".to_string(),
                        role,
                    },
                    false,
                    String::new(),
                )),
                escritas: std::sync::Mutex::new(0),
            }
        }

        fn estado(&self) -> AccountModeration {
            let (account, blocked, reason) = self.estado.clone().expect("estado");
            AccountModeration {
                account,
                blocked,
                reason,
                blocked_at_ms: blocked.then_some(1),
            }
        }
    }

    impl Accounts for FakeModeration {
        async fn find_by_login(&self, _login: &str) -> Result<Account, DbError> {
            Err(DbError::NotFoundKey { entity: "account" })
        }
        async fn find_by_id(&self, _id: i64) -> Result<Account, DbError> {
            Err(DbError::NotFoundKey { entity: "account" })
        }
        async fn create_account(&self, _login: &str, _role: Role) -> Result<Account, DbError> {
            Err(DbError::NotFoundKey { entity: "account" })
        }
        fn moderation_state<'a>(
            &'a self,
            _login: &'a str,
        ) -> std::pin::Pin<
            Box<dyn std::future::Future<Output = Result<AccountModeration, DbError>> + Send + 'a>,
        > {
            let estado = self.estado();
            Box::pin(async move { Ok(estado) })
        }
        fn set_role<'a>(
            &'a self,
            _login: &'a str,
            role: Role,
        ) -> std::pin::Pin<
            Box<dyn std::future::Future<Output = Result<AccountModeration, DbError>> + Send + 'a>,
        > {
            *self.escritas.lock().expect("fake") += 1;
            let mut estado = self.estado();
            estado.account.role = role;
            Box::pin(async move { Ok(estado) })
        }
        fn set_blocked<'a>(
            &'a self,
            _login: &'a str,
            blocked: bool,
            reason: &'a str,
            _by: i64,
        ) -> std::pin::Pin<
            Box<dyn std::future::Future<Output = Result<AccountModeration, DbError>> + Send + 'a>,
        > {
            *self.escritas.lock().expect("fake") += 1;
            let mut estado = self.estado();
            estado.blocked = blocked;
            estado.reason = reason.to_string();
            estado.blocked_at_ms = blocked.then_some(1);
            Box::pin(async move { Ok(estado) })
        }
    }

    fn sessao(role: Role) -> Option<Session> {
        Some(Session {
            account_id: 7,
            role,
        })
    }

    #[test]
    fn moderador_nao_troca_papel_e_a_negacao_e_auditada() {
        let repo = FakeModeration::com(Role::Player);
        let sink = VecSink::default();
        let erro = block_on(set_role(
            sessao(Role::Moderator),
            "alvo",
            Role::Admin,
            &repo,
            &sink,
        ));
        assert_eq!(
            erro,
            Err(AdminError::Forbidden {
                reason: "só admin troca papel"
            })
        );
        assert_eq!(*repo.escritas.lock().expect("fake"), 0, "nada escrito");
        assert_eq!(sink.entries()[0].action, "auth.denied:accounts.set_role");
    }

    #[test]
    fn admin_nao_muda_o_proprio_papel() {
        let repo = FakeModeration::com(Role::Player);
        let sink = VecSink::default();
        let mut alvo = repo.estado();
        alvo.account.id = 7;
        let repo = FakeModeration {
            estado: Some((alvo.account, false, String::new())),
            escritas: std::sync::Mutex::new(0),
        };
        let erro = block_on(set_role(
            sessao(Role::Admin),
            "eu",
            Role::Player,
            &repo,
            &sink,
        ));
        assert_eq!(
            erro,
            Err(AdminError::Conflict {
                reason: "não altere o próprio papel"
            })
        );
        assert!(sink.entries().is_empty());
    }

    #[test]
    fn bloqueio_sem_motivo_nao_escreve() {
        let repo = FakeModeration::com(Role::Player);
        let sink = VecSink::default();
        let erro = block_on(set_blocked(
            sessao(Role::Moderator),
            "alvo",
            true,
            "   ",
            &repo,
            &sink,
        ));
        assert_eq!(
            erro,
            Err(AdminError::InvalidInput {
                reason: "motivo do bloqueio obrigatório"
            })
        );
        assert_eq!(*repo.escritas.lock().expect("fake"), 0);
    }

    #[test]
    fn bloqueio_de_jogador_e_auditado_com_motivo() {
        let repo = FakeModeration::com(Role::Player);
        let sink = VecSink::default();
        let estado = block_on(set_blocked(
            sessao(Role::Moderator),
            "alvo",
            true,
            "trapaça confirmada",
            &repo,
            &sink,
        ))
        .expect("moderador bloqueia jogador");
        assert!(estado.blocked);
        let entradas = sink.entries();
        assert_eq!(entradas.len(), 1);
        assert_eq!(entradas[0].action, "accounts.block");
        assert_eq!(entradas[0].target, "account:9");
        assert!(entradas[0].detail.contains("trapaça confirmada"));
    }

    #[test]
    fn moderador_nao_bloqueia_staff() {
        let repo = FakeModeration::com(Role::Moderator);
        let sink = VecSink::default();
        let erro = block_on(set_blocked(
            sessao(Role::Moderator),
            "outro-staff",
            true,
            "motivo",
            &repo,
            &sink,
        ));
        assert_eq!(
            erro,
            Err(AdminError::Forbidden {
                reason: "só admin bloqueia staff"
            })
        );
        assert_eq!(*repo.escritas.lock().expect("fake"), 0);
    }
}
