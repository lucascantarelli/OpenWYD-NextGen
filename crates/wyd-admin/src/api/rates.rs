//! `content_rates/*` — taxas de refino, SancRate, composição, multiplicadores e
//! refino de acessório, lidas/gravadas pelo portal (WEB-06).
//!
//! Toda escrita exige fonte (`content_source`) e passa por `ConfigStore`, que grava o
//! override, incrementa `config_revision` e registra `config_change` na mesma transação.
//! Nenhuma rota aqui toca personagem vivo: é configuração de mundo.

use wyd_db::{AccessoryRuleRow, ConfigRateRow};

use crate::audit::{log_action, AuditSink};
use crate::auth::Session;
use crate::error::AdminError;
use crate::transport::json;

use super::accounts::{staff_or_deny, AccountSummary};
use super::donate::{
    AccountLookupRow, DebitResult, DonateBuyerRow, DonateLedgerRow, DonateRevenueRow,
};

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

    /// Página de NPCs por id para a lista do portal (WEB-05, só leitura).
    fn npcs(&self, limit: i64, after_id: Option<i32>) -> Result<Vec<NpcRow>, AdminError>;

    /// Templates de mob por nome para a lista do portal (WEB-05, só leitura).
    fn mob_templates(
        &self,
        limit: i64,
        after: Option<&str>,
    ) -> Result<Vec<MobTemplateRow>, AdminError>;

    /// Efeitos de item por nome para a lista do portal (WEB-05, só leitura).
    fn effects(&self, limit: i64, after: Option<&str>) -> Result<Vec<EffectRow>, AdminError>;

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

    /// Recompensas diárias em páginas por id (WEB-06, só leitura).
    fn daily_rewards(
        &self,
        limit: i64,
        after_id: Option<i32>,
    ) -> Result<Vec<DailyRewardRow>, AdminError>;

    /// Cria (`id = None`) ou substitui uma recompensa diária (auditado).
    fn upsert_daily_reward(&self, edit: &DailyEdit, actor: i64) -> Result<i64, AdminError>;

    /// Remove uma recompensa diária (auditado).
    fn delete_daily_reward(&self, id: i32, actor: i64) -> Result<i64, AdminError>;

    /// Loja de donate em páginas por id (WEB-06, só leitura).
    fn donate_items(
        &self,
        limit: i64,
        after_id: Option<i32>,
    ) -> Result<Vec<DonateItemRow>, AdminError>;

    /// Cria (`id = None`) ou substitui um item da loja (auditado).
    fn upsert_donate_item(&self, edit: &DonateEdit, actor: i64) -> Result<i64, AdminError>;

    /// Liga/desliga um item da loja (auditado).
    fn set_donate_item_enabled(
        &self,
        id: i32,
        enabled: bool,
        actor: i64,
    ) -> Result<i64, AdminError>;

    /// Remove um item da loja (auditado).
    fn delete_donate_item(&self, id: i32, actor: i64) -> Result<i64, AdminError>;

    /// Eventos de mundo com estado (WEB-06, só leitura).
    fn world_events(&self) -> Result<Vec<WorldEventRow>, AdminError>;

    /// Horários de todos os eventos (WEB-06, só leitura).
    fn event_schedules(&self) -> Result<Vec<EventScheduleRow>, AdminError>;

    /// Liga/desliga um evento (auditado; efeito no próximo leitor do dado).
    fn set_world_event_enabled(
        &self,
        code: &str,
        enabled: bool,
        actor: i64,
    ) -> Result<i64, AdminError>;

    /// Contas em páginas por id, sem segredo (WEB-06, só leitura).
    fn accounts(
        &self,
        limit: i64,
        after_id: Option<i64>,
    ) -> Result<Vec<AccountSummary>, AdminError>;

    /// Uma conta por id, sem segredo.
    fn account(&self, id: i64) -> Result<Option<AccountSummary>, AdminError>;

    /// Credita saldo de donate na carteira da conta, idempotente por chave.
    ///
    /// Devolve `(razão, replay)`: mesma chave = mesmo razão, sem crédito duplo.
    fn credit_donate(
        &self,
        account_id: i64,
        credits: i64,
        key: &str,
        actor: i64,
    ) -> Result<(i64, bool), AdminError>;

    /// Receita de donate: créditos, débitos e contagens (WEB-06, só leitura).
    fn donate_revenue(&self) -> Result<DonateRevenueRow, AdminError>;

    /// Extrato de recargas, mais recentes primeiro.
    fn donate_credits(&self, limit: i64) -> Result<Vec<DonateLedgerRow>, AdminError>;

    /// Extrato de débitos, mais recentes primeiro.
    fn donate_debits(&self, limit: i64) -> Result<Vec<DonateLedgerRow>, AdminError>;

    /// Contas que mais compraram créditos.
    fn donate_top_buyers(&self, limit: i64) -> Result<Vec<DonateBuyerRow>, AdminError>;

    /// Debita saldo de donate da conta, idempotente por chave.
    fn debit_donate(
        &self,
        account_id: i64,
        credits: i64,
        key: &str,
        reason: &str,
        actor: i64,
    ) -> Result<DebitResult, AdminError>;

    /// Busca conta por prefixo de login (id + papel, sem segredo).
    fn search_accounts(&self, term: &str, limit: i64) -> Result<Vec<AccountLookupRow>, AdminError>;

    fn set_route(
        &self,
        id: i32,
        cost_gold: Option<i32>,
        req_level: Option<i16>,
        enabled: Option<bool>,
        actor: i64,
    ) -> Result<i64, AdminError>;

    /// Preços de NPC (`item_price`), por item quando informado (WEB-05, só leitura).
    fn item_prices(
        &self,
        item_index: Option<i32>,
        limit: i64,
    ) -> Result<Vec<ItemPriceRow>, AdminError>;

    /// Grava o preço de (item, NPC) e devolve a revisão (auditado).
    fn set_item_price(&self, edit: &ItemPriceEdit, actor: i64) -> Result<i64, AdminError>;
}

