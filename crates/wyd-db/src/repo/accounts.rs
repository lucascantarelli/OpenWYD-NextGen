//! Porta de contas — o que o Cold Path precisa, sem SQL aqui.
//!
//! `async fn` nativo em trait (estável desde 1.75, sem dependência externa);
//! a implementação `pg` com `sqlx` chega na 04-3 (exige rede + ADR).

use std::future::Future;
use std::pin::Pin;

use crate::error::DbError;
use crate::model::{Account, AccountModeration, ItemRow, Role};

/// Credencial de login: a conta e o hash guardado.
///
/// O hash **nunca** sai da borda de autenticação (`wyd-admin`): o BFF só recebe
/// `ok`/`role`, nunca `password_hash`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CredentialRecord {
    pub account: Account,
    pub password_hash: String,
    /// Conta bloqueada pela moderação: a borda de autenticação recusa antes de
    /// comparar a senha, tanto no jogo quanto no portal.
    pub blocked: bool,
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

    /// Hash da senha numérica (PIN) da conta; `None` = primeira entrada, ainda não cadastrado.
    ///
    /// Conta inexistente é `NotFoundKey` — a borda de autenticação traduz para a mesma recusa.
    fn pin_hash(
        &self,
        login: &str,
    ) -> impl Future<Output = Result<Option<String>, DbError>> + Send {
        let _ = login;
        async { Err(DbError::NotFoundKey { entity: "account" }) }
    }

    /// Grava o hash do PIN (Argon2id calculado na borda, nunca texto plano).
    fn set_pin_hash(
        &self,
        login: &str,
        pin_hash: &str,
    ) -> impl Future<Output = Result<(), DbError>> + Send {
        let _ = (login, pin_hash);
        async { Err(DbError::NotFoundKey { entity: "account" }) }
    }

    /// Estado do bloqueio do PIN: `(falhas acumuladas, segundos restantes)`.
    ///
    /// Persistido em `accounts` — reiniciar o processo do canal não zera tentativas.
    fn pin_guard(&self, login: &str) -> impl Future<Output = Result<(i64, i64), DbError>> + Send {
        let _ = login;
        async { Err(DbError::NotFoundKey { entity: "account" }) }
    }

    /// Soma uma falha de PIN e arma o bloqueio ao atingir `limite`.
    ///
    /// Devolve `(falhas, segundos de bloqueio)`; `0` segundos = ainda não bloqueado.
    fn pin_failure(
        &self,
        login: &str,
        limite: i64,
        segundos: i64,
    ) -> impl Future<Output = Result<(i64, i64), DbError>> + Send {
        let _ = (login, limite, segundos);
        async { Err(DbError::NotFoundKey { entity: "account" }) }
    }

    /// Zera falhas e bloqueio (PIN correto ou troca concluída).
    fn pin_success(&self, login: &str) -> impl Future<Output = Result<(), DbError>> + Send {
        let _ = login;
        async { Err(DbError::NotFoundKey { entity: "account" }) }
    }

    /// Estado de moderação pelo login (painel de staff).
    ///
    /// Fonte: `accounts` + `account_moderation` (0031). Conta sem linha de moderação
    /// está liberada — ausência de bloqueio, nunca bloqueio implícito.
    fn moderation_state<'a>(
        &'a self,
        login: &'a str,
    ) -> Pin<Box<dyn Future<Output = Result<AccountModeration, DbError>> + Send + 'a>> {
        let _ = login;
        Box::pin(async { Err(DbError::NotFoundKey { entity: "account" }) })
    }

    /// Troca o papel da conta (`UPDATE accounts`), devolvendo o estado novo.
    ///
    /// A permissão (só admin) é decidida na borda `wyd-admin`, não aqui.
    fn set_role<'a>(
        &'a self,
        login: &'a str,
        role: Role,
    ) -> Pin<Box<dyn Future<Output = Result<AccountModeration, DbError>> + Send + 'a>> {
        let _ = (login, role);
        Box::pin(async { Err(DbError::NotFoundKey { entity: "account" }) })
    }

    /// Bloqueia/libera a conta com motivo e autor (id do staff na sessão).
    fn set_blocked<'a>(
        &'a self,
        login: &'a str,
        blocked: bool,
        reason: &'a str,
        by: i64,
    ) -> Pin<Box<dyn Future<Output = Result<AccountModeration, DbError>> + Send + 'a>> {
        let _ = (login, blocked, reason, by);
        Box::pin(async { Err(DbError::NotFoundKey { entity: "account" }) })
    }

    /// Cofre da conta (`vault_items`) no `Cargo[128]` do `MSG_DBCNFAccountLogin`.
    ///
    /// Sem linha a resposta é vazia — o pacote manda slots vazios, nunca item sintético.
    fn vault_items(
        &self,
        account_id: i64,
    ) -> impl Future<Output = Result<Vec<ItemRow>, DbError>> + Send {
        let _ = account_id;
        async { Ok(Vec::new()) }
    }

    /// Moeda da conta (`account_wallet.coin`); `None` = conta sem linha.
    ///
    /// Separada de `characters.gold`: o `Coin` do pacote de login é saldo **da conta**.
    fn account_coin(
        &self,
        account_id: i64,
    ) -> impl Future<Output = Result<Option<i64>, DbError>> + Send {
        let _ = account_id;
        async { Ok(None) }
    }

    /// Grava o cofre da conta (`vault_items`), upsert por slot.
    ///
    /// O depósito/retirada do jogo entra pela porta de jogo; esta escrita existe para o
    /// cofre ter lastro persistido e o login poder ser conferido contra o banco.
    fn store_vault_items(
        &self,
        account_id: i64,
        items: &[ItemRow],
    ) -> impl Future<Output = Result<(), DbError>> + Send {
        let _ = (account_id, items);
        async { Ok(()) }
    }

    /// Grava o `Coin` da conta (`account_wallet`); teto `i32::MAX` porque o campo do fio é `int`.
    fn set_account_coin(
        &self,
        account_id: i64,
        coin: i64,
    ) -> impl Future<Output = Result<(), DbError>> + Send {
        let _ = (account_id, coin);
        async { Ok(()) }
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
