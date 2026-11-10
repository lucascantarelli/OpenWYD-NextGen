//! `drop_admin/*` — peso de drop por (template de monstro, item).
//!
//! Drop no original é **peso**, não probabilidade: o servidor sorteia 1..1.000.000 por
//! item (`docs/wyd/sistemas/drops.md`). O portal edita o peso e a quantidade; a taxa
//! efetiva continua sendo derivada no Hot Path, nunca congelada aqui.

use crate::audit::{log_action, AuditSink};
use crate::auth::Session;
use crate::error::AdminError;
use crate::transport::json;

use super::accounts::staff_or_deny;
use super::rates::{ConfigStore, DropEdit, DropRow, MAX_SOURCE};

/// Teto de linhas por leitura.
pub const MAX_LIMIT: i64 = 200;

/// Lista pesos de drop (filtro opcional por template), auditada.
pub fn list_drops(
    session: Option<Session>,
    template: Option<&str>,
    limit: i64,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<DropRow>, AdminError> {
    let staff = staff_or_deny(session, sink, "config.drops")?;
    if !(1..=MAX_LIMIT).contains(&limit) {
        return Err(AdminError::InvalidInput {
            reason: "limite de drops fora da faixa",
        });
    }
    let template = template.map(str::trim).filter(|value| !value.is_empty());
    let drops = store.drops(template, limit)?;
    log_action(
        sink,
        &staff,
        "config.drops",
        template.unwrap_or("*"),
        &format!("{} linhas", drops.len()),
    );
    Ok(drops)
}

/// Grava o peso de um item no template, com fonte obrigatória.
pub fn upsert_drop(
    session: Option<Session>,
    edit: &DropEdit,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.drop.set")?;
    let template = edit.template_name.trim();
    if template.is_empty() || template.len() > 120 || edit.item_index <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "template ou item inválido",
        });
    }
    if edit.weight <= 0 || edit.min_qty <= 0 || edit.max_qty < edit.min_qty {
        return Err(AdminError::InvalidInput {
            reason: "peso/quantidade inválidos",
        });
    }
    let source = edit.source.trim();
    if source.is_empty() || source.chars().count() > MAX_SOURCE {
        return Err(AdminError::InvalidInput {
            reason: "fonte do drop obrigatória (até 160 caracteres)",
        });
    }
    let revision = store.set_drop(edit, staff.moderator_id())?;
    log_action(
        sink,
        &staff,
        "config.drop.set",
        &format!("{template}#{}", edit.item_index),
        &format!("peso {} (revisão {revision})", edit.weight),
    );
    Ok(revision)
}

/// Remove o peso editado.
pub fn reset_drop(
    session: Option<Session>,
    template: &str,
    item_index: i32,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.drop.reset")?;
    let revision = store.reset_drop(template.trim(), item_index, staff.moderator_id())?;
    log_action(
        sink,
        &staff,
        "config.drop.reset",
        &format!("{}#{item_index}", template.trim()),
        &format!("peso removido (revisão {revision})"),
    );
    Ok(revision)
}

/// JSON de uma linha de drop.
#[must_use]
pub fn drop_json(row: &DropRow) -> String {
    format!(
        "{{\"template_name\":{},\"item_index\":{},\"item_name\":{},\"weight\":{},\
\"min_qty\":{},\"max_qty\":{},\"source\":{}}}",
        json::quote(&row.template_name),
        row.item_index,
        json::quote(&row.item_name),
        row.weight,
        row.min_qty,
        row.max_qty,
        json::quote(&row.source)
    )
}

/// Lista JSON.
#[must_use]
pub fn drops_json(rows: &[DropRow]) -> String {
    let itens: Vec<String> = rows.iter().map(drop_json).collect();
    format!("[{}]", itens.join(","))
}
