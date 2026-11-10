//! `teleport_admin/*` — custo, nível e disponibilidade de rotas de teleporte.
//!
//! A rota em si (origem/destino) vem do conteúdo capturado; o portal edita o que é
//! política de servidor: preço, requisito e ligar/desligar. Criar destino novo por
//! formulário seria inventar dado do mapa, então não existe rota de escrita para isso.

use crate::audit::{log_action, AuditSink};
use crate::auth::Session;
use crate::error::AdminError;

use super::accounts::staff_or_deny;
use super::rates::{ConfigStore, RouteRow};

/// Teto de rotas por leitura.
pub const MAX_LIMIT: i64 = 500;

/// Lista rotas, auditada.
pub fn list_routes(
    session: Option<Session>,
    limit: i64,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<RouteRow>, AdminError> {
    let staff = staff_or_deny(session, sink, "config.teleport")?;
    if !(1..=MAX_LIMIT).contains(&limit) {
        return Err(AdminError::InvalidInput {
            reason: "limite de rotas fora da faixa",
        });
    }
    let routes = store.routes(limit)?;
    log_action(
        sink,
        &staff,
        "config.teleport",
        "teleport_route",
        &format!("{} rotas", routes.len()),
    );
    Ok(routes)
}

/// Edita custo/nível/disponibilidade de uma rota existente.
pub fn set_route(
    session: Option<Session>,
    id: i32,
    cost_gold: Option<i32>,
    req_level: Option<i16>,
    enabled: Option<bool>,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.teleport.set")?;
    if id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "rota inválida",
        });
    }
    if let Some(cost) = cost_gold {
        if cost < 0 {
            return Err(AdminError::InvalidInput {
                reason: "custo negativo",
            });
        }
    }
    if let Some(level) = req_level {
        if level < 1 {
            return Err(AdminError::InvalidInput {
                reason: "nível mínimo inválido",
            });
        }
    }
    if cost_gold.is_none() && req_level.is_none() && enabled.is_none() {
        return Err(AdminError::InvalidInput {
            reason: "nada para alterar",
        });
    }
    let revision = store.set_route(id, cost_gold, req_level, enabled, staff.moderator_id())?;
    log_action(
        sink,
        &staff,
        "config.teleport.set",
        &format!("route:{id}"),
        &format!("revisão {revision}"),
    );
    Ok(revision)
}

/// JSON de uma rota.
#[must_use]
pub fn route_json(row: &RouteRow) -> String {
    let npc = row
        .npc_id
        .map_or_else(|| "null".to_string(), |id| id.to_string());
    format!(
        "{{\"id\":{},\"from_map_id\":{},\"from_x\":{},\"from_y\":{},\"to_map_id\":{},\
\"to_x\":{},\"to_y\":{},\"npc_id\":{},\"cost_gold\":{},\"req_level\":{},\"enabled\":{}}}",
        row.id,
        row.from_map_id,
        row.from_x,
        row.from_y,
        row.to_map_id,
        row.to_x,
        row.to_y,
        npc,
        row.cost_gold,
        row.req_level,
        row.enabled
    )
}

/// Lista JSON.
#[must_use]
pub fn routes_json(rows: &[RouteRow]) -> String {
    let itens: Vec<String> = rows.iter().map(route_json).collect();
    format!("[{}]", itens.join(","))
}
