//! `content_rates/*` — taxas de refino, SancRate, composição, multiplicadores e
//! refino de acessório, lidas/gravadas pelo portal (FASE-07-6).
//!
//! Toda escrita exige fonte (`content_source`) e passa por `ConfigStore`, que grava o
//! override, incrementa `config_revision` e registra `config_change` na mesma transação.
//! Nenhuma rota aqui toca personagem vivo: é configuração de mundo.

use wyd_db::{AccessoryRuleRow, ConfigRateRow};

use crate::audit::{log_action, AuditSink};
use crate::auth::Session;
use crate::error::AdminError;
use crate::transport::json;

use super::accounts::staff_or_deny;

/// Teto do texto de fonte/nota aceito do staff.
pub const MAX_SOURCE: usize = 160;

/// Foto completa da configuração de taxas.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RatesSnapshot {
    pub revision: i64,
    pub rates: Vec<ConfigRateRow>,
    pub accessories: Vec<AccessoryRuleRow>,
}

/// Porta de configuração do gateway.
///
/// Síncrona de propósito: o roteador não tem runtime (ADR-0006); o binário embrulha as
/// chamadas `sqlx` com `block_on`, como já faz em login e presença.
pub trait ConfigStore: Send + Sync {
    fn rates_snapshot(&self) -> Result<RatesSnapshot, AdminError>;

    fn set_rate(
        &self,
        domain: &str,
        key: &str,
        value: i64,
        source: &str,
        actor: i64,
    ) -> Result<i64, AdminError>;

    fn reset_rate(&self, domain: &str, key: &str, actor: i64) -> Result<i64, AdminError>;

    fn drops(&self, template: Option<&str>, limit: i64) -> Result<Vec<DropRow>, AdminError>;

    fn set_drop(&self, edit: &DropEdit, actor: i64) -> Result<i64, AdminError>;

    fn reset_drop(&self, template: &str, item_index: i32, actor: i64) -> Result<i64, AdminError>;

    fn npc(&self, id: i32) -> Result<Option<NpcRow>, AdminError>;

    fn set_npc(
        &self,
        id: i32,
        map_id: i16,
        pos_x: i16,
        pos_y: i16,
        enabled: bool,
        actor: i64,
    ) -> Result<i64, AdminError>;

    fn routes(&self, limit: i64) -> Result<Vec<RouteRow>, AdminError>;

    fn set_route(
        &self,
        id: i32,
        cost_gold: Option<i32>,
        req_level: Option<i16>,
        enabled: Option<bool>,
        actor: i64,
    ) -> Result<i64, AdminError>;
}

/// Edição de um peso de drop (agrupada para manter a assinatura curta).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DropEdit {
    pub template_name: String,
    pub item_index: i32,
    pub weight: i64,
    pub min_qty: i16,
    pub max_qty: i16,
    pub source: String,
}

/// Linha de drop exposta ao portal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DropRow {
    pub template_name: String,
    pub item_index: i32,
    pub item_name: String,
    pub weight: i64,
    pub min_qty: i16,
    pub max_qty: i16,
    pub source: String,
}

/// NPC exposto ao portal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NpcRow {
    pub id: i32,
    pub slug: String,
    pub display_name: String,
    pub map_id: i16,
    pub pos_x: i16,
    pub pos_y: i16,
    pub enabled: bool,
    pub merchant: i16,
}

/// Rota de teleporte exposta ao portal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RouteRow {
    pub id: i32,
    pub from_map_id: i16,
    pub from_x: i16,
    pub from_y: i16,
    pub to_map_id: i16,
    pub to_x: i16,
    pub to_y: i16,
    pub npc_id: Option<i32>,
    pub cost_gold: i32,
    pub req_level: i16,
    pub enabled: bool,
}

/// Domínios aceitos por `upsert_rate`/`reset_rate`.
pub const DOMAINS: &[&str] = &[
    "refine",
    "sanc",
    "combine",
    "multiplier",
    "setting",
    "accessory",
];

fn valid_domain(domain: &str) -> Result<(), AdminError> {
    if DOMAINS.contains(&domain) {
        return Ok(());
    }
    Err(AdminError::InvalidInput {
        reason: "domínio de taxa desconhecido",
    })
}

