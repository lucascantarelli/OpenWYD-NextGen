//! Engine Hot Path (FASE-03): tick determinístico, ECS contíguo, grade espacial.
//!
//! Invariantes que não podem ser quebradas:
//!
//! - Um único processo por nó de mundo; tick em frequência fixa (20–50 Hz).
//! - Estado como dados contíguos (`ecs`), nunca OOP espalhada no heap.
//! - Sem GC, sem mutex global, sem I/O bloqueante e sem alocação no quente.
//! - Servidor é autoridade (validação server-side chega em 03-5/FASE-10).
//! - Tempo entra pela porta [`ports::Clock`]; o scheduler puro recebe ms.
//!
//! Detalhes em `pre-release-rust.md` FASE 3 e `docs/rust/00-visao-estrutura.md`.

pub mod ecs;
pub mod error;
pub mod metrics;
pub mod npc;
pub mod ports;
pub mod spatial;
pub mod systems;
pub mod telemetry;
pub mod tick;

/// Ticks por segundo do game loop deterministico.
pub const TICK_HZ: u32 = 20;

/// Intervalo entre ticks em milissegundos.
#[must_use]
pub fn tick_interval_ms() -> u64 {
    1000 / u64::from(TICK_HZ)
}

pub use ecs::{
    gem_effect_code, gem_value, AppliedGem, ComponentStore, Entity, EntityKind, Inventory, Item,
    Position, Stats, BAG_SLOTS, EQUIP_SLOTS, GEM_EFFECT_ABSORB, GEM_EFFECT_DROP, GEM_EFFECT_EXP,
    GEM_EFFECT_NONE, GEM_EFFECT_PIERCE, GEM_MAX_REFINE, GEM_MIN_REFINE,
};
pub use ecs::{DomainEvent, DomainEvents, GemEffect, EVENT_QUEUE_CAPACITY};
pub use error::CoreError;
pub use metrics::{Metrics, MetricsServer, MetricsSource, TickSnapshot};
pub use npc::{Decision, Intention, NpcMode, NpcProfile, NpcState, NpcStates};
pub use ports::{elapsed_ms, Clock, ManualClock, SystemClock};
pub use spatial::{Grid, GridConfig};
pub use systems::{
    anct, apply_powder, attack, aylin, cast, downgrade_refine, equip, equip_into, jewel_effect,
    move_item, odin, plus12_failure, powder_failure, roll_115, roll_199, step, unequip,
    AttackRules, Cast, ComposeOutcome, ComposeRules, DamageModel, EquipmentRules, Equipped, Hit,
    ItemFacts, ItemRules, MoveRules, OdinRecipe, Plus12Failure, PowderFailure, PowderOutcome,
    SkillBook, SkillDef, Unequipped,
};
pub use tick::{Scheduler, SchedulerConfig, Tick, MAX_HZ, MAX_STEPS_PER_ADVANCE, MIN_HZ};

/// Reexporta o marcador autoritativo de cena para o game loop.
pub use wyd_protocol::ESCENE_FIELD;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tick_interval_matches_20hz() {
        assert_eq!(TICK_HZ, 20);
        assert_eq!(tick_interval_ms(), 50);
    }
}
