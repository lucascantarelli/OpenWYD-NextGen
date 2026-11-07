//! Sessão de staff — identidade vem do cookie `httpOnly`, nunca do corpo.
//!
//! O portal emite/verifica o token (HS256); aqui a sessão já verificada vira
//! contexto tipado. `moderator_id` sai de `StaffSession::moderator_id()` —
//! nenhum handler recebe id de autor por parâmetro.

use wyd_db::Role;

use crate::error::AdminError;

/// Sessão verificada pelo BFF (sub do JWT + papel do login).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Session {
    pub account_id: i64,
    pub role: Role,
}

/// Contexto de staff: só `moderator`/`admin` constroem (player é 403).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct StaffSession {
    session: Session,
}

impl StaffSession {
    /// Promove a sessão verificada; fora de staff → `Forbidden`.
    pub fn require(session: Session) -> Result<Self, AdminError> {
        match session.role {
            Role::Moderator | Role::Admin => Ok(Self { session }),
            Role::Player => Err(AdminError::Forbidden {
                reason: "papel de staff necessário",
            }),
        }
    }

    /// Id do autor para auditoria e `moderator_id` das intenções.
    #[must_use]
    pub fn moderator_id(&self) -> i64 {
        self.session.account_id
    }

    #[must_use]
    pub fn role(&self) -> Role {
        self.session.role
    }

    #[must_use]
    pub fn is_admin(&self) -> bool {
        self.session.role == Role::Admin
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn session(role: Role) -> Session {
        Session {
            account_id: 7,
            role,
        }
    }

    #[test]
    fn player_is_forbidden_and_staff_passes() {
        assert_eq!(
            StaffSession::require(session(Role::Player)),
            Err(AdminError::Forbidden {
                reason: "papel de staff necessário"
            })
        );
        let staff = StaffSession::require(session(Role::Moderator)).expect("moderador é staff");
        assert_eq!(staff.moderator_id(), 7);
        assert!(!staff.is_admin());
        let admin = StaffSession::require(session(Role::Admin)).expect("admin é staff");
        assert!(admin.is_admin());
    }
}
