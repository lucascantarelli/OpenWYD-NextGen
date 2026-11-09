//! Serviço de grants: regra + entrega durável e idempotente.
//!
//! Fluxo: valida → enfileira na `delivery_queue` pela porta [`GrantQueue`] →
//! devolve recibo com o id da linha. A idempotência é **do banco**
//! (`idempotency_key` UNIQUE, migration 0032): repetir a chave devolve o mesmo
//! `grant_id` com `replay = true` e não cria segunda entrega. Quem aplica é o
//! servidor de jogo (`Grants::pending`/`claim_gold`), nunca o portal.

use std::sync::{Arc, Mutex};

use wyd_db::{DbError, Grant, QueuedGrantRow};

use crate::api::{GrantKind, GrantReceipt, GrantRequest};
use crate::auth::StaffSession;
use crate::error::AdminError;

/// Porta de enfileiramento durável (o gateway liga no pool; teste usa memória).
///
/// Síncrona por desenho: o roteador do Cold Path não tem runtime (ADR-0006) e o
/// gateway entrega o futuro já resolvido, como faz com `Accounts`/`AuditTrail`.
pub trait GrantQueue: Send + Sync {
    /// Grava o grant e devolve `(id, replay)`; a mesma chave nunca gera duas linhas.
    fn enqueue(&self, key: &str, by: i64, grant: &Grant) -> Result<(i64, bool), DbError>;
}

/// Leitura da fila para a tela de staff (somente leitura).
///
/// Porta **síncrona**: o roteador do Cold Path não tem runtime e as telas de
/// admin conferem o que foi pedido, sem escrever estado de entrega.
pub trait GrantTrail: Send + Sync {
    fn list_for_character(
        &self,
        character_id: i64,
        limit: usize,
    ) -> Result<Vec<QueuedGrantRow>, DbError>;
}

/// Fila em memória com a **mesma semântica** de idempotência da tabela.
///
/// Para teste e desenvolvimento local; produção usa `Grants::enqueue` no pool.
#[derive(Debug, Default)]
pub struct MemoryGrantQueue {
    rows: Mutex<Vec<(String, i64, Grant)>>,
}

impl MemoryGrantQueue {
    #[must_use]
    pub fn len(&self) -> usize {
        self.rows.lock().expect("fila acessível").len()
    }

    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.len() == 0
    }
}

impl GrantTrail for MemoryGrantQueue {
    fn list_for_character(
        &self,
        character_id: i64,
        limit: usize,
    ) -> Result<Vec<QueuedGrantRow>, DbError> {
        let rows = self.rows.lock().expect("fila acessível");
        let mut out: Vec<QueuedGrantRow> = rows
            .iter()
            .filter(|(_, _, grant)| match grant {
                Grant::Item {
                    character_id: id, ..
                }
                | Grant::Gold {
                    character_id: id, ..
                }
                | Grant::Level {
                    character_id: id, ..
                } => *id == character_id,
            })
            .map(|(key, id, grant)| {
                let (kind, item_id, qty, gold, level_delta) = match grant {
                    Grant::Item { item_id, qty, .. } => {
                        ("item", Some(*item_id), Some(*qty), 0, None)
                    }
                    Grant::Gold { amount, .. } => ("gold", None, None, *amount, None),
                    Grant::Level { delta, .. } => ("level", None, None, 0, Some(*delta)),
                };
                QueuedGrantRow {
                    id: *id,
                    kind: kind.to_string(),
                    item_id,
                    qty,
                    gold,
                    level_delta,
                    status: "pending".to_string(),
                    idempotency_key: Some(key.clone()),
                    created_by: None,
                    at_ms: 0,
                }
            })
            .collect();
        out.reverse();
        out.truncate(limit);
        Ok(out)
    }
}

impl GrantQueue for MemoryGrantQueue {
    fn enqueue(&self, key: &str, _by: i64, grant: &Grant) -> Result<(i64, bool), DbError> {
        let mut rows = self.rows.lock().expect("fila acessível");
        if let Some((_, id, _)) = rows.iter().find(|(k, _, _)| k == key) {
            return Ok((*id, true));
        }
        let id = rows.len() as i64 + 1;
        rows.push((key.to_string(), id, grant.clone()));
        Ok((id, false))
    }
}

/// Orquestra grants com idempotência durável (Cold Path; sem estado de jogo).
pub struct GrantService {
    queue: Mutex<Option<Arc<dyn GrantQueue>>>,
}

impl Default for GrantService {
    fn default() -> Self {
        Self::new()
    }
}

impl std::fmt::Debug for GrantService {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let conectada = self.queue.lock().map(|q| q.is_some()).unwrap_or(false);
        f.debug_struct("GrantService")
            .field("queue_connected", &conectada)
            .finish()
    }
}

impl GrantService {
    #[must_use]
    pub fn new() -> Self {
        Self {
            queue: Mutex::new(None),
        }
    }

    /// Conecta a fila durável (configuração fria, uma vez).
    pub fn attach_queue(&self, queue: Arc<dyn GrantQueue>) {
        *self.queue.lock().expect("serviço acessível") = Some(queue);
    }

    /// Enfileira o grant ou devolve o recibo do replay (idempotente por chave).
    pub fn apply(
        &self,
        staff: StaffSession,
        req: &GrantRequest,
    ) -> Result<GrantReceipt, AdminError> {
        let grant = to_domain(req);
        let guard = self.queue.lock().expect("serviço acessível");
        let queue = guard.as_ref().ok_or(AdminError::InvalidInput {
            reason: "fila de entrega não conectada",
        })?;
        let (grant_id, replay) = queue
            .enqueue(&req.idempotency_key, staff.moderator_id(), &grant)
            .map_err(AdminError::from)?;
        Ok(GrantReceipt {
            idempotency_key: req.idempotency_key.clone(),
            grant_id,
            replay,
        })
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
        let queue = Arc::new(MemoryGrantQueue::default());
        service.attach_queue(queue.clone());
        let first = service
            .apply(staff(), &req("op-1"))
            .expect("primeira aplica");
        assert!(!first.replay);
        let second = service
            .apply(staff(), &req("op-1"))
            .expect("replay devolve recibo");
        assert!(second.replay);
        assert_eq!(first.grant_id, second.grant_id);
        assert_eq!(queue.len(), 1, "replay não enfileira segunda entrega");
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
