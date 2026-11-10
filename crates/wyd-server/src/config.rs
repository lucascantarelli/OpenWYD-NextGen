//! Configuração viva do servidor de jogo (FASE-07-6): snapshot em memória, revisão e recarga.
//!
//! O portal grava `refine_rate_override`, `sanc_rate_override`, `combine_value_override`,
//! `content_rate_override`, `drop_rate_override`, `npc_definition`, `teleport_route` e
//! incrementa `config_revision` na mesma transação. Este módulo é o lado do **processo de
//! jogo**: lê a revisão **fora** do tick, monta um [`GameConfig`] imutável e troca o
//! `Arc` publicado de uma vez.
//!
//! Invariantes:
//!
//! - **Sem I/O no tick**: a thread `wyd-config` faz o `SELECT`; o tick só clona um `Arc`.
//! - **Troca atômica**: leitores veem a foto antiga ou a nova inteira, nunca mistura de
//!   revisões (o snapshot é imutável e o ponteiro é trocado com write-lock curto).
//! - **Falha não derruba nem zera**: erro de banco, revisão inválida ou snapshot incompleto
//!   mantém o último snapshot bom; o motivo fica em log e em [`LiveConfig::last_error`].
//! - **Sem default silencioso**: o snapshot exige as 15 taxas de refino, ao menos uma célula
//!   de SancRate e um valor de composição. Snapshot incompleto é recusado, não completado
//!   com zero.
//!
//! O resolvedor ([`GameConfig`]) é a única porta que o gameplay usa para taxa de refino,
//! SancRate, composição, multiplicadores, drops, NPC e teleporte. Nenhum número de economia
//! é constante no handler.

use std::collections::BTreeMap;
use std::fmt;
use std::sync::atomic::{AtomicBool, AtomicI64, AtomicU64, Ordering};
use std::sync::{Arc, Mutex, RwLock};
use std::thread::{self, JoinHandle};
use std::time::Duration;

use wyd_db::{
    AccessoryRuleRow, ConfigRateRow, DropConfigRow, NpcConfigRow, PgBackend, RouteConfigRow,
};
use wyd_sim::formula::refine::{powder_success_bps, Powder, RefinePolicy};

/// Multiplicador neutro (`10000` bps = 100%) quando a tabela não tem override.
pub const DEFAULT_MULTIPLIER_BPS: u32 = 10_000;
/// Variável que controla o intervalo de observação da revisão.
pub const ENV_CONFIG_POLL_MS: &str = "WYD_CONFIG_POLL_MS";
/// Intervalo default entre leituras de `config_revision` (1 s).
pub const DEFAULT_CONFIG_POLL_MS: u64 = 1_000;
/// Menor intervalo aceito (evita laço de banco).
pub const MIN_CONFIG_POLL_MS: u64 = 50;
/// Maior intervalo aceito (uma leitura por minuto).
pub const MAX_CONFIG_POLL_MS: u64 = 60_000;
/// Teto de linhas lidas de drops/NPCs/rotas por snapshot.
const DROP_LIMIT: i64 = 200;
const NPC_LIMIT: i64 = 100_000;
const ROUTE_LIMIT: i64 = 500;

/// Peso de drop editado, com quantidade sorteada.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct DropRule {
    pub weight: i64,
    pub min_qty: i16,
    pub max_qty: i16,
}

/// NPC definido no mundo: posição/visibilidade **e** o que o `STRUCT_MOB` do template diz.
///
/// O spawn, o diálogo e a recompensa de abate saem daqui — nunca de constante no core. Os
/// campos vitais são normalizados para o contrato de [`Stats`]: nível 0 e `max_hp` 0 não
/// existem em entidade viva, então viram 1 (piso estrutural, não valor de balanceamento).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NpcRule {
    pub id: i32,
    pub slug: String,
    pub display_name: String,
    /// Nome do registro em `npc_template` (origem do `STRUCT_MOB`).
    pub template_name: String,
    /// Nome do template (`STRUCT_MOB.Name`), usado quando `display_name` está vazio.
    pub name: String,
    /// EXP que o abate concede (`Mob.Exp` do template); 0 = não é criatura de combate.
    pub exp: i64,
    pub level: u16,
    pub hp: u32,
    pub max_hp: u32,
    pub attack: u32,
    pub defense: u32,
    /// `true` quando o template tem EXP e a definição não é de comerciante.
    pub monster: bool,
    pub map_id: i16,
    pub pos_x: i16,
    pub pos_y: i16,
    pub enabled: bool,
    pub merchant: i16,
    /// `STRUCT_MOB.Equip[i].sIndex` do template — a aparência que o cliente desenha.
    pub equip: [u16; 16],
    /// `STRUCT_MOB.CurrentScore.AttackRun`: velocidade da animação de andar no cliente.
    pub attack_run: u8,
    /// `STRUCT_MOB.CurrentScore.Direction`: direção inicial da criatura.
    pub direction: u8,
}

