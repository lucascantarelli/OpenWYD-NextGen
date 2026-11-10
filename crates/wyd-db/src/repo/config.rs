//! Configuração editável pelo portal (FASE-07-6): taxas, drops, NPC e teleporte.
//!
//! Duas camadas de propósito: o **seed** continua sendo a fonte histórica e a edição do
//! staff vira `*_override`, com autor, fonte e data. O `reset` apaga o override e o valor
//! documentado volta — sem reescrever migração e sem perder a proveniência.
//!
//! Toda escrita faz três coisas **na mesma transação**: grava o dado, incrementa
//! `config_revision` (o servidor recarrega fora do tick quando a versão muda) e registra
//! `config_change` com o valor anterior. Nada aqui toca personagem vivo.

use sqlx::Row;

use crate::error::DbError;
use crate::pool::pg::{map_error, PgBackend};

/// Uma linha de taxa como o portal a edita (já com o override aplicado).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ConfigRateRow {
    /// `refine`, `sanc`, `combine`, `multiplier` ou `setting`.
    pub domain: String,
    /// Chave estável dentro do domínio (`12`, `PO:5`, `Odin.PLUS12_BASE`, `drop`…).
    pub key: String,
    pub value: i64,
    pub unit: String,
    pub source: String,
    pub overridden: bool,
}

/// Regra de refino de acessório (uma por passo e receita).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccessoryRuleRow {
    pub kind: String,
    pub recipe: String,
    pub step_from: i16,
    pub step_to: i16,
    pub rate_bps: i32,
    pub fail_mode: String,
    pub fail_bonus_bps: i32,
    pub gold_cost: i64,
    pub material_name: String,
    pub material_qty: i16,
    pub source: String,
    pub notes: String,
}

/// Peso de drop por (template de monstro, item), como o portal edita.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DropConfigRow {
    pub template_name: String,
    pub item_index: i32,
    pub item_name: String,
    pub weight: i64,
    pub min_qty: i16,
    pub max_qty: i16,
    pub source: String,
}

/// NPC com posição editável (não é estado vivo: é definição de mundo).
///
/// `template` é o `STRUCT_MOB` de 816 B de `npc_template`: é dele que saem nome, EXP,
/// nível e atributos que o servidor usa no spawn e no abate. O template vem junto com a
/// definição para que o snapshot não precise de uma segunda consulta por NPC.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NpcConfigRow {
    pub id: i32,
    pub slug: String,
    pub display_name: String,
    pub template_name: String,
    pub template: Vec<u8>,
    pub map_id: i16,
    pub pos_x: i16,
    pub pos_y: i16,
    pub enabled: bool,
    pub merchant: i16,
}

/// NPC para a lista do portal (sem o `blob` do template: a lista não precisa de 816 B por
/// linha; quem edita um NPC abre `config_npc` por id).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NpcListRow {
    pub id: i32,
    pub slug: String,
    pub display_name: String,
    pub template_name: String,
    pub map_id: i16,
    pub pos_x: i16,
    pub pos_y: i16,
    pub enabled: bool,
    pub merchant: i16,
}

/// Template de mob para a lista do portal (vem de `npc_template`, que tem 1.991 linhas
/// semeadas; `mob_template_stat` continua vazia até a extração do `STRUCT_MOB`, FASE-10).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MobTemplateRow {
    pub template_name: String,
    /// `name_recovered` quando existe, senão o próprio nome do template.
    pub name: String,
    pub source_size: i16,
}

/// Edição de recompensa diária (cria com `id = None`, substitui com `Some`).
///
/// Agrupa os 7 campos do upsert num struct só: o clippy da série limita funções a
/// 7 argumentos e o `actor` de auditoria já ocupa um.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DailyRewardEdit {
    pub id: Option<i32>,
    pub day: Option<i16>,
    pub item_index: Option<i32>,
    pub gold: i64,
    pub qty: i16,
    pub account_wide: bool,
}

/// Recompensa diária para a lista do portal (`daily_reward_item`).
///
/// Sem `enabled` nem `content_source`: a tabela 0010 não tem essas colunas e a série de
/// migrations não usa `ALTER TABLE` no `up` — ligar/desligar dia específico segue `501`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DailyRewardRow {
    pub id: i32,
    pub day: Option<i16>,
    pub item_index: Option<i32>,
    pub gold: i64,
    pub qty: i16,
    pub account_wide: bool,
}

/// Item da loja de donate para a lista do portal (`donate_shop_item`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DonateItemRow {
    pub id: i32,
    pub item_index: i32,
    pub price_credits: i32,
    pub duration_days: Option<i16>,
    pub enabled: bool,
}

/// Evento de mundo para o painel (`world_event` + `event_schedule`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct WorldEventRow {
    pub code: String,
    pub name: String,
    pub kind: String,
    pub enabled: bool,
    pub requires_city: bool,
    pub description: String,
}

/// Horário de um evento (`event_schedule`, `timezone` explícito por linha).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct EventScheduleRow {
    pub id: i32,
    pub event_code: String,
    pub weekday: i16,
    /// `HH:MM:SS` como o banco devolve.
    pub start_time: String,
    pub duration_minutes: i16,
    pub timezone: String,
    pub every_n_weeks: i16,
    pub enabled: bool,
}

/// Conta para a lista do portal: sem hash, PIN, segredo ou token.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccountListRow {
    pub id: i64,
    pub login: String,
    pub role: String,
}

/// Efeito de item para a lista do portal (`item_effect`, chave `EF_*`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct EffectRow {
    pub name: String,
    pub label: String,
    pub category: String,
    pub description: String,
    pub confirmed: bool,
}

/// Rota de teleporte editável.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RouteConfigRow {
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

fn invalid(table: &'static str, constraint: &'static str) -> DbError {
    DbError::ConstraintViolation { table, constraint }
}

