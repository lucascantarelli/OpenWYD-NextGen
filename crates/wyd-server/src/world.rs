//! Motor do canal dentro do processo (FASE-03-7) e mundo vivo do personagem (FASE-03-8).
//!
//! O `Engine` é **dono** do estado do mundo deste processo: `ComponentStore` (ECS SoA),
//! grade espacial, acumulador de tick e o mapa `personagem → entidade`. Ele não conhece
//! rede, banco nem socket — a borda entrega intenção por **fila de comandos** e o tick só
//! mexe em memória (invariante 1 do `AGENTS.md`).
//!
//! - Frequência vem de `WYD_TICK_HZ` (20–50, default 20) e é validada no boot; valor fora da
//!   faixa é erro fatal, nunca correção silenciosa.
//! - O tempo real entra por `Instant` na thread do tick; o `Scheduler` do core recebe só
//!   `elapsed_ms`, então o passo continua determinístico e testável sem relógio real.
//! - `Metrics::record_tick` publica `wyd_ticks_total`, `wyd_tick_duration_microseconds` e
//!   `wyd_entities_active` — os nomes da FASE-06-1, com número medido.
//! - Entrada/saída (03-8) passam por [`EngineHandle`]: a thread de rede envia o comando e
//!   espera a resposta; o tick nunca bloqueia em I/O e nenhum mutex global é necessário.
//!
//! O que este módulo **ainda não** faz (issues donas): movimento/combate do jogador no fio
//! (FASE-03-5/08) e `ESCENE_FIELD` (03-9). O save usa a fila de `wyd-db` (`SaveQueue`), que
//! é *try_send*: nunca escreve no tick.

use std::collections::HashMap;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::mpsc::{self, Receiver, Sender, SyncSender};
use std::sync::Arc;
use std::thread::{self, JoinHandle};
use std::time::{Duration, Instant};

use wyd_core::systems::combat::{self as combat_system, AttackRules, DamageModel};
use wyd_core::{
    ComponentStore, CoreError, Entity, EntityKind, Grid, GridConfig, Inventory, Metrics, Position,
    Scheduler, SchedulerConfig, SpawnSpec, Stats, Tick,
};
use wyd_db::{Character, SaveQueue, SaveRequest};
use wyd_sim::formula::damage::{damage_chain, DamageInput};

use crate::config::GameConfig;

/// Variável que escolhe a frequência do tick.
pub const ENV_TICK_HZ: &str = "WYD_TICK_HZ";
/// Frequência default (a mesma do `wyd_core::TICK_HZ`).
pub const DEFAULT_TICK_HZ: u32 = 20;
/// Largura do mundo provisória até o mapa virar dado (`map_definition`, FASE-03-9).
pub const WORLD_WIDTH: i32 = 4096;
/// Altura do mundo provisória (idem).
pub const WORLD_HEIGHT: i32 = 4096;
/// Lado da célula da grade espacial (idem).
pub const GRID_CELL: i32 = 128;
/// Intervalo mínimo entre logs de resumo do tick.
const LOG_EVERY: Duration = Duration::from_secs(10);
/// Teto do sono da thread entre verificações (mantém a parada responsiva).
const MAX_SLEEP: Duration = Duration::from_millis(10);
/// Tempo máximo que a borda espera uma resposta do tick (um tick a 20 Hz + folga).
const COMMAND_TIMEOUT: Duration = Duration::from_millis(2_000);
/// Comandos aplicados por volta da thread (não deixa a fila crescer sem limite).
const COMMANDS_PER_TURN: usize = 64;

/// Distância máxima (Chebyshev) entre o personagem e o NPC clicado para abrir diálogo.
///
/// É validação **do servidor** sobre o `TargetX/TargetY` do `_MSG_Action`; o cliente não
/// define alcance.
pub const TALK_RANGE_TILES: u32 = 5;
/// Alcance do golpe corpo a corpo validado no servidor (não vem do `_MSG_Attack`).
pub const ATTACK_RANGE_TILES: u32 = 2;
/// Intervalo mínimo entre dois golpes da mesma entidade.
pub const ATTACK_COOLDOWN_MS: u64 = 800;
/// Regras de combate do canal (constantes válidas por construção).
const ATTACK_RULES: AttackRules = AttackRules {
    range_tiles: ATTACK_RANGE_TILES,
    cooldown_ms: ATTACK_COOLDOWN_MS,
};

/// Frequência pedida no ambiente (default 20), já validada pelo contrato do scheduler.
pub fn tick_hz_from_env() -> Result<u32, String> {
    let valor = std::env::var(ENV_TICK_HZ).ok();
    tick_hz_from_value(valor.as_deref())
}

/// Interpreta o valor da variável; `None`/vazio usa o default.
///
/// Separado do ambiente para o teste provar 19, 51 e `"abc"` sem mexer em variável de processo.
pub fn tick_hz_from_value(valor: Option<&str>) -> Result<u32, String> {
    let hz = match valor.map(str::trim) {
        None | Some("") => DEFAULT_TICK_HZ,
        Some(bruto) => bruto.parse::<u32>().map_err(|_| {
            format!("{ENV_TICK_HZ} precisa ser um número entre 20 e 50, recebido {bruto:?}")
        })?,
    };
    SchedulerConfig::validated(hz)
        .map(|config| config.hz)
        .map_err(|error| format!("{ENV_TICK_HZ}={hz}: {error}"))
}

/// Recusas de entrada no mundo — todas explícitas, nenhuma cria entidade pela metade.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum EnterWorldError {
    /// O personagem já tem entidade viva neste processo.
    AlreadyInWorld { character_id: i64 },
    /// Estado persistido não cabe no mundo (stats inválidos/posição fora do mapa).
    InvalidState(String),
    /// O tick não respondeu (thread parada ou sobrecarregada).
    Unavailable,
    /// Falha estrutural do core (spawn/grade) — propagada, nunca silenciada.
    Core(CoreError),
}