/// Preço de NPC exposto ao portal (`item_price`).
///
/// Tabela vazia = nenhum preço definido pelo dono do servidor: a tela mostra ausência,
/// nunca zero nem estimativa.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ItemPriceRow {
    pub item_index: i32,
    pub item_name: String,
    pub npc_id: i32,
    pub npc_slug: String,
    pub buy_price: i64,
    pub sell_price: i64,
    pub source: String,
}

/// Edição de preço de NPC (`item_index`/`npc_id` obrigatórios: a chave é o par).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ItemPriceEdit {
    pub item_index: i32,
    pub npc_id: i32,
    pub buy_price: i64,
    pub sell_price: i64,
    pub source: String,
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

/// Template de mob exposto ao portal (nome recuperado quando existe).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MobTemplateRow {
    pub template_name: String,
    pub name: String,
    pub source_size: i16,
}

/// Efeito de item exposto ao portal (símbolo `EF_*`, sem código numérico).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct EffectRow {
    pub name: String,
    pub label: String,
    pub category: String,
    pub description: String,
    pub confirmed: bool,
}

/// Recompensa diária exposta ao portal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DailyRewardRow {
    pub id: i32,
    pub day: Option<i16>,
    pub item_index: Option<i32>,
    pub gold: i64,
    pub qty: i16,
    pub account_wide: bool,
}

/// Edição de recompensa diária (`id = None` cria).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DailyEdit {
    pub id: Option<i32>,
    pub day: Option<i16>,
    pub item_index: Option<i32>,
    pub gold: i64,
    pub qty: i16,
    pub account_wide: bool,
}

/// Item da loja de donate exposto ao portal.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DonateItemRow {
    pub id: i32,
    pub item_index: i32,
    pub price_credits: i32,
    pub duration_days: Option<i16>,
    pub enabled: bool,
}

/// Edição de item da loja (`id = None` cria).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DonateEdit {
    pub id: Option<i32>,
    pub item_index: i32,
    pub price_credits: i32,
    pub duration_days: Option<i16>,
    pub enabled: bool,
}

/// Evento de mundo com estado.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct WorldEventRow {
    pub code: String,
    pub name: String,
    pub kind: String,
    pub enabled: bool,
    pub requires_city: bool,
    pub description: String,
}