/// Política de uma rota de teleporte.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct RouteRule {
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

/// Regra de refino de acessório carregada no snapshot.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AccessoryRule {
    pub kind: String,
    pub recipe: String,
    pub step_from: u8,
    pub step_to: u8,
    pub rate_bps: u16,
    pub fail_mode: String,
    pub fail_bonus_bps: u16,
    pub gold_cost: i64,
    pub material_name: String,
    pub material_qty: i16,
    pub source: String,
}

/// Motivo de recusa de um snapshot — tipado, sem vazar SQL/DSN.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ConfigError {
    /// `config_revision.version` fora de `>= 1`.
    Revision(i64),
    /// Família obrigatória veio vazia.
    Empty(&'static str),
    /// Passo/célula obrigatória ausente (não é completada com zero).
    Missing { family: &'static str, key: String },
    /// Valor fora da faixa da família.
    OutOfRange {
        family: &'static str,
        key: String,
        value: i64,
    },
    /// Chave que o loader não sabe interpretar.
    InvalidKey { family: &'static str, key: String },
    /// O `STRUCT_MOB` de `npc_template` não decodificou em 816 B.
    BadTemplate { npc_id: i32 },
    /// Falha de leitura (banco indisponível, migração ausente…).
    Source(String),
}

impl fmt::Display for ConfigError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Revision(version) => write!(f, "config_revision inválida: {version}"),
            Self::Empty(family) => write!(f, "snapshot sem linhas de {family}"),
            Self::Missing { family, key } => {
                write!(f, "snapshot sem {family} obrigatório: {key}")
            }
            Self::OutOfRange { family, key, value } => {
                write!(f, "{family}:{key} fora da faixa ({value})")
            }
            Self::InvalidKey { family, key } => write!(f, "chave inválida em {family}: {key}"),
            Self::BadTemplate { npc_id } => {
                write!(f, "NPC {npc_id} com STRUCT_MOB inválido no npc_template")
            }
            Self::Source(reason) => write!(f, "leitura de configuração falhou: {reason}"),
        }
    }
}

impl std::error::Error for ConfigError {}

/// Foto imutável de toda a configuração que o jogo lê.
///
/// Quem chama nunca recebe `&mut`: uma revisão publicada não muda; a próxima vira outro
/// `Arc<GameConfig>`.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct GameConfig {
    revision: i64,
    refine: BTreeMap<u8, u16>,
    sanc: BTreeMap<(String, u8), u8>,
    combine: BTreeMap<(String, String), i32>,
    multipliers: BTreeMap<String, u32>,
    settings: BTreeMap<String, i64>,
    drops: BTreeMap<(String, i32), DropRule>,
    npcs: BTreeMap<i32, NpcRule>,
    routes: BTreeMap<i32, RouteRule>,
    accessories: Vec<AccessoryRule>,
}

