//! Fila de entrega em memória: grants do wyd-portal/admin → dreno do servidor.
//!
//! Backpressure por canal limitado (`sync_channel`): fila cheia recusa com
//! `PoolSaturated` em vez de crescer sem limite. Dreno não-bloqueante
//! (`drain`): o servidor consome sem esperar a web. Persistência da fila na
//! tabela `delivery_queue` (migration 0006); esta estrutura é o transporte.

use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::mpsc::{sync_channel, Receiver, SyncSender, TrySendError};

use crate::error::DbError;
use crate::repo::Grant;

/// Pedido de grant com identidade (id da linha persistida, para `mark_claimed`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct QueuedGrant {
    pub id: i64,
    pub grant: Grant,
}

/// Fila limitada de grants (produtor: web/admin; consumidor: dreno do servidor).
#[derive(Debug)]
pub struct DeliveryQueue {
    tx: SyncSender<QueuedGrant>,
    rx: Receiver<QueuedGrant>,
    pending: AtomicUsize,
}

impl DeliveryQueue {
    /// Cria a fila com teto de pendentes (capacidade 0 é recusada: fila que
    /// nunca entrega esconde perda — falha explícita).
    pub fn bounded(capacity: usize) -> Result<Self, DbError> {
        if capacity == 0 {
            return Err(DbError::ConstraintViolation {
                table: "delivery_queue",
                constraint: "capacity > 0",
            });
        }
        let (tx, rx) = sync_channel(capacity);
        Ok(Self {
            tx,
            rx,
            pending: AtomicUsize::new(0),
        })
    }

    /// Enfileira sem bloquear; cheia → `PoolSaturated` (backpressure explícito).
    pub fn enqueue(&self, grant: QueuedGrant) -> Result<(), DbError> {
        self.tx.try_send(grant).map_err(|e| match e {
            TrySendError::Full(_) => DbError::PoolSaturated { max: 0 },
            TrySendError::Disconnected(_) => DbError::Transport {
                op: "delivery.enqueue",
            },
        })?;
        self.pending.fetch_add(1, Ordering::Relaxed);
        Ok(())
    }

    /// Drena até `limit` itens sem bloquear (chamado pelo loop do servidor).
    pub fn drain(&self, limit: usize) -> Vec<QueuedGrant> {
        let got: Vec<QueuedGrant> = self.rx.try_iter().take(limit).collect();
        self.pending.fetch_sub(got.len(), Ordering::Relaxed);
        got
    }

    /// Pendentes agora (observabilidade; contador atômico — fora do quente).
    #[must_use]
    pub fn len(&self) -> usize {
        self.pending.load(Ordering::Relaxed)
    }

    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.len() == 0
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn gold(id: i64) -> QueuedGrant {
        QueuedGrant {
            id,
            grant: Grant::Gold {
                character_id: 1,
                amount: 100,
            },
        }
    }

    #[test]
    fn bounded_rejects_zero_and_full_applies_backpressure() {
        assert!(DeliveryQueue::bounded(0).is_err());
        let q = DeliveryQueue::bounded(2).expect("capacidade válida");
        assert!(q.enqueue(gold(1)).is_ok());
        assert!(q.enqueue(gold(2)).is_ok());
        assert_eq!(q.enqueue(gold(3)), Err(DbError::PoolSaturated { max: 0 }));
        assert_eq!(q.len(), 2);
    }

    #[test]
    fn drain_is_fifo_and_non_blocking() {
        let q = DeliveryQueue::bounded(8).expect("capacidade válida");
        assert!(q.drain(4).is_empty());
        for id in 1..=5 {
            q.enqueue(gold(id)).expect("cabe");
        }
        let got = q.drain(3);
        assert_eq!(got.iter().map(|g| g.id).collect::<Vec<_>>(), vec![1, 2, 3]);
        let rest = q.drain(10);
        assert_eq!(rest.iter().map(|g| g.id).collect::<Vec<_>>(), vec![4, 5]);
        assert!(q.is_empty());
    }
}