impl core::fmt::Display for EnterWorldError {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::AlreadyInWorld { character_id } => {
                write!(f, "personagem {character_id} já está vivo neste canal")
            }
            Self::InvalidState(motivo) => write!(f, "estado do personagem inválido: {motivo}"),
            Self::Unavailable => write!(f, "motor do canal indisponível"),
            Self::Core(error) => write!(f, "core recusou o spawn: {error}"),
        }
    }
}

impl std::error::Error for EnterWorldError {}

/// Estado vivo serializável do personagem — o que o save grava e o teste inspeciona.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct LiveState {
    pub character_id: i64,
    pub level: u16,
    pub exp: i64,
    pub hp: u32,
    pub mp: u32,
    pub x: i32,
    pub y: i32,
}

/// Destino do save (implementado em `main` sobre `SaveQueue`; teste injeta um coletor).
pub trait SaveSink: Send + Sync {
    /// Enfileira o snapshot; falha é registrada, nunca bloqueia a borda.
    fn save(&self, state: LiveState);
}

/// Save real: converte o estado vivo no `Character` que o `SaveDrain` grava.
#[derive(Debug)]
pub struct SaveQueueSink {
    queue: Arc<SaveQueue>,
}

impl SaveQueueSink {
    #[must_use]
    pub fn new(queue: Arc<SaveQueue>) -> Self {
        Self { queue }
    }
}

impl SaveSink for SaveQueueSink {
    fn save(&self, state: LiveState) {
        let character = Character {
            id: state.character_id,
            level: i64::from(state.level),
            exp: state.exp,
            hp: i64::from(state.hp),
            mp: i64::from(state.mp),
            pos_x: i64::from(state.x),
            pos_y: i64::from(state.y),
            ..Character::default()
        };
        // Ouro **não** entra no save do mundo: quem escreve `gold` é o claim de grant, e
        // sobrescrever aqui poderia regredir uma concessão já auditada (FASE-07-4).
        if let Err(error) = self.queue.enqueue(SaveRequest::Character(character)) {
            println!(
                "level=warn event=save result=enqueue_failed character={} reason={error}",
                state.character_id
            );
        }
    }
}

/// Um NPC/monstro materializado no mundo a partir de `npc_definition` do snapshot.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NpcSpec {
    pub id: i32,
    pub name: String,
    /// Nome do registro em `npc_template`: é a chave da tabela de drop do abate.
    pub template: String,
    pub position: Position,
    pub stats: Stats,
    pub kind: EntityKind,
    /// EXP do template; 0 = NPC sem combate.
    pub exp: i64,
    pub merchant: i16,
    /// Aparência do template (`STRUCT_MOB.Equip[i].sIndex`): o cliente desenha o mob por ela.
    pub equip: [u16; 16],
    /// `STRUCT_MOB.CurrentScore.AttackRun`: velocidade da animação de andar no cliente.
    pub attack_run: u8,
    /// `STRUCT_MOB.CurrentScore.Direction`: direção inicial da criatura.
    pub direction: u8,
}

/// NPC como o cliente precisa vê-lo: `_MSG_CreateMob` é montado a partir daqui.
///
/// Sai do mundo vivo (posição autoritativa), não da foto do snapshot: a recarga pode já ter
/// movido o NPC e a cena que vale é a do motor.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct NpcView {
    pub id: i32,
    pub name: String,
    pub position: Position,
    pub level: u16,
    pub hp: u32,
    pub max_hp: u32,
    pub attack: u32,
    pub defense: u32,
    /// `STRUCT_SCORE.Merchant` do template: NPC de loja tem nome sempre visível.
    pub merchant: u8,
    pub attack_run: u8,
    pub direction: u8,
    pub equip: [u16; 16],
}

/// Foto da cena de NPCs + a geração que a produziu.
///
/// A geração só sobe quando `sync_npcs` materializa ou remove alguma criatura: a sessão
/// compara o número e busca a foto só quando ela mudou (nada de varrer 6 100 NPCs por
/// pacote).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SceneSnapshot {
    pub generation: u64,
    pub npcs: Vec<NpcView>,
}

/// Resultado de reconciliar o mundo com uma foto de `npc_definition`.
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NpcSync {
    pub spawned: usize,
    pub removed: usize,
    pub kept: usize,
    /// Definicoes que o motor recusou (posição fora do mapa/Stats inválido).
    pub rejected: usize,
}

/// Quem respondeu ao clique de `_MSG_Action` (dado de `npc_definition`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct TalkTarget {
    pub id: i32,
    pub name: String,
    pub merchant: i16,
    pub exp: i64,
}

/// Resultado de um golpe aceito: dano do servidor e a EXP que o abate libera.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct AttackOutcome {
    pub target_npc: i32,
    /// Template do alvo (`npc_template.template_name`), usado para resolver o loot.
    pub template: String,
    pub damage: u32,
    pub remaining_hp: u32,
    pub killed: bool,
    /// EXP base do template (o multiplicador entra depois, na sessão).
    pub exp: i64,
}

/// Estado vivo de um NPC: entidade no armazém + a definição que a originou.
#[derive(Debug, Clone)]
struct NpcLive {
    entity: Entity,
    spec: NpcSpec,
}

