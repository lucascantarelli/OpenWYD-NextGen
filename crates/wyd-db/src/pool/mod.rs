//! Pool de conexões (configuração dimensionada; pool `sqlx` na 04-3).
//!
//! Só declara e re-exporta; os limites moram em `config`.

pub mod config;

pub use config::PoolConfig;
