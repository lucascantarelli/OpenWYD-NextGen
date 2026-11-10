//! `donate_admin/*` — loja de donate e crédito de saldo (FASE-07-6).
//!
//! A loja vende itens por créditos já conciliados (dinheiro real nunca entra aqui).
//! `credit_donate_balance` credita `account_wallet` (saldo de **conta**, nunca personagem
//! vivo) com idempotência por chave via `donate_credit`: mesma chave devolve o mesmo razão
//! sem creditar duas vezes. Toda escrita é auditada com autor/alvo/antes/depois.

use crate::audit::AuditSink;
use crate::auth::Session;
use crate::error::AdminError;

use super::accounts::staff_or_deny;
use super::rates::{ConfigStore, DonateEdit, DonateItemRow};

/// Teto de itens por leitura.
pub const MAX_LIMIT: i64 = 200;

/// Lista itens da loja em páginas por id.
pub fn list_shop_items(
    session: Option<Session>,
    limit: i64,
    after_id: Option<i32>,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<DonateItemRow>, AdminError> {
    let _staff = staff_or_deny(session, sink, "config.donate_items")?;
    if !(1..=MAX_LIMIT).contains(&limit) {
        return Err(AdminError::InvalidInput {
            reason: "limite da loja fora da faixa",
        });
    }
    if after_id.is_some_and(|id| id < 0) {
        return Err(AdminError::InvalidInput {
            reason: "cursor da loja inválido",
        });
    }
    store.donate_items(limit, after_id)
}

/// Cria (`edit.id = None`) ou substitui um item da loja.
pub fn upsert_shop_item(
    session: Option<Session>,
    edit: &DonateEdit,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.donate.set")?;
    validar_edicao(edit)?;
    let revision = store.upsert_donate_item(edit, staff.moderator_id())?;
    crate::audit::log_action(
        sink,
        &staff,
        "config.donate.set",
        &edit
            .id
            .map_or_else(|| "shop:novo".to_string(), |id| format!("shop:{id}")),
        &format!(
            "item={} preco={} enabled={}",
            edit.item_index, edit.price_credits, edit.enabled
        ),
    );
    Ok(revision)
}

/// Liga/desliga um item da loja.
pub fn set_shop_item_enabled(
    session: Option<Session>,
    id: i32,
    enabled: bool,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.donate.set")?;
    if id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "item da loja inválido",
        });
    }
    let revision = store.set_donate_item_enabled(id, enabled, staff.moderator_id())?;
    crate::audit::log_action(
        sink,
        &staff,
        "config.donate.set",
        &format!("shop:{id}"),
        &format!("enabled={enabled}"),
    );
    Ok(revision)
}

/// Remove um item da loja (não apaga compra já feita).
pub fn delete_shop_item(
    session: Option<Session>,
    id: i32,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.donate.delete")?;
    if id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "item da loja inválido",
        });
    }
    let revision = store.delete_donate_item(id, staff.moderator_id())?;
    crate::audit::log_action(
        sink,
        &staff,
        "config.donate.delete",
        &format!("shop:{id}"),
        "removido",
    );
    Ok(revision)
}

/// Recibo do crédito (mesma chave, mesmo razão — prova de idempotência).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DonateCreditReceipt {
    pub idempotency_key: String,
    pub credit_id: i64,
    pub replay: bool,
}

/// Credita saldo de donate na conta, idempotente por chave.
pub fn credit_balance(
    session: Option<Session>,
    account_id: i64,
    credits: i64,
    key: &str,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<DonateCreditReceipt, AdminError> {
    let staff = staff_or_deny(session, sink, "config.donate.credit")?;
    if account_id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "conta inválida",
        });
    }
    if credits <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "créditos precisam ser positivos",
        });
    }
    let key = key.trim();
    if key.is_empty() || key.len() > 120 {
        return Err(AdminError::InvalidInput {
            reason: "chave de idempotência obrigatória (até 120 caracteres)",
        });
    }
    let (credit_id, replay) =
        store.credit_donate(account_id, credits, key, staff.moderator_id())?;
    crate::audit::log_action(
        sink,
        &staff,
        if replay {
            "config.donate.replay"
        } else {
            "config.donate.credit"
        },
        &format!("account:{account_id}"),
        &format!("{credits} créditos (chave {key})"),
    );
    Ok(DonateCreditReceipt {
        idempotency_key: key.to_string(),
        credit_id,
        replay,
    })
}

fn validar_edicao(edit: &DonateEdit) -> Result<(), AdminError> {
    if edit.item_index <= 0 || edit.price_credits <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "item ou preço inválido",
        });
    }
    if edit.duration_days.is_some_and(|d| d <= 0) {
        return Err(AdminError::InvalidInput {
            reason: "duração inválida",
        });
    }
    Ok(())
}

/// JSON de um item da loja.
#[must_use]
pub fn shop_item_json(row: &DonateItemRow) -> String {
    let duration = row
        .duration_days
        .map_or_else(|| "null".to_string(), |d| d.to_string());
    format!(
        "{{\"id\":{},\"item_index\":{},\"price_credits\":{},\"duration_days\":{duration},\"enabled\":{}}}",
        row.id, row.item_index, row.price_credits, row.enabled
    )
}

/// Lista JSON da loja.
#[must_use]
pub fn shop_items_json(rows: &[DonateItemRow]) -> String {
    let itens: Vec<String> = rows.iter().map(shop_item_json).collect();
    format!("[{}]", itens.join(","))
}
