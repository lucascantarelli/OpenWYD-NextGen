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

    /// Cria a conta (login validado antes; UNIQUE estoura como `ConstraintViolation`).
    fn create(
        &self,
        login: &str,
        role: Role,
    ) -> impl Future<Output = Result<Account, DbError>> + Send;
}