/// Comando aceito pela thread do tick.
pub enum Command {
    /// Encarna o personagem lido do banco (a leitura acontece na borda, fora do tick).
    Enter {
        character: Box<Character>,
        reply: SyncSender<Result<Entity, EnterWorldError>>,
    },
    /// Tira a entidade do mundo e devolve o estado para o save.
    Leave {
        entity: Entity,
        reply: SyncSender<Option<LiveState>>,
    },
    /// Foto de todos os vivos (save periódico/shutdown).
    Snapshots { reply: SyncSender<Vec<LiveState>> },
    /// Materializa a foto de `npc_definition` (boot e cada recarga de configuração).
    SyncNpcs {
        specs: Vec<NpcSpec>,
        reply: SyncSender<NpcSync>,
    },
    /// Foto da cena de NPCs (o que o cliente deve ver) + geração.
    Npcs { reply: SyncSender<SceneSnapshot> },
    /// Golpe corpo a corpo contra o NPC/monstro mais próximo do ponto clicado.
    Attack {
        attacker: Entity,
        x: i32,
        y: i32,
        now_ms: u64,
        reply: SyncSender<Result<Option<AttackOutcome>, CoreError>>,
    },
    /// Clique de `_MSG_Action`: quem está no ponto, dentro do alcance do personagem.
    Talk {
        speaker: Entity,
        x: i32,
        y: i32,
        reply: SyncSender<Option<TalkTarget>>,
    },
}

/// Dono do mundo deste processo — um por canal, nenhum mutex global.
pub struct Engine {
    store: ComponentStore,
    grid: Grid,
    scheduler: Scheduler,
    metrics: Arc<Metrics>,
    events_last_tick: usize,
    hz: u32,
    interval_ms: u64,
    live: HashMap<u32, LiveState>,
    by_character: HashMap<i64, Entity>,
    /// NPCs vivos por id de `npc_definition` (o mapa que a recarga reconcilia).
    npcs: HashMap<i32, NpcLive>,
    /// Sobe a cada `sync_npcs` que materializou/removeu criatura: a borda difunde o delta.
    npcs_generation: u64,
}

impl core::fmt::Debug for Engine {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        f.debug_struct("Engine")
            .field("hz", &self.hz)
            .field("entities", &self.store.live_count())
            .field("players", &self.live.len())
            .finish()
    }
}

impl Engine {
    /// Cria o mundo vazio com a frequência pedida (validada).
    pub fn new(hz: u32, metrics: Arc<Metrics>) -> Result<Self, CoreError> {
        let config = SchedulerConfig::validated(hz)?;
        // O orçamento publicado vem da frequência real do processo: 50 ms a 20 Hz, 20 ms a 50 Hz.
        metrics.set_tick_budget_us(1_000_000 / u64::from(config.hz));
        let grid = Grid::new(GridConfig::validated(WORLD_WIDTH, WORLD_HEIGHT, GRID_CELL)?);
        Ok(Self {
            store: ComponentStore::default(),
            grid,
            scheduler: Scheduler::new(config),
            metrics,
            events_last_tick: 0,
            hz: config.hz,
            interval_ms: config.interval_ms(),
            live: HashMap::new(),
            by_character: HashMap::new(),
            npcs: HashMap::new(),
            npcs_generation: 0,
        })
    }

    /// Frequência efetiva (Hz).
    #[must_use]
    pub fn hz(&self) -> u32 {
        self.hz
    }

    /// Intervalo entre ticks em ms (inteiro, como o scheduler usa).
    #[must_use]
    pub fn interval_ms(&self) -> u64 {
        self.interval_ms
    }

    /// Spawn autoritativo: entra no armazém e na grade, na mesma transação de memória.
    ///
    /// Se a posição está fora do mapa, a entidade **não** fica no armazém (rollback explícito).
    pub fn spawn(&mut self, spec: SpawnSpec) -> Result<Entity, CoreError> {
        let posicao = spec.position;
        if !(0..WORLD_WIDTH).contains(&posicao.x) || !(0..WORLD_HEIGHT).contains(&posicao.y) {
            return Err(CoreError::OutOfBounds {
                x: posicao.x,
                y: posicao.y,
            });
        }
        let entity = self
            .store
            .spawn(spec.kind, posicao, spec.stats, Inventory::default())?;
        if let Err(error) = self.grid.insert(entity, posicao.x, posicao.y) {
            self.store.despawn(entity);
            return Err(error);
        }
        Ok(entity)
    }

    /// Remove a entidade do armazém **e** da grade (posição lida antes do despawn).
    pub fn despawn(&mut self, entity: Entity) -> bool {
        if let Some(posicao) = self.store.position_of(entity) {
            let _ = self.grid.remove(entity, posicao.x, posicao.y);
        }
        self.live.remove(&entity.index);
        self.by_character.retain(|_, atual| *atual != entity);
        self.npcs.retain(|_, npc| npc.entity != entity);
        self.store.despawn(entity)
    }

    /// Reconcilia o mundo com a foto de `npc_definition` do snapshot.
    ///
    /// Operação fria (boot e recarga de configuração), nunca por tick com tamanho variável.
    /// NPC que sumiu ou mudou de posição/Stats sai e volta a nascer; o que não mudou fica
    /// com a mesma entidade. Recusa do motor (posição fora do mapa) é contada, não silenciada.
    pub fn sync_npcs(&mut self, specs: Vec<NpcSpec>) -> NpcSync {
        let mut report = NpcSync::default();
        let mut querem: HashMap<i32, NpcSpec> =
            specs.into_iter().map(|spec| (spec.id, spec)).collect();
        let existentes: Vec<i32> = self.npcs.keys().copied().collect();
        for id in existentes {
            let manter = match (self.npcs.get(&id), querem.get(&id)) {
                (Some(atual), Some(novo)) => {
                    novo.position == atual.spec.position && novo.stats == atual.spec.stats
                }
                _ => false,
            };
            if manter {
                querem.remove(&id);
                report.kept += 1;
                continue;
            }
            if let Some(atual) = self.npcs.remove(&id) {
                self.despawn(atual.entity);
                report.removed += 1;
            }
        }
        for (id, spec) in querem {
            match self.spawn(SpawnSpec {
                kind: spec.kind,
                position: spec.position,
                stats: spec.stats,
            }) {
                Ok(entity) => {
                    self.npcs.insert(id, NpcLive { entity, spec });
                    report.spawned += 1;
                }
                Err(_) => report.rejected += 1,
            }
        }
        if report.spawned > 0 || report.removed > 0 {
            self.npcs_generation = self.npcs_generation.wrapping_add(1);
        }
        report
    }

