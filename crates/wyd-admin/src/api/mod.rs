//! Endpoints de controle por domínio (nunca estado vivo de personagem).
//!
//! Só declara e re-exporta; cada domínio mora no seu módulo.

pub mod accounts;
pub mod grants;

pub use accounts::{create_account, find_account, AccountSummary};
pub use grants::{request_grant, GrantKind, GrantReceipt, GrantRequest};
