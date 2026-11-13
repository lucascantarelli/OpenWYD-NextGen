//! Personagem persistido — espelho da tabela `characters` (migration 0002).

use crate::error::DbError;

/// Slot de personagem na conta (0–3, espelha o CHECK).
pub const MAX_SLOTS: i64 = 3;

/// `MAX_EQUIP`: primeiros slots de `character_items` são o equipamento.
pub const EQUIP_SLOTS: usize = 16;
/// `MAX_CARRY`: mochila depois do equipamento.
pub const CARRY_SLOTS: usize = 64;
/// `slot` válido em `character_items`/`vault_items` (CHECK 0–127).
pub const ITEM_SLOTS: i64 = 128;
/// Primeiro slot de mochila: `16 + índice do Carry` (o `Equip` ocupa 0–15).
pub const CARRY_FIRST_SLOT: i64 = 16;
/// `sizeof(ShortSkill)` no `MSG_CNFClientCharacterLogin`.
pub const SHORT_SKILL_LEN: usize = 16;
/// `memset(ShortSkill, -1, 16)` da criação (`CFileDB.cpp`).
pub const SHORT_SKILL_EMPTY: u8 = 0xFF;

/// `LearnedSkill`: 24 skills por classe (`g_pSpell[24 * classe + pos]`).
pub const SKILL_SLOTS: usize = 24;
/// Máscara de `Learn`: o `long long` do `MSG_UpdateEtc` transporta 8 B, o C só usa 24 bits.
pub const LEARNED_SKILL_MASK: i64 = (1 << SKILL_SLOTS) - 1;
/// Posições de tier (`_MSG_ApplyBonus.cpp`): uma skill de linha por classe.
pub const SKILL_TIER_POSITIONS: [usize; 3] = [7, 15, 23];
/// Ouro cobrado ao aprender skill de tier (`_MSG_ApplyBonus.cpp`).
pub const SKILL_TIER_COIN: i64 = 50_000_000;
/// `ScoreBonus >= 300` gasta 100 pontos de uma vez (`_MSG_ApplyBonus.cpp`).
pub const SCORE_BONUS_BULK: i64 = 300;
/// `Special[4]` do `STRUCT_SCORE`: maestria por atributo.
pub const SPECIAL_SLOTS: usize = 4;
/// Teto padrão por maestria (`_MSG_ApplyBonus.cpp`).
pub const SPECIAL_MAX: i64 = 200;
/// Teto quando a skill de linha correspondente está aprendida (`_MSG_ApplyBonus.cpp`).
pub const SPECIAL_MAX_LINEAGE: i64 = 255;
/// Skill de linha que libera o teto 255: `Detail` 1/2/3 → bits 7/15/23 do `LearnedSkill`.
/// `Detail` 0 (Str) não tem linha e fica sempre em 200.
pub const SPECIAL_LINEAGE_SKILL: [usize; SPECIAL_SLOTS] = [usize::MAX, 7, 15, 23];

/// Estado que `_MSG_ApplyBonus` altera — lido e gravado como um conjunto só.
///
/// Mora junto de `characters` (atributos, vitais, ouro, pontos) e de
/// `character_learned_skill` (máscara de skill); a escrita é uma transação única.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Progression {
    /// Nível persistido (**1-based**): o fio manda `level - 1` (`personagens::wire_level`).
    pub level: i64,
    /// EXP acumulada total (`STRUCT_MOB.Exp`), comparada com a curva de `g_pNextLevel`.
    pub exp: i64,
    pub strength: i64,
    pub intelligence: i64,
    pub dexterity: i64,
    pub constitution: i64,
    pub hp_max: i64,
    pub mp_max: i64,
    pub hp: i64,
    pub mp: i64,
    pub gold: i64,
    pub score_bonus: i64,
    pub skill_bonus: i64,
    pub learned_skill: i64,
    /// `Special[0..3]` (maestria por atributo); zero = o `BaseMob` de todas as classes.
    pub special: [i64; SPECIAL_SLOTS],
    /// Pontos de maestria ainda não gastos (`SpecialBonus@790`).
    pub special_bonus: i64,
}

/// Progresso de uma quest por personagem (`character_quest`, `0013_quests`).
///
/// `stage` é o `Newbie` do legado (0..=4, um passo por treinador); no banco vira
/// `step = stage + 1`. `reward_mask` ocupa `progress` e tem um bit por etapa já paga —
/// a trava de idempotência contra recompensa dupla.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct QuestState {
    pub quest_code: String,
    pub stage: i16,
    pub completed: bool,
    pub reward_mask: i32,
}

/// Linha de item normalizada — colunas idênticas em `character_items` e `vault_items` (0003).///
/// `slot` 0–15 = `Equip[i]`, 16–79 = `Carry[i − 16]`; 80–127 fica reservado. Refino, Ancient
/// e gema são estado **da linha**, não do catálogo (`docs/wyd/sistemas/refino.md` §3).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct ItemRow {
    pub slot: i64,
    pub item_id: i64,
    pub qty: i64,
    /// Reservado ao catálogo/economia (efeitos de item ainda não portados).
    pub flags: i64,
    pub refine: i64,
    pub ancient: bool,
    pub gem_effect: i64,
    pub gem_value: i64,
}

impl ItemRow {
    /// Índice em `STRUCT_MOB.Equip[16]`, quando a linha é de equipamento.
    #[must_use]
    pub fn equip_slot(&self) -> Option<usize> {
        usize::try_from(self.slot)
            .ok()
            .filter(|slot| *slot < EQUIP_SLOTS)
    }

    /// Índice em `STRUCT_MOB.Carry[64]`, quando a linha é de mochila.
    #[must_use]
    pub fn carry_slot(&self) -> Option<usize> {
        usize::try_from(self.slot - CARRY_FIRST_SLOT)
            .ok()
            .filter(|slot| *slot < CARRY_SLOTS)
    }
}

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
    /// CP (Chaos Points, 0048): `-75..=75`, 0 = neutro. A cor do nome no cliente é
    /// `cp + 75`, calculada na borda — o banco guarda o CP real.
    pub cp: i64,
    /// Atributos-base (0032): `BaseSIDCHM[classe]` no nível 1, mais o que o jogador
    /// distribuiu. HP/MP máximos derivam de `Con`/`Int` em `BASE_GetHpMp`.
    pub strength: i64,
    pub intelligence: i64,
    pub dexterity: i64,
    pub constitution: i64,
    /// Pontos de atributo e de skill ainda não gastos (5 e 3 por nível no MORTAL).
    pub score_bonus: i64,
    pub skill_bonus: i64,
    /// Guilda vem de `guild_members`/`guilds` (0004), preenchida no roster;
    /// `create`/`top_by_level` devolvem `None`/vazio.
    pub guild_id: Option<i64>,
    pub guild_name: String,
}

impl Character {
    /// Invariantes que o código verifica antes do banco (dupla checagem barata).
    pub fn validate(&self) -> Result<(), DbError> {
        if !(0..=MAX_SLOTS).contains(&self.slot)
            || self.level < 1
            || self.exp < 0
            || self.strength < 0
            || self.intelligence < 0
            || self.dexterity < 0
            || self.constitution < 0
            || self.score_bonus < 0
            || self.skill_bonus < 0
        {
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
        if !(-75..=75).contains(&self.cp) {
            return Err(DbError::ConstraintViolation {
                table: "characters",
                constraint: "characters_cp",
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