impl GameConfig {
    /// Monta e valida o snapshot a partir das linhas do banco.
    ///
    /// A ordem não importa: mapas ordenados dão uma foto determinística (bom para log e
    /// teste). Qualquer inconsistência recusa a foto inteira — nada de completar taxa zero.
    pub fn from_rows(
        revision: i64,
        rates: &[ConfigRateRow],
        accessories: Vec<AccessoryRuleRow>,
        drops: Vec<DropConfigRow>,
        npcs: Vec<NpcConfigRow>,
        routes: Vec<RouteConfigRow>,
    ) -> Result<Self, ConfigError> {
        if revision < 1 {
            return Err(ConfigError::Revision(revision));
        }

        let mut refine = BTreeMap::new();
        let mut sanc = BTreeMap::new();
        let mut combine = BTreeMap::new();
        let mut multipliers = BTreeMap::new();
        let mut settings = BTreeMap::new();

        for row in rates {
            match row.domain.as_str() {
                "refine" => {
                    let step: u8 = row.key.parse().map_err(|_| ConfigError::InvalidKey {
                        family: "refine",
                        key: row.key.clone(),
                    })?;
                    if !(1..=15).contains(&step) {
                        return Err(ConfigError::OutOfRange {
                            family: "refine",
                            key: row.key.clone(),
                            value: i64::from(step),
                        });
                    }
                    if !(0..=10_000).contains(&row.value) {
                        return Err(ConfigError::OutOfRange {
                            family: "refine",
                            key: row.key.clone(),
                            value: row.value,
                        });
                    }
                    refine.insert(step, u16::try_from(row.value).unwrap_or(10_000));
                }
                "sanc" => {
                    let (powder, index) =
                        row.key
                            .split_once(':')
                            .ok_or_else(|| ConfigError::InvalidKey {
                                family: "sanc",
                                key: row.key.clone(),
                            })?;
                    if !matches!(powder, "PO" | "PL" | "AMAGO") {
                        return Err(ConfigError::InvalidKey {
                            family: "sanc",
                            key: row.key.clone(),
                        });
                    }
                    let index: u8 = index.parse().map_err(|_| ConfigError::InvalidKey {
                        family: "sanc",
                        key: row.key.clone(),
                    })?;
                    if !(0..=15).contains(&index) || !(0..=100).contains(&row.value) {
                        return Err(ConfigError::OutOfRange {
                            family: "sanc",
                            key: row.key.clone(),
                            value: row.value,
                        });
                    }
                    sanc.insert(
                        (powder.to_string(), index),
                        u8::try_from(row.value).unwrap_or(0),
                    );
                }
                "combine" => {
                    let (family, key) =
                        row.key
                            .split_once('.')
                            .ok_or_else(|| ConfigError::InvalidKey {
                                family: "combine",
                                key: row.key.clone(),
                            })?;
                    if family.is_empty() || key.is_empty() || row.value < 0 {
                        return Err(ConfigError::OutOfRange {
                            family: "combine",
                            key: row.key.clone(),
                            value: row.value,
                        });
                    }
                    combine.insert(
                        (family.to_string(), key.to_string()),
                        i32::try_from(row.value).unwrap_or(i32::MAX),
                    );
                }
                "multiplier" => {
                    if !matches!(row.key.as_str(), "exp" | "drop" | "gold" | "skill_point")
                        || !(0..=1_000_000).contains(&row.value)
                    {
                        return Err(ConfigError::OutOfRange {
                            family: "multiplier",
                            key: row.key.clone(),
                            value: row.value,
                        });
                    }
                    multipliers.insert(
                        row.key.clone(),
                        u32::try_from(row.value).unwrap_or(DEFAULT_MULTIPLIER_BPS),
                    );
                }
                "setting" => {
                    let (section, key) =
                        row.key
                            .split_once('.')
                            .ok_or_else(|| ConfigError::InvalidKey {
                                family: "setting",
                                key: row.key.clone(),
                            })?;
                    if section.is_empty() || key.is_empty() {
                        return Err(ConfigError::InvalidKey {
                            family: "setting",
                            key: row.key.clone(),
                        });
                    }
                    settings.insert(row.key.clone(), row.value);
                }
                // A regra de acessório vem na lista própria (`accessories`).
                "accessory" => {}
                other => {
                    return Err(ConfigError::InvalidKey {
                        family: "rate",
                        key: format!("{other}:{}", row.key),
                    })
                }
            }
        }

        for step in 1..=15u8 {
            if !refine.contains_key(&step) {
                return Err(ConfigError::Missing {
                    family: "refine",
                    key: step.to_string(),
                });
            }
        }
        if sanc.is_empty() {
            return Err(ConfigError::Empty("sanc"));
        }
        if combine.is_empty() {
            return Err(ConfigError::Empty("combine"));
        }

        let mut accessory_rules = Vec::with_capacity(accessories.len());
        for row in accessories {
            if !matches!(
                row.kind.as_str(),
                "earring" | "necklace" | "belt" | "amulet" | "ankh" | "spirit_stone"
            ) || row.recipe.is_empty()
                || !(0..=14).contains(&row.step_from)
                || row.step_to != row.step_from + 1
                || !(0..=10_000).contains(&row.rate_bps)
                || !(0..=10_000).contains(&row.fail_bonus_bps)
            {
                return Err(ConfigError::OutOfRange {
                    family: "accessory",
                    key: format!("{}:{}:{}", row.kind, row.recipe, row.step_from),
                    value: i64::from(row.rate_bps),
                });
            }
            accessory_rules.push(AccessoryRule {
                kind: row.kind,
                recipe: row.recipe,
                step_from: u8::try_from(row.step_from).unwrap_or(0),
                step_to: u8::try_from(row.step_to).unwrap_or(0),
                rate_bps: u16::try_from(row.rate_bps).unwrap_or(10_000),
                fail_mode: row.fail_mode,
                fail_bonus_bps: u16::try_from(row.fail_bonus_bps).unwrap_or(0),
                gold_cost: row.gold_cost,
                material_name: row.material_name,
                material_qty: row.material_qty,
                source: row.source,
            });
        }

        let mut drop_rules = BTreeMap::new();
        for row in drops {
            if row.weight <= 0 || row.min_qty <= 0 || row.max_qty < row.min_qty {
                return Err(ConfigError::OutOfRange {
                    family: "drop",
                    key: format!("{}#{}", row.template_name, row.item_index),
                    value: row.weight,
                });
            }
            drop_rules.insert(
                (row.template_name, row.item_index),
                DropRule {
                    weight: row.weight,
                    min_qty: row.min_qty,
                    max_qty: row.max_qty,
                },
            );
        }

        let mut npc_rules = BTreeMap::new();
        for row in npcs {
            let mob = crate::personagens::Mob::decode(&row.template)
                .map_err(|_| ConfigError::BadTemplate { npc_id: row.id })?;
            let score = mob.current_score;
            let exp = mob.exp.max(0);
            let max_hp = u32::try_from(score.max_hp).unwrap_or(0).max(1);
            let hp = u32::try_from(score.hp).unwrap_or(0).min(max_hp);
            let level = u16::try_from(score.level).unwrap_or(1).max(1);
            npc_rules.insert(
                row.id,
                NpcRule {
                    id: row.id,
                    slug: row.slug,
                    display_name: row.display_name,
                    template_name: row.template_name,
                    name: template_name(&mob.name),
                    exp,
                    level,
                    hp,
                    max_hp,
                    attack: u32::try_from(score.damage).unwrap_or(0),
                    defense: u32::try_from(score.ac).unwrap_or(0),
                    monster: exp > 0 && row.merchant == 0,
                    map_id: row.map_id,
                    pos_x: row.pos_x,
                    pos_y: row.pos_y,
                    enabled: row.enabled,
                    merchant: row.merchant,
                    equip: core::array::from_fn(|indice| {
                        u16::try_from(mob.equip[indice].index).unwrap_or(0)
                    }),
                    attack_run: score.attack_run,
                    direction: score.direction,
                },
            );
        }

        let mut route_rules = BTreeMap::new();
        for row in routes {
            route_rules.insert(
                row.id,
                RouteRule {
                    id: row.id,
                    from_map_id: row.from_map_id,
                    from_x: row.from_x,
                    from_y: row.from_y,
                    to_map_id: row.to_map_id,
                    to_x: row.to_x,
                    to_y: row.to_y,
                    npc_id: row.npc_id,
                    cost_gold: row.cost_gold,
                    req_level: row.req_level,
                    enabled: row.enabled,
                },
            );
        }

        Ok(Self {
            revision,
            refine,
            sanc,
            combine,
            multipliers,
            settings,
            drops: drop_rules,
            npcs: npc_rules,
            routes: route_rules,
            accessories: accessory_rules,
        })
    }

