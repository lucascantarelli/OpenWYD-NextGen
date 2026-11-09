//! Porta de personagens — o que o Cold Path precisa, sem SQL aqui.

use std::future::Future;

use crate::error::DbError;
use crate::model::Character;

/// Operações sobre `characters` (soft-delete: apaga marcando `deleted_at`).
pub trait Characters {
    /// Personagens vivos da conta, ordenados por slot.
    fn list_by_account(
        &self,
        account_id: i64,
    ) -> impl Future<Output = Result<Vec<Character>, DbError>> + Send;

    /// Ranking por nível (desc) e nome (asc) — leitura pública, sem `account_id`.
    fn top_by_level(
        &self,
        limit: usize,
    ) -> impl Future<Output = Result<Vec<Character>, DbError>> + Send;

    /// Personagens vivos cujo nome contém `term` (vazio = todos), por nome.
    ///
    /// Leitura de staff para a tela de entregas: devolve o destino com `gold`
    /// atual, para a concessão ser conferida antes de enfileirar.
    fn search_by_name(
        &self,
        term: &str,
        limit: usize,
    ) -> impl Future<Output = Result<Vec<Character>, DbError>> + Send;

    /// Cria validando invariantes (o banco impõe o resto).
    ///
    /// Nome sem ambiguidade com `Accounts::create_account` (mesmo adapter, duas portas).
    fn create_character(
        &self,
        character: &Character,
    ) -> impl Future<Output = Result<Character, DbError>> + Send;

    /// Apaga marcando `deleted_at` (nome/slot liberados pelo índice parcial).
    fn soft_delete(&self, id: i64) -> impl Future<Output = Result<(), DbError>> + Send;
}
