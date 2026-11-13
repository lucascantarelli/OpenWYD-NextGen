//! Auditoria: quem, o quê, quando — em toda ação admin.
//!
//! O destino (`AuditSink`) é porta: teste usa `VecSink` em memória, produção
//! grava `audit_log` (migration 0005). Tentativa negada por falta de papel é
//! registrada (`auth.denied`); chamada sem sessão (401) não tem autor.

use std::sync::Mutex;
use std::time::{SystemTime, UNIX_EPOCH};

use wyd_db::Role;

use crate::auth::StaffSession;

/// Registro de auditoria (espelha `audit_log`; `at_ms` em milissegundos UTC).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AuditEntry {
    pub at_ms: u64,
    pub actor_id: i64,
    pub actor_role: Role,
    pub action: String,
    pub target: String,
    pub detail: String,
}

/// Destino da trilha (teste: memória; produção: tabela `audit_log`).
pub trait AuditSink {
    fn record(&self, entry: AuditEntry);
}

/// Coletor em memória para testes (ordem de inserção preservada).
#[derive(Debug, Default)]
pub struct VecSink {
    entries: Mutex<Vec<AuditEntry>>,
}

impl VecSink {
    #[must_use]
    pub fn entries(&self) -> Vec<AuditEntry> {
        self.entries.lock().expect("auditoria acessível").clone()
    }
}

impl AuditSink for VecSink {
    fn record(&self, entry: AuditEntry) {
        self.entries
            .lock()
            .expect("auditoria acessível")
            .push(entry);
    }
}

fn now_ms() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d: std::time::Duration| d.as_millis() as u64)
        .unwrap_or(0)
}

/// Registra uma ação autorizada de staff (autor sempre da sessão).
pub fn log_action(
    sink: &impl AuditSink,
    staff: &StaffSession,
    action: &str,
    target: &str,
    detail: &str,
) {
    sink.record(AuditEntry {
        at_ms: now_ms(),
        actor_id: staff.moderator_id(),
        actor_role: staff.role(),
        action: action.to_string(),
        target: target.to_string(),
        detail: detail.to_string(),
    });
}

/// Registra leitura/solicitação do painel do jogador (autor autenticado, qualquer papel).
///
/// `target` nunca carrega login/nome: só o identificador interno da conta.
pub fn log_panel(
    sink: &impl AuditSink,
    session: crate::auth::Session,
    action: &str,
    target: &str,
    detail: &str,
) {
    sink.record(AuditEntry {
        at_ms: now_ms(),
        actor_id: session.account_id,
        actor_role: session.role,
        action: action.to_string(),
        target: target.to_string(),
        detail: detail.to_string(),
    });
}

/// Registra tentativa negada de conta autenticada sem o papel necessário.
///
/// O papel vem da sessão, não de um `Player` fixo: a linha `auth.denied:*`
/// precisa dizer **quem** tentou (`3:moderator`), senão a revisão não distingue
/// um jogador de um moderador que estourou o próprio alcance.
pub fn log_denied(sink: &impl AuditSink, account_id: i64, role: Role, action: &str, detail: &str) {
    sink.record(AuditEntry {
        at_ms: now_ms(),
        actor_id: account_id,
        actor_role: role,
        action: format!("auth.denied:{action}"),
        target: String::new(),
        detail: detail.to_string(),
    });
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::auth::Session;

    #[test]
    fn actions_carry_author_and_timestamp() {
        let sink = VecSink::default();
        let staff = StaffSession::require(Session {
            account_id: 3,
            role: Role::Admin,
        })
        .expect("admin é staff");
        log_action(&sink, &staff, "grant.item", "char:9", "item 42 x1");
        let entries = sink.entries();
        assert_eq!(entries.len(), 1);
        assert_eq!(entries[0].actor_id, 3);
        assert_eq!(entries[0].actor_role, Role::Admin);
        assert!(entries[0].at_ms > 0);
    }

    #[test]
    fn negacao_preserva_o_papel_de_quem_tentou() {
        let sink = VecSink::default();
        log_denied(
            &sink,
            3,
            Role::Moderator,
            "accounts.set_role",
            "papel além do alcance",
        );
        let entries = sink.entries();
        assert_eq!(entries[0].action, "auth.denied:accounts.set_role");
        assert_eq!(
            entries[0].actor_role,
            Role::Moderator,
            "papel real na trilha"
        );
        assert_eq!(entries[0].actor_id, 3);
    }
}
