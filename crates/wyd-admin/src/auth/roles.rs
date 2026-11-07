//! Papéis e escopos — quem pode o quê (regra server-side, duas vezes).
//!
//! A guarda de rota do portal é UX; a permissão real é checada aqui, por chamada.

use wyd_db::Role;

/// Ações de moderação podem mirar a conta?
pub fn may_target(actor: Role, target: Role) -> bool {
    match (actor, target) {
        // Admin mira qualquer um; moderador só jogador comum.
        (Role::Admin, _) => true,
        (Role::Moderator, Role::Player) => true,
        (Role::Moderator, _) | (Role::Player, _) => false,
    }
}

/// Troca de papel exige admin (promover/rebaixar nunca é ato de moderador).
pub fn may_change_role(actor: Role) -> bool {
    actor == Role::Admin
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn moderator_only_touches_players_and_never_roles() {
        assert!(may_target(Role::Admin, Role::Admin));
        assert!(may_target(Role::Moderator, Role::Player));
        assert!(!may_target(Role::Moderator, Role::Moderator));
        assert!(!may_target(Role::Player, Role::Player));
        assert!(may_change_role(Role::Admin));
        assert!(!may_change_role(Role::Moderator));
    }
}
