//! Serviços: regra de negócio sobre portas e fila (sem transporte).
//!
//! Só declara e re-exporta; cada serviço mora no seu módulo.

pub mod grant_service;
pub mod pin;

pub use grant_service::{GrantQueue, GrantService, GrantTrail, MemoryGrantQueue};
pub use pin::{change_pin, valid_pin, PinError};
