//! Estatísticas de economia do painel (FASE-07-3).
//!
//! O número vem de fonte auditável, nunca de estimativa: imposto somado do
//! livro-caixa (`city_tax_ledger`) e concessões contadas na fila de entrega
//! (`delivery_queue`). Tabela sem movimento devolve zero/lista vazia — o painel
//! mostra a fonte e a ausência de dados.

use wyd_db::{Economy, EconomySummary};

use crate::audit::{log_action, AuditSink};
use crate::auth::Session;
use crate::error::AdminError;

use super::accounts::staff_or_deny;

/// Foto da economia auditada (`economy.summary`), legível por qualquer staff.
pub async fn economy_summary(
    session: Option<Session>,
    repo: &(dyn Economy + Send + Sync),
    sink: &impl AuditSink,
) -> Result<EconomySummary, AdminError> {
    let staff = staff_or_deny(session, sink, "economy.summary")?;
    let resumo = repo.summary().await?;
    log_action(
        sink,
        &staff,
        "economy.summary",
        "economy",
        &format!(
            "{} cidades, {} tipos de grant",
            resumo.tax_by_city.len(),
            resumo.grants.len()
        ),
    );
    Ok(resumo)
}
