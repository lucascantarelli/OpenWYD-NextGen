//! Cérebro da criatura: percepção → alvo → intenção, tudo inteiro e determinístico.
//!
//! P1 vê candidatos dentro do alcance de aggro (inclusivo), P2 escolhe o alvo pela tabela de
//! ameaça (empate pelo menor `entity`) e P3 devolve **uma** intenção. Nenhuma intenção é
//! aplicada aqui: quem move/ataca são `systems/movement.rs` e `systems/combat.rs`, com as
//! mesmas validações que um pedido de cliente enfrentaria (ADR-0016).

use crate::ecs::components::{Entity, Position};
use crate::error::CoreError;
use crate::npc::behavior::transition;
use crate::npc::state::{NpcMode, NpcState};
use crate::systems::movement::distance;

/// Perfil de comportamento da criatura — **configuração do servidor**, não constante da engine.
///
/// Vem do conteúdo carregado no boot (hoje `npc_definition`/futura `npc_behavior`; até lá o
/// chamador informa). Nenhum número de balanceamento é fixado no core.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct NpcProfile {
    /// Distância (tiles) em que a criatura percebe um alvo.
    pub aggro_range: u32,
    /// Distância máxima da âncora antes de voltar (leash).
    pub leash_range: u32,
    /// Distância para atacar.
    pub attack_range: u32,
    /// HP (em bps de 10.000) abaixo do qual foge.
    pub flee_hp_bps: u32,
    /// Raio de patrulha em torno da âncora.
    pub patrol_radius: u32,
}

impl NpcProfile {
    /// Perfil válido ou erro explícito (alcance zero, leash menor que aggro, fuga > 100%).
    pub fn new(
        aggro_range: u32,
        leash_range: u32,
        attack_range: u32,
        flee_hp_bps: u32,
        patrol_radius: u32,
    ) -> Result<Self, CoreError> {
        if aggro_range == 0 {
            return Err(CoreError::InvalidNpcProfile {
                field: "aggro_range",
                value: 0,
            });
        }
        if leash_range < aggro_range {
            return Err(CoreError::InvalidNpcProfile {
                field: "leash_range",
                value: leash_range,
            });
        }
        if attack_range == 0 || attack_range > aggro_range {
            return Err(CoreError::InvalidNpcProfile {
                field: "attack_range",
                value: attack_range,
            });
        }
        if flee_hp_bps > 10_000 {
            return Err(CoreError::InvalidNpcProfile {
                field: "flee_hp_bps",
                value: flee_hp_bps,
            });
        }
        Ok(Self {
            aggro_range,
            leash_range,
            attack_range,
            flee_hp_bps,
            patrol_radius,
        })
    }
}

/// O que a criatura vê de uma entidade (dado puro, sem `Stats` completo).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Observed {
    /// Identidade estável.
    pub entity: Entity,
    /// Posição autoritativa.
    pub position: Position,
    /// HP em bps (10.000 = cheio).
    pub hp_bps: u32,
}

/// Intenção emitida pela IA — o sistema valida antes de aplicar.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Intention {
    /// Nada a fazer.
    Idle,
    /// Mover um passo em direção ao ponto (o orçamento de velocidade é do `step`).
    MoveTo { to: Position },
    /// Atacar o alvo (alcance/cooldown conferidos pelo `attack`).
    Attack { target: u32 },
    /// Fugir do alvo.
    Flee { from: u32, to: Position },
    /// Voltar para a âncora.
    Return { to: Position },
}

impl Intention {
    /// Nome estável para o golden (`kind` da linha `<tick>,<entity>,<kind>,<arg0>,<arg1>`).
    #[must_use]
    pub fn kind(&self) -> &'static str {
        match self {
            Self::Idle => "idle",
            Self::MoveTo { .. } => "move",
            Self::Attack { .. } => "attack",
            Self::Flee { .. } => "flee",
            Self::Return { .. } => "return",
        }
    }
}

/// Decisão completa do ciclo: estado novo + intenção.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Decision {
    /// Estado da FSM depois da transição.
    pub mode: NpcMode,
    /// Alvo escolhido (`Entity.index`).
    pub target: Option<u32>,
    /// Intenção para os sistemas aplicarem.
    pub intention: Intention,
}

/// Um passo determinístico em direção a `to`, limitado a `tiles` por eixo.
#[must_use]
fn step_towards(from: Position, to: Position, tiles: u32) -> Position {
    let limite = i32::try_from(tiles.max(1)).unwrap_or(i32::MAX);
    Position {
        x: from.x + (to.x - from.x).clamp(-limite, limite),
        y: from.y + (to.y - from.y).clamp(-limite, limite),
    }
}

