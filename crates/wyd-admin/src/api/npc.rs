//! `npc_admin/get_npc` e `npc_admin/upsert_npc` — posição e visibilidade de NPC.
//!
//! NPC é definição de mundo (não estado vivo de personagem): mover e desligar aqui é
//! edição de conteúdo auditada, e o servidor recarrega quando `config_revision` muda.
//! O espaço de coordenadas é o do cliente: mapa 0..255, x/y 0..4095.

use crate::audit::{log_action, AuditSink};
use crate::auth::Session;
use crate::error::AdminError;
use crate::transport::json;

use super::accounts::staff_or_deny;
use super::rates::{ConfigStore, NpcRow};

/// Teto de NPCs por leitura (a definição tem 6.100 linhas; sem paginação o portal trava).
pub const MAX_LIMIT: i64 = 200;

/// Lista NPCs em páginas por id (só leitura, FASE-07-5).
///
/// Sem escrita em `audit_log`: leitura pura não é ação auditável (a negação de acesso
/// continua auditada por `staff_or_deny`). `after_id` é o cursor (`None` = primeira página).
pub fn list_npcs(
    session: Option<Session>,
    limit: i64,
    after_id: Option<i32>,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<NpcRow>, AdminError> {
    let _staff = staff_or_deny(session, sink, "config.npcs")?;
    if !(1..=MAX_LIMIT).contains(&limit) {
        return Err(AdminError::InvalidInput {
            reason: "limite de NPCs fora da faixa",
        });
    }
    if after_id.is_some_and(|id| id < 0) {
        return Err(AdminError::InvalidInput {
            reason: "cursor de NPC inválido",
        });
    }
    store.npcs(limit, after_id)
}

/// Leitura de um NPC por id.
pub fn get_npc(
    session: Option<Session>,
    id: i32,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Option<NpcRow>, AdminError> {
    let staff = staff_or_deny(session, sink, "config.npc")?;
    if id <= 0 {
        return Err(AdminError::InvalidInput {
            reason: "id de NPC inválido",
        });
    }
    let npc = store.npc(id)?;
    log_action(
        sink,
        &staff,
        "config.npc",
        &format!("npc:{id}"),
        if npc.is_some() { "leitura" } else { "ausente" },
    );
    Ok(npc)
}

/// Move/ativa um NPC, validando o espaço de coordenadas antes do banco.
pub fn set_npc(
    session: Option<Session>,
    id: i32,
    (map_id, pos_x, pos_y): (i16, i16, i16),
    enabled: bool,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.npc.set")?;
    if id <= 0 || !(0..=255).contains(&map_id) {
        return Err(AdminError::InvalidInput {
            reason: "NPC ou mapa inválido",
        });
    }
    if !(0..=4095).contains(&pos_x) || !(0..=4095).contains(&pos_y) {
        return Err(AdminError::InvalidInput {
            reason: "posição fora do espaço 0..4095",
        });
    }
    let revision = store.set_npc(id, map_id, pos_x, pos_y, enabled, staff.moderator_id())?;
    log_action(
        sink,
        &staff,
        "config.npc.set",
        &format!("npc:{id}"),
        &format!("map={map_id} x={pos_x} y={pos_y} enabled={enabled} (revisão {revision})"),
    );
    Ok(revision)
}

/// Lista JSON de NPCs.
#[must_use]
pub fn npcs_json(rows: &[NpcRow]) -> String {
    let itens: Vec<String> = rows.iter().map(npc_json).collect();
    format!("[{}]", itens.join(","))
}

/// JSON de um NPC.
#[must_use]
pub fn npc_json(row: &NpcRow) -> String {
    format!(
        "{{\"id\":{},\"slug\":{},\"display_name\":{},\"map_id\":{},\"pos_x\":{},\
\"pos_y\":{},\"enabled\":{},\"merchant\":{}}}",
        row.id,
        json::quote(&row.slug),
        json::quote(&row.display_name),
        row.map_id,
        row.pos_x,
        row.pos_y,
        row.enabled,
        row.merchant
    )
}
