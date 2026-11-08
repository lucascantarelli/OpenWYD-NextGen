//! Sessão de rede: o que decide se uma conexão entra no mundo.
//!
//! Árvore da ADR-0002 §11: `state.rs` (typestate `Connected → Handshaked → Authenticated`) e
//! `handshake.rs` (INITCODE + rate-limit). Hoje existe o gate de versão do login — o resto do
//! handshake chega com o transporte (FASE-08).

pub mod handshake;

pub use handshake::{VersionGate, VersionPolicy};
