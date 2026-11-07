//! Repositórios: portas (traits) que o Cold Path consome.
//!
//! Só declara e re-exporta; a implementação `pg` chega na 04-3 (sqlx via ADR).

pub mod accounts;
pub mod characters;
pub mod grants;

pub use accounts::Accounts;
pub use characters::Characters;
pub use grants::{Grant, Grants};
