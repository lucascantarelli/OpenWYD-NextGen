//! Integração da IA com o tick — ordem documentada, sem I/O e sem aplicar efeito.
//!
//! Fluxo por ciclo: entidade viva → é criatura → cadência de 5 Hz → orçamento do tick →
//! percepção/alvo/intenção ([`crate::npc::brain`]) → devolve a intenção. Nada é movido nem
//! atacado aqui: `movement::step` e `combat::attack` recebem a intenção como receberiam um
//! pedido de cliente, com as mesmas validações (ADR-0014/ADR-0016).
//!
//! Recusa é atômica: as checagens acontecem **antes** do primeiro `ensure`, então um pedido
//! forjado (entidade morta, jogador, índice inválido) não cria estado nem consome orçamento.

use crate::ecs::components::{ComponentStore, Entity, EntityKind};
use crate::error::CoreError;
use crate::npc::brain::{decide as decidir, Decision, NpcProfile, Observed};
use crate::npc::budget::{should_decide, DecisionBudget};
use crate::npc::state::NpcStates;

/// Proporção de HP em bps (10.000 = cheio); `max_hp = 0` nunca ocorre em entidade viva.
#[must_use]
fn hp_bps(hp: u32, max_hp: u32) -> u32 {
    if max_hp == 0 {
        return 0;
    }
    hp.min(max_hp).saturating_mul(10_000) / max_hp
}

/// Decide o ciclo de uma criatura. `Ok(None)` = fora da cadência ou sem orçamento no tick.
///
/// `candidates` é o conjunto que o mundo conhece (ordem de índice SoA); só entidades vivas do
/// tipo `Player` viram alvo, e a seleção desempata pelo menor `entity`, não pela ordem da fatia.
pub fn decide(
    store: &ComponentStore,
    entity: Entity,
    tick: u32,
    profile: NpcProfile,
    candidates: &[Entity],
    states: &mut NpcStates,
    budget: &mut DecisionBudget,
) -> Result<Option<Decision>, CoreError> {
    if !store.is_alive(entity) {
        return Err(CoreError::EntityNotAlive {
            index: entity.index,
            generation: entity.generation,
        });
    }
    let kind = store.kind_of(entity).ok_or(CoreError::EntityNotAlive {
        index: entity.index,
        generation: entity.generation,
    })?;
    if !matches!(kind, EntityKind::Npc | EntityKind::Monster) {
        return Err(CoreError::NotACreature {
            index: entity.index,
        });
    }
    if !should_decide(tick, entity.index) {
        return Ok(None);
    }
    if !budget.try_consume() {
        return Ok(None);
    }

    let position = store.position_of(entity).ok_or(CoreError::EntityNotAlive {
        index: entity.index,
        generation: entity.generation,
    })?;
    let stats = store.stats_of(entity).ok_or(CoreError::EntityNotAlive {
        index: entity.index,
        generation: entity.generation,
    })?;

    let mut vistos = [Observed {
        entity,
        position,
        hp_bps: 0,
    }; 32];
    let mut total = 0usize;
    for candidato in candidates.iter().take(31) {
        if candidato.index == entity.index || !store.is_alive(*candidato) {
            continue;
        }
        if store.kind_of(*candidato) != Some(EntityKind::Player) {
            continue;
        }
        let (Some(posicao), Some(alvo_stats)) =
            (store.position_of(*candidato), store.stats_of(*candidato))
        else {
            continue;
        };
        vistos[total] = Observed {
            entity: *candidato,
            position: posicao,
            hp_bps: hp_bps(alvo_stats.hp, alvo_stats.max_hp),
        };
        total += 1;
    }

    let eu = Observed {
        entity,
        position,
        hp_bps: hp_bps(stats.hp, stats.max_hp),
    };
    let estado = states.ensure(entity.index, position, u64::from(entity.generation));
    Ok(Some(decidir(estado, eu, &vistos[..total], tick, profile)))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ecs::components::{Inventory, Position, Stats};

    fn stats(hp: u32, max_hp: u32) -> Stats {
        Stats {
            level: 1,
            hp,
            max_hp,
            mp: 0,
            max_mp: 0,
            attack: 1,
            defense: 0,
        }
    }

    fn loja() -> (ComponentStore, Entity) {
        let mut store = ComponentStore::default();
        let npc = store
            .spawn(
                EntityKind::Monster,
                Position { x: 10, y: 10 },
                stats(100, 100),
                Inventory::default(),
            )
            .expect("spawn");
        (store, npc)
    }

    #[test]
    fn hp_bps_escala_inteira() {
        assert_eq!(hp_bps(100, 100), 10_000);
        assert_eq!(hp_bps(50, 100), 5_000);
        assert_eq!(hp_bps(0, 100), 0);
        assert_eq!(hp_bps(1, 0), 0);
    }

    #[test]
    fn jogador_nao_recebe_decisao_de_ia() {
        let mut store = ComponentStore::default();
        let player = store
            .spawn(
                EntityKind::Player,
                Position::default(),
                stats(100, 100),
                Inventory::default(),
            )
            .expect("spawn");
        let mut states = NpcStates::default();
        let mut budget = DecisionBudget::new(8);
        assert_eq!(
            decide(
                &store,
                player,
                0,
                NpcProfile::new(5, 10, 1, 0, 0).expect("perfil"),
                &[],
                &mut states,
                &mut budget
            ),
            Err(CoreError::NotACreature { index: 0 })
        );
        assert!(states.is_empty(), "recusa não cria estado");
        assert_eq!(budget.used, 0, "recusa não consome orçamento");
    }

    #[test]
    fn fora_da_cadencia_nao_decide() {
        let (store, npc) = loja();
        let mut states = NpcStates::default();
        let mut budget = DecisionBudget::new(8);
        let decisao = decide(
            &store,
            npc,
            1 + (npc.index % 4),
            NpcProfile::new(5, 10, 1, 0, 0).expect("perfil"),
            &[],
            &mut states,
            &mut budget,
        )
        .expect("vivo");
        assert_eq!(decisao, None);
        assert_eq!(budget.used, 0);
    }
}
