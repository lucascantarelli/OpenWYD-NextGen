//! Personagem persistido — espelho da tabela `characters` (migration 0002).

use crate::error::DbError;

/// Slot de personagem na conta (0–3, espelha o CHECK).
pub const MAX_SLOTS: i64 = 3;

/// Personagem (posição incluída: o save/pos mora aqui, não no tick).
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Character {
    pub id: i64,
    pub account_id: i64,
    pub slot: i64,
    pub name: String,
    pub class: i64,
    pub level: i64,
    pub exp: i64,
    pub hp: i64,
    pub mp: i64,
    pub pos_x: i64,
    pub pos_y: i64,
    /// Ouro do personagem (migration 0031); gasto/ganho passam pelo servidor.
    pub gold: i64,
    /// Máximo de HP/MP (0031): sem eles a barra da seleção nascia com máximo 0.
    pub hp_max: i64,
    pub mp_max: i64,
    /// Pontos de PvP (0031); ranking/season consome depois.
    pub pvp_points: i64,
    /// Guilda vem de `guild_members`/`guilds` (0004), preenchida no roster;
    /// `create`/`top_by_level` devolvem `None`/vazio.
    pub guild_id: Option<i64>,
    pub guild_name: String,
}

impl Character {
    /// Invariantes que o código verifica antes do banco (dupla checagem barata).
    pub fn validate(&self) -> Result<(), DbError> {
        if !(0..=MAX_SLOTS).contains(&self.slot) || self.level < 1 || self.exp < 0 {
            return Err(DbError::ConstraintViolation {
                table: "characters",
                constraint: "slot/level/exp",
            });
        }
        if self.name.trim().is_empty() {
            return Err(DbError::ConstraintViolation {
                table: "characters",
                constraint: "characters_name_nonempty",
            });
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn valid() -> Character {
        Character {
            id: 1,
            account_id: 1,
            slot: 0,
            name: "Kae".to_string(),
            class: 0,
            level: 1,
            exp: 0,
            hp: 100,
            mp: 10,
            pos_x: 2100,
            pos_y: 2100,
            hp_max: 100,
            mp_max: 10,
            ..Character::default()
        }
    }

    #[test]
    fn validation_rejects_bad_slot_level_and_blank_name() {
        assert!(valid().validate().is_ok());
        assert!(Character { slot: 4, ..valid() }.validate().is_err());
        assert!(Character {
            level: 0,
            ..valid()
        }
        .validate()
        .is_err());
        assert!(Character {
            name: "  ".to_string(),
            ..valid()
        }
        .validate()
        .is_err());
    }
}