    /// Cena atual: cada NPC vivo com a posição que o motor conhece, ordenada por id.
    ///
    /// A ordem determinística importa para o delta da borda: duas leituras iguais devolvem a
    /// mesma lista, então a sessão só emite o que realmente mudou.
    #[must_use]
    pub fn scene_snapshot(&self) -> SceneSnapshot {
        let mut npcs: Vec<NpcView> = self
            .npcs
            .values()
            .filter_map(|npc| {
                let position = self.store.position_of(npc.entity)?;
                Some(NpcView {
                    id: npc.spec.id,
                    name: npc.spec.name.clone(),
                    position,
                    level: npc.spec.stats.level,
                    hp: npc.spec.stats.hp,
                    max_hp: npc.spec.stats.max_hp,
                    attack: npc.spec.stats.attack,
                    defense: npc.spec.stats.defense,
                    merchant: u8::try_from(npc.spec.merchant).unwrap_or(0),
                    attack_run: npc.spec.attack_run,
                    direction: npc.spec.direction,
                    equip: npc.spec.equip,
                })
            })
            .collect();
        npcs.sort_by_key(|npc| npc.id);
        SceneSnapshot {
            generation: self.npcs_generation,
            npcs,
        }
    }

    /// NPC/monstro mais próximo do ponto clicado, dentro de `range` tiles.
    ///
    /// Desempate pelo menor id — a mesma regra determinística usada na IA. `only_monster`
    /// separa combate (`_MSG_Attack`) de diálogo (`_MSG_Action`).
    fn npc_near(&self, x: i32, y: i32, range: u32, only_monster: bool) -> Option<&NpcLive> {
        self.npcs
            .values()
            .filter(|npc| !only_monster || npc.spec.kind == EntityKind::Monster)
            .filter_map(|npc| {
                let posicao = self.store.position_of(npc.entity)?;
                let distancia = posicao.x.abs_diff(x).max(posicao.y.abs_diff(y));
                (distancia <= range).then_some((distancia, npc.spec.id, npc))
            })
            .min_by_key(|(distancia, id, _)| (*distancia, *id))
            .map(|(_, _, npc)| npc)
    }

    /// Golpe corpo a corpo autoritativo no ponto clicado.
    ///
    /// `Ok(None)` = não há criatura de combate no alcance (o cliente não escolhe alvo fora
    /// do alcance). O dano sai de [`TemplateDamage`] sobre os `Stats` das duas entidades; o
    /// cliente só diz **onde** golpeou. Abate devolve a EXP do template, que a sessão aplica
    /// com o multiplicador do snapshot.
    pub fn attack_at(
        &mut self,
        attacker: Entity,
        x: i32,
        y: i32,
        now_ms: u64,
    ) -> Result<Option<AttackOutcome>, CoreError> {
        let Some((entity, npc_id, exp, template)) =
            self.npc_near(x, y, ATTACK_RANGE_TILES, true).map(|npc| {
                (
                    npc.entity,
                    npc.spec.id,
                    npc.spec.exp,
                    npc.spec.template.clone(),
                )
            })
        else {
            return Ok(None);
        };
        let golpe = combat_system::attack(
            &mut self.store,
            attacker,
            entity,
            now_ms,
            ATTACK_RULES,
            &TemplateDamage,
        )?;
        let killed = golpe.remaining_hp == 0;
        // Morte real: a criatura sai do mundo no mesmo comando. Deixá-la no mapa com HP 0
        // seria um cadáver que o cliente desenha e o jogador não consegue mais golpear; a
        // geração sobe para a borda emitir `_MSG_RemoveMob` (tipo morte). O loot do abate é
        // resolvido na sessão, com a tabela do snapshot e o sorteio do momento.
        if killed {
            self.despawn(entity);
            self.npcs_generation = self.npcs_generation.wrapping_add(1);
        }
        Ok(Some(AttackOutcome {
            target_npc: npc_id,
            template,
            damage: golpe.amount,
            remaining_hp: golpe.remaining_hp,
            killed,
            exp: if killed { exp } else { 0 },
        }))
    }

    /// Quem o personagem clicou em `_MSG_Action`, se houver NPC no alcance.
    pub fn talk_at(&self, speaker: Entity, x: i32, y: i32) -> Option<TalkTarget> {
        let posicao = self.store.position_of(speaker)?;
        let distancia = posicao.x.abs_diff(x).max(posicao.y.abs_diff(y));
        if distancia > TALK_RANGE_TILES {
            return None;
        }
        let npc = self.npc_near(x, y, TALK_RANGE_TILES, false)?;
        Some(TalkTarget {
            id: npc.spec.id,
            name: npc.spec.name.clone(),
            merchant: npc.spec.merchant,
            exp: npc.spec.exp,
        })
    }

    /// Quantos NPCs vivem neste mundo (observabilidade/teste).
    #[must_use]
    pub fn npc_actors(&self) -> usize {
        self.npcs.len()
    }

