//! Transferência web→jogo: fila de grants (única ponte permitida).
//!
//! Só declara e re-exporta; o transporte mora em `delivery_queue`.

pub mod delivery_queue;

pub use delivery_queue::{DeliveryQueue, QueuedGrant};
