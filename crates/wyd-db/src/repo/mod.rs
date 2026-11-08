//! Repositórios: portas (traits) que o Cold Path consome + adapter `pg`.
//!
//! As portas não conhecem SQL; `pg` implementa as três sobre o [`crate::PgBackend`].

pub mod accounts;
pub mod characters;
pub mod content;
pub mod grants;
pub mod pg;

pub use accounts::Accounts;
pub use characters::Characters;
pub use grants::{Grant, Grants};
pub use pg::NO_CREDENTIAL;
