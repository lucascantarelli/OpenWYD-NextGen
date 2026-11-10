//! Endpoints de controle por domínio (nunca estado vivo de personagem).
//!
//! Só declara e re-exporta; cada domínio mora no seu módulo.

pub mod accounts;
pub mod catalog;
pub mod daily;
pub mod donate;
pub mod drop;
pub mod economy;
pub mod event;
pub mod grants;
pub mod item_effect;
pub mod mob;
pub mod moderation;
pub mod npc;
pub mod panel;
pub mod presence;
pub mod rates;
pub mod teleport;

pub use accounts::{account_json, accounts_json, create_account, find_account, AccountSummary};
pub use catalog::items_json;
pub use daily::rewards_json;
pub use donate::{shop_items_json, DonateCreditReceipt};
pub use economy::economy_summary;
pub use event::config_json as event_config_json;
pub use grants::{request_grant, GrantKind, GrantReceipt, GrantRequest};
pub use item_effect::{effect_json, effects_json};
pub use mob::{mob_template_json, mob_templates_json};
pub use moderation::{moderation, recent_audit, set_blocked, set_role, MAX_REASON};
pub use npc::npcs_json;
pub use panel::{
    own_account, own_characters, request_recovery, top_rankings, PanelAccount, PanelCharacter,
    RankingEntry, MAX_RANKING,
};
pub use presence::{list_channel_presence, ChannelPresence, PresenceSnapshot};
pub use rates::{
    ConfigStore, DailyEdit, DailyRewardRow, DonateEdit, DonateItemRow, DropEdit, DropRow,
    EffectRow, EventScheduleRow, MobTemplateRow, NpcRow, RatesSnapshot, RouteRow, WorldEventRow,
};
