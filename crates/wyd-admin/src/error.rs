//! Erros do control plane — cada variante vira um status HTTP no BFF.
//!
//! Espelha o `AdminResult` do portal (`portal/src/server/admin-result.ts`):
//! OK=1→200, FORBIDDEN=2→403, INVALID=3→422, NOT_FOUND=4→404, CONTENT_OWNED=5→409.
//! Falha de transporte/infra é 500 com mensagem genérica (sem stack/paths).

use core::fmt;

use wyd_db::DbError;

/// Erro de domínio administrativo (nunca carrega segredo ou DSN).
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum AdminError {
    /// Sem sessão de staff (equivale a 401 do BFF).
    Unauthorized,
    /// Sessão válida sem papel suficiente (equivale a 403).
    Forbidden { reason: &'static str },
    /// Alvo inexistente (equivale a 404).
    NotFound { entity: &'static str, id: i64 },
    /// Entrada inválida (equivale a 422).
    InvalidInput { reason: &'static str },
    /// Conflito de negócio — ex.: última conta admin (equivale a 409).
    Conflict { reason: &'static str },
}

impl AdminError {
    /// Status HTTP que o BFF devolve para cada variante.
    #[must_use]
    pub fn http_status(&self) -> u16 {
        match self {
            Self::Unauthorized => 401,
            Self::Forbidden { .. } => 403,
            Self::NotFound { .. } => 404,
            Self::InvalidInput { .. } => 422,
            Self::Conflict { .. } => 409,
        }
    }
}

impl fmt::Display for AdminError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Unauthorized => write!(f, "sessão de staff ausente ou inválida"),
            Self::Forbidden { reason } => write!(f, "acesso negado: {reason}"),
            Self::NotFound { entity, id } => write!(f, "{entity} id={id} não encontrado"),
            Self::InvalidInput { reason } => write!(f, "entrada inválida: {reason}"),
            Self::Conflict { reason } => write!(f, "conflito: {reason}"),
        }
    }
}

impl std::error::Error for AdminError {}

/// Mapeia erro de persistência para o contrato HTTP (sem vazar detalhe de banco).
impl From<DbError> for AdminError {
    fn from(e: DbError) -> Self {
        match e {
            DbError::InvalidDsn | DbError::Transport { .. } | DbError::PoolSaturated { .. } => {
                Self::InvalidInput {
                    reason: "falha temporária, tente de novo",
                }
            }
            DbError::ConstraintViolation { .. } => Self::InvalidInput {
                reason: "dados conflitam com as regras",
            },
            DbError::NotFound { entity, id } => Self::NotFound { entity, id },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn status_matches_portal_admin_result_map() {
        assert_eq!(AdminError::Unauthorized.http_status(), 401);
        assert_eq!(AdminError::Forbidden { reason: "x" }.http_status(), 403);
        assert_eq!(
            AdminError::NotFound { entity: "x", id: 1 }.http_status(),
            404
        );
        assert_eq!(AdminError::InvalidInput { reason: "x" }.http_status(), 422);
        assert_eq!(AdminError::Conflict { reason: "x" }.http_status(), 409);
    }

    #[test]
    fn db_errors_never_leak_transport_details() {
        let mapped: AdminError = DbError::Transport { op: "x" }.into();
        assert_eq!(
            mapped,
            AdminError::InvalidInput {
                reason: "falha temporária, tente de novo"
            }
        );
    }
}