    /// Encarna o personagem persistido como entidade `Player` na posição salva.
    ///
    /// Toda validação acontece **antes** do spawn: dado inconsistente devolve erro e o
    /// personagem continua fora do mundo (nada de entidade com HP 0 ou posição inválida).
    pub fn enter(&mut self, character: &Character) -> Result<Entity, EnterWorldError> {
        if self.by_character.contains_key(&character.id) {
            return Err(EnterWorldError::AlreadyInWorld {
                character_id: character.id,
            });
        }
        let stats = stats_from(character)?;
        let x = i32::try_from(character.pos_x)
            .map_err(|_| EnterWorldError::InvalidState(format!("pos_x {}", character.pos_x)))?;
        let y = i32::try_from(character.pos_y)
            .map_err(|_| EnterWorldError::InvalidState(format!("pos_y {}", character.pos_y)))?;
        let entity = self
            .spawn(SpawnSpec {
                kind: wyd_core::EntityKind::Player,
                position: wyd_core::Position { x, y },
                stats,
            })
            .map_err(EnterWorldError::Core)?;
        let state = LiveState {
            character_id: character.id,
            level: stats.level,
            exp: character.exp,
            hp: stats.hp,
            mp: stats.mp,
            x,
            y,
        };
        self.live.insert(entity.index, state);
        self.by_character.insert(character.id, entity);
        Ok(entity)
    }

    /// Tira o personagem do mundo e devolve o estado final (o chamador decide salvar).
    pub fn leave(&mut self, entity: Entity) -> Option<LiveState> {
        let state = self.live.remove(&entity.index);
        if let Some(state) = state {
            self.by_character.remove(&state.character_id);
            self.despawn(entity);
        }
        state
    }

    /// Foto de todos os personagens vivos (save periódico).
    #[must_use]
    pub fn snapshots(&self) -> Vec<LiveState> {
        let mut estados: Vec<LiveState> = self.live.values().copied().collect();
        estados.sort_unstable_by_key(|state| state.character_id);
        estados
    }

    /// Aplica um comando da borda sem bloquear o tick.
    fn apply(&mut self, command: Command) {
        match command {
            Command::Enter { character, reply } => {
                let _ = reply.send(self.enter(&character));
            }
            Command::Leave { entity, reply } => {
                let _ = reply.send(self.leave(entity));
            }
            Command::Snapshots { reply } => {
                let _ = reply.send(self.snapshots());
            }
            Command::Npcs { reply } => {
                let _ = reply.send(self.scene_snapshot());
            }
            Command::SyncNpcs { specs, reply } => {
                let _ = reply.send(self.sync_npcs(specs));
            }
            Command::Attack {
                attacker,
                x,
                y,
                now_ms,
                reply,
            } => {
                let _ = reply.send(self.attack_at(attacker, x, y, now_ms));
            }
            Command::Talk {
                speaker,
                x,
                y,
                reply,
            } => {
                let _ = reply.send(self.talk_at(speaker, x, y));
            }
        }
    }

    /// Um passo fixo de simulação: drena eventos parados e publica o gauge de entidades.
    fn step(&mut self) {
        self.events_last_tick = self.store.events().len();
        if self.events_last_tick > 0 {
            // Consumidor de borda entra com o fio de cena (03-9); até lá, drenar impede a
            // fila de crescer.
            self.events_last_tick = self.store.drain_events().len();
        }
    }

    /// Avança o tempo real acumulado e executa os passos devidos (0..=5).
    pub fn advance(&mut self, elapsed_ms: u64) -> u64 {
        let steps = self.scheduler.advance(elapsed_ms);
        for _ in 0..steps {
            let inicio = Instant::now();
            self.step();
            let micros = u64::try_from(inicio.elapsed().as_micros()).unwrap_or(u64::MAX);
            let entidades = u64::try_from(self.store.live_count()).unwrap_or(u64::MAX);
            self.metrics.record_tick(micros, entidades);
        }
        steps
    }

    /// Tick atual do processo.
    #[must_use]
    pub fn tick(&self) -> Tick {
        self.scheduler.tick()
    }

    /// Entidades vivas no mundo.
    #[must_use]
    pub fn entities(&self) -> usize {
        self.store.live_count()
    }

    /// Personagens vivos (jogadores) no mundo.
    #[must_use]
    pub fn players(&self) -> usize {
        self.live.len()
    }

    /// Eventos drenados no último passo (consumidor definitivo em 03-9).
    #[must_use]
    pub fn events_last_tick(&self) -> usize {
        self.events_last_tick
    }

    /// Leitura do armazém para os sistemas do core.
    #[must_use]
    pub fn store(&self) -> &ComponentStore {
        &self.store
    }

    /// Registro de métricas do motor (compartilhado com o `/metrics` do processo).
    #[must_use]
    pub fn metrics(&self) -> &Arc<Metrics> {
        &self.metrics
    }
}

/// Conversão explícita do estado persistido para o componente do core.
fn stats_from(character: &Character) -> Result<Stats, EnterWorldError> {
    let nivel = u16::try_from(character.level).map_err(|_| {
        EnterWorldError::InvalidState(format!("nível {} fora de u16", character.level))
    })?;
    let hp = u32::try_from(character.hp)
        .map_err(|_| EnterWorldError::InvalidState(format!("hp {}", character.hp)))?;
    let mp = u32::try_from(character.mp)
        .map_err(|_| EnterWorldError::InvalidState(format!("mp {}", character.mp)))?;
    let max_hp = u32::try_from(character.hp_max)
        .map_err(|_| EnterWorldError::InvalidState(format!("hp_max {}", character.hp_max)))?;
    let max_mp = u32::try_from(character.mp_max)
        .map_err(|_| EnterWorldError::InvalidState(format!("mp_max {}", character.mp_max)))?;
    // Ataque/defesa correntes do `STRUCT_MOB` da classe (`CurrentScore.Damage`/`Ac`): é o
    // valor que o próprio login envia ao cliente, não uma fórmula nova. Equipamento e bônus
    // entram na FASE-10.
    let modelo = crate::personagens::template_mob(character.class);
    let stats = Stats {
        level: nivel,
        hp,
        max_hp,
        mp,
        max_mp,
        attack: u32::try_from(modelo.current_score.damage.max(0)).unwrap_or(0),
        defense: u32::try_from(modelo.current_score.ac.max(0)).unwrap_or(0),
    };
    stats
        .validate()
        .map_err(|error| EnterWorldError::InvalidState(error.to_string()))?;
    Ok(stats)
}

