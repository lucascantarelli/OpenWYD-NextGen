//! Endpoints de controle por domínio (nunca estado vivo de personagem).
//!
//! Só declara e re-exporta; cada domínio mora no seu módulo.

pub mod accounts;
pub mod catalog;
pub mod drop;
pub mod economy;
pub mod grants;
pub mod moderation;
pub mod npc;
pub mod panel;
pub mod presence;
pub mod rates;
pub mod teleport;

pub use accounts::{create_account, find_account, AccountSummary};
pub use catalog::items_json;
pub use economy::economy_summary;
pub use grants::{request_grant, GrantKind, GrantReceipt, GrantRequest};
pub use moderation::{moderation, recent_audit, set_blocked, set_role, MAX_REASON};
pub use panel::{
    own_account, own_characters, request_recovery, top_rankings, PanelAccount, PanelCharacter,
    RankingEntry, MAX_RANKING,
};
pub use presence::{list_channel_presence, ChannelPresence, PresenceSnapshot};
pub use rates::{ConfigStore, DropEdit, DropRow, NpcRow, RatesSnapshot, RouteRow};
