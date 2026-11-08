//! Pool dimensionado (`config`) e o adapter Postgres (`pg`).
//!
//! A config é dado puro e testável; o adapter é a única peça que fala `sqlx`.

pub mod config;
pub mod pg;

pub use config::PoolConfig;
pub use pg::PgBackend;
