//! Persistência Cold Path (`wyd-db`): modelo, portas, fila e config.
//!
//! PostgreSQL via migrations versionadas (`migrations/`, seeds em `seeds/`
//! só com dados de desenvolvimento). O pool `sqlx`/async e a implementação
//! `pg` chegam na 04-3 (exigem rede + ADR — ver `docs/rust/01-dependencias.md`).
//!
//! DSN real nunca versionado: vem de ambiente e só o esquema é validado aqui.

pub mod error;
pub mod model;
pub mod pool;
pub mod repo;
pub mod transfer;

/// Versão do schema = última migration aplicada (`0006_delivery_queue`).
pub const SCHEMA_VERSION: u32 = 6;

/// Esquema esperado da string de conexão.
pub const EXPECTED_SCHEME: &str = "postgres://";

/// Verifica apenas o esquema do DSN (validação completa vira com `sqlx`).
#[must_use]
pub fn is_postgres_dsn(dsn: &str) -> bool {
    dsn.starts_with(EXPECTED_SCHEME)
}

pub use error::DbError;
pub use model::{Account, AuditEntry, Character, Guild, GuildMember, Role, MAX_SLOTS};
pub use pool::PoolConfig;
pub use repo::{Accounts, Characters, Grant, Grants};
pub use transfer::{DeliveryQueue, QueuedGrant};

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn accepts_only_postgres_scheme() {
        assert!(is_postgres_dsn("postgres://localhost/wyd"));
        assert!(!is_postgres_dsn("mysql://localhost/wyd"));
    }

    #[test]
    fn schema_version_matches_latest_migration() {
        assert_eq!(SCHEMA_VERSION, 6);
    }
}