    /// Revisão publicada por esta foto.
    #[must_use]
    pub fn revision(&self) -> i64 {
        self.revision
    }

    /// Taxa de sucesso do passo `step` (+0→+1 é `1`), em basis points.
    #[must_use]
    pub fn refine_bps(&self, step: u8) -> Option<u16> {
        self.refine.get(&step).copied()
    }

    /// Célula de SancRate (`powder`, `refine_index`), em pontos percentuais.
    #[must_use]
    pub fn sanc_percent(&self, powder: Powder, index: u8) -> Option<u8> {
        self.sanc.get(&(powder.code().to_string(), index)).copied()
    }

    /// Taxa efetiva do refino com pó, já com a piedade e o passo especial.
    ///
    /// A tabela do legado é lida em `nível + 1`; célula ausente devolve `None` (não vira
    /// zero silencioso). A fórmula é a de [`powder_success_bps`], não uma cópia local.
    #[must_use]
    pub fn powder_rate_bps(&self, powder: Powder, level: u8, pity: u8) -> Option<u16> {
        let table = self.sanc_percent(powder, level.checked_add(1)?)?;
        powder_success_bps(RefinePolicy::LEGACY, powder, level, pity, table)
    }

    /// Valor de composição (`combine_value`).
    #[must_use]
    pub fn combine_value(&self, family: &str, key: &str) -> Option<i32> {
        self.combine
            .get(&(family.to_string(), key.to_string()))
            .copied()
    }

