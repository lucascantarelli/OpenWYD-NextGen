//! Porta de personagens — o que o Cold Path precisa, sem SQL aqui.

use std::future::Future;

use crate::error::DbError;
use crate::model::{Character, ItemRow, Progression, SHORT_SKILL_LEN};

/// Operações sobre `characters` (soft-delete: apaga marcando `deleted_at`).
pub trait Characters {
    /// Personagens vivos da conta, ordenados por slot.
    fn list_by_account(
        &self,
        account_id: i64,
    ) -> impl Future<Output = Result<Vec<Character>, DbError>> + Send;

    /// Ranking por nível (desc) e nome (asc) — leitura pública, sem `account_id`.
    fn top_by_level(
        &self,
        limit: usize,
    ) -> impl Future<Output = Result<Vec<Character>, DbError>> + Send;

    /// Personagens vivos cujo nome contém `term` (vazio = todos), por nome.
    ///
    /// Leitura de staff para a tela de entregas: devolve o destino com `gold`
    /// atual, para a concessão ser conferida antes de enfileirar.
    fn search_by_name(
        &self,
        term: &str,
        limit: usize,
    ) -> impl Future<Output = Result<Vec<Character>, DbError>> + Send;

    /// Cria validando invariantes (o banco impõe o resto).
    ///
    /// Nome sem ambiguidade com `Accounts::create_account` (mesmo adapter, duas portas).
    fn create_character(
        &self,
        character: &Character,
    ) -> impl Future<Output = Result<Character, DbError>> + Send;

    /// Apaga marcando `deleted_at` (nome/slot liberados pelo índice parcial).
    fn soft_delete(&self, id: i64) -> impl Future<Output = Result<(), DbError>> + Send;

    /// Inventário persistido (`character_items`), por slot.
    ///
    /// É a **fonte da verdade** do equipamento/mochila: o template da classe só é copiado
    /// na criação (`memcpy` do `g_pBaseSet`), nunca relido no login.
    fn items(
        &self,
        character_id: i64,
    ) -> impl Future<Output = Result<Vec<ItemRow>, DbError>> + Send {
        // Default para fakes do Cold Path (portal): sem inventário persistido naquele adapter.
        let _ = character_id;
        async { Ok(Vec::new()) }
    }

    /// Grava linhas de inventário que ainda não existem (idempotente por slot).
    ///
    /// A criação semeia o template + kill mark; a leitura de um personagem anterior a esta
    /// entrega materializa o mesmo conjunto uma vez. Slot já ocupado nunca é sobrescrito.
    fn insert_items(
        &self,
        character_id: i64,
        items: &[ItemRow],
    ) -> impl Future<Output = Result<(), DbError>> + Send {
        let _ = (character_id, items);
        async { Ok(()) }
    }

    /// `ShortSkill[16]` persistido; `None` = personagem sem linha (anterior à 0033).
    fn short_skill(
        &self,
        character_id: i64,
    ) -> impl Future<Output = Result<Option<[u8; SHORT_SKILL_LEN]>, DbError>> + Send {
        let _ = character_id;
        async { Ok(None) }
    }

    /// Grava o `ShortSkill` do personagem (criação e `_MSG_SetShortSkill` no futuro).
    fn set_short_skill(
        &self,
        character_id: i64,
        skill: &[u8; SHORT_SKILL_LEN],
    ) -> impl Future<Output = Result<(), DbError>> + Send {
        let _ = (character_id, skill);
        async { Ok(()) }
    }

    /// Progressão mutável: atributos, vitais, ouro, pontos e `LearnedSkill`.
    ///
    /// É o estado que `_MSG_ApplyBonus` lê antes de decidir e regrava depois — o cliente
    /// nunca informa quanto ponto tem. Default para os fakes do Cold Path (progressão zerada).
    fn progression(
        &self,
        character_id: i64,
    ) -> impl Future<Output = Result<Progression, DbError>> + Send {
        let _ = character_id;
        async { Ok(Progression::default()) }
    }

    /// Grava a progressão em **uma** transação (`characters` + `character_learned_skill`).
    fn store_progression(
        &self,
        character_id: i64,
        progression: &Progression,
    ) -> impl Future<Output = Result<(), DbError>> + Send {
        let _ = (character_id, progression);
        async { Ok(()) }
    }
}