impl PgBackend {
    /// Taxas de refino, SancRate, composição, multiplicadores e configurações de servidor.
    ///
    /// Uma leitura só: o portal monta a tela inteira com uma foto coerente.
    pub async fn config_rates(
        &self,
    ) -> Result<(Vec<ConfigRateRow>, Vec<AccessoryRuleRow>), DbError> {
        let mut rows = Vec::new();

        let refine = sqlx::query(
            "SELECT r.step::int AS step, \
             COALESCE(o.rate_bps, r.rate_bps)::int AS rate_bps, \
             COALESCE(o.content_source, r.content_source) AS source, \
             (o.step IS NOT NULL) AS overridden \
             FROM refine_rate r LEFT JOIN refine_rate_override o ON o.step = r.step \
             ORDER BY r.step",
        )
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.rates.refine", "refine_rate"))?;
        for row in &refine {
            rows.push(ConfigRateRow {
                domain: "refine".to_string(),
                key: row
                    .try_get::<i32, _>("step")
                    .unwrap_or_default()
                    .to_string(),
                value: i64::from(row.try_get::<i32, _>("rate_bps").unwrap_or_default()),
                unit: "bps".to_string(),
                source: row.try_get("source").unwrap_or_default(),
                overridden: row.try_get("overridden").unwrap_or(false),
            });
        }

        let sanc = sqlx::query(
            "SELECT s.powder, s.refine_index::int AS refine_index, \
             COALESCE(o.percent, s.percent)::int AS percent, \
             COALESCE(o.content_source, s.content_source) AS source, \
             (o.powder IS NOT NULL) AS overridden \
             FROM sanc_rate s \
             LEFT JOIN sanc_rate_override o \
               ON o.powder = s.powder AND o.refine_index = s.refine_index \
             ORDER BY s.powder, s.refine_index",
        )
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.rates.sanc", "sanc_rate"))?;
        for row in &sanc {
            let powder: String = row.try_get("powder").unwrap_or_default();
            let index: i32 = row.try_get("refine_index").unwrap_or_default();
            rows.push(ConfigRateRow {
                domain: "sanc".to_string(),
                key: format!("{powder}:{index}"),
                value: i64::from(row.try_get::<i32, _>("percent").unwrap_or_default()),
                unit: "percent".to_string(),
                source: row.try_get("source").unwrap_or_default(),
                overridden: row.try_get("overridden").unwrap_or(false),
            });
        }

        let combine = sqlx::query(
            "SELECT c.family, c.key, \
             COALESCE(o.value, c.value)::bigint AS value, \
             COALESCE(o.unit, c.unit) AS unit, \
             COALESCE(o.content_source, c.content_source) AS source, \
             (o.family IS NOT NULL) AS overridden \
             FROM combine_value c \
             LEFT JOIN combine_value_override o ON o.family = c.family AND o.key = c.key \
             ORDER BY c.family, c.key",
        )
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.rates.combine", "combine_value"))?;
        for row in &combine {
            let family: String = row.try_get("family").unwrap_or_default();
            let key: String = row.try_get("key").unwrap_or_default();
            rows.push(ConfigRateRow {
                domain: "combine".to_string(),
                key: format!("{family}.{key}"),
                value: row.try_get::<i64, _>("value").unwrap_or_default(),
                unit: row.try_get("unit").unwrap_or_default(),
                source: row.try_get("source").unwrap_or_default(),
                overridden: row.try_get("overridden").unwrap_or(false),
            });
        }

        let multipliers = sqlx::query(
            "SELECT kind, rate_bps::bigint AS rate_bps, content_source \
             FROM content_rate_override ORDER BY kind",
        )
        .fetch_all(self.pool())
        .await
        .map_err(map_error(
            "config.rates.multiplier",
            "content_rate_override",
        ))?;
        for row in &multipliers {
            rows.push(ConfigRateRow {
                domain: "multiplier".to_string(),
                key: row.try_get("kind").unwrap_or_default(),
                value: row.try_get::<i64, _>("rate_bps").unwrap_or_default(),
                unit: "bps".to_string(),
                source: row.try_get("content_source").unwrap_or_default(),
                overridden: true,
            });
        }

        let settings = sqlx::query(
            "SELECT section, key, value_int FROM game_setting \
             WHERE value_int IS NOT NULL \
               AND (key ~* '(rate|drop|exp|tax|bonus|hour|nightmare|sapphire|fame|evon|evitem|evrate|evindex|evdelete|kefra)') \
             ORDER BY section, key",
        )
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.rates.setting", "game_setting"))?;
        for row in &settings {
            let section: String = row.try_get("section").unwrap_or_default();
            let key: String = row.try_get("key").unwrap_or_default();
            rows.push(ConfigRateRow {
                domain: "setting".to_string(),
                key: format!("{section}.{key}"),
                value: row.try_get::<i64, _>("value_int").unwrap_or_default(),
                unit: "int".to_string(),
                source: "game_setting".to_string(),
                overridden: false,
            });
        }

        let rules = sqlx::query(
            "SELECT kind, recipe, step_from::int AS step_from, step_to::int AS step_to, \
             rate_bps, fail_mode, fail_bonus_bps, gold_cost::bigint AS gold_cost, \
             material_name, material_qty::int AS material_qty, content_source, notes \
             FROM accessory_refine_rule \
             ORDER BY kind, recipe, step_from",
        )
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.rates.accessory", "accessory_refine_rule"))?;
        let mut accessories = Vec::with_capacity(rules.len());
        for row in &rules {
            accessories.push(AccessoryRuleRow {
                kind: row.try_get("kind").unwrap_or_default(),
                recipe: row.try_get("recipe").unwrap_or_default(),
                step_from: row.try_get::<i32, _>("step_from").unwrap_or_default() as i16,
                step_to: row.try_get::<i32, _>("step_to").unwrap_or_default() as i16,
                rate_bps: row.try_get("rate_bps").unwrap_or_default(),
                fail_mode: row.try_get("fail_mode").unwrap_or_default(),
                fail_bonus_bps: row.try_get("fail_bonus_bps").unwrap_or_default(),
                gold_cost: row.try_get("gold_cost").unwrap_or_default(),
                material_name: row.try_get("material_name").unwrap_or_default(),
                material_qty: row.try_get::<i32, _>("material_qty").unwrap_or_default() as i16,
                source: row.try_get("content_source").unwrap_or_default(),
                notes: row.try_get("notes").unwrap_or_default(),
            });
        }

        Ok((rows, accessories))
    }