    /// Multiplicador de conteúdo em bps (`10000` = neutro).
    ///
    /// Ausência de linha é o default documentado (100%), não falha de leitura: a falha de
    /// leitura mantém o snapshot anterior inteiro.
    #[must_use]
    pub fn multiplier_bps(&self, kind: &str) -> u32 {
        self.multipliers
            .get(kind)
            .copied()
            .unwrap_or(DEFAULT_MULTIPLIER_BPS)
    }

    /// Aplica o multiplicador `kind` a um valor base, saturando no teto de `u64`.
    #[must_use]
    pub fn apply_multiplier(&self, kind: &str, base: u64) -> u64 {
        let produto = u128::from(base) * u128::from(self.multiplier_bps(kind));
        u64::try_from(produto / u128::from(DEFAULT_MULTIPLIER_BPS)).unwrap_or(u64::MAX)
    }
    /// Peso de drop editado para (template, item), se houver override.
    #[must_use]
    pub fn drop_rule(&self, template: &str, item_index: i32) -> Option<DropRule> {
        self.drops.get(&(template.to_string(), item_index)).copied()
    }

    /// Tabela efetiva de drop de um template, ordenada por item.
    ///
    /// Vazia significa que o template não tem nenhuma linha — o resolvedor de loot trata
    /// isso como erro explícito, não como "não dropa nada" silencioso (ver [`crate::gameplay`]).
    #[must_use]
    pub fn drops_for(&self, template: &str) -> Vec<(i32, DropRule)> {
        self.drops
            .iter()
            .filter(|((nome, _), _)| nome == template)
            .map(|((_, item), rule)| (*item, *rule))
            .collect()
    }

    /// Chance efetiva do peso no sorteio de 1..1.000.000, em bps.
    ///
    /// `None` quando o item não tem override; peso 500.000 → 5.000 bps (50%).
    #[must_use]
    pub fn drop_chance_bps(&self, template: &str, item_index: i32) -> Option<u32> {
        let rule = self.drop_rule(template, item_index)?;
        let chance = u128::try_from(rule.weight).unwrap_or(0) * 10_000 / 1_000_000;
        Some(u32::try_from(chance).unwrap_or(10_000).min(10_000))
    }

    /// NPC carregado (posição/visibilidade/template).
    #[must_use]
    pub fn npc(&self, id: i32) -> Option<&NpcRule> {
        self.npcs.get(&id)
    }

    /// Todos os NPCs carregados, em ordem de id (foto determinística).
    pub fn npcs(&self) -> impl Iterator<Item = &NpcRule> {
        self.npcs.values()
    }

    /// NPCs habilitados de um mapa — o conjunto que o mundo daquele mapa materializa.
    pub fn npcs_on_map(&self, map_id: i16) -> impl Iterator<Item = &NpcRule> {
        self.npcs
            .values()
            .filter(move |npc| npc.enabled && npc.map_id == map_id)
    }

    /// Rotas de teleporte carregadas.
    pub fn routes(&self) -> impl Iterator<Item = &RouteRule> {
        self.routes.values()
    }

    /// Uma rota por id.
    #[must_use]
    pub fn route(&self, id: i32) -> Option<&RouteRule> {
        self.routes.get(&id)
    }

    /// Regra de refino de acessório por (tipo, receita, passo de origem).
    #[must_use]
    pub fn accessory_rate_bps(&self, kind: &str, recipe: &str, step_from: u8) -> Option<u16> {
        self.accessories
            .iter()
            .find(|rule| rule.kind == kind && rule.recipe == recipe && rule.step_from == step_from)
            .map(|rule| rule.rate_bps)
    }

