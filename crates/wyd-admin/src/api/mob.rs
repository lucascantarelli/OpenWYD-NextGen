//! `mob_admin/list_mob_templates` — templates de mob em páginas por nome (só leitura).
//!
//! A fonte é `npc_template` (1.991 linhas semeadas): `mob_template_stat` continua vazia até
//! a extração do `STRUCT_MOB` (FASE-10) e não serve lista nenhuma. Sem escrita em
//! `audit_log`: leitura pura não é ação auditável.

use crate::audit::AuditSink;
use crate::auth::Session;
use crate::error::AdminError;
use crate::transport::json;

use super::accounts::staff_or_deny;
use super::rates::{ConfigStore, MobTemplateRow};

/// Teto de templates por leitura.
pub const MAX_LIMIT: i64 = 200;

/// Lista templates em páginas por nome. `after` é o cursor (`None` = primeira página).
pub fn list_mob_templates(
    session: Option<Session>,
    limit: i64,
    after: Option<&str>,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<MobTemplateRow>, AdminError> {
    let _staff = staff_or_deny(session, sink, "config.mob_templates")?;
    if !(1..=MAX_LIMIT).contains(&limit) {
        return Err(AdminError::InvalidInput {
            reason: "limite de templates fora da faixa",
        });
    }
    let after = after.map(str::trim).filter(|value| !value.is_empty());
    store.mob_templates(limit, after)
}

/// JSON de um template.
#[must_use]
pub fn mob_template_json(row: &MobTemplateRow) -> String {
    format!(
        "{{\"template_name\":{},\"name\":{},\"source_size\":{}}}",
        json::quote(&row.template_name),
        json::quote(&row.name),
        row.source_size
    )
}

/// Lista JSON de templates.
#[must_use]
pub fn mob_templates_json(rows: &[MobTemplateRow]) -> String {
    let itens: Vec<String> = rows.iter().map(mob_template_json).collect();
    format!("[{}]", itens.join(","))
}
