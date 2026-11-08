//! Transferência web→jogo e Hot→Cold: fila de grants (única ponte permitida) e fila
//! de save (persistência fora do tick).
//!
//! Só declara e re-exporta; o transporte mora nos módulos.

pub mod delivery_queue;
pub mod save;

pub use delivery_queue::{DeliveryQueue, QueuedGrant};
pub use save::{FlushReport, SaveDrain, SavePolicy, SaveQueue, SaveRequest};
