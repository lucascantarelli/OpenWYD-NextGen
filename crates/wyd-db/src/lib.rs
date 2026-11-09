//! Persistência Cold Path (`wyd-db`): modelo, portas, adapters, filas e config.
//!
//! PostgreSQL via migrations versionadas (`migrations/`, seeds em `seeds/`
//! só com dados de desenvolvimento). O pool `sqlx` e as implementações `pg` das portas
//! entraram na FASE-04-3 (ADR-0021); a fila de save (`SaveQueue`/`SaveDrain`) leva o
//! estado do Hot Path ao banco **fora** do tick.
//!
//! Duas pontes, nenhuma no quente:
//!
//! - `DeliveryQueue` (web → jogo): grants que o portal enfileira e o servidor drena;
//! - `SaveQueue` (jogo → banco): snapshots que o tick enfileira sem bloquear e o dreno
//!   Cold Path grava em transação curta.
//!
//! DSN real nunca versionado: vem de ambiente e só o esquema é validado aqui.

pub mod error;
pub mod model;
pub mod pool;
pub mod repo;
pub mod transfer;

/// Versão do schema = número da última migration (`0031_moderacao`).
///
/// Invariante cobrada por `tests/catalogo.rs`: este número tem de acompanhar o último
/// arquivo da série. Antes ficou em 6 com 25 migrations aplicadas — catálogo e save mentindo
/// juntos é pior que não ter versão.
pub const SCHEMA_VERSION: u32 = 31;

/// Esquema esperado da string de conexão.
pub const EXPECTED_SCHEME: &str = "postgres://";

/// Verifica apenas o esquema do DSN (validação completa vira com `sqlx`).
#[must_use]
pub fn is_postgres_dsn(dsn: &str) -> bool {
    dsn.starts_with(EXPECTED_SCHEME)
}

pub use error::DbError;
pub use model::{
    Account, AccountModeration, AuditEntry, Character, Guild, GuildMember, Role, MAX_SLOTS,
};
pub use pool::{PgBackend, PoolConfig};
pub use repo::{
    Accounts, AuditRow, AuditTrail, CatalogItemRow, CatalogSnapshot, Characters, CityTaxTotal,
    ClaimOutcome, CredentialRecord, Economy, EconomySummary, Grant, GrantTotal, Grants,
    NewAuditEntry, QueuedGrantRow, ServerChannel, Sessions, NO_CREDENTIAL,
};
pub use transfer::{
    DeliveryQueue, FlushReport, QueuedGrant, SaveDrain, SavePolicy, SaveQueue, SaveRequest,
};

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
        assert_eq!(SCHEMA_VERSION, 31);
        // A amarração de verdade (arquivo mais alto da série) está em tests/catalogo.rs.
    }
}
