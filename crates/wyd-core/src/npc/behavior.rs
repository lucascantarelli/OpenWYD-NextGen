//! FSM da criatura — transições puras, uma função, sem I/O.
//!
//! Ordem de precedência (documentada, não incidental):
//! `Flee` (HP baixo) → `Return` (fora do leash) → `Attack` (no alcance) → `Chase` (alvo vivo)
//! → `Aggro` (alvo novo) → `Patrol` (rota ancorada) → `Idle`.
//!
//! A ordem importa: fuga vence tudo (criatura ferida não segura linha), leash vence perseguição
//! (não existe mob atravessando o mapa) e ataque só ocorre com alvo no alcance declarado.

use crate::ecs::components::Position;
use crate::npc::brain::NpcProfile;
use crate::npc::state::NpcMode;
use crate::systems::movement::distance;

/// Transição de estado para este ciclo de decisão.
#[must_use]
pub fn transition(
    current: NpcMode,
    hp_bps: u32,
    anchor: Position,
    position: Position,
    target: Option<(u32, Position)>,
    profile: NpcProfile,
) -> NpcMode {
    if hp_bps <= profile.flee_hp_bps {
        return NpcMode::Flee;
    }
    if distance(anchor, position) > profile.leash_range {
        return NpcMode::Return;
    }
    match target {
        Some((_, alvo)) => {
            let distancia = distance(position, alvo);
            if NpcMode::should_aggro(distancia, profile.attack_range) {
                NpcMode::Attack
            } else if NpcMode::should_aggro(distancia, profile.aggro_range) {
                NpcMode::Chase
            } else {
                // Alvo saiu do alcance de aggro: solta e volta a patrulhar.
                NpcMode::Aggro
            }
        }
        None => match current {
            NpcMode::Patrol | NpcMode::Aggro | NpcMode::Chase | NpcMode::Attack => NpcMode::Patrol,
            NpcMode::Flee | NpcMode::Return => NpcMode::Return,
            NpcMode::Idle => NpcMode::Idle,
        },
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn perfil() -> NpcProfile {
        NpcProfile::new(6, 12, 2, 2_000, 4).expect("perfil válido")
    }

    #[test]
    fn hp_baixo_foge_antes_de_tudo() {
        assert_eq!(
            transition(
                NpcMode::Attack,
                1_500,
                Position { x: 0, y: 0 },
                Position { x: 0, y: 0 },
                Some((1, Position { x: 1, y: 0 })),
                perfil()
            ),
            NpcMode::Flee
        );
    }

    #[test]
    fn fora_do_leash_volta_para_a_ancora() {
        assert_eq!(
            transition(
                NpcMode::Chase,
                10_000,
                Position { x: 0, y: 0 },
                Position { x: 30, y: 0 },
                Some((1, Position { x: 31, y: 0 })),
                perfil()
            ),
            NpcMode::Return
        );
    }

    #[test]
    fn alcance_de_ataque_vence_perseguicao() {
        assert_eq!(
            transition(
                NpcMode::Chase,
                10_000,
                Position { x: 0, y: 0 },
                Position { x: 0, y: 0 },
                Some((1, Position { x: 2, y: 0 })),
                perfil()
            ),
            NpcMode::Attack
        );
        assert_eq!(
            transition(
                NpcMode::Chase,
                10_000,
                Position { x: 0, y: 0 },
                Position { x: 0, y: 0 },
                Some((1, Position { x: 5, y: 0 })),
                perfil()
            ),
            NpcMode::Chase
        );
    }

    #[test]
    fn sem_alvo_patrulha_ou_fica_idle() {
        assert_eq!(
            transition(
                NpcMode::Idle,
                10_000,
                Position::default(),
                Position::default(),
                None,
                perfil()
            ),
            NpcMode::Idle
        );
        assert_eq!(
            transition(
                NpcMode::Patrol,
                10_000,
                Position::default(),
                Position::default(),
                None,
                perfil()
            ),
            NpcMode::Patrol
        );
    }
}