/// Decide o ciclo: percepção (P1), alvo (P2) e intenção (P3). Função pura.
///
/// `candidates` deve vir em ordem de índice de entidade (o chamador itera o SoA); a seleção
/// ignora a ordem porque desempata pelo menor `entity`.
pub fn decide(
    state: &mut NpcState,
    me: Observed,
    candidates: &[Observed],
    tick: u32,
    profile: NpcProfile,
) -> Decision {
    // P1 — percepção: só quem está dentro do alcance vira candidato.
    for candidato in candidates {
        if candidato.entity.index == me.entity.index {
            continue;
        }
        let distancia = distance(me.position, candidato.position);
        if NpcMode::should_aggro(distancia, profile.aggro_range)
            && state.threat.threat_of(candidato.entity.index).is_none()
        {
            // Ameaça mínima por proximidade; o dano real soma por `combat` (D-IA02).
            let _ = state.threat.add(candidato.entity.index, 1, tick);
        }
    }
    let _ = state.threat.prune(tick, profile.leash_range);

    // P2 — alvo pela tabela, nunca por ordem de chegada.
    let alvo_index = state.threat.select_target(tick, 0);
    let alvo = alvo_index.and_then(|index| {
        candidates
            .iter()
            .find(|candidato| candidato.entity.index == index)
            .copied()
    });

    // P3 — estado e intenção.
    let mode = transition(
        state.mode,
        me.hp_bps,
        state.anchor,
        me.position,
        alvo.map(|candidato| (candidato.entity.index, candidato.position)),
        profile,
    );
    let intention = match mode {
        NpcMode::Attack => Intention::Attack {
            target: alvo.map_or(0, |candidato| candidato.entity.index),
        },
        NpcMode::Chase => Intention::MoveTo {
            to: alvo.map_or(me.position, |candidato| {
                step_towards(me.position, candidato.position, profile.attack_range)
            }),
        },
        NpcMode::Flee => Intention::Flee {
            from: alvo.map_or(0, |candidato| candidato.entity.index),
            to: step_towards(me.position, state.anchor, profile.aggro_range),
        },
        NpcMode::Return => Intention::Return {
            to: step_towards(me.position, state.anchor, profile.aggro_range),
        },
        NpcMode::Patrol | NpcMode::Aggro | NpcMode::Idle => Intention::Idle,
    };

    state.mode = mode;
    state.target = alvo.map(|candidato| candidato.entity.index);
    state.next_decision_tick = tick.saturating_add(crate::npc::budget::DECISION_EVERY_TICKS);
    Decision {
        mode,
        target: state.target,
        intention,
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn entity(index: u32) -> Entity {
        Entity {
            index,
            generation: 0,
        }
    }

    fn perfil() -> NpcProfile {
        NpcProfile::new(6, 20, 2, 2_000, 3).expect("perfil")
    }

    fn npc(anchor: Position) -> NpcState {
        NpcState::spawn(anchor, 7, 0)
    }

    #[test]
    fn ve_alvo_no_limite_inclusivo_do_alcance() {
        let mut estado = npc(Position { x: 10, y: 10 });
        let eu = Observed {
            entity: entity(0),
            position: Position { x: 10, y: 10 },
            hp_bps: 10_000,
        };
        let no_limite = Observed {
            entity: entity(1),
            position: Position { x: 16, y: 10 },
            hp_bps: 10_000,
        };
        let decisao = decide(&mut estado, eu, &[no_limite], 0, perfil());
        assert_eq!(decisao.target, Some(1));
        assert_eq!(decisao.mode, NpcMode::Chase, "fora do alcance de ataque");
    }

    #[test]
    fn nao_ve_alvo_fora_do_alcance() {
        let mut estado = npc(Position { x: 10, y: 10 });
        let eu = Observed {
            entity: entity(0),
            position: Position { x: 10, y: 10 },
            hp_bps: 10_000,
        };
        let longe = Observed {
            entity: entity(1),
            position: Position { x: 17, y: 10 },
            hp_bps: 10_000,
        };
        let decisao = decide(&mut estado, eu, &[longe], 0, perfil());
        assert_eq!(decisao.target, None);
        assert_eq!(decisao.intention, Intention::Idle);
    }

    #[test]
    fn ataca_quando_o_alvo_esta_no_alcance() {
        let mut estado = npc(Position { x: 10, y: 10 });
        let eu = Observed {
            entity: entity(0),
            position: Position { x: 10, y: 10 },
            hp_bps: 10_000,
        };
        let perto = Observed {
            entity: entity(4),
            position: Position { x: 11, y: 10 },
            hp_bps: 10_000,
        };
        let decisao = decide(&mut estado, eu, &[perto], 0, perfil());
        assert_eq!(decisao.intention, Intention::Attack { target: 4 });
    }

    #[test]
    fn hp_baixo_foge_na_direcao_da_ancora() {
        let mut estado = npc(Position { x: 10, y: 10 });
        let eu = Observed {
            entity: entity(0),
            position: Position { x: 12, y: 10 },
            hp_bps: 1_000,
        };
        let perto = Observed {
            entity: entity(4),
            position: Position { x: 13, y: 10 },
            hp_bps: 10_000,
        };
        let decisao = decide(&mut estado, eu, &[perto], 0, perfil());
        assert_eq!(
            decisao.intention,
            Intention::Flee {
                from: 4,
                to: Position { x: 10, y: 10 }
            }
        );
    }

    #[test]
    fn perfil_invalido_falha_no_boot() {
        assert!(NpcProfile::new(0, 10, 1, 0, 0).is_err());
        assert!(NpcProfile::new(10, 5, 1, 0, 0).is_err(), "leash < aggro");
        assert!(NpcProfile::new(10, 20, 11, 0, 0).is_err(), "ataque > aggro");
        assert!(
            NpcProfile::new(10, 20, 2, 10_001, 0).is_err(),
            "fuga > 100%"
        );
    }
}