/// Horário de um evento.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct EventScheduleRow {
    pub id: i32,
    pub event_code: String,
    pub weekday: i16,
    pub start_time: String,
    pub duration_minutes: i16,
    pub timezone: String,
    pub every_n_weeks: i16,
    pub enabled: bool,
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

        fn npcs(&self, _limit: i64, _after_id: Option<i32>) -> Result<Vec<NpcRow>, AdminError> {
            Ok(Vec::new())
        }

        fn mob_templates(
            &self,
            _limit: i64,
            _after: Option<&str>,
        ) -> Result<Vec<MobTemplateRow>, AdminError> {
            Ok(Vec::new())
        }

        fn effects(&self, _limit: i64, _after: Option<&str>) -> Result<Vec<EffectRow>, AdminError> {
            Ok(Vec::new())
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

        fn daily_rewards(
            &self,
            _limit: i64,
            _after_id: Option<i32>,
        ) -> Result<Vec<DailyRewardRow>, AdminError> {
            Ok(Vec::new())
        }

        fn upsert_daily_reward(&self, _edit: &DailyEdit, _actor: i64) -> Result<i64, AdminError> {
            Ok(14)
        }

        fn delete_daily_reward(&self, _id: i32, _actor: i64) -> Result<i64, AdminError> {
            Ok(15)
        }

        fn donate_items(
            &self,
            _limit: i64,
            _after_id: Option<i32>,
        ) -> Result<Vec<DonateItemRow>, AdminError> {
            Ok(Vec::new())
        }

        fn upsert_donate_item(&self, _edit: &DonateEdit, _actor: i64) -> Result<i64, AdminError> {
            Ok(16)
        }

        fn set_donate_item_enabled(
            &self,
            _id: i32,
            _enabled: bool,
            _actor: i64,
        ) -> Result<i64, AdminError> {
            Ok(17)
        }

        fn delete_donate_item(&self, _id: i32, _actor: i64) -> Result<i64, AdminError> {
            Ok(18)
        }

        fn world_events(&self) -> Result<Vec<WorldEventRow>, AdminError> {
            Ok(Vec::new())
        }

        fn event_schedules(&self) -> Result<Vec<EventScheduleRow>, AdminError> {
            Ok(Vec::new())
        }

        fn set_world_event_enabled(
            &self,
            _code: &str,
            _enabled: bool,
            _actor: i64,
        ) -> Result<i64, AdminError> {
            Ok(19)
        }

        fn accounts(
            &self,
            _limit: i64,
            _after_id: Option<i64>,
        ) -> Result<Vec<AccountSummary>, AdminError> {
            Ok(Vec::new())
        }

        fn account(&self, _id: i64) -> Result<Option<AccountSummary>, AdminError> {
            Ok(None)
        }

        fn credit_donate(
            &self,
            _account_id: i64,
            _credits: i64,
            _key: &str,
            _actor: i64,
        ) -> Result<(i64, bool), AdminError> {
            Ok((20, false))
        }

        fn donate_revenue(&self) -> Result<DonateRevenueRow, AdminError> {
            Ok(DonateRevenueRow::default())
        }

        fn donate_credits(&self, _limit: i64) -> Result<Vec<DonateLedgerRow>, AdminError> {
            Ok(Vec::new())
        }

        fn donate_debits(&self, _limit: i64) -> Result<Vec<DonateLedgerRow>, AdminError> {
            Ok(Vec::new())
        }

        fn donate_top_buyers(&self, _limit: i64) -> Result<Vec<DonateBuyerRow>, AdminError> {
            Ok(Vec::new())
        }

        fn debit_donate(
            &self,
            _account_id: i64,
            _credits: i64,
            _key: &str,
            _reason: &str,
            _actor: i64,
        ) -> Result<DebitResult, AdminError> {
            Ok(DebitResult::Applied(21))
        }

        fn search_accounts(
            &self,
            _term: &str,
            _limit: i64,
        ) -> Result<Vec<AccountLookupRow>, AdminError> {
            Ok(Vec::new())
        }

        fn item_prices(
            &self,
            _item_index: Option<i32>,
            _limit: i64,
        ) -> Result<Vec<ItemPriceRow>, AdminError> {
            Ok(Vec::new())
        }

        fn set_item_price(&self, _edit: &ItemPriceEdit, _actor: i64) -> Result<i64, AdminError> {
            Ok(21)
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
        // WEB-11: a trilha carrega o autor real (conta de staff da sessão),
        // nunca o corpo da chamada.
        assert_eq!(sink.entries()[0].action, "config.rate.set");
        assert_eq!(sink.entries()[0].actor_id, 7);
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