    /// Chave inteira de `game_setting` (formato `seção.chave`).
    #[must_use]
    pub fn setting_int(&self, section: &str, key: &str) -> Option<i64> {
        self.settings.get(&format!("{section}.{key}")).copied()
    }

    /// Quantos passos de refino a foto tem (deve ser 15).
    #[must_use]
    pub fn refine_steps(&self) -> usize {
        self.refine.len()
    }

    /// Quantas células de SancRate a foto tem.
    #[must_use]
    pub fn sanc_cells(&self) -> usize {
        self.sanc.len()
    }

    /// Quantos valores de composição a foto tem.
    #[must_use]
    pub fn combine_values(&self) -> usize {
        self.combine.len()
    }

    /// Quantos multiplicadores explícitos a foto tem.
    #[must_use]
    pub fn multiplier_rows(&self) -> usize {
        self.multipliers.len()
    }

    /// Quantos overrides de drop a foto tem.
    #[must_use]
    pub fn drop_overrides(&self) -> usize {
        self.drops.len()
    }

    /// Quantos NPCs a foto tem.
    #[must_use]
    pub fn npc_count(&self) -> usize {
        self.npcs.len()
    }

    /// Quantas rotas de teleporte a foto tem.
    #[must_use]
    pub fn route_count(&self) -> usize {
        self.routes.len()
    }

    /// Quantas regras de acessório a foto tem.
    #[must_use]
    pub fn accessory_count(&self) -> usize {
        self.accessories.len()
    }
}

/// Nome do `STRUCT_MOB` até o primeiro `NUL`.
///
/// Decodificação tolerante: parte dos templates veio de CP949 lido como ISO-8859-1. O texto
/// legível que o diálogo usa é `display_name`; este nome só preenche quando ele está vazio.
fn template_name(raw: &[u8; 16]) -> String {
    let fim = raw.iter().position(|byte| *byte == 0).unwrap_or(raw.len());
    String::from_utf8_lossy(&raw[..fim]).to_string()
}

/// Fonte de snapshot: banco real ou duplo de teste.
///
/// `revision` é a consulta barata feita a cada janela; `load` só roda quando a versão mudou.
pub trait ConfigSource: Send + Sync {
    fn revision(&self) -> Result<i64, String>;
    fn load(&self, revision: i64) -> Result<GameConfig, String>;
}

/// Fonte real: lê o pool do Postgres no runtime do processo (fora do tick).
#[derive(Clone)]
pub struct PgConfigSource {
    backend: PgBackend,
    runtime: Arc<tokio::runtime::Runtime>,
}

impl PgConfigSource {
    #[must_use]
    pub fn new(backend: PgBackend, runtime: Arc<tokio::runtime::Runtime>) -> Self {
        Self { backend, runtime }
    }

    /// Lê a revisão e o snapshot na mesma janela (usado no boot).
    pub fn load_current(&self) -> Result<GameConfig, String> {
        self.read_snapshot()
    }

    /// Monta uma foto coerente: relê a revisão no fim e repete se uma escrita commitou
    /// no meio das quatro consultas. Sem isso, taxas de uma revisão poderiam conviver com
    /// drops de outra mesmo com a troca do `Arc` sendo atômica.
    fn read_snapshot(&self) -> Result<GameConfig, String> {
        for _ in 0..5 {
            let revision = self.revision()?;
            let (rates, accessories) = self
                .runtime
                .block_on(self.backend.config_rates())
                .map_err(|error| error.to_string())?;
            let drops = self
                .runtime
                .block_on(self.backend.config_drops(None, DROP_LIMIT))
                .map_err(|error| error.to_string())?;
            let npcs = self
                .runtime
                .block_on(self.backend.config_npcs(NPC_LIMIT))
                .map_err(|error| error.to_string())?;
            let routes = self
                .runtime
                .block_on(self.backend.config_routes(ROUTE_LIMIT))
                .map_err(|error| error.to_string())?;
            if self.revision()? != revision {
                continue;
            }
            return GameConfig::from_rows(revision, &rates, accessories, drops, npcs, routes)
                .map_err(|error| error.to_string());
        }
        Err("config_revision mudou durante a leitura cinco vezes".to_string())
    }
}

