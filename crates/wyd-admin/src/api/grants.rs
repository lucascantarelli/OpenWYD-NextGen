//! Grants: intenções do portal/admin → `delivery_queue` (idempotente).
//!
//! A web nunca escreve estado vivo: o grant entra na fila e o servidor drena.
//! Idempotência por chave rastreável — retry do browser não duplica entrega.

use crate::audit::{log_action, AuditSink};
use crate::auth::{Session, StaffSession};
use crate::error::AdminError;
use crate::service::GrantService;

use super::accounts::staff_or_deny;

/// Pedido de grant vindo do BFF (chave gerada no portal por operação).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GrantRequest {
    /// Idempotência: mesma chave = mesma entrega (nunca duas).
    pub idempotency_key: String,
    pub character_id: i64,
    pub kind: GrantKind,
}

/// Tipo de benefício (espelha os `kind` da `delivery_queue`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum GrantKind {
    Item { item_id: i64, qty: i64 },
    Gold { amount: i64 },
    Level { delta: i64 },
}

/// Recibo da entrega (mesma chave, mesmo recibo — prova de idempotência).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GrantReceipt {
    pub idempotency_key: String,
    pub grant_id: i64,
    pub replay: bool,
}

/// Enfileira o grant (staff; auditado; replay devolve o recibo original).
pub fn request_grant(
    session: Option<Session>,
    req: GrantRequest,
    service: &GrantService,
    sink: &impl AuditSink,
) -> Result<GrantReceipt, AdminError> {
    let staff: StaffSession = staff_or_deny(session, sink, "grants.request")?;
    validate_request(&req)?;
    let receipt = service.apply(staff, &req)?;
    log_action(
        sink,
        &staff,
        if receipt.replay {
            "grants.replay"
        } else {
            "grants.request"
        },
        &format!("char:{}", req.character_id),
        &format!("chave {}", req.idempotency_key),
    );
    Ok(receipt)
}

fn validate_request(req: &GrantRequest) -> Result<(), AdminError> {
    if req.idempotency_key.trim().is_empty() {
        return Err(AdminError::InvalidInput {
            reason: "chave de idempotência vazia",
        });
    }
    if req.character_id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "personagem inválido",
        });
    }
    let bad = match &req.kind {
        GrantKind::Item { item_id, qty } => *item_id <= 0 || *qty <= 0,
        GrantKind::Gold { amount } => *amount <= 0,
        GrantKind::Level { delta } => *delta <= 0,
    };
    if bad {
        return Err(AdminError::InvalidInput {
            reason: "quantidade do grant inválida",
        });
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use wyd_db::Role;

    #[test]
    fn empty_key_and_bad_amounts_rejected() {
        let service = GrantService::new();
        let sink = crate::audit::VecSink::default();
        let staff = Some(Session {
            account_id: 1,
            role: Role::Admin,
        });
        let base = GrantRequest {
            idempotency_key: "k".to_string(),
            character_id: 9,
            kind: GrantKind::Gold { amount: 100 },
        };
        assert_eq!(
            request_grant(
                staff,
                GrantRequest {
                    idempotency_key: "  ".to_string(),
                    ..base.clone()
                },
                &service,
                &sink
            ),
            Err(AdminError::InvalidInput {
                reason: "chave de idempotência vazia"
            })
        );
        assert_eq!(
            request_grant(
                staff,
                GrantRequest {
                    kind: GrantKind::Gold { amount: 0 },
                    ..base
                },
                &service,
                &sink
            ),
            Err(AdminError::InvalidInput {
                reason: "quantidade do grant inválida"
            })
        );
    }
}
