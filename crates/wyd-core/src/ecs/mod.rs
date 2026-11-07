//! ECS/DOD: componentes contíguos (dados) + armazém SoA.
//!
//! Só declara e re-exporta; a forma dos dados mora em cada módulo.

pub mod components;

pub use components::{
    ComponentStore, Entity, EntityKind, Inventory, Item, Position, Stats, BAG_SLOTS, EQUIP_SLOTS,
};