/// Porta da borda para o tick: `Enter`/`Leave`/`Snapshots` com resposta síncrona curta.
///
/// Clonável (a thread de rede clona por conexão). Não guarda estado do mundo, só a fila —
/// por isso não existe mutex global protegendo o `Engine`.
#[derive(Debug, Clone)]
pub struct EngineHandle {
    tx: Sender<Command>,
}

impl EngineHandle {
    /// Envia o comando e espera a resposta (usado **fora** do tick).
    fn call<T>(
        &self,
        build: impl FnOnce(SyncSender<T>) -> Command,
        timeout: Duration,
    ) -> Option<T> {
        let (reply, rx) = mpsc::sync_channel(1);
        if self.tx.send(build(reply)).is_err() {
            return None;
        }
        rx.recv_timeout(timeout).ok()
    }

    /// Encarna o personagem; erro é recusa tipada (nada de entidade parcial).
    pub fn enter(&self, character: &Character) -> Result<Entity, EnterWorldError> {
        self.call(
            |reply| Command::Enter {
                character: Box::new(character.clone()),
                reply,
            },
            COMMAND_TIMEOUT,
        )
        .unwrap_or(Err(EnterWorldError::Unavailable))
    }

    /// Retira a entidade do mundo e devolve o estado final.
    #[must_use]
    pub fn leave(&self, entity: Entity) -> Option<LiveState> {
        self.call(|reply| Command::Leave { entity, reply }, COMMAND_TIMEOUT)
            .flatten()
    }

    /// Foto de todos os vivos (save periódico).
    #[must_use]
    pub fn snapshots(&self) -> Vec<LiveState> {
        self.call(|reply| Command::Snapshots { reply }, COMMAND_TIMEOUT)
            .unwrap_or_default()
    }

    /// Materializa a foto de NPCs do snapshot (boot e recarga).
    #[must_use]
    pub fn sync_npcs(&self, specs: Vec<NpcSpec>) -> NpcSync {
        self.call(|reply| Command::SyncNpcs { specs, reply }, COMMAND_TIMEOUT)
            .unwrap_or_default()
    }

    /// Cena vigente do motor (NPCs com posição autoritativa + geração).
    #[must_use]
    pub fn npcs(&self) -> SceneSnapshot {
        self.call(|reply| Command::Npcs { reply }, COMMAND_TIMEOUT)
            .unwrap_or_default()
    }

    /// Golpe autoritativo no ponto clicado. `None` = sem criatura no alcance.
    pub fn attack(
        &self,
        attacker: Entity,
        x: i32,
        y: i32,
        now_ms: u64,
    ) -> Result<Option<AttackOutcome>, CoreError> {
        match self.call(
            |reply| Command::Attack {
                attacker,
                x,
                y,
                now_ms,
                reply,
            },
            COMMAND_TIMEOUT,
        ) {
            Some(resultado) => resultado,
            None => Err(CoreError::EntityNotAlive {
                index: attacker.index,
                generation: attacker.generation,
            }),
        }
    }

    /// NPC clicado por `_MSG_Action` (fora do alcance devolve `None`).
    #[must_use]
    pub fn talk(&self, speaker: Entity, x: i32, y: i32) -> Option<TalkTarget> {
        self.call(
            |reply| Command::Talk {
                speaker,
                x,
                y,
                reply,
            },
            COMMAND_TIMEOUT,
        )
        .flatten()
    }
}

/// Dano do canal: `attack` do atacante contra o `defense` do alvo na cadeia portada do
/// `wyd-sim` (`DamageInput`/`damage_chain`), com piso 1 para o golpe nunca ser nulo.
///
/// O piso é regra de andamento do canal, não balanceamento: ambos os lados podem nascer com
/// ataque 0 antes da FASE-10 e um golpe de 0 travaria o combate para sempre.
struct TemplateDamage;

impl DamageModel for TemplateDamage {
    fn damage(&self, attacker: Stats, defender: Stats) -> u32 {
        let entrada = DamageInput {
            base: attacker.attack,
            defense: defender.defense,
            ..DamageInput::new(0)
        };
        damage_chain(&entrada).max(1)
    }
}

/// Materializa `npc_definition` do snapshot como pedidos de spawn do mapa dado.
///
/// Nome do diálogo: `display_name` (texto do portal); template só preenche quando ele está
/// vazio. Criatura de combate é a que tem `Exp` no template e não é comerciante.
pub fn npc_specs(config: &GameConfig, map_id: i16) -> Vec<NpcSpec> {
    config
        .npcs_on_map(map_id)
        .map(|npc| NpcSpec {
            id: npc.id,
            name: if npc.display_name.is_empty() {
                npc.name.clone()
            } else {
                npc.display_name.clone()
            },
            template: npc.template_name.clone(),
            position: Position {
                x: i32::from(npc.pos_x),
                y: i32::from(npc.pos_y),
            },
            stats: Stats {
                level: npc.level,
                hp: npc.hp,
                max_hp: npc.max_hp,
                mp: 0,
                max_mp: 0,
                attack: npc.attack,
                defense: npc.defense,
            },
            kind: if npc.monster {
                EntityKind::Monster
            } else {
                EntityKind::Npc
            },
            exp: npc.exp,
            merchant: npc.merchant,
            equip: npc.equip,
            attack_run: npc.attack_run,
            direction: npc.direction,
        })
        .collect()
}

