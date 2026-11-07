//! Serviço de grants: regra + idempotência por transação rastreável.
//!
//! Fluxo: valida → checa chave (replay devolve recibo) → enfileira na
//! `DeliveryQueue` → guarda recibo. Retry do browser nunca duplica entrega.
//! O `grant_id` aqui é sequencial em memória; a implementação `pg` (04-3)
//! usa o `RETURNING id` da `delivery_queue` (deduplicação durável via UNIQUE
//! na chave — pendência registrada, não improvisada).

use std::collections::HashMap;
use std::sync::{
    atomic::{AtomicI64, Ordering},
    Mutex,
};

use wyd_db::{DeliveryQueue, Grant, QueuedGrant};

use crate::api::{GrantKind, GrantReceipt, GrantRequest};
use crate::auth::StaffSession;
use crate::error::AdminError;

/// Orquestra grants com idempotência em memória (Cold Path; Mutex local, sem global).
#[derive(Debug, Default)]
pub struct GrantService {
    queue: Mutex<Option<DeliveryQueue>>,
    seen: Mutex<HashMap<String, GrantReceipt>>,
    next_id: AtomicI64,
}

impl GrantService {
    #[must_use]
    pub fn new() -> Self {
        Self {
            queue: Mutex::new(None),
            seen: Mutex::new(HashMap::new()),
            next_id: AtomicI64::new(1),
        }
    }

    /// Conecta a fila de entrega (configuração fria, uma vez).
    pub fn attach_queue(&self, queue: DeliveryQueue) {
        *self.queue.lock().expect("serviço acessível") = Some(queue);
    }

    /// Aplica o grant ou devolve o recibo do replay (idempotente).
    pub fn apply(
        &self,
        _staff: StaffSession,
        req: &GrantRequest,
    ) -> Result<GrantReceipt, AdminError> {
        if let Some(receipt) = self
            .seen
            .lock()
            .expect("serviço acessível")
            .get(&req.idempotency_key)
        {
            let mut replay = receipt.clone();
            replay.replay = true;
            return Ok(replay);
        }
        let grant = to_domain(req);
        let id = self.next_id.fetch_add(1, Ordering::Relaxed);
        let guard = self.queue.lock().expect("serviço acessível");
        let queue = guard.as_ref().ok_or(AdminError::InvalidInput {
            reason: "fila de entrega não conectada",
        })?;
        queue
            .enqueue(QueuedGrant { id, grant })
            .map_err(AdminError::from)?;
        let receipt = GrantReceipt {
            idempotency_key: req.idempotency_key.clone(),
            grant_id: id,
            replay: false,
        };
        self.seen
            .lock()
            .expect("serviço acessível")
            .insert(req.idempotency_key.clone(), receipt.clone());
        Ok(receipt)
    }
}

fn to_domain(req: &GrantRequest) -> Grant {
    match &req.kind {
        GrantKind::Item { item_id, qty } => Grant::Item {
            character_id: req.character_id,
            item_id: *item_id,
            qty: *qty,
        },
        GrantKind::Gold { amount } => Grant::Gold {
            character_id: req.character_id,
            amount: *amount,
        },
        GrantKind::Level { delta } => Grant::Level {
            character_id: req.character_id,
            delta: *delta,
        },
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::auth::Session;
    use wyd_db::Role;

    fn staff() -> StaffSession {
        StaffSession::require(Session {
            account_id: 7,
            role: Role::Admin,
        })
        .expect("admin é staff")
    }

    fn req(key: &str) -> GrantRequest {
        GrantRequest {
            idempotency_key: key.to_string(),
            character_id: 9,
            kind: GrantKind::Gold { amount: 100 },
        }
    }

    #[test]
    fn retry_returns_original_receipt_without_second_enqueue() {
        let service = GrantService::new();
        let queue = DeliveryQueue::bounded(8).expect("fila válida");
        service.attach_queue(queue);
        let first = service
            .apply(staff(), &req("op-1"))
            .expect("primeira aplica");
        assert!(!first.replay);
        let second = service
            .apply(staff(), &req("op-1"))
            .expect("replay devolve recibo");
        assert!(second.replay);
        assert_eq!(first.grant_id, second.grant_id);
    }

    #[test]
    fn without_queue_fails_explicitly() {
        let service = GrantService::new();
        assert_eq!(
            service.apply(staff(), &req("op-x")),
            Err(AdminError::InvalidInput {
                reason: "fila de entrega não conectada"
            })
        );
    }
}