/// Leitura completa (`content_rates/get_rates`), auditada.
pub fn get_rates(
    session: Option<Session>,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<RatesSnapshot, AdminError> {
    let staff = staff_or_deny(session, sink, "config.rates")?;
    let snapshot = store.rates_snapshot()?;
    log_action(
        sink,
        &staff,
        "config.rates",
        "rates",
        &format!(
            "revisão {} ({} taxas, {} regras de acessório)",
            snapshot.revision,
            snapshot.rates.len(),
            snapshot.accessories.len()
        ),
    );
    Ok(snapshot)
}

/// Grava uma taxa; a fonte é obrigatória e o valor não pode ser negativo.
pub fn upsert_rate(
    session: Option<Session>,
    domain: &str,
    key: &str,
    value: i64,
    source: &str,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.rate.set")?;
    valid_domain(domain)?;
    if key.trim().is_empty() || key.len() > 120 {
        return Err(AdminError::InvalidInput {
            reason: "chave da taxa vazia ou longa demais",
        });
    }
    let source = source.trim();
    if source.is_empty() || source.chars().count() > MAX_SOURCE {
        return Err(AdminError::InvalidInput {
            reason: "fonte da taxa obrigatória (até 160 caracteres)",
        });
    }
    if value < 0 {
        return Err(AdminError::InvalidInput {
            reason: "taxa não pode ser negativa",
        });
    }
    let revision = store.set_rate(domain, key.trim(), value, source, staff.moderator_id())?;
    log_action(
        sink,
        &staff,
        "config.rate.set",
        &format!("{domain}:{key}"),
        &format!("valor {value} (revisão {revision})"),
    );
    Ok(revision)
}

/// Remove o override e devolve o valor documentado.
pub fn reset_rate(
    session: Option<Session>,
    domain: &str,
    key: &str,
    store: &dyn ConfigStore,
    sink: &impl AuditSink,
) -> Result<i64, AdminError> {
    let staff = staff_or_deny(session, sink, "config.rate.reset")?;
    valid_domain(domain)?;
    let revision = store.reset_rate(domain, key.trim(), staff.moderator_id())?;
    log_action(
        sink,
        &staff,
        "config.rate.reset",
        &format!("{domain}:{key}"),
        &format!("default restaurado (revisão {revision})"),
    );
    Ok(revision)
}

/// Bandeiras de servidor derivadas das configurações com valor inteiro.
#[must_use]
pub fn flags_from(snapshot: &RatesSnapshot) -> Vec<(&str, i64)> {
    snapshot
        .rates
        .iter()
        .filter(|row| row.domain == "setting")
        .map(|row| (row.key.as_str(), row.value))
        .collect()
}

/// JSON de uma taxa.
#[must_use]
pub fn rate_json(row: &ConfigRateRow) -> String {
    format!(
        "{{\"domain\":{},\"key\":{},\"value\":{},\"unit\":{},\"source\":{},\"overridden\":{}}}",
        json::quote(&row.domain),
        json::quote(&row.key),
        row.value,
        json::quote(&row.unit),
        json::quote(&row.source),
        row.overridden
    )
}

/// JSON de uma regra de refino de acessório.
#[must_use]
pub fn accessory_json(row: &AccessoryRuleRow) -> String {
    format!(
        "{{\"kind\":{},\"recipe\":{},\"step_from\":{},\"step_to\":{},\"rate_bps\":{},\
\"fail_mode\":{},\"fail_bonus_bps\":{},\"gold_cost\":{},\"material_name\":{},\
\"material_qty\":{},\"source\":{},\"notes\":{}}}",
        json::quote(&row.kind),
        json::quote(&row.recipe),
        row.step_from,
        row.step_to,
        row.rate_bps,
        json::quote(&row.fail_mode),
        row.fail_bonus_bps,
        row.gold_cost,
        json::quote(&row.material_name),
        row.material_qty,
        json::quote(&row.source),
        json::quote(&row.notes)
    )
}

/// JSON da foto completa.
#[must_use]
pub fn snapshot_json(snapshot: &RatesSnapshot) -> String {
    let rates: Vec<String> = snapshot.rates.iter().map(rate_json).collect();
    let accessories: Vec<String> = snapshot.accessories.iter().map(accessory_json).collect();
    format!(
        "\"revision\":{},\"rates\":[{}],\"accessories\":[{}]",
        snapshot.revision,
        rates.join(","),
        accessories.join(",")
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::audit::VecSink;
    use crate::auth::Session;
    use wyd_db::Role::Moderator;

    struct Loja {
        chamadas: std::sync::Mutex<Vec<String>>,
    }

    impl ConfigStore for Loja {
        fn rates_snapshot(&self) -> Result<RatesSnapshot, AdminError> {
            Ok(RatesSnapshot {
                revision: 7,
                rates: Vec::new(),
                accessories: Vec::new(),
            })
        }

        fn set_rate(
            &self,
            domain: &str,
            key: &str,
            value: i64,
            _source: &str,
            _actor: i64,
        ) -> Result<i64, AdminError> {
            self.chamadas
                .lock()
                .expect("lock")
                .push(format!("{domain}:{key}={value}"));
            Ok(8)
        }

        fn reset_rate(&self, _domain: &str, _key: &str, _actor: i64) -> Result<i64, AdminError> {
            Ok(9)
        }

        fn drops(&self, _template: Option<&str>, _limit: i64) -> Result<Vec<DropRow>, AdminError> {
            Ok(Vec::new())
        }

        fn set_drop(&self, _edit: &DropEdit, _actor: i64) -> Result<i64, AdminError> {
            Ok(10)
        }

        fn reset_drop(
            &self,
            _template: &str,
            _item_index: i32,
            _actor: i64,
        ) -> Result<i64, AdminError> {
            Ok(11)
        }

        fn npc(&self, _id: i32) -> Result<Option<NpcRow>, AdminError> {
            Ok(None)
        }

        fn set_npc(
            &self,
            _id: i32,
            _map_id: i16,
            _pos_x: i16,
            _pos_y: i16,
            _enabled: bool,
            _actor: i64,
        ) -> Result<i64, AdminError> {
            Ok(12)
        }

        fn routes(&self, _limit: i64) -> Result<Vec<RouteRow>, AdminError> {
            Ok(Vec::new())
        }

        fn set_route(
            &self,
            _id: i32,
            _cost_gold: Option<i32>,
            _req_level: Option<i16>,
            _enabled: Option<bool>,
            _actor: i64,
        ) -> Result<i64, AdminError> {
            Ok(13)
        }
    }

    fn staff() -> Session {
        Session {
            account_id: 7,
            role: Moderator,
        }
    }

    fn loja() -> Loja {
        Loja {
            chamadas: std::sync::Mutex::new(Vec::new()),
        }
    }

    #[test]
    fn taxa_sem_fonte_e_recusada_antes_do_banco() {
        let store = loja();
        let sink = VecSink::default();
        let erro = upsert_rate(Some(staff()), "refine", "12", 4_000, "  ", &store, &sink)
            .expect_err("fonte vazia");
        assert!(matches!(erro, AdminError::InvalidInput { .. }));
        assert!(store.chamadas.lock().expect("lock").is_empty());
    }

    #[test]
    fn dominio_desconhecido_nao_chega_ao_repositorio() {
        let store = loja();
        let sink = VecSink::default();
        assert!(upsert_rate(Some(staff()), "drops", "x", 1, "doc", &store, &sink).is_err());
        assert!(store.chamadas.lock().expect("lock").is_empty());
    }

    #[test]
    fn taxa_valida_grava_e_audita() {
        let store = loja();
        let sink = VecSink::default();
        let revision = upsert_rate(
            Some(staff()),
            "refine",
            "12",
            4_000,
            "WYD International",
            &store,
            &sink,
        )
        .expect("grava");
        assert_eq!(revision, 8);
        assert_eq!(store.chamadas.lock().expect("lock")[0], "refine:12=4000");
        assert_eq!(sink.entries()[0].action, "config.rate.set");
    }

    #[test]
    fn jogador_nao_le_taxas() {
        let store = loja();
        let sink = VecSink::default();
        let jogador = Session {
            account_id: 3,
            role: wyd_db::Role::Player,
        };
        assert!(get_rates(Some(jogador), &store, &sink).is_err());
        assert!(store.chamadas.lock().expect("lock").is_empty());
    }
}
