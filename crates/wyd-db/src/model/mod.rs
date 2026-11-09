//! Modelos de domínio — dados que espelham as tabelas, sem SQL.
//!
//! Só declara e re-exporta; cada entidade mora no seu módulo.

pub mod account;
pub mod audit;
pub mod character;
pub mod guild;

pub use account::{Account, AccountModeration, Role};
pub use audit::AuditEntry;
pub use character::{Character, MAX_SLOTS};
pub use guild::{Guild, GuildMember};