/// Thread do tick com parada limpa — sem `unsafe`, sem `tokio` no laço.
#[derive(Debug)]
pub struct EngineThread {
    stop: Arc<AtomicBool>,
    handle: Option<JoinHandle<()>>,
    metrics: Arc<Metrics>,
}

impl EngineThread {
    /// Sobe a thread `wyd-tick` e devolve a porta da borda.
    ///
    /// `interval_ms` é o intervalo do scheduler (50 ms a 20 Hz). Criar thread é operação de
    /// boot: falha de SO vira `Err` e o processo não finge que está simulando.
    pub fn spawn(mut engine: Engine, interval_ms: u64) -> Result<(Self, EngineHandle), String> {
        let (tx, rx) = mpsc::channel::<Command>();
        let handle = EngineHandle { tx };
        let stop = Arc::new(AtomicBool::new(false));
        let flag = Arc::clone(&stop);
        let metrics = Arc::clone(engine.metrics());
        let resumo_metrics = Arc::clone(&metrics);
        let intervalo = Duration::from_millis(interval_ms.max(1));
        let thread_handle = thread::Builder::new()
            .name("wyd-tick".to_string())
            .spawn(move || {
                run_loop(&mut engine, &rx, &flag, intervalo, &resumo_metrics);
            })
            .map_err(|error| format!("thread do tick: {error}"))?;
        Ok((
            Self {
                stop,
                handle: Some(thread_handle),
                metrics,
            },
            handle,
        ))
    }

    /// Registro do motor (para o `/metrics` do processo).
    #[must_use]
    pub fn metrics(&self) -> &Arc<Metrics> {
        &self.metrics
    }

    /// Sinaliza parada e espera a thread terminar (idempotente).
    pub fn stop_and_join(&mut self) {
        self.stop.store(true, Ordering::Relaxed);
        if let Some(handle) = self.handle.take() {
            let _ = handle.join();
        }
    }
}

impl Drop for EngineThread {
    fn drop(&mut self) {
        self.stop_and_join();
    }
}

