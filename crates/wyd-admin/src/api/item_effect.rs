//! `item_effect_admin/list_effects` — efeitos de item em páginas por nome (só leitura).
//!
//! Mostra o símbolo `EF_*`, o rótulo e a categoria que o seed declara. O código numérico do
//! `ItemEffect.h` não foi provado (`confirmed = false` em parte das linhas), então a lista
//! não inventa número nenhum. Sem escrita em `audit_log`: leitura pura não é ação auditável.

use crate::audit::AuditSink;
use crate::auth::Session;
use crate::error::AdminError;
use crate::transport::json;

use super::accounts::staff_or_deny;
use super::rates::{ConfigStore, EffectRow};

/// Teto de efeitos por leitura.
pub const MAX_LIMIT: i64 = 200;

/// Lista efeitos em páginas por nome. `after` é o cursor (`None` = primeira página).
pub fn list_effects(
    session: Option<Session>,
    limit: i64,
    after: Option<&str>,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<EffectRow>, AdminError> {
    let _staff = staff_or_deny(session, sink, "config.effects")?;
    if !(1..=MAX_LIMIT).contains(&limit) {
        return Err(AdminError::InvalidInput {
            reason: "limite de efeitos fora da faixa",
        });
    }
    let after = after.map(str::trim).filter(|value| !value.is_empty());
    store.effects(limit, after)
}

/// JSON de um efeito.
#[must_use]
pub fn effect_json(row: &EffectRow) -> String {
    format!(
        "{{\"name\":{},\"label\":{},\"category\":{},\"description\":{},\"confirmed\":{}}}",
        json::quote(&row.name),
        json::quote(&row.label),
        json::quote(&row.category),
        json::quote(&row.description),
        row.confirmed
    )
}

/// Lista JSON de efeitos.
#[must_use]
pub fn effects_json(rows: &[EffectRow]) -> String {
    let itens: Vec<String> = rows.iter().map(effect_json).collect();
    format!("[{}]", itens.join(","))
}
