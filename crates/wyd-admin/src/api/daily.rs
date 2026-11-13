//! `daily_reward_admin/*` — recompensa diária (CRUD auditado, WEB-06).
//!
//! Linha de configuração (`day` 1..7 ou `None` para todo dia; item e/ou ouro): o jogo
//! lê a tabela, nunca recebe push. Escrita audita autor/alvo/antes/depois; leitura é
//! paginada e não audita. Ligar/desligar dia específico não tem coluna (a série proíbe
//! `ALTER TABLE` no `up`) — segue `501` declarado.

use crate::audit::AuditSink;
use crate::auth::Session;
use crate::error::AdminError;

use super::accounts::staff_or_deny;
use super::rates::{ConfigStore, DailyEdit, DailyRewardRow};

/// Teto de recompensas por leitura.
pub const MAX_LIMIT: i64 = 200;

/// Lista recompensas em páginas por id (`None` = primeira página).
pub fn list_rewards(
    session: Option<Session>,
    limit: i64,
    after_id: Option<i32>,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<DailyRewardRow>, AdminError> {
    let _staff = staff_or_deny(session, sink, "config.daily_rewards")?;
    if !(1..=MAX_LIMIT).contains(&limit) {
        return Err(AdminError::InvalidInput {
            reason: "limite de recompensas fora da faixa",
        });
    }
    if after_id.is_some_and(|id| id < 0) {
        return Err(AdminError::InvalidInput {
            reason: "cursor de recompensa inválido",
        });
    }
    store.daily_rewards(limit, after_id)
}

/// Cria (`edit.id = None`) ou substitui uma recompensa, com `qty >= 1` e item ou ouro.
pub fn upsert_reward(
    session: Option<Session>,
    edit: &DailyEdit,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.daily.set")?;
    validar_edicao(edit)?;
    let revision = store.upsert_daily_reward(edit, staff.moderator_id())?;
    crate::audit::log_action(
        sink,
        &staff,
        "config.daily.set",
        &edit
            .id
            .map_or_else(|| "reward:nova".to_string(), |id| format!("reward:{id}")),
        &format!(
            "day={} item={} gold={} qty={}",
            edit.day.map_or_else(|| "*".to_string(), |d| d.to_string()),
            edit.item_index
                .map_or_else(|| "-".to_string(), |i| i.to_string()),
            edit.gold,
            edit.qty
        ),
    );
    Ok(revision)
}

/// Remove uma recompensa.
pub fn delete_reward(
    session: Option<Session>,
    id: i32,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.daily.delete")?;
    if id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "recompensa inválida",
        });
    }
    let revision = store.delete_daily_reward(id, staff.moderator_id())?;
    crate::audit::log_action(
        sink,
        &staff,
        "config.daily.delete",
        &format!("reward:{id}"),
        "removida",
    );
    Ok(revision)
}

fn validar_edicao(edit: &DailyEdit) -> Result<(), AdminError> {
    if edit.day.is_some_and(|day| !(1..=7).contains(&day)) {
        return Err(AdminError::InvalidInput {
            reason: "day fora de 1..7 (ou ausente para todo dia)",
        });
    }
    if edit.item_index.is_some_and(|item| item <= 0) {
        return Err(AdminError::InvalidInput {
            reason: "item_index inválido",
        });
    }
    if edit.gold < 0 || edit.qty < 1 {
        return Err(AdminError::InvalidInput {
            reason: "gold/qty inválidos",
        });
    }
    if edit.item_index.is_none() && edit.gold == 0 {
        return Err(AdminError::InvalidInput {
            reason: "recompensa precisa de item ou ouro",
        });
    }
    Ok(())
}

/// JSON de uma recompensa.
#[must_use]
pub fn reward_json(row: &DailyRewardRow) -> String {
    let day = row
        .day
        .map_or_else(|| "null".to_string(), |d| d.to_string());
    let item = row
        .item_index
        .map_or_else(|| "null".to_string(), |i| i.to_string());
    format!(
        "{{\"id\":{},\"day\":{day},\"item_index\":{item},\"gold\":{},\"qty\":{},\"account_wide\":{}}}",
        row.id, row.gold, row.qty, row.account_wide
    )
}

/// Lista JSON de recompensas.
#[must_use]
pub fn rewards_json(rows: &[DailyRewardRow]) -> String {
    let itens: Vec<String> = rows.iter().map(reward_json).collect();
    format!("[{}]", itens.join(","))
}
