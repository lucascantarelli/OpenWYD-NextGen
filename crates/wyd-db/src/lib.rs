//! Persistencia async Cold Path (esqueleto).
//!
//! Destino: PostgreSQL via `sqlx` com migrations versionadas, sem I/O
//! bloqueante no tick loop. Sem dependencias externas neste esqueleto.

/// Esquema esperado da string de conexao neste esqueleto.
pub const EXPECTED_SCHEME: &str = "postgres://";

/// Verifica apenas o esquema do DSN (validacao completa vira com `sqlx`).
#[must_use]
pub fn is_postgres_dsn(dsn: &str) -> bool {
    dsn.starts_with(EXPECTED_SCHEME)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn accepts_only_postgres_scheme() {
        assert!(is_postgres_dsn("postgres://localhost/wyd"));
        assert!(!is_postgres_dsn("mysql://localhost/wyd"));
    }
}
