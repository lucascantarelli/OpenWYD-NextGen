//! Repositórios: portas (traits) que o Cold Path consome + adapter `pg`.
//!
//! As portas não conhecem SQL; `pg` implementa as três sobre o [`crate::PgBackend`].

pub mod accounts;
pub mod audit;
pub mod characters;
pub mod content;
pub mod economy;
pub mod grants;
pub mod pg;
pub mod sessions;

pub use accounts::{Accounts, CredentialRecord};
pub use audit::{AuditRow, AuditTrail, NewAuditEntry};
pub use characters::Characters;
pub use content::{CatalogItemRow, CatalogSnapshot, ServerChannel, SkillRule, SkillRules};
pub use economy::{CityTaxTotal, Economy, EconomySummary, GrantTotal};
pub use grants::{Grant, Grants, QueuedGrantRow};
pub use pg::NO_CREDENTIAL;
pub use sessions::{ClaimOutcome, Sessions};
