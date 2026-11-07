//! Conta de acesso — espelho da tabela `accounts` (migration 0001).

use crate::error::DbError;

/// Papel de acesso (espelha o CHECK `accounts_role`).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Role {
    Player,
    Moderator,
    Admin,
}

impl Role {
    #[must_use]
    pub fn as_db(&self) -> &'static str {
        match self {
            Self::Player => "player",
            Self::Moderator => "moderator",
            Self::Admin => "admin",
        }
    }
}

/// Conta persistida (hashes opacos: nunca logar, nunca devolver ao cliente).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Account {
    pub id: i64,
    pub login: String,
    pub role: Role,
}

impl Account {
    /// Invariante estrutural mínima (o banco impõe o resto via CHECK/UNIQUE).
    pub fn validate_login(login: &str) -> Result<(), DbError> {
        if login.trim().is_empty() || login.len() > 32 {
            return Err(DbError::ConstraintViolation {
                table: "accounts",
                constraint: "accounts_login_nonempty",
            });
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn login_validation_rejects_blank_and_too_long() {
        assert!(Account::validate_login("test").is_ok());
        assert!(Account::validate_login("   ").is_err());
        assert!(Account::validate_login(&"a".repeat(33)).is_err());
    }
}
