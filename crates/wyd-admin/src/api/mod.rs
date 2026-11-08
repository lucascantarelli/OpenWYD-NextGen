//! Endpoints de controle por domínio (nunca estado vivo de personagem).
//!
//! Só declara e re-exporta; cada domínio mora no seu módulo.

pub mod accounts;
pub mod catalog;
pub mod grants;
pub mod panel;

pub use accounts::{create_account, find_account, AccountSummary};
pub use catalog::items_json;
pub use grants::{request_grant, GrantKind, GrantReceipt, GrantRequest};
pub use panel::{
    own_account, own_characters, request_recovery, top_rankings, PanelAccount, PanelCharacter,
    RankingEntry, MAX_RANKING,
};
