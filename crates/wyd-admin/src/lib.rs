//! Control plane Cold Path (`wyd-admin`): contrato administrativo do portal.
//!
//! Camadas (dentro→fora): `error` (HTTP por variante) → `auth` (sessão/papéis)
//! → `audit` (quem/o quê/quando) → `api` (endpoints por domínio) → `service`
//! (regra + idempotência). Transporte (HTTP/gRPC) chega com runtime async
//! (04-3/FASE-05: exige rede + ADR); a regra já é testável sem ele.
//!
//! Invariantes: BFF puro — nunca estado vivo de personagem (só intenções para
//! `delivery_queue`); `moderator_id` sempre da sessão, nunca do corpo.

pub mod api;
pub mod audit;
pub mod auth;
pub mod error;
pub mod service;

/// Prefixo das rotas administrativas neste esqueleto.
pub const ADMIN_SCOPE: &str = "/admin";

/// Verifica se o caminho pertence ao escopo admin.
#[must_use]
pub fn is_admin_path(path: &str) -> bool {
    path == ADMIN_SCOPE || path.starts_with("/admin/")
}

pub use api::{
    create_account, find_account, request_grant, AccountSummary, GrantKind, GrantReceipt,
    GrantRequest,
};
pub use audit::{log_action, log_denied, AuditEntry, AuditSink, VecSink};
pub use auth::{may_change_role, may_target, Session, StaffSession};
pub use error::AdminError;
pub use service::GrantService;
pub use wyd_db::{DeliveryQueue, Grant, QueuedGrant, Role};

/// Executor mínimo para testes: futuros prontos, sem runtime.
#[cfg(test)]
pub(crate) fn block_on<F: std::future::Future>(f: F) -> F::Output {
    use std::task::{Context, Poll, Waker};
    let waker: Waker = Waker::noop().clone();
    let mut cx = Context::from_waker(&waker);
    match Box::pin(f).as_mut().poll(&mut cx) {
        Poll::Ready(v) => v,
        Poll::Pending => panic!("futuro de teste completa de primeira"),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn matches_admin_scope_only() {
        assert!(is_admin_path("/admin"));
        assert!(is_admin_path("/admin/users"));
        assert!(!is_admin_path("/player"));
    }
}
