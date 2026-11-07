//! Limites do transporte: tetos por conexão/IP e backpressure.
//!
//! Só declara e re-exporta; balde e porte moram em `rate`.

pub mod rate;

pub use rate::{default_bucket, ConnectionGate, Deny, TokenBucket};