impl ConfigSource for PgConfigSource {
    fn revision(&self) -> Result<i64, String> {
        self.runtime
            .block_on(self.backend.config_revision())
            .map_err(|error| error.to_string())
    }

    fn load(&self, revision: i64) -> Result<GameConfig, String> {
        let config = self.read_snapshot()?;
        if config.revision() < revision {
            return Err(format!(
                "snapshot na revisão {} para uma leitura a partir de {revision}",
                config.revision()
            ));
        }
        Ok(config)
    }
}

/// Resultado de uma tentativa de recarga.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ReloadOutcome {
    /// A revisão do banco é a mesma já publicada.
    Unchanged,
    /// Uma revisão nova foi validada e publicada.
    Applied { from: i64, to: i64 },
    /// A leitura ou a validação falhou; o snapshot anterior continua valendo.
    Failed { reason: String },
}

/// Snapshot publicado + contadores da recarga.
#[derive(Debug)]
pub struct LiveConfig {
    snapshot: RwLock<Arc<GameConfig>>,
    revision_seen: AtomicI64,
    reloads: AtomicU64,
    failures: AtomicU64,
    last_error: Mutex<Option<String>>,
}

impl LiveConfig {
    /// Publica a primeira foto (boot). Um snapshot já validado nunca é vazio.
    #[must_use]
    pub fn new(config: GameConfig) -> Self {
        let revision = config.revision();
        Self {
            snapshot: RwLock::new(Arc::new(config)),
            revision_seen: AtomicI64::new(revision),
            reloads: AtomicU64::new(0),
            failures: AtomicU64::new(0),
            last_error: Mutex::new(None),
        }
    }

    /// Foto vigente. O `Arc` garante que a revisão não muda durante o uso.
    #[must_use]
    pub fn snapshot(&self) -> Arc<GameConfig> {
        self.snapshot
            .read()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .clone()
    }

    /// Revisão publicada.
    #[must_use]
    pub fn revision(&self) -> i64 {
        self.revision_seen.load(Ordering::Acquire)
    }

    /// Recargas aplicadas desde o boot.
    #[must_use]
    pub fn reloads(&self) -> u64 {
        self.reloads.load(Ordering::Relaxed)
    }

    /// Recargas recusadas desde o boot.
    #[must_use]
    pub fn failures(&self) -> u64 {
        self.failures.load(Ordering::Relaxed)
    }

    /// Último motivo de recusa (para `/health`, log e teste).
    #[must_use]
    pub fn last_error(&self) -> Option<String> {
        self.last_error
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
            .clone()
    }

    /// Compara a revisão e, se mudou, carrega e publica a nova foto.
    ///
    /// Nenhum caminho de erro altera o snapshot: revisão ilegível, `load` falho, revisão
    /// divergente ou snapshot inválido contam em [`Self::failures`] e mantêm o anterior.
    pub fn refresh(&self, source: &dyn ConfigSource) -> ReloadOutcome {
        let current = self.revision();
        let revision = match source.revision() {
            Ok(revision) => revision,
            Err(reason) => return self.reject(reason),
        };
        if revision <= current {
            return ReloadOutcome::Unchanged;
        }
        let config = match source.load(revision) {
            Ok(config) => config,
            Err(reason) => return self.reject(reason),
        };
        let novo = config.revision();
        if novo <= current {
            return self.reject(format!(
                "snapshot na revisão {novo} não avança a revisão publicada {current}"
            ));
        }
        let mut guard = self
            .snapshot
            .write()
            .unwrap_or_else(std::sync::PoisonError::into_inner);
        *guard = Arc::new(config);
        self.revision_seen.store(novo, Ordering::Release);
        self.reloads.fetch_add(1, Ordering::Relaxed);
        println!(
            "level=info event=config_reload result=applied from={current} to={novo} \
             refine={} sanc={} combine={} drops={} npcs={} routes={} accessories={} \
             multiplier_exp={}",
            guard.refine_steps(),
            guard.sanc_cells(),
            guard.combine_values(),
            guard.drop_overrides(),
            guard.npc_count(),
            guard.route_count(),
            guard.accessory_count(),
            // Valor efetivo que passa a valer no processo: é o que o portal acabou de editar.
            guard.multiplier_bps("exp")
        );
        ReloadOutcome::Applied {
            from: current,
            to: novo,
        }
    }

