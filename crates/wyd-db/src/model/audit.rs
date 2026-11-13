//! Trilha de auditoria — espelho append-only de `audit_log` (migration 0005).

/// Entrada de auditoria (sem FK de propósito: nunca falha por integridade alheia).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AuditEntry {
    pub actor: String,
    pub action: String,
    pub target: String,
    pub detail: String,
}
