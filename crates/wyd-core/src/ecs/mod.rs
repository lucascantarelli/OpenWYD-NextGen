//! ECS/DOD: componentes contíguos (dados) + armazém SoA.
//!
//! Só declara e re-exporta; a forma dos dados mora em cada módulo.

pub mod components;
pub mod resources;

pub use components::{
    gem_effect_code, gem_value, AppliedGem, ComponentStore, Entity, EntityKind, Inventory, Item,
    Position, Stats, BAG_SLOTS, EQUIP_SLOTS, GEM_EFFECT_ABSORB, GEM_EFFECT_DROP, GEM_EFFECT_EXP,
    GEM_EFFECT_NONE, GEM_EFFECT_PIERCE, GEM_MAX_REFINE, GEM_MIN_REFINE,
};
pub use resources::{DomainEvent, DomainEvents, GemEffect, EVENT_QUEUE_CAPACITY};
