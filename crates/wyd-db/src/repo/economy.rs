//! Porta de leitura da economia real (WEB-03).
//!
//! Duas fontes, nenhuma inventada:
//!
//! - `city_tax_ledger` (migration 0010): livro-caixa do imposto, uma linha por
//!   recolhimento — a soma por cidade é o que existe, não uma projeção;
//! - `delivery_queue` (migration 0006): tudo o que o staff enfileirou para o
//!   servidor drenar (item/ouro/nível), a fonte auditável do que foi concedido.
//!
//! Tabela vazia é resposta legítima (`0`, lista vazia): o painel mostra a fonte
//! e diz que não há movimento, em vez de inventar número.

use std::future::Future;
use std::pin::Pin;

use crate::error::DbError;

/// Imposto arrecadado por cidade (soma do livro-caixa).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CityTaxTotal {
    pub city_code: String,
    pub total: i64,
}

/// Concessões por tipo de grant na fila de entrega.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GrantTotal {
    pub kind: String,
    pub count: i64,
    /// Soma de `gold_amount` (zero para `item`/`level`).
    pub gold: i64,
}

/// Foto da economia auditada.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct EconomySummary {
    pub tax_by_city: Vec<CityTaxTotal>,
    pub grants: Vec<GrantTotal>,
}

/// Leitura da economia real (Cold Path; nunca toca estado vivo de personagem).
pub trait Economy {
    fn summary(&self)
        -> Pin<Box<dyn Future<Output = Result<EconomySummary, DbError>> + Send + '_>>;
}
