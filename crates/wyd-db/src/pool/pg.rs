//! Pool `sqlx` — o **adapter**, não a porta (PROC-16, ADR-0021).
//!
//! Regras que este arquivo garante:
//!
//! - aquisição com timeout curto: contenção vira `PoolSaturated`, nunca espera infinita;
//! - transação curta com `SET LOCAL statement_timeout` (o teto não vaza para a conexão
//!   devolvida ao pool);
//! - nenhum `await` alcançável do tick: quem chama `begin`/`flush` é o `SaveDrain`
//!   (Cold Path), nunca `systems/`;
//! - DSN não é armazenado nem aparece no `DbError` — o erro carrega só o nome da operação.

use std::str::FromStr;
use std::time::Duration;

use sqlx::postgres::{PgConnectOptions, PgPoolOptions};
use sqlx::{PgPool, Postgres, Transaction};

use crate::error::DbError;
use crate::pool::config::PoolConfig;

/// Backend Postgres: pool + config validada (o pool já nasce dimensionado).
#[derive(Debug, Clone)]
pub struct PgBackend {
    pool: PgPool,
    config: PoolConfig,
}

impl PgBackend {
    /// Conecta validando esquema do DSN e faixas do [`PoolConfig`].
    ///
    /// DSN malformado é `InvalidDsn` (não `Transport`): é erro de configuração, e o
    /// operador precisa da distinção para não procurar o problema no servidor de banco.
    pub async fn connect(dsn: &str, config: PoolConfig) -> Result<Self, DbError> {
        if !crate::is_postgres_dsn(dsn) {
            return Err(DbError::InvalidDsn);
        }
        let config = config.validated()?;
        let options = PgConnectOptions::from_str(dsn).map_err(|_| DbError::InvalidDsn)?;
        let pool = PgPoolOptions::new()
            .max_connections(config.max_connections)
            .acquire_timeout(Duration::from_millis(config.acquire_timeout_ms))
            .connect_with(options)
            .await
            .map_err(map_error("pool.connect", "pool"))?;
        Ok(Self { pool, config })
    }

    #[must_use]
    pub fn pool(&self) -> &PgPool {
        &self.pool
    }

    #[must_use]
    pub fn config(&self) -> PoolConfig {
        self.config
    }

    /// Prova que o pool responde (`SELECT 1`) — base do `/health` do gateway (ADR-0022).
    ///
    /// Não é `SELECT 1` decorativo: é a única forma de o healthcheck saber que a conexão
    /// ainda vale (banco reiniciado deixa o pool com conexões mortas até alguém tocar).
    pub async fn ping(&self) -> Result<(), DbError> {
        sqlx::query_scalar::<_, i32>("SELECT 1")
            .fetch_one(&self.pool)
            .await
            .map(|_| ())
            .map_err(map_error("pool.ping", "pool"))
    }

    /// Abre transação curta com o teto de statement do [`PoolConfig`].
    ///
    /// `SET LOCAL` vale só para a transação: a conexão volta ao pool sem carregar o
    /// timeout de quem a usou antes.
    pub async fn begin(&self) -> Result<Transaction<'_, Postgres>, DbError> {
        let mut tx = self
            .pool
            .begin()
            .await
            .map_err(map_error("pool.begin", "pool"))?;
        let ms = self.config.statement_timeout_ms;
        sqlx::query(&format!("SET LOCAL statement_timeout = {ms}"))
            .execute(&mut *tx)
            .await
            .map_err(map_error("pool.begin", "pool"))?;
        Ok(tx)
    }

    /// Fecha o pool esperando as conexões em uso (chamado no shutdown).
    pub async fn close(self) {
        self.pool.close().await;
    }
}

/// Mapeia `sqlx::Error` para o erro de domínio.
///
/// `table` só é usada em violação de constraint — o único caso em que o chamador
/// consegue corrigir alguma coisa. Todo o resto é `Transport { op }`, que é o que se
/// pode dizer sem vazar DSN, host ou SQL.
pub(crate) fn map_error(op: &'static str, table: &'static str) -> impl Fn(sqlx::Error) -> DbError {
    move |e| match e {
        sqlx::Error::PoolTimedOut | sqlx::Error::PoolClosed => DbError::PoolSaturated { max: 0 },
        sqlx::Error::RowNotFound => DbError::NotFound {
            entity: table,
            id: 0,
        },
        sqlx::Error::Database(db) => match db.code().as_deref() {
            Some("23505") => DbError::ConstraintViolation {
                table,
                constraint: "unique",
            },
            Some("23503") => DbError::ConstraintViolation {
                table,
                constraint: "foreign_key",
            },
            Some("23514") => DbError::ConstraintViolation {
                table,
                constraint: "check",
            },
            Some("23502") => DbError::ConstraintViolation {
                table,
                constraint: "not_null",
            },
            _ => DbError::Transport { op },
        },
        _ => DbError::Transport { op },
    }
}

#[cfg(test)]
mod tests {
    use std::future::Future;

    use super::*;

    #[test]
    fn rejects_non_postgres_dsn_before_touching_the_network() {
        let err = block_on(PgBackend::connect(
            "mysql://localhost/wyd",
            PoolConfig::local(),
        ));
        assert_eq!(err.err(), Some(DbError::InvalidDsn));
    }

    #[test]
    fn rejects_invalid_pool_config_before_connecting() {
        let bad = PoolConfig {
            max_connections: 0,
            ..PoolConfig::local()
        };
        let err = block_on(PgBackend::connect("postgres://127.0.0.1:1/wyd", bad));
        assert!(matches!(
            err.err(),
            Some(DbError::ConstraintViolation { table: "pool", .. })
        ));
    }

    /// Runtime mínimo de teste: os casos acima falham antes de qualquer I/O, mas a
    /// assinatura é `async`, então precisa de executor.
    fn block_on<F: Future>(f: F) -> F::Output {
        tokio::runtime::Builder::new_current_thread()
            .enable_all()
            .build()
            .expect("runtime de teste")
            .block_on(f)
    }
}