/// Laço do tick: comandos da borda (máx. [`COMMANDS_PER_TURN`]) → avanço do relógio.
fn run_loop(
    engine: &mut Engine,
    rx: &Receiver<Command>,
    stop: &AtomicBool,
    intervalo: Duration,
    metrics: &Arc<Metrics>,
) {
    let mut proximo = Instant::now() + intervalo;
    let mut ultimo = Instant::now();
    let mut ultimo_log = Instant::now();
    while !stop.load(Ordering::Relaxed) {
        for command in rx.try_iter().take(COMMANDS_PER_TURN) {
            engine.apply(command);
        }
        let agora = Instant::now();
        if agora >= proximo {
            let elapsed =
                u64::try_from(agora.duration_since(ultimo).as_millis()).unwrap_or(u64::MAX);
            ultimo = agora;
            engine.advance(elapsed);
            proximo += intervalo;
            if agora >= proximo {
                // Stall longo (pausa da VM, debugger): não tenta "recuperar" o atraso.
                proximo = agora + intervalo;
            }
        } else {
            thread::sleep((proximo - agora).min(MAX_SLEEP));
        }
        if ultimo_log.elapsed() >= LOG_EVERY {
            ultimo_log = Instant::now();
            let resumo = metrics.tick_snapshot();
            println!(
                "level=info event=tick ticks={} entities={} players={} p50_us={} p99_us={} mean_us={}",
                metrics.ticks(),
                engine.entities(),
                engine.players(),
                resumo.p50,
                resumo.p99,
                resumo.mean
            );
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn personagem(id: i64, x: i64, y: i64) -> Character {
        Character {
            id,
            level: 10,
            exp: 1_500,
            hp: 90,
            mp: 40,
            hp_max: 100,
            mp_max: 50,
            pos_x: x,
            pos_y: y,
            ..Character::default()
        }
    }

    fn engine() -> Engine {
        Engine::new(20, Arc::new(Metrics::new())).expect("motor com 20 Hz")
    }

    #[test]
    fn hz_fora_da_faixa_falha_na_interpretacao() {
        assert_eq!(tick_hz_from_value(None), Ok(DEFAULT_TICK_HZ));
        assert_eq!(tick_hz_from_value(Some("  ")), Ok(DEFAULT_TICK_HZ));
        assert_eq!(tick_hz_from_value(Some("50")), Ok(50));
        for invalido in ["19", "51", "0", "abc", "-20"] {
            assert!(
                tick_hz_from_value(Some(invalido)).is_err(),
                "{invalido} deveria ser recusado"
            );
        }
    }

    #[test]
    fn entrar_cria_entidade_na_posicao_do_banco_e_sair_devolve_o_estado() {
        let mut engine = engine();
        let personagem = personagem(7, 300, 400);
        let entity = engine.enter(&personagem).expect("entrada válida");
        assert_eq!(engine.players(), 1);
        assert_eq!(engine.entities(), 1);
        let posicao = engine.store().position_of(entity).expect("vivo");
        assert_eq!((posicao.x, posicao.y), (300, 400));

        let estado = engine.leave(entity).expect("estado final");
        assert_eq!(estado.character_id, 7);
        assert_eq!(
            (estado.x, estado.y, estado.hp, estado.mp),
            (300, 400, 90, 40)
        );
        assert_eq!(engine.players(), 0);
        assert_eq!(engine.entities(), 0);
        assert!(engine.snapshots().is_empty());
    }

    #[test]
    fn personagem_duplicado_e_estado_invalido_nao_criam_entidade() {
        let mut engine = engine();
        let vivo = personagem(9, 10, 10);
        engine.enter(&vivo).expect("primeira entrada");
        assert_eq!(
            engine.enter(&vivo),
            Err(EnterWorldError::AlreadyInWorld { character_id: 9 })
        );

        let mut quebrado = personagem(10, 10, 10);
        quebrado.hp = 200; // acima do max_hp
        assert!(matches!(
            engine.enter(&quebrado),
            Err(EnterWorldError::InvalidState(_))
        ));
        let mut fora = personagem(11, -5, 10);
        fora.pos_x = -5;
        assert!(matches!(
            engine.enter(&fora),
            Err(EnterWorldError::Core(CoreError::OutOfBounds { .. }))
        ));
        assert_eq!(engine.players(), 1, "só o personagem válido entrou");
    }

    /// NPC de teste: `exp > 0` vira `Monster`, senão `Npc` de loja/diálogo.
    fn npc(id: i32, name: &str, x: i32, y: i32, exp: i64) -> NpcSpec {
        NpcSpec {
            id,
            name: name.to_string(),
            template: name.to_string(),
            position: Position { x, y },
            stats: Stats {
                level: 1,
                hp: if exp > 0 { 1 } else { 10 },
                max_hp: if exp > 0 { 1 } else { 10 },
                mp: 0,
                max_mp: 0,
                attack: 1,
                defense: 0,
            },
            kind: if exp > 0 {
                EntityKind::Monster
            } else {
                EntityKind::Npc
            },
            exp,
            merchant: if exp > 0 { 0 } else { 1 },
            equip: [0; 16],
            attack_run: 82,
            direction: 3,
        }
    }

    #[test]
    fn npc_do_snapshot_nasce_e_a_recarga_move_sem_duplicar() {
        let mut engine = engine();
        let aylin = npc(7, "Aylin", 40, 40, 0);
        let primeiro = engine.sync_npcs(vec![aylin.clone()]);
        assert_eq!((primeiro.spawned, primeiro.rejected), (1, 0));
        assert_eq!(engine.npc_actors(), 1);

        // Mesma foto: mantém a entidade (nada de duplicar no mapa).
        let segundo = engine.sync_npcs(vec![aylin.clone()]);
        assert_eq!((segundo.spawned, segundo.kept), (0, 1));
        assert_eq!(engine.npc_actors(), 1);

        // Posição nova: nasce outra entidade e a antiga sai.
        let movido = NpcSpec {
            position: Position { x: 50, y: 60 },
            ..aylin
        };
        let terceiro = engine.sync_npcs(vec![movido]);
        assert_eq!((terceiro.spawned, terceiro.removed), (1, 1));

        // Foto vazia (NPC desabilitado no portal) tira todos do mundo.
        let vazio = engine.sync_npcs(Vec::new());
        assert_eq!(vazio.removed, 1);
        assert_eq!(engine.npc_actors(), 0);
    }

    #[test]
    fn dialogo_exige_alcance_e_usa_o_nome_da_definicao() {
        let mut engine = engine();
        engine.sync_npcs(vec![npc(7, "Aylin", 40, 40, 0)]);
        let jogador = engine.enter(&personagem(1, 42, 40)).expect("entrada");

        let alvo = engine.talk_at(jogador, 40, 40).expect("NPC no alcance");
        assert_eq!(
            (alvo.id, alvo.name.as_str(), alvo.merchant),
            (7, "Aylin", 1)
        );

        // `TargetX/TargetY` longe do personagem não abre diálogo nenhum.
        assert_eq!(engine.talk_at(jogador, 90, 90), None);
        // Ponto sem NPC no alcance também não.
        assert_eq!(engine.talk_at(jogador, 50, 50), None);
    }

    #[test]
    fn abate_usa_a_exp_do_template_e_npc_sem_combate_nao_e_alvo() {
        let mut engine = engine();
        engine.sync_npcs(vec![
            npc(1_000, "Orc Cavaleiro", 41, 40, 5_000),
            npc(7, "Aylin", 70, 70, 0),
        ]);
        let jogador = engine.enter(&personagem(1, 40, 40)).expect("entrada");

        // `Npc` de loja nunca é alvo de `_MSG_Attack`: clicar nele não acha criatura.
        assert_eq!(
            engine.attack_at(jogador, 70, 70, 1_000).expect("sem erro"),
            None,
            "NPC sem EXP não entra em combate"
        );

        // O monstro cai no golpe do servidor e libera a EXP que o template declara.
        let golpe = engine
            .attack_at(jogador, 41, 40, 1_000)
            .expect("sem erro")
            .expect("monstro no alcance");
        assert!(golpe.killed);
        assert_eq!((golpe.target_npc, golpe.exp), (1_000, 5_000));
        assert_eq!(golpe.remaining_hp, 0);

        // Morte tira a criatura do mundo: não sobra cadáver no mapa nem alvo para um
        // segundo golpe, e a cena publica a remoção pela geração nova.
        assert_eq!(
            engine.attack_at(jogador, 41, 40, 9_000).expect("sem erro"),
            None,
            "o NPC abatido deixa de existir no mundo"
        );
        assert_eq!(engine.npc_actors(), 1, "só o NPC de loja continua vivo");
        let cena = engine.scene_snapshot();
        assert_eq!(cena.npcs.len(), 1);
        assert!(cena.npcs.iter().all(|npc| npc.id == 7));
        assert_eq!(cena.generation, 2, "o abate incrementa a geração da cena");
    }

    #[test]
    fn handle_da_borda_entra_sai_e_fotografa() {
        let (mut thread, handle) = EngineThread::spawn(engine(), 10).expect("thread do tick");
        let personagem = personagem(21, 100, 200);
        let entity = handle.enter(&personagem).expect("entrada pelo handle");
        let fotos = handle.snapshots();
        assert_eq!(fotos.len(), 1);
        assert_eq!(fotos[0].character_id, 21);
        let estado = handle.leave(entity).expect("saída pelo handle");
        assert_eq!(estado.character_id, 21);
        assert!(handle.snapshots().is_empty());
        thread.stop_and_join();
    }
}
