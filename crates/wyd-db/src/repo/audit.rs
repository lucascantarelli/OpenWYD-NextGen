//! Porta da trilha de auditoria (`audit_log`, migration 0005).
//!
//! Duas operações, ambas Cold Path: **gravar** o que o staff fez e **ler** as
//! linhas recentes para o painel. A tabela é append-only: não existe update nem
//! delete aqui.
//!
//! O `actor` é gravado como `id:role` (`7:moderator`) sobre a coluna TEXT que a
//! série já tinha; a leitura separa os dois campos. A tabela nasceu sem FK de
//! propósito (0005) — auditoria nunca falha por causa da conta auditada.
//!
//! A porta usa futuro empacotado (`Pin<Box<...>>`) porque o roteador do gateway
//! guarda o adapter como trait object: o transporte não tem runtime (ADR-0006) e
//! não pode ser genérico sobre o pool.

use std::future::Future;
use std::pin::Pin;

use crate::error::DbError;
use crate::model::Role;

/// Linha nova a gravar (quem/o quê/alvo/detalhe). `at` é do banco.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NewAuditEntry {
    pub actor_id: i64,
    pub actor_role: Role,
    pub action: String,
    pub target: String,
    pub detail: String,
}

/// Linha lida da trilha (somente leitura para o painel).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AuditRow {
    /// Milissegundos desde a época UTC (o banco guarda `TIMESTAMPTZ`).
    pub at_ms: u64,
    pub actor_id: i64,
    pub actor_role: String,
    pub action: String,
    pub target: String,
    pub detail: String,
}

/// Trilha append-only: grava uma linha e devolve as mais recentes.
pub trait AuditTrail {
    /// Grava uma linha; erro de banco é do chamador (nunca engolido na porta).
    fn record<'a>(
        &'a self,
        entry: &'a NewAuditEntry,
    ) -> Pin<Box<dyn Future<Output = Result<(), DbError>> + Send + 'a>>;

    /// Últimas `limit` linhas, da mais nova para a mais antiga.
    fn recent(
        &self,
        limit: usize,
    ) -> Pin<Box<dyn Future<Output = Result<Vec<AuditRow>, DbError>> + Send + '_>>;

    /// Últimas `limit` linhas do ator, opcionalmente filtradas por ação.
    ///
    /// É o histórico do **próprio** jogador na área dele (`auth.login`); a
    /// consulta filtra pelo id do ator, nunca por login em texto.
    fn by_actor(
        &self,
        actor_id: i64,
        action: Option<&str>,
        limit: usize,
    ) -> Pin<Box<dyn Future<Output = Result<Vec<AuditRow>, DbError>> + Send + '_>>;
}
