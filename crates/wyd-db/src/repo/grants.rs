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

/// Linha da fila vista pelo staff (estado + rastro), sem estado vivo de jogo.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct QueuedGrantRow {
    pub id: i64,
    pub kind: String,
    pub item_id: Option<i64>,
    pub qty: Option<i64>,
    /// Ouro concedido (`gold_amount`); 0 para item/nível.
    pub gold: i64,
    pub level_delta: Option<i64>,
    /// `pending` (aguardando o dreno) ou `claimed` (aplicado na back).
    pub status: String,
    pub idempotency_key: Option<String>,
    pub created_by: Option<i64>,
    /// Instante de criação em milissegundos (epoch), para a tela ordenar.
    pub at_ms: i64,
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

    /// Enfileira o grant de forma **durável e idempotente** (Cold Path).
    ///
    /// `key` é a chave do pedido do portal: a primeira gravação cria a linha;
    /// repetir a mesma chave devolve o mesmo `id` com `replay = true` e **não**
    /// cria segunda entrega. `by` é a conta de staff que pediu.
    fn enqueue(
        &self,
        key: &str,
        by: i64,
        grant: &Grant,
    ) -> impl Future<Output = Result<(i64, bool), DbError>> + Send;

    /// Aplica o gold da linha pendente ao personagem e marca `claimed`.
    ///
    /// Uma instrução só: o crédito e a baixa acontecem juntos, então um erro no
    /// meio não deixa gold entregue sem baixa (nem baixa sem gold). Devolve
    /// `None` quando a linha não é um gold pendente — já entregue ou outro tipo.
    fn claim_gold(
        &self,
        id: i64,
    ) -> impl Future<Output = Result<Option<(i64, i64)>, DbError>> + Send;

    /// Linhas da fila de um personagem (mais recentes primeiro), com estado.
    ///
    /// Somente leitura: staff confere o que foi pedido e o que já foi entregue;
    /// o portal nunca marca nada como aplicado.
    fn list_for_character(
        &self,
        character_id: i64,
        limit: usize,
    ) -> impl Future<Output = Result<Vec<QueuedGrantRow>, DbError>> + Send;
}
