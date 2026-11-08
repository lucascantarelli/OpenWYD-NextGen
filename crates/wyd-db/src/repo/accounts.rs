//! Porta de contas — o que o Cold Path precisa, sem SQL aqui.
//!
//! `async fn` nativo em trait (estável desde 1.75, sem dependência externa);
//! a implementação `pg` com `sqlx` chega na 04-3 (exige rede + ADR).

use std::future::Future;

use crate::error::DbError;
use crate::model::{Account, Role};

/// Operações sobre `accounts` (implementação `pg` futura; fakes em teste).
pub trait Accounts {
    /// Busca por login exato (retorna `NotFound`, nunca `None` silencioso).
    fn find_by_login(&self, login: &str) -> impl Future<Output = Result<Account, DbError>> + Send;

    /// Busca por id (painel do jogador — leitura da própria conta).
    fn find_by_id(&self, id: i64) -> impl Future<Output = Result<Account, DbError>> + Send;

    /// Cria a conta (login validado antes; UNIQUE estoura como `ConstraintViolation`).
    ///
    /// Nome sem ambiguidade: `PgBackend` implementa também `Characters`, e dois `create`
    /// no mesmo tipo obrigariam todo chamador a qualificar a trait.
    fn create_account(
        &self,
        login: &str,
        role: Role,
    ) -> impl Future<Output = Result<Account, DbError>> + Send;
}
