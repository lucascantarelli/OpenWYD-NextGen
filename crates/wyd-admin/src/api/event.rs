//! `world_event_admin/*` — estado de eventos de mundo (FASE-07-6).
//!
//! Linha de configuração: ligar/desligar vale para o próximo leitor do dado
//! (boot/tick que lê a tabela) — nenhum push para o Hot Path. Horários vêm de
//! `event_schedule` com `timezone` explícita. Toda escrita é auditada.

use crate::audit::AuditSink;
use crate::auth::Session;
use crate::error::AdminError;
use crate::transport::json;

use super::accounts::staff_or_deny;
use super::rates::{ConfigStore, EventScheduleRow, WorldEventRow};

/// Evento com os horários (o portal monta a grade a partir daqui).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct WorldEventFull {
    pub event: WorldEventRow,
    pub schedules: Vec<EventScheduleRow>,
}

/// Foto dos eventos + horários (só leitura, sem auditoria de leitura).
pub fn get_config(
    session: Option<Session>,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<Vec<WorldEventFull>, AdminError> {
    let _staff = staff_or_deny(session, sink, "config.events")?;
    let eventos = store.world_events()?;
    let horarios = store.event_schedules()?;
    Ok(eventos
        .into_iter()
        .map(|event| {
            let schedules = horarios
                .iter()
                .filter(|h| h.event_code == event.code)
                .cloned()
                .collect();
            WorldEventFull { event, schedules }
        })
        .collect())
}

/// Liga/desliga um evento.
pub fn set_enabled(
    session: Option<Session>,
    code: &str,
    enabled: bool,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.event.set")?;
    let code = code.trim();
    if code.is_empty() || code.len() > 60 {
        return Err(AdminError::InvalidInput {
            reason: "código de evento inválido",
        });
    }
    let revision = store.set_world_event_enabled(code, enabled, staff.moderator_id())?;
    crate::audit::log_action(
        sink,
        &staff,
        "config.event.set",
        &format!("event:{code}"),
        &format!("enabled={enabled}"),
    );
    Ok(revision)
}

/// JSON da foto (eventos com horários).
#[must_use]
pub fn config_json(full: &[WorldEventFull]) -> String {
    let itens: Vec<String> = full.iter().map(|item| {
        let horarios: Vec<String> = item
            .schedules
            .iter()
            .map(|h| {
                format!(
                    "{{\"id\":{},\"weekday\":{},\"start_time\":{},\"duration_minutes\":{},\"timezone\":{},\"every_n_weeks\":{},\"enabled\":{}}}",
                    h.id,
                    h.weekday,
                    json::quote(&h.start_time),
                    h.duration_minutes,
                    json::quote(&h.timezone),
                    h.every_n_weeks,
                    h.enabled
                )
            })
            .collect();
        format!(
            "{{\"code\":{},\"name\":{},\"kind\":{},\"enabled\":{},\"requires_city\":{},\"description\":{},\"schedules\":[{}]}}",
            json::quote(&item.event.code),
            json::quote(&item.event.name),
            json::quote(&item.event.kind),
            item.event.enabled,
            item.event.requires_city,
            json::quote(&item.event.description),
            horarios.join(",")
        )
    }).collect();
    format!("[{}]", itens.join(","))
}
