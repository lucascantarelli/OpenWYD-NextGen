//! Serviços: regra de negócio sobre portas e fila (sem transporte).
//!
//! Só declara e re-exporta; cada serviço mora no seu módulo.

pub mod grant_service;

pub use grant_service::GrantService;
