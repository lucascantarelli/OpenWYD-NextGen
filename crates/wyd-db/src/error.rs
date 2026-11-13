//! Erros de persistência — nunca carregam DSN, senha ou segredo no `Display`.
//!
//! O DSN vive fora do repo (variável de ambiente); nenhuma variante o armazena.

use core::fmt;

/// Erro de domínio da persistência (fronteira que o produziu no nome da variante).
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum DbError {
    /// DSN ausente ou com esquema inválido (esperado `postgres://`).
    InvalidDsn,
    /// Constraint violada (FK/CHECK/UNIQUE) — falha explícita, nunca correção.
    ConstraintViolation {
        table: &'static str,
        constraint: &'static str,
    },
    /// Entidade inexistente para leitura/escrita pontual.
    NotFound { entity: &'static str, id: i64 },
    /// Entidade inexistente em busca por chave natural (id desconhecido).
    NotFoundKey { entity: &'static str },
    /// Pool esgotado (backpressure: o chamador aguarda, nunca abre nova conexão).
    PoolSaturated { max: u32 },
    /// Fila de save cheia (Hot Path recusa em vez de bloquear o tick).
    SaveQueueFull { capacity: usize },
    /// Shutdown não conseguiu esvaziar a fila de save no tempo da política.
    SaveTimeout { ms: u64 },
    /// Falha de transporte/execução (operação nomeada, sem detalhe sensível).
    Transport { op: &'static str },
}

impl fmt::Display for DbError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::InvalidDsn => {
                write!(f, "DSN inválido (esperado esquema postgres://)")
            }
            Self::ConstraintViolation { table, constraint } => {
                write!(f, "constraint violada em {table}: {constraint}")
            }
            Self::NotFound { entity, id } => {
                write!(f, "{entity} id={id} não encontrado")
            }
            Self::NotFoundKey { entity } => {
                write!(f, "{entity} não encontrado")
            }
            Self::PoolSaturated { max } => {
                write!(f, "pool esgotado (máximo {max} conexões)")
            }
            Self::SaveQueueFull { capacity } => {
                write!(f, "fila de save cheia ({capacity} pendentes)")
            }
            Self::SaveTimeout { ms } => {
                write!(f, "shutdown não esvaziou a fila de save em {ms} ms")
            }
            Self::Transport { op } => {
                write!(f, "falha de persistência em {op}")
            }
        }
    }
}

impl std::error::Error for DbError {}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn display_never_leaks_secrets() {
        // Nenhuma variante carrega DSN/senha: o pior caso é o nome da operação.
        let msgs = [
            DbError::InvalidDsn.to_string(),
            DbError::ConstraintViolation {
                table: "characters",
                constraint: "characters_account_slot_unique",
            }
            .to_string(),
            DbError::NotFound {
                entity: "account",
                id: 7,
            }
            .to_string(),
            DbError::NotFoundKey { entity: "account" }.to_string(),
            DbError::PoolSaturated { max: 10 }.to_string(),
            DbError::SaveQueueFull { capacity: 4_096 }.to_string(),
            DbError::SaveTimeout { ms: 3_000 }.to_string(),
            DbError::Transport {
                op: "accounts.create",
            }
            .to_string(),
        ];
        for m in msgs {
            // O nome do esquema é público; segredo seria credencial/host.
            assert!(!m.contains('@'), "vazamento de DSN: {m}");
            assert!(!m.to_lowercase().contains("password"), "vazamento: {m}");
            assert!(!m.to_lowercase().contains("secret"), "vazamento: {m}");
        }
    }
}
