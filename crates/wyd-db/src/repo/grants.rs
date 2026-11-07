//! Porta de grants — leitura da `delivery_queue` pelo dreno do servidor.
//!
//! A web/admin ENFILEIRA (Cold Path); o servidor DRENA e aplica via
//! `transfer::DeliveryQueue` (sem web escrevendo estado vivo — AGENTS.md §3.7).

use std::future::Future;

use crate::error::DbError;

/// Grant validado vindo da fila (espelha as colunas tipadas da tabela).
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Grant {
    Item {
        character_id: i64,
        item_id: i64,
        qty: i64,
    },
    Gold {
        character_id: i64,
        amount: i64,
    },
    Level {
        character_id: i64,
        delta: i64,
    },
}

/// Leitura e baixa da fila persistida.
pub trait Grants {
    /// Próximos grants `pending` (ordem de criação, no máximo `limit`).
    fn pending(
        &self,
        limit: u32,
    ) -> impl Future<Output = Result<Vec<(i64, Grant)>, DbError>> + Send;

    /// Marca como `claimed` após aplicação (idempotente por id).
    fn mark_claimed(&self, id: i64) -> impl Future<Output = Result<(), DbError>> + Send;
}
