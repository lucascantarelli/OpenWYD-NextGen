//! Clã e membros — espelho das tabelas `guilds`/`guild_members` (migration 0004).

/// Clã (líder pode nascer na mesma transação: FK DEFERRABLE no banco).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Guild {
    pub id: i64,
    pub name: String,
    pub master_character_id: Option<i64>,
}

/// Vínculo personagem↔clã (um clã por personagem: UNIQUE no banco).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GuildMember {
    pub guild_id: i64,
    pub character_id: i64,
    pub rank: i64,
}