    fn reject(&self, reason: String) -> ReloadOutcome {
        self.failures.fetch_add(1, Ordering::Relaxed);
        *self
            .last_error
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner) = Some(reason.clone());
        println!(
            "level=warn event=config_reload result=failed revision={} reason={reason}",
            self.revision()
        );
        ReloadOutcome::Failed { reason }
    }

    /// Texto Prometheus do snapshot e da recarga.
    #[must_use]
    pub fn render_metrics(&self) -> String {
        let config = self.snapshot();
        let mut out = String::new();
        out.push_str("# HELP wyd_config_revision Revisao de configuracao publicada no processo.\n");
        out.push_str("# TYPE wyd_config_revision gauge\n");
        out.push_str(&format!("wyd_config_revision {}\n", config.revision()));
        out.push_str("# HELP wyd_config_reloads_total Recargas de configuracao aplicadas.\n");
        out.push_str("# TYPE wyd_config_reloads_total counter\n");
        out.push_str(&format!("wyd_config_reloads_total {}\n", self.reloads()));
        out.push_str(
            "# HELP wyd_config_reload_failures_total Recargas recusadas (snapshot anterior mantido).\n",
        );
        out.push_str("# TYPE wyd_config_reload_failures_total counter\n");
        out.push_str(&format!(
            "wyd_config_reload_failures_total {}\n",
            self.failures()
        ));
        for (name, value, help) in [
            (
                "wyd_config_refine_steps",
                config.refine_steps(),
                "Passos de refino carregados.",
            ),
            (
                "wyd_config_sanc_cells",
                config.sanc_cells(),
                "Celulas de SancRate carregadas.",
            ),
            (
                "wyd_config_combine_values",
                config.combine_values(),
                "Valores de composicao carregados.",
            ),
            (
                "wyd_config_drop_overrides",
                config.drop_overrides(),
                "Pesos de drop editados.",
            ),
            ("wyd_config_npcs", config.npc_count(), "NPCs carregados."),
            (
                "wyd_config_routes",
                config.route_count(),
                "Rotas de teleporte carregadas.",
            ),
            (
                "wyd_config_accessories",
                config.accessory_count(),
                "Regras de acessorio carregadas.",
            ),
        ] {
            out.push_str(&format!(
                "# HELP {name} {help}\n# TYPE {name} gauge\n{name} {value}\n"
            ));
        }
        out
    }
}

/// Sobe a thread `wyd-config`, que só lê a revisão e troca o snapshot.
///
/// O laço dorme `interval` (limitado a [`MIN_CONFIG_POLL_MS`]..=[`MAX_CONFIG_POLL_MS`]) e
/// nunca toca no tick. Falha de banco é logada e o último snapshot bom continua servindo.
pub fn spawn_reloader(
    config: Arc<LiveConfig>,
    source: Arc<dyn ConfigSource>,
    interval: Duration,
    stop: Arc<AtomicBool>,
) -> Result<JoinHandle<()>, String> {
    let intervalo = interval.clamp(
        Duration::from_millis(MIN_CONFIG_POLL_MS),
        Duration::from_millis(MAX_CONFIG_POLL_MS),
    );
    thread::Builder::new()
        .name("wyd-config".to_string())
        .spawn(move || {
            while !stop.load(Ordering::Relaxed) {
                thread::sleep(intervalo);
                config.refresh(source.as_ref());
            }
        })
        .map_err(|error| format!("thread de configuracao: {error}"))
}

/// Intervalo de observação vindo do ambiente, validado no boot.
pub fn poll_interval_from_env() -> Result<Duration, String> {
    let raw =
        std::env::var(ENV_CONFIG_POLL_MS).unwrap_or_else(|_| DEFAULT_CONFIG_POLL_MS.to_string());
    let millis: u64 = raw.trim().parse().map_err(|_| {
        format!("{ENV_CONFIG_POLL_MS} precisa ser um número entre {MIN_CONFIG_POLL_MS} e {MAX_CONFIG_POLL_MS}")
    })?;
    if !(MIN_CONFIG_POLL_MS..=MAX_CONFIG_POLL_MS).contains(&millis) {
        return Err(format!(
            "{ENV_CONFIG_POLL_MS}={millis} fora de {MIN_CONFIG_POLL_MS}..={MAX_CONFIG_POLL_MS}"
        ));
    }
    Ok(Duration::from_millis(millis))
}
