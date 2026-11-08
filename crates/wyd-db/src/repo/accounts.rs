//! Porta de contas — o que o Cold Path precisa, sem SQL aqui.
//!
//! `async fn` nativo em trait (estável desde 1.75, sem dependência externa);
//! a implementação `pg` com `sqlx` chega na 04-3 (exige rede + ADR).

use std::future::Future;

use crate::error::DbError;
use crate::model::{Account, Role};

/// Credencial de login: a conta e o hash guardado.
///
/// O hash **nunca** sai da borda de autenticação (`wyd-admin`): o BFF só recebe
/// `ok`/`role`, nunca `password_hash`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CredentialRecord {
    pub account: Account,
    pub password_hash: String,
}

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

    /// Hash da credencial para a verificação de login (borda de autenticação).
    ///
    /// Conta inexistente é `NotFoundKey` — quem chama responde a mesma recusa
    /// genérica de senha errada, sem vazar existência.
    fn credentials(
        &self,
        login: &str,
    ) -> impl Future<Output = Result<CredentialRecord, DbError>> + Send {
        let _ = login;
        async { Err(DbError::NotFoundKey { entity: "account" }) }
    }

    /// Define/atualiza a credencial de uma conta (seed de staff, troca de senha).
    ///
    /// `password_hash` já vem calculado (argon2id); a porta não decide algoritmo.
    fn set_credentials(
        &self,
        login: &str,
        password_hash: &str,
        role: Role,
    ) -> impl Future<Output = Result<Account, DbError>> + Send {
        let _ = (login, password_hash, role);
        async { Err(DbError::NotFoundKey { entity: "account" }) }
    }
}
