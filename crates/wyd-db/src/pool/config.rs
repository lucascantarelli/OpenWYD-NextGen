//! Configuração dimensionada do pool (dados validados; o pool `sqlx` chega na 04-3).
//!
//! Transações curtas e timeouts agressivos: persistência é Cold Path, mas nunca
//! pode segurar o servidor. DSN vem de ambiente — nunca versionado, nunca aqui.

use crate::error::DbError;

/// Limites do pool (valores em milissegundos onde indicado).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct PoolConfig {
    /// Conexões simultâneas (1–64: servidor local não precisa de mais).
    pub max_connections: u32,
    /// Tempo para conseguir conexão antes de `PoolSaturated`.
    pub acquire_timeout_ms: u64,
    /// Teto de cada statement (trava query fugitiva).
    pub statement_timeout_ms: u64,
    /// Teto de cada transação (curtas por contrato).
    pub transaction_max_ms: u64,
}

impl PoolConfig {
    /// Padrão do servidor local: 10 conexões, timeouts curtos.
    #[must_use]
    pub fn local() -> Self {
        Self {
            max_connections: 10,
            acquire_timeout_ms: 2_000,
            statement_timeout_ms: 5_000,
            transaction_max_ms: 10_000,
        }
    }

    /// Valida faixas (zero/desligado é recusado: falha explícita, nunca default
    /// silencioso que esconda contenção).
    pub fn validated(self) -> Result<Self, DbError> {
        if !(1..=64).contains(&self.max_connections) {
            return Err(DbError::ConstraintViolation {
                table: "pool",
                constraint: "max_connections 1–64",
            });
        }
        if self.acquire_timeout_ms == 0 || self.statement_timeout_ms == 0 {
            return Err(DbError::ConstraintViolation {
                table: "pool",
                constraint: "timeouts > 0",
            });
        }
        if self.transaction_max_ms < self.statement_timeout_ms {
            return Err(DbError::ConstraintViolation {
                table: "pool",
                constraint: "transaction_max_ms >= statement_timeout_ms",
            });
        }
        Ok(self)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn local_is_valid_and_bad_ranges_rejected() {
        assert!(PoolConfig::local().validated().is_ok());
        let zero_conn = PoolConfig {
            max_connections: 0,
            ..PoolConfig::local()
        };
        assert!(zero_conn.validated().is_err());
        let inverted = PoolConfig {
            statement_timeout_ms: 20_000,
            transaction_max_ms: 1_000,
            ..PoolConfig::local()
        };
        assert!(inverted.validated().is_err());
    }
}
