//! Gameplay do canal: refino, loot, EXP, quest, portões e inventário.
//!
//! ADR-0026 (etapa 4): a fachada só declara o domínio; as regras ficam em
//! [`service`], que lê a configuração viva no instante da chamada.

mod service;

pub use service::*;
