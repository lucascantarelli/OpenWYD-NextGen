//! Modelos de domínio — dados que espelham as tabelas, sem SQL.
//!
//! Só declara e re-exporta; cada entidade mora no seu módulo.

pub mod account;
pub mod audit;
pub mod character;
pub mod guild;

pub use account::{Account, AccountModeration, Role};
pub use audit::AuditEntry;
pub use character::{
    Character, ItemRow, Progression, CARRY_FIRST_SLOT, CARRY_SLOTS, EQUIP_SLOTS, ITEM_SLOTS,
    LEARNED_SKILL_MASK, MAX_SLOTS, SCORE_BONUS_BULK, SHORT_SKILL_EMPTY, SHORT_SKILL_LEN,
    SKILL_SLOTS, SKILL_TIER_COIN, SKILL_TIER_POSITIONS, SPECIAL_LINEAGE_SKILL, SPECIAL_MAX,
    SPECIAL_MAX_LINEAGE, SPECIAL_SLOTS,
};
pub use guild::{Guild, GuildMember};