    /// Grava uma taxa (domínio + chave) e devolve a nova revisão de configuração.
    #[allow(clippy::too_many_lines)]
    pub async fn config_set_rate(
        &self,
        domain: &str,
        key: &str,
        value: i64,
        source: &str,
        actor: i64,
    ) -> Result<i64, DbError> {
        if value < 0 {
            return Err(invalid("config_rate", "value_non_negative"));
        }
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "config.set_rate",
        })?;

        let (target, old) = match domain {
            "refine" => {
                let step: i16 = key
                    .parse()
                    .map_err(|_| invalid("refine_rate_override", "step"))?;
                if !(1..=15).contains(&step) || value > 10_000 {
                    return Err(invalid("refine_rate_override", "rate_bps"));
                }
                let old: Option<i32> = sqlx::query_scalar(
                    "SELECT COALESCE(o.rate_bps, r.rate_bps)::int \
                     FROM refine_rate r LEFT JOIN refine_rate_override o ON o.step = r.step \
                     WHERE r.step = $1",
                )
                .bind(step)
                .fetch_optional(&mut *tx)
                .await
                .map_err(map_error("config.set_rate.refine", "refine_rate"))?;
                sqlx::query(
                    "INSERT INTO refine_rate_override (step, rate_bps, content_source, updated_by) \
                     VALUES ($1, $2, $3, $4) \
                     ON CONFLICT (step) DO UPDATE SET rate_bps = EXCLUDED.rate_bps, \
                       content_source = EXCLUDED.content_source, \
                       updated_by = EXCLUDED.updated_by, updated_at = now()",
                )
                .bind(step)
                .bind(i32::try_from(value).map_err(|_| invalid("refine_rate_override", "rate_bps"))?)
                .bind(source)
                .bind(actor)
                .execute(&mut *tx)
                .await
                .map_err(map_error("config.set_rate.refine.write", "refine_rate_override"))?;
                (
                    format!("refine:{step}"),
                    old.map_or_else(String::new, |v| v.to_string()),
                )
            }
            "sanc" => {
                let (powder, index) = key
                    .split_once(':')
                    .ok_or_else(|| invalid("sanc_rate_override", "key"))?;
                if !matches!(powder, "PO" | "PL" | "AMAGO") || value > 100 {
                    return Err(invalid("sanc_rate_override", "percent"));
                }
                let index: i16 = index
                    .parse()
                    .map_err(|_| invalid("sanc_rate_override", "refine_index"))?;
                let old: Option<i32> = sqlx::query_scalar(
                    "SELECT COALESCE(o.percent, s.percent)::int FROM sanc_rate s \
                     LEFT JOIN sanc_rate_override o \
                       ON o.powder = s.powder AND o.refine_index = s.refine_index \
                     WHERE s.powder = $1 AND s.refine_index = $2",
                )
                .bind(powder)
                .bind(index)
                .fetch_optional(&mut *tx)
                .await
                .map_err(map_error("config.set_rate.sanc", "sanc_rate"))?;
                sqlx::query(
                    "INSERT INTO sanc_rate_override \
                       (powder, refine_index, percent, content_source, updated_by) \
                     VALUES ($1, $2, $3, $4, $5) \
                     ON CONFLICT (powder, refine_index) DO UPDATE SET percent = EXCLUDED.percent, \
                       content_source = EXCLUDED.content_source, \
                       updated_by = EXCLUDED.updated_by, updated_at = now()",
                )
                .bind(powder)
                .bind(index)
                .bind(i16::try_from(value).map_err(|_| invalid("sanc_rate_override", "percent"))?)
                .bind(source)
                .bind(actor)
                .execute(&mut *tx)
                .await
                .map_err(map_error(
                    "config.set_rate.sanc.write",
                    "sanc_rate_override",
                ))?;
                (
                    format!("sanc:{key}"),
                    old.map_or_else(String::new, |v| v.to_string()),
                )
            }
            "combine" => {
                let (family, item_key) = key
                    .split_once('.')
                    .ok_or_else(|| invalid("combine_value_override", "key"))?;
                let unit: Option<String> = sqlx::query_scalar(
                    "SELECT COALESCE(o.unit, c.unit) FROM combine_value c \
                     LEFT JOIN combine_value_override o \
                       ON o.family = c.family AND o.key = c.key \
                     WHERE c.family = $1 AND c.key = $2",
                )
                .bind(family)
                .bind(item_key)
                .fetch_optional(&mut *tx)
                .await
                .map_err(map_error("config.set_rate.combine", "combine_value"))?;
                let unit = unit.unwrap_or_else(|| "percent".to_string());
                if !matches!(unit.as_str(), "percent" | "coin") {
                    return Err(invalid("combine_value_override", "unit"));
                }
                let old: Option<i64> = sqlx::query_scalar(
                    "SELECT COALESCE(o.value, c.value)::bigint FROM combine_value c \
                     LEFT JOIN combine_value_override o \
                       ON o.family = c.family AND o.key = c.key \
                     WHERE c.family = $1 AND c.key = $2",
                )
                .bind(family)
                .bind(item_key)
                .fetch_optional(&mut *tx)
                .await
                .map_err(map_error("config.set_rate.combine_value", "combine_value"))?;
                sqlx::query(
                    "INSERT INTO combine_value_override \
                       (family, key, value, unit, content_source, updated_by) \
                     VALUES ($1, $2, $3, $4, $5, $6) \
                     ON CONFLICT (family, key) DO UPDATE SET value = EXCLUDED.value, \
                       unit = EXCLUDED.unit, content_source = EXCLUDED.content_source, \
                       updated_by = EXCLUDED.updated_by, updated_at = now()",
                )
                .bind(family)
                .bind(item_key)
                .bind(i32::try_from(value).map_err(|_| invalid("combine_value_override", "value"))?)
                .bind(&unit)
                .bind(source)
                .bind(actor)
                .execute(&mut *tx)
                .await
                .map_err(map_error(
                    "config.set_rate.combine.write",
                    "combine_value_override",
                ))?;
                (
                    format!("combine:{key}"),
                    old.map_or_else(String::new, |v| v.to_string()),
                )
            }
            "multiplier" => {
                if !matches!(key, "exp" | "drop" | "gold" | "skill_point") || value > 1_000_000 {
                    return Err(invalid("content_rate_override", "rate_bps"));
                }
                let old: Option<i64> = sqlx::query_scalar(
                    "SELECT rate_bps::bigint FROM content_rate_override WHERE kind = $1",
                )
                .bind(key)
                .fetch_optional(&mut *tx)
                .await
                .map_err(map_error(
                    "config.set_rate.multiplier",
                    "content_rate_override",
                ))?;
                sqlx::query(
                    "INSERT INTO content_rate_override (kind, rate_bps, content_source) \
                     VALUES ($1, $2, $3) \
                     ON CONFLICT (kind) DO UPDATE SET rate_bps = EXCLUDED.rate_bps, \
                       content_source = EXCLUDED.content_source",
                )
                .bind(key)
                .bind(
                    i32::try_from(value)
                        .map_err(|_| invalid("content_rate_override", "rate_bps"))?,
                )
                .bind(source)
                .execute(&mut *tx)
                .await
                .map_err(map_error(
                    "config.set_rate.multiplier.write",
                    "content_rate_override",
                ))?;
                (
                    format!("multiplier:{key}"),
                    old.map_or_else(String::new, |v| v.to_string()),
                )
            }
            "setting" => {
                let (section, setting_key) = key
                    .split_once('.')
                    .ok_or_else(|| invalid("game_setting", "key"))?;
                let old: Option<i64> = sqlx::query_scalar(
                    "SELECT value_int FROM game_setting WHERE section = $1 AND key = $2",
                )
                .bind(section)
                .bind(setting_key)
                .fetch_optional(&mut *tx)
                .await
                .map_err(map_error("config.set_rate.setting", "game_setting"))?;
                let affected = sqlx::query(
                    "UPDATE game_setting SET value_int = $3 WHERE section = $1 AND key = $2",
                )
                .bind(section)
                .bind(setting_key)
                .bind(value)
                .execute(&mut *tx)
                .await
                .map_err(map_error("config.set_rate.setting.write", "game_setting"))?
                .rows_affected();
                if affected == 0 {
                    return Err(DbError::NotFoundKey {
                        entity: "game_setting",
                    });
                }
                (
                    format!("setting:{key}"),
                    old.map_or_else(String::new, |v| v.to_string()),
                )
            }
            "accessory" => {
                let parts: Vec<&str> = key.split(':').collect();
                let [kind, recipe, step] = parts.as_slice() else {
                    return Err(invalid("accessory_refine_rule", "key"));
                };
                if value > 10_000 {
                    return Err(invalid("accessory_refine_rule", "rate_bps"));
                }
                let step: i16 = step
                    .parse()
                    .map_err(|_| invalid("accessory_refine_rule", "step"))?;
                let old: Option<i32> = sqlx::query_scalar(
                    "SELECT rate_bps FROM accessory_refine_rule \
                     WHERE kind = $1 AND recipe = $2 AND step_from = $3",
                )
                .bind(kind)
                .bind(recipe)
                .bind(step)
                .fetch_optional(&mut *tx)
                .await
                .map_err(map_error(
                    "config.set_rate.accessory",
                    "accessory_refine_rule",
                ))?;
                let affected = sqlx::query(
                    "UPDATE accessory_refine_rule SET rate_bps = $4, content_source = $5, \
                       updated_by = $6, updated_at = now() \
                     WHERE kind = $1 AND recipe = $2 AND step_from = $3",
                )
                .bind(kind)
                .bind(recipe)
                .bind(step)
                .bind(
                    i32::try_from(value)
                        .map_err(|_| invalid("accessory_refine_rule", "rate_bps"))?,
                )
                .bind(source)
                .bind(actor)
                .execute(&mut *tx)
                .await
                .map_err(map_error(
                    "config.set_rate.accessory.write",
                    "accessory_refine_rule",
                ))?
                .rows_affected();
                if affected == 0 {
                    return Err(DbError::NotFoundKey {
                        entity: "accessory_refine_rule",
                    });
                }
                (
                    format!("accessory:{key}"),
                    old.map_or_else(String::new, |v| v.to_string()),
                )
            }
            _ => return Err(invalid("config_rate", "domain")),
        };

        let version = self
            .bump_revision(&mut tx, actor, domain, &target, &old, &value.to_string())
            .await?;
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "config.set_rate.commit",
        })?;
        Ok(version)
    }

    /// Remove o override e devolve o valor documentado.
    pub async fn config_reset_rate(
        &self,
        domain: &str,
        key: &str,
        actor: i64,
    ) -> Result<i64, DbError> {
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "config.reset_rate",
        })?;
        let (target, old) = match domain {
            "refine" => {
                let step: i16 = key
                    .parse()
                    .map_err(|_| invalid("refine_rate_override", "step"))?;
                let old: Option<i32> =
                    sqlx::query_scalar("SELECT rate_bps FROM refine_rate_override WHERE step = $1")
                        .bind(step)
                        .fetch_optional(&mut *tx)
                        .await
                        .map_err(map_error(
                            "config.reset_rate.refine",
                            "refine_rate_override",
                        ))?;
                sqlx::query("DELETE FROM refine_rate_override WHERE step = $1")
                    .bind(step)
                    .execute(&mut *tx)
                    .await
                    .map_err(map_error(
                        "config.reset_rate.refine.write",
                        "refine_rate_override",
                    ))?;
                (format!("refine:{step}"), old)
            }
            "sanc" => {
                let (powder, index) = key
                    .split_once(':')
                    .ok_or_else(|| invalid("sanc_rate_override", "key"))?;
                let index: i16 = index
                    .parse()
                    .map_err(|_| invalid("sanc_rate_override", "refine_index"))?;
                let old: Option<i16> = sqlx::query_scalar(
                    "SELECT percent FROM sanc_rate_override \
                     WHERE powder = $1 AND refine_index = $2",
                )
                .bind(powder)
                .bind(index)
                .fetch_optional(&mut *tx)
                .await
                .map_err(map_error("config.reset_rate.sanc", "sanc_rate_override"))?;
                sqlx::query(
                    "DELETE FROM sanc_rate_override WHERE powder = $1 AND refine_index = $2",
                )
                .bind(powder)
                .bind(index)
                .execute(&mut *tx)
                .await
                .map_err(map_error(
                    "config.reset_rate.sanc.write",
                    "sanc_rate_override",
                ))?;
                (format!("sanc:{key}"), old.map(i32::from))
            }
            "combine" => {
                let (family, item_key) = key
                    .split_once('.')
                    .ok_or_else(|| invalid("combine_value_override", "key"))?;
                let old: Option<i32> = sqlx::query_scalar(
                    "SELECT value FROM combine_value_override WHERE family = $1 AND key = $2",
                )
                .bind(family)
                .bind(item_key)
                .fetch_optional(&mut *tx)
                .await
                .map_err(map_error(
                    "config.reset_rate.combine",
                    "combine_value_override",
                ))?;
                sqlx::query("DELETE FROM combine_value_override WHERE family = $1 AND key = $2")
                    .bind(family)
                    .bind(item_key)
                    .execute(&mut *tx)
                    .await
                    .map_err(map_error(
                        "config.reset_rate.combine.write",
                        "combine_value_override",
                    ))?;
                (format!("combine:{key}"), old)
            }
            "multiplier" => {
                let old: Option<i32> = sqlx::query_scalar(
                    "SELECT rate_bps FROM content_rate_override WHERE kind = $1",
                )
                .bind(key)
                .fetch_optional(&mut *tx)
                .await
                .map_err(map_error(
                    "config.reset_rate.multiplier",
                    "content_rate_override",
                ))?;
                sqlx::query("DELETE FROM content_rate_override WHERE kind = $1")
                    .bind(key)
                    .execute(&mut *tx)
                    .await
                    .map_err(map_error(
                        "config.reset_rate.multiplier.write",
                        "content_rate_override",
                    ))?;
                (format!("multiplier:{key}"), old)
            }
            // `game_setting` é a própria tabela de configuração: não existe default a
            // restaurar sem reexecutar o seed. O portal edita, não reseta.
            "setting" | "accessory" => return Err(invalid("config_rate", "reset_not_supported")),
            _ => return Err(invalid("config_rate", "domain")),
        };

        let new = if domain == "multiplier" {
            String::new()
        } else {
            "default".to_string()
        };
        let version = self
            .bump_revision(
                &mut tx,
                actor,
                domain,
                &target,
                &old.map_or_else(String::new, |v| v.to_string()),
                &new,
            )
            .await?;
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "config.reset_rate.commit",
        })?;
        Ok(version)
    }

    async fn bump_revision(
        &self,
        tx: &mut sqlx::Transaction<'_, sqlx::Postgres>,
        actor: i64,
        domain: &str,
        target: &str,
        old_value: &str,
        new_value: &str,
    ) -> Result<i64, DbError> {
        let version: i64 = sqlx::query_scalar(
            "UPDATE config_revision SET version = version + 1, updated_at = now(), \
             updated_by = $1 WHERE id = 1 RETURNING version",
        )
        .bind(actor)
        .fetch_one(&mut **tx)
        .await
        .map_err(map_error("config.revision", "config_revision"))?;
        sqlx::query(
            "INSERT INTO config_change (domain, target, old_value, new_value, actor_id) \
             VALUES ($1, $2, $3, $4, $5)",
        )
        .bind(domain)
        .bind(target)
        .bind(old_value)
        .bind(new_value)
        .bind(actor)
        .execute(&mut **tx)
        .await
        .map_err(map_error("config.change", "config_change"))?;
        Ok(version)
    }

    /// Pesos de drop por template de monstro (filtro opcional, teto de linhas).
    pub async fn config_drops(
        &self,
        template: Option<&str>,
        limit: i64,
    ) -> Result<Vec<DropConfigRow>, DbError> {
        let rows = sqlx::query(
            "SELECT d.template_name, d.item_index, COALESCE(i.name, '') AS item_name, \
             d.weight::bigint AS weight, d.min_qty::int AS min_qty, d.max_qty::int AS max_qty, \
             d.content_source \
             FROM drop_rate_override d \
             LEFT JOIN item_catalog i ON i.item_index = d.item_index \
             WHERE ($1::text IS NULL OR d.template_name = $1) \
             ORDER BY d.template_name, d.item_index LIMIT $2",
        )
        .bind(template)
        .bind(limit.clamp(1, 200))
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.drops", "drop_rate_override"))?;
        Ok(rows
            .iter()
            .map(|row| DropConfigRow {
                template_name: row.try_get("template_name").unwrap_or_default(),
                item_index: row.try_get("item_index").unwrap_or_default(),
                item_name: row.try_get("item_name").unwrap_or_default(),
                weight: row.try_get("weight").unwrap_or_default(),
                min_qty: row.try_get::<i32, _>("min_qty").unwrap_or_default() as i16,
                max_qty: row.try_get::<i32, _>("max_qty").unwrap_or_default() as i16,
                source: row.try_get("content_source").unwrap_or_default(),
            })
            .collect())
    }

    /// Grava o peso de drop de (template, item) e devolve a nova revisão.
    pub async fn config_set_drop(
        &self,
        template: &str,
        item_index: i32,
        weight: i64,
        qty: (i16, i16),
        source: &str,
        actor: i64,
    ) -> Result<i64, DbError> {
        let (min_qty, max_qty) = qty;
        if weight <= 0 || min_qty <= 0 || max_qty < min_qty {
            return Err(invalid("drop_rate_override", "weight"));
        }
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "config.set_drop",
        })?;
        let old: Option<i64> = sqlx::query_scalar(
            "SELECT weight::bigint FROM drop_rate_override \
             WHERE template_name = $1 AND item_index = $2",
        )
        .bind(template)
        .bind(item_index)
        .fetch_optional(&mut *tx)
        .await
        .map_err(map_error("config.set_drop.read", "drop_rate_override"))?;
        sqlx::query(
            "INSERT INTO drop_rate_override \
               (template_name, item_index, weight, min_qty, max_qty, content_source) \
             VALUES ($1, $2, $3, $4, $5, $6) \
             ON CONFLICT (template_name, item_index) DO UPDATE SET weight = EXCLUDED.weight, \
               min_qty = EXCLUDED.min_qty, max_qty = EXCLUDED.max_qty, \
               content_source = EXCLUDED.content_source",
        )
        .bind(template)
        .bind(item_index)
        .bind(weight)
        .bind(min_qty)
        .bind(max_qty)
        .bind(source)
        .execute(&mut *tx)
        .await
        .map_err(map_error("config.set_drop.write", "drop_rate_override"))?;
        let target = format!("{template}#{item_index}");
        let version = self
            .bump_revision(
                &mut tx,
                actor,
                "drop",
                &target,
                &old.map_or_else(String::new, |v| v.to_string()),
                &weight.to_string(),
            )
            .await?;
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "config.set_drop.commit",
        })?;
        Ok(version)
    }

    /// Remove o peso de drop editado (volta a não cair por esta tabela).
    pub async fn config_reset_drop(
        &self,
        template: &str,
        item_index: i32,
        actor: i64,
    ) -> Result<i64, DbError> {
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "config.reset_drop",
        })?;
        let old: Option<i64> = sqlx::query_scalar(
            "SELECT weight::bigint FROM drop_rate_override \
             WHERE template_name = $1 AND item_index = $2",
        )
        .bind(template)
        .bind(item_index)
        .fetch_optional(&mut *tx)
        .await
        .map_err(map_error("config.reset_drop.read", "drop_rate_override"))?;
        sqlx::query("DELETE FROM drop_rate_override WHERE template_name = $1 AND item_index = $2")
            .bind(template)
            .bind(item_index)
            .execute(&mut *tx)
            .await
            .map_err(map_error("config.reset_drop.write", "drop_rate_override"))?;
        let target = format!("{template}#{item_index}");
        let version = self
            .bump_revision(
                &mut tx,
                actor,
                "drop",
                &target,
                &old.map_or_else(String::new, |v| v.to_string()),
                "",
            )
            .await?;
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "config.reset_drop.commit",
        })?;
        Ok(version)
    }

    /// NPCs definidos no mundo (posição e visibilidade), para o snapshot do servidor.
    ///
    /// O portal edita um por id; esta leitura em lote é o que o processo de jogo usa para
    /// trocar a foto de NPCs numa recarga, sem consulta no tick.
    pub async fn config_npcs(&self, limit: i64) -> Result<Vec<NpcConfigRow>, DbError> {
        let rows = sqlx::query(
            "SELECT d.id, d.slug, d.display_name, d.template_name, t.blob, \
             d.map_id::int AS map_id, d.pos_x::int AS pos_x, \
             d.pos_y::int AS pos_y, d.enabled, d.merchant::int AS merchant \
             FROM npc_definition d \
             JOIN npc_template t ON t.template_name = d.template_name \
             ORDER BY d.id LIMIT $1",
        )
        .bind(limit.clamp(1, 100_000))
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.npcs", "npc_definition"))?;
        Ok(rows
            .iter()
            .map(|row| NpcConfigRow {
                id: row.try_get("id").unwrap_or_default(),
                slug: row.try_get("slug").unwrap_or_default(),
                display_name: row.try_get("display_name").unwrap_or_default(),
                template_name: row.try_get("template_name").unwrap_or_default(),
                template: row.try_get("blob").unwrap_or_default(),
                map_id: row.try_get::<i32, _>("map_id").unwrap_or_default() as i16,
                pos_x: row.try_get::<i32, _>("pos_x").unwrap_or_default() as i16,
                pos_y: row.try_get::<i32, _>("pos_y").unwrap_or_default() as i16,
                enabled: row.try_get("enabled").unwrap_or(false),
                merchant: row.try_get::<i32, _>("merchant").unwrap_or_default() as i16,
            })
            .collect())
    }

    /// Um NPC por id (posição e visibilidade).
    pub async fn config_npc(&self, id: i32) -> Result<Option<NpcConfigRow>, DbError> {
        let row = sqlx::query(
            "SELECT d.id, d.slug, d.display_name, d.template_name, t.blob, \
             d.map_id::int AS map_id, d.pos_x::int AS pos_x, \
             d.pos_y::int AS pos_y, d.enabled, d.merchant::int AS merchant \
             FROM npc_definition d \
             JOIN npc_template t ON t.template_name = d.template_name \
             WHERE d.id = $1",
        )
        .bind(id)
        .fetch_optional(self.pool())
        .await
        .map_err(map_error("config.npc", "npc_definition"))?;
        Ok(row.map(|row| NpcConfigRow {
            id: row.try_get("id").unwrap_or_default(),
            slug: row.try_get("slug").unwrap_or_default(),
            display_name: row.try_get("display_name").unwrap_or_default(),
            template_name: row.try_get("template_name").unwrap_or_default(),
            template: row.try_get("blob").unwrap_or_default(),
            map_id: row.try_get::<i32, _>("map_id").unwrap_or_default() as i16,
            pos_x: row.try_get::<i32, _>("pos_x").unwrap_or_default() as i16,
            pos_y: row.try_get::<i32, _>("pos_y").unwrap_or_default() as i16,
            enabled: row.try_get("enabled").unwrap_or(false),
            merchant: row.try_get::<i32, _>("merchant").unwrap_or_default() as i16,
        }))
    }

    /// Move/ativa um NPC. Recusa posição fora do espaço 0..4095 e mapa fora de 0..255.
    pub async fn config_set_npc(
        &self,
        id: i32,
        map_id: i16,
        pos_x: i16,
        pos_y: i16,
        enabled: bool,
        actor: i64,
    ) -> Result<i64, DbError> {
        if !(0..=255).contains(&map_id)
            || !(0..=4095).contains(&pos_x)
            || !(0..=4095).contains(&pos_y)
        {
            return Err(invalid("npc_definition", "position"));
        }
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "config.set_npc",
        })?;
        let old: Option<String> = sqlx::query_scalar(
            "SELECT format('map=%s x=%s y=%s enabled=%s', map_id, pos_x, pos_y, enabled) \
             FROM npc_definition WHERE id = $1",
        )
        .bind(id)
        .fetch_optional(&mut *tx)
        .await
        .map_err(map_error("config.set_npc.read", "npc_definition"))?;
        let Some(old) = old else {
            return Err(DbError::NotFound {
                entity: "npc_definition",
                id: i64::from(id),
            });
        };
        sqlx::query(
            "UPDATE npc_definition SET map_id = $2, pos_x = $3, pos_y = $4, enabled = $5 \
             WHERE id = $1",
        )
        .bind(id)
        .bind(map_id)
        .bind(pos_x)
        .bind(pos_y)
        .bind(enabled)
        .execute(&mut *tx)
        .await
        .map_err(map_error("config.set_npc.write", "npc_definition"))?;
        let new = format!("map={map_id} x={pos_x} y={pos_y} enabled={enabled}");
        let version = self
            .bump_revision(&mut tx, actor, "npc", &format!("npc:{id}"), &old, &new)
            .await?;
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "config.set_npc.commit",
        })?;
        Ok(version)
    }

    /// Rotas de teleporte (teto de linhas).
    pub async fn config_routes(&self, limit: i64) -> Result<Vec<RouteConfigRow>, DbError> {
        let rows = sqlx::query(
            "SELECT id, from_map_id::int AS from_map_id, from_x::int AS from_x, \
             from_y::int AS from_y, to_map_id::int AS to_map_id, to_x::int AS to_x, \
             to_y::int AS to_y, npc_id, cost_gold, req_level::int AS req_level, enabled \
             FROM teleport_route ORDER BY id LIMIT $1",
        )
        .bind(limit.clamp(1, 500))
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.routes", "teleport_route"))?;
        Ok(rows
            .iter()
            .map(|row| RouteConfigRow {
                id: row.try_get("id").unwrap_or_default(),
                from_map_id: row.try_get::<i32, _>("from_map_id").unwrap_or_default() as i16,
                from_x: row.try_get::<i32, _>("from_x").unwrap_or_default() as i16,
                from_y: row.try_get::<i32, _>("from_y").unwrap_or_default() as i16,
                to_map_id: row.try_get::<i32, _>("to_map_id").unwrap_or_default() as i16,
                to_x: row.try_get::<i32, _>("to_x").unwrap_or_default() as i16,
                to_y: row.try_get::<i32, _>("to_y").unwrap_or_default() as i16,
                npc_id: row.try_get("npc_id").ok(),
                cost_gold: row.try_get("cost_gold").unwrap_or_default(),
                req_level: row.try_get::<i32, _>("req_level").unwrap_or_default() as i16,
                enabled: row.try_get("enabled").unwrap_or(false),
            })
            .collect())
    }

    /// Edita custo, nível mínimo e disponibilidade de uma rota (não cria destino).
    pub async fn config_set_route(
        &self,
        id: i32,
        cost_gold: Option<i32>,
        req_level: Option<i16>,
        enabled: Option<bool>,
        actor: i64,
    ) -> Result<i64, DbError> {
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "config.set_route",
        })?;
        let old: Option<String> = sqlx::query_scalar(
            "SELECT format('cost=%s level=%s enabled=%s', cost_gold, req_level, enabled) \
             FROM teleport_route WHERE id = $1",
        )
        .bind(id)
        .fetch_optional(&mut *tx)
        .await
        .map_err(map_error("config.set_route.read", "teleport_route"))?;
        let Some(old) = old else {
            return Err(DbError::NotFound {
                entity: "teleport_route",
                id: i64::from(id),
            });
        };
        sqlx::query(
            "UPDATE teleport_route SET cost_gold = COALESCE($2, cost_gold), \
             req_level = COALESCE($3, req_level), enabled = COALESCE($4, enabled) WHERE id = $1",
        )
        .bind(id)
        .bind(cost_gold)
        .bind(req_level)
        .bind(enabled)
        .execute(&mut *tx)
        .await
        .map_err(map_error("config.set_route.write", "teleport_route"))?;
        let new = format!(
            "cost={} level={} enabled={}",
            cost_gold.map_or_else(|| "-".to_string(), |v| v.to_string()),
            req_level.map_or_else(|| "-".to_string(), |v| v.to_string()),
            enabled.map_or_else(|| "-".to_string(), |v| v.to_string())
        );
        let version = self
            .bump_revision(
                &mut tx,
                actor,
                "teleport",
                &format!("route:{id}"),
                &old,
                &new,
            )
            .await?;
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "config.set_route.commit",
        })?;
        Ok(version)
    }

    /// NPCs definidos no mundo em páginas por id (lista do portal, FASE-07-5).
    ///
    /// `after_id` é o cursor: `None` começa do primeiro. Sem o `blob` do template para a
    /// lista de 6.100 caber no teto sem pesar a resposta.
    pub async fn config_npc_page(
        &self,
        limit: i64,
        after_id: Option<i32>,
    ) -> Result<Vec<NpcListRow>, DbError> {
        let rows = sqlx::query(
            "SELECT id, slug, display_name, template_name, map_id::int AS map_id, \
             pos_x::int AS pos_x, pos_y::int AS pos_y, enabled, merchant::int AS merchant \
             FROM npc_definition \
             WHERE ($1::int IS NULL OR id > $1) \
             ORDER BY id LIMIT $2",
        )
        .bind(after_id)
        .bind(limit.clamp(1, 200))
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.npc_page", "npc_definition"))?;
        Ok(rows
            .iter()
            .map(|row| NpcListRow {
                id: row.try_get("id").unwrap_or_default(),
                slug: row.try_get("slug").unwrap_or_default(),
                display_name: row.try_get("display_name").unwrap_or_default(),
                template_name: row.try_get("template_name").unwrap_or_default(),
                map_id: row.try_get::<i32, _>("map_id").unwrap_or_default() as i16,
                pos_x: row.try_get::<i32, _>("pos_x").unwrap_or_default() as i16,
                pos_y: row.try_get::<i32, _>("pos_y").unwrap_or_default() as i16,
                enabled: row.try_get("enabled").unwrap_or(false),
                merchant: row.try_get::<i32, _>("merchant").unwrap_or_default() as i16,
            })
            .collect())
    }

    /// Templates de mob em páginas por nome (lista do portal, FASE-07-5).
    ///
    /// A fonte é `npc_template` (semeada); `mob_template_stat`/`mob_template_equip` estão
    /// vazias até a FASE-10 e não servem lista nenhuma. Cursor sobre `template_name`.
    pub async fn config_mob_templates(
        &self,
        limit: i64,
        after: Option<&str>,
    ) -> Result<Vec<MobTemplateRow>, DbError> {
        let rows = sqlx::query(
            "SELECT template_name, COALESCE(NULLIF(name_recovered, ''), template_name) AS name, \
             source_size \
             FROM npc_template \
             WHERE ($1::text IS NULL OR template_name > $1) \
             ORDER BY template_name LIMIT $2",
        )
        .bind(after)
        .bind(limit.clamp(1, 200))
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.mob_templates", "npc_template"))?;
        Ok(rows
            .iter()
            .map(|row| MobTemplateRow {
                template_name: row.try_get("template_name").unwrap_or_default(),
                name: row.try_get("name").unwrap_or_default(),
                source_size: row.try_get("source_size").unwrap_or_default(),
            })
            .collect())
    }

    /// Efeitos de item em páginas por nome (lista do portal, FASE-07-5).
    ///
    /// O código numérico do `ItemEffect.h` não foi provado (`confirmed = false`): a lista
    /// mostra o símbolo `EF_*`, o rótulo e a categoria que o seed declara, sem número.
    pub async fn config_effects(
        &self,
        limit: i64,
        after: Option<&str>,
    ) -> Result<Vec<EffectRow>, DbError> {
        let rows = sqlx::query(
            "SELECT name, label, category, description, confirmed \
             FROM item_effect \
             WHERE ($1::text IS NULL OR name > $1) \
             ORDER BY name LIMIT $2",
        )
        .bind(after)
        .bind(limit.clamp(1, 200))
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.effects", "item_effect"))?;
        Ok(rows
            .iter()
            .map(|row| EffectRow {
                name: row.try_get("name").unwrap_or_default(),
                label: row.try_get("label").unwrap_or_default(),
                category: row.try_get("category").unwrap_or_default(),
                description: row.try_get("description").unwrap_or_default(),
                confirmed: row.try_get("confirmed").unwrap_or(false),
            })
            .collect())
    }

    /// Recompensas diárias em páginas por id (lista do portal, FASE-07-6).
    ///
    /// Base semeada vazia: lista vazia é resposta válida, nunca erro.
    pub async fn config_daily_rewards(
        &self,
        limit: i64,
        after_id: Option<i32>,
    ) -> Result<Vec<DailyRewardRow>, DbError> {
        let rows = sqlx::query(
            "SELECT id, day::int AS day, item_index, gold, qty::int AS qty, account_wide \
             FROM daily_reward_item \
             WHERE ($1::int IS NULL OR id > $1) \
             ORDER BY id LIMIT $2",
        )
        .bind(after_id)
        .bind(limit.clamp(1, 200))
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.daily_rewards", "daily_reward_item"))?;
        Ok(rows
            .iter()
            .map(|row| DailyRewardRow {
                id: row.try_get("id").unwrap_or_default(),
                day: row
                    .try_get::<Option<i32>, _>("day")
                    .ok()
                    .flatten()
                    .map(|d| d as i16),
                item_index: row.try_get("item_index").ok().flatten(),
                gold: row.try_get("gold").unwrap_or_default(),
                qty: row.try_get::<i32, _>("qty").unwrap_or_default() as i16,
                account_wide: row.try_get("account_wide").unwrap_or(true),
            })
            .collect())
    }

    /// Cria (`id = None`) ou substitui (`Some`) uma recompensa diária.
    ///
    /// A validação de faixa mora no `CHECK` da tabela (`day` 1..=7, item ou ouro,
    /// `qty > 0`); violação vira `ConstraintViolation`, nunca `panic!`. Devolve a revisão.
    pub async fn config_upsert_daily_reward(
        &self,
        edit: &DailyRewardEdit,
        actor: i64,
    ) -> Result<i64, DbError> {
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "config.set_daily_reward",
        })?;
        let version =
            match edit.id {
                Some(id) => {
                    let old: Option<String> = sqlx::query_scalar(
                    "SELECT format('day=%s item=%s gold=%s qty=%s', day, item_index, gold, qty) \
                     FROM daily_reward_item WHERE id = $1",
                )
                .bind(id)
                .fetch_optional(&mut *tx)
                .await
                .map_err(map_error("config.set_daily_reward.read", "daily_reward_item"))?;
                    let Some(old) = old else {
                        return Err(DbError::NotFound {
                            entity: "daily_reward_item",
                            id: i64::from(id),
                        });
                    };
                    sqlx::query(
                        "UPDATE daily_reward_item SET day = $2, item_index = $3, gold = $4, \
                     qty = $5, account_wide = $6 WHERE id = $1",
                    )
                    .bind(id)
                    .bind(edit.day.map(i32::from))
                    .bind(edit.item_index)
                    .bind(edit.gold)
                    .bind(i32::from(edit.qty))
                    .bind(edit.account_wide)
                    .execute(&mut *tx)
                    .await
                    .map_err(map_error(
                        "config.set_daily_reward.write",
                        "daily_reward_item",
                    ))?;
                    let new = format!(
                        "day={} item={} gold={} qty={}",
                        edit.day.map_or_else(|| "*".to_string(), |d| d.to_string()),
                        edit.item_index
                            .map_or_else(|| "-".to_string(), |i| i.to_string()),
                        edit.gold,
                        edit.qty
                    );
                    self.bump_revision(&mut tx, actor, "daily", &format!("reward:{id}"), &old, &new)
                        .await?
                }
                None => {
                    let row: (i32,) = sqlx::query_as(
                        "INSERT INTO daily_reward_item (day, item_index, gold, qty, account_wide) \
                     VALUES ($1, $2, $3, $4, $5) RETURNING id",
                    )
                    .bind(edit.day.map(i32::from))
                    .bind(edit.item_index)
                    .bind(edit.gold)
                    .bind(i32::from(edit.qty))
                    .bind(edit.account_wide)
                    .fetch_one(&mut *tx)
                    .await
                    .map_err(map_error(
                        "config.set_daily_reward.write",
                        "daily_reward_item",
                    ))?;
                    self.bump_revision(
                        &mut tx,
                        actor,
                        "daily",
                        &format!("reward:{}", row.0),
                        "",
                        &format!("gold={} qty={}", edit.gold, edit.qty),
                    )
                    .await?
                }
            };
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "config.set_daily_reward.commit",
        })?;
        Ok(version)
    }

    /// Remove uma recompensa diária (volta a não existir; o seed não a recria sozinho).
    pub async fn config_delete_daily_reward(&self, id: i32, actor: i64) -> Result<i64, DbError> {
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "config.delete_daily_reward",
        })?;
        let feito = sqlx::query("DELETE FROM daily_reward_item WHERE id = $1")
            .bind(id)
            .execute(&mut *tx)
            .await
            .map_err(map_error(
                "config.delete_daily_reward.write",
                "daily_reward_item",
            ))?;
        if feito.rows_affected() == 0 {
            return Err(DbError::NotFound {
                entity: "daily_reward_item",
                id: i64::from(id),
            });
        }
        let version = self
            .bump_revision(
                &mut tx,
                actor,
                "daily",
                &format!("reward:{id}"),
                "existe",
                "",
            )
            .await?;
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "config.delete_daily_reward.commit",
        })?;
        Ok(version)
    }

    /// Loja de donate em páginas por id (lista do portal, FASE-07-6).
    pub async fn config_donate_items(
        &self,
        limit: i64,
        after_id: Option<i32>,
    ) -> Result<Vec<DonateItemRow>, DbError> {
        let rows = sqlx::query(
            "SELECT id, item_index, price_credits, duration_days::int AS duration_days, enabled \
             FROM donate_shop_item \
             WHERE ($1::int IS NULL OR id > $1) \
             ORDER BY id LIMIT $2",
        )
        .bind(after_id)
        .bind(limit.clamp(1, 200))
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.donate_items", "donate_shop_item"))?;
        Ok(rows
            .iter()
            .map(|row| DonateItemRow {
                id: row.try_get("id").unwrap_or_default(),
                item_index: row.try_get("item_index").unwrap_or_default(),
                price_credits: row.try_get("price_credits").unwrap_or_default(),
                duration_days: row
                    .try_get::<Option<i32>, _>("duration_days")
                    .ok()
                    .flatten()
                    .map(|d| d as i16),
                enabled: row.try_get("enabled").unwrap_or(false),
            })
            .collect())
    }

    /// Cria (`id = None`) ou substitui (`Some`) um item da loja de donate.
    pub async fn config_upsert_donate_item(
        &self,
        id: Option<i32>,
        item_index: i32,
        price_credits: i32,
        duration_days: Option<i16>,
        enabled: bool,
        actor: i64,
    ) -> Result<i64, DbError> {
        if item_index <= 0 || price_credits <= 0 {
            return Err(invalid("donate_shop_item", "item_ou_preco"));
        }
        if duration_days.is_some_and(|d| d <= 0) {
            return Err(invalid("donate_shop_item", "duration_days"));
        }
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "config.set_donate_item",
        })?;
        let alvo = match id {
            Some(id) => {
                let existe: Option<i32> =
                    sqlx::query_scalar("SELECT id FROM donate_shop_item WHERE id = $1")
                        .bind(id)
                        .fetch_optional(&mut *tx)
                        .await
                        .map_err(map_error("config.set_donate_item.read", "donate_shop_item"))?;
                if existe.is_none() {
                    return Err(DbError::NotFound {
                        entity: "donate_shop_item",
                        id: i64::from(id),
                    });
                }
                sqlx::query(
                    "UPDATE donate_shop_item SET item_index = $2, price_credits = $3, \
                     duration_days = $4, enabled = $5 WHERE id = $1",
                )
                .bind(id)
                .bind(item_index)
                .bind(price_credits)
                .bind(duration_days.map(i32::from))
                .bind(enabled)
                .execute(&mut *tx)
                .await
                .map_err(map_error(
                    "config.set_donate_item.write",
                    "donate_shop_item",
                ))?;
                format!("shop:{id}")
            }
            None => {
                let row: (i32,) = sqlx::query_as(
                    "INSERT INTO donate_shop_item (item_index, price_credits, duration_days, enabled) \
                     VALUES ($1, $2, $3, $4) RETURNING id",
                )
                .bind(item_index)
                .bind(price_credits)
                .bind(duration_days.map(i32::from))
                .bind(enabled)
                .fetch_one(&mut *tx)
                .await
                .map_err(map_error("config.set_donate_item.write", "donate_shop_item"))?;
                format!("shop:{}", row.0)
            }
        };
        let version = self
            .bump_revision(
                &mut tx,
                actor,
                "donate",
                &alvo,
                "",
                &format!("item={item_index}"),
            )
            .await?;
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "config.set_donate_item.commit",
        })?;
        Ok(version)
    }

    /// Liga/desliga um item da loja sem tocar no preço.
    pub async fn config_set_donate_item_enabled(
        &self,
        id: i32,
        enabled: bool,
        actor: i64,
    ) -> Result<i64, DbError> {
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "config.set_donate_item_enabled",
        })?;
        let feito = sqlx::query("UPDATE donate_shop_item SET enabled = $2 WHERE id = $1")
            .bind(id)
            .bind(enabled)
            .execute(&mut *tx)
            .await
            .map_err(map_error(
                "config.set_donate_item_enabled.write",
                "donate_shop_item",
            ))?;
        if feito.rows_affected() == 0 {
            return Err(DbError::NotFound {
                entity: "donate_shop_item",
                id: i64::from(id),
            });
        }
        let version = self
            .bump_revision(
                &mut tx,
                actor,
                "donate",
                &format!("shop:{id}"),
                &format!("enabled={}", !enabled),
                &format!("enabled={enabled}"),
            )
            .await?;
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "config.set_donate_item_enabled.commit",
        })?;
        Ok(version)
    }

    /// Remove um item da loja (não apaga compra já feita; só some da vitrine).
    pub async fn config_delete_donate_item(&self, id: i32, actor: i64) -> Result<i64, DbError> {
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "config.delete_donate_item",
        })?;
        let feito = sqlx::query("DELETE FROM donate_shop_item WHERE id = $1")
            .bind(id)
            .execute(&mut *tx)
            .await
            .map_err(map_error(
                "config.delete_donate_item.write",
                "donate_shop_item",
            ))?;
        if feito.rows_affected() == 0 {
            return Err(DbError::NotFound {
                entity: "donate_shop_item",
                id: i64::from(id),
            });
        }
        let version = self
            .bump_revision(
                &mut tx,
                actor,
                "donate",
                &format!("shop:{id}"),
                "existe",
                "",
            )
            .await?;
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "config.delete_donate_item.commit",
        })?;
        Ok(version)
    }

    /// Eventos de mundo com estado (lista do portal, FASE-07-6).
    pub async fn config_world_events(&self) -> Result<Vec<WorldEventRow>, DbError> {
        let rows = sqlx::query(
            "SELECT code, name, kind, enabled, requires_city, description \
             FROM world_event ORDER BY code",
        )
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.world_events", "world_event"))?;
        Ok(rows
            .iter()
            .map(|row| WorldEventRow {
                code: row.try_get("code").unwrap_or_default(),
                name: row.try_get("name").unwrap_or_default(),
                kind: row.try_get("kind").unwrap_or_default(),
                enabled: row.try_get("enabled").unwrap_or(false),
                requires_city: row.try_get("requires_city").unwrap_or(false),
                description: row.try_get("description").unwrap_or_default(),
            })
            .collect())
    }

    /// Horários de todos os eventos (o portal monta a grade a partir daqui).
    pub async fn config_event_schedules(&self) -> Result<Vec<EventScheduleRow>, DbError> {
        let rows = sqlx::query(
            "SELECT id, event_code, weekday::int AS weekday, \
             to_char(start_time, 'HH24:MI:SS') AS start_time, \
             duration_minutes::int AS duration_minutes, timezone, \
             every_n_weeks::int AS every_n_weeks, enabled \
             FROM event_schedule ORDER BY event_code, weekday, start_time",
        )
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.event_schedules", "event_schedule"))?;
        Ok(rows
            .iter()
            .map(|row| EventScheduleRow {
                id: row.try_get("id").unwrap_or_default(),
                event_code: row.try_get("event_code").unwrap_or_default(),
                weekday: row.try_get::<i32, _>("weekday").unwrap_or_default() as i16,
                start_time: row.try_get("start_time").unwrap_or_default(),
                duration_minutes: row
                    .try_get::<i32, _>("duration_minutes")
                    .unwrap_or_default() as i16,
                timezone: row.try_get("timezone").unwrap_or_default(),
                every_n_weeks: row.try_get::<i32, _>("every_n_weeks").unwrap_or_default() as i16,
                enabled: row.try_get("enabled").unwrap_or(false),
            })
            .collect())
    }

    /// Liga/desliga um evento. Efeito só no próximo leitor do dado (boot/tick que lê a
    /// tabela) — nenhum push para o Hot Path.
    pub async fn config_set_world_event_enabled(
        &self,
        code: &str,
        enabled: bool,
        actor: i64,
    ) -> Result<i64, DbError> {
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "config.set_world_event",
        })?;
        let feito = sqlx::query("UPDATE world_event SET enabled = $2 WHERE code = $1")
            .bind(code.trim())
            .bind(enabled)
            .execute(&mut *tx)
            .await
            .map_err(map_error("config.set_world_event.write", "world_event"))?;
        if feito.rows_affected() == 0 {
            return Err(DbError::NotFoundKey {
                entity: "world_event",
            });
        }
        let version = self
            .bump_revision(
                &mut tx,
                actor,
                "event",
                &format!("event:{}", code.trim()),
                &format!("enabled={}", !enabled),
                &format!("enabled={enabled}"),
            )
            .await?;
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "config.set_world_event.commit",
        })?;
        Ok(version)
    }

    /// Contas em páginas por id, sem hash/PIN/segredo (lista do portal, FASE-07-6).
    pub async fn config_accounts(
        &self,
        limit: i64,
        after_id: Option<i64>,
    ) -> Result<Vec<AccountListRow>, DbError> {
        let rows = sqlx::query(
            "SELECT id, login, role FROM accounts \
             WHERE ($1::bigint IS NULL OR id > $1) \
             ORDER BY id LIMIT $2",
        )
        .bind(after_id)
        .bind(limit.clamp(1, 200))
        .fetch_all(self.pool())
        .await
        .map_err(map_error("config.accounts", "accounts"))?;
        Ok(rows
            .iter()
            .map(|row| AccountListRow {
                id: row.try_get("id").unwrap_or_default(),
                login: row.try_get("login").unwrap_or_default(),
                role: row.try_get("role").unwrap_or_default(),
            })
            .collect())
    }

    /// Uma conta por id, sem hash/PIN/segredo.
    pub async fn config_account(&self, id: i64) -> Result<Option<AccountListRow>, DbError> {
        let row = sqlx::query("SELECT id, login, role FROM accounts WHERE id = $1")
            .bind(id)
            .fetch_optional(self.pool())
            .await
            .map_err(map_error("config.account", "accounts"))?;
        Ok(row.map(|row| AccountListRow {
            id: row.try_get("id").unwrap_or_default(),
            login: row.try_get("login").unwrap_or_default(),
            role: row.try_get("role").unwrap_or_default(),
        }))
    }

    /// Credita saldo de donate na carteira da conta, com idempotência por chave.
    ///
    /// Uma transação só: a linha do razão (`donate_credit`, migration 0042) sai junto com
    /// o crédito em `account_wallet` (teto `i32::MAX`, como o campo do fio). Repetir a
    /// mesma chave devolve o mesmo razão com `replay = true` e **não** credita duas vezes.
    /// É saldo de **conta** (Cold Path), nunca escrita em personagem vivo.
    pub async fn config_credit_donate(
        &self,
        account_id: i64,
        credits: i64,
        key: &str,
        actor: i64,
    ) -> Result<(i64, bool), DbError> {
        if credits <= 0 {
            return Err(invalid("donate_credit", "credits"));
        }
        let existe: Option<i64> = sqlx::query_scalar("SELECT id FROM accounts WHERE id = $1")
            .bind(account_id)
            .fetch_optional(self.pool())
            .await
            .map_err(map_error("config.credit_donate.read", "accounts"))?;
        if existe.is_none() {
            return Err(DbError::NotFound {
                entity: "accounts",
                id: account_id,
            });
        }
        let mut tx = self.begin().await.map_err(|_e| DbError::Transport {
            op: "config.credit_donate",
        })?;
        let inserido: Option<(i64,)> = sqlx::query_as(
            "INSERT INTO donate_credit (idempotency_key, account_id, credits, created_by) \
             VALUES ($1, $2, $3, $4) ON CONFLICT (idempotency_key) DO NOTHING RETURNING id",
        )
        .bind(key.trim())
        .bind(account_id)
        .bind(credits)
        .bind(actor)
        .fetch_optional(&mut *tx)
        .await
        .map_err(map_error("config.credit_donate.write", "donate_credit"))?;
        let (razao, replay) = match inserido {
            Some((id,)) => {
                sqlx::query(
                    "INSERT INTO account_wallet (account_id, coin) VALUES ($1, $2) \
                     ON CONFLICT (account_id) DO UPDATE SET \
                       coin = LEAST(account_wallet.coin + EXCLUDED.coin, 2147483647)",
                )
                .bind(account_id)
                .bind(credits)
                .execute(&mut *tx)
                .await
                .map_err(map_error("config.credit_donate.write", "account_wallet"))?;
                (id, false)
            }
            None => {
                let row: Option<(i64,)> =
                    sqlx::query_as("SELECT id FROM donate_credit WHERE idempotency_key = $1")
                        .bind(key.trim())
                        .fetch_optional(&mut *tx)
                        .await
                        .map_err(map_error("config.credit_donate.read", "donate_credit"))?;
                let Some((id,)) = row else {
                    return Err(DbError::Transport {
                        op: "config.credit_donate.replay",
                    });
                };
                (id, true)
            }
        };
        tx.commit().await.map_err(|_e| DbError::Transport {
            op: "config.credit_donate.commit",
        })?;
        Ok((razao, replay))
    }

    /// Versão atual da configuração (o servidor compara antes de recarregar).
    pub async fn config_revision(&self) -> Result<i64, DbError> {
        sqlx::query_scalar("SELECT version FROM config_revision WHERE id = 1")
            .fetch_one(self.pool())
            .await
            .map_err(map_error("config.revision.read", "config_revision"))
    }
}
