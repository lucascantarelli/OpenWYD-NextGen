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
//! Cena autoritativa (FASE-03-9): [`Engine::scene_snapshot`] devolve **uma** lista de
//! entidades visíveis — NPCs do mapa e personagens no mundo — com o **índice de entidade**
//! como id de cena (espaço único: o cliente indexa `MobID`, então NPC e jogador não podem
//! colidir) e [`plan_scene`] calcula o delta de entrada/saída por sessão, com teto de pacotes
//! por varredura. Quem escreve no fio é a borda, sempre com `HEADER.ID = ESCENE_FIELD`.
//!
//! O que este módulo **ainda não** faz (issues donas): movimento do jogador no fio (a tabela
//! de opcodes provada não tem `_MSG_Move`, FASE-08-6) e atualização de barra de outro jogador.
//! O save usa a fila de `wyd-db` (`SaveQueue`), que é *try_send*: nunca escreve no tick.

use std::collections::HashMap;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::mpsc::{self, Receiver, Sender, SyncSender};
use std::sync::Arc;
use std::thread::{self, JoinHandle};
use std::time::{Duration, Instant};

use wyd_core::systems::combat::{self as combat_system, AttackRules, DamageModel};
use wyd_core::systems::movement::{self as movement_system, MoveRules};
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
/// `MAX_USER` do legado (`Basedef.h`): id de cena **abaixo** deste é jogador.
///
/// O cliente decide "é personagem" pela faixa do `MobID`/`ClientID` (o porte Go do
/// mesmo alvo usa `IsPlayer(id) = id < MaxUser`). Por isso o id de cena não pode ser o
/// índice cru do armazém: os NPCs nascem primeiro e empurrariam o jogador para a faixa
/// de criatura.
pub const MAX_USER: u16 = 1_000;
/// `MAX_MOB` do legado: teto do espaço de id de cena (jogador `< MAX_USER`, criatura
/// `MAX_USER..MAX_MOB`).
pub const MAX_MOB: u16 = 25_000;
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
/// Raio (tiles) em que um clique conta como "no sprite" do NPC.
///
/// O sprite do 7662 ocupa mais de um tile; 2 cobre a base sem pegar o vizinho. Clicar no
/// NPC vale mesmo com o personagem longe: o `_MSG_Action` é destino de caminhada e o
/// diálogo/loja abre ao chegar — recusar por distância deixava toda loja da cidade sem
/// resposta (o clique virava passo recusado e `no_npc`).
pub const CLICK_RANGE_TILES: u32 = 2;
/// Alcance do golpe corpo a corpo validado no servidor (não vem do `_MSG_Attack`).
pub const ATTACK_RANGE_TILES: u32 = 2;
/// Intervalo mínimo entre dois golpes da mesma entidade.
pub const ATTACK_COOLDOWN_MS: u64 = 800;

/// Tiles por segundo do personagem no `_MSG_Action` de movimento.
///
/// **UNVERIFIED:** a velocidade do cliente 7662 ainda não foi medida; entra por ambiente
/// (`WYD_MOVE_TILES_PER_SECOND`, padrão 4) e quem decide continua sendo o servidor — o
/// pacote traz destino, nunca velocidade.
pub const ENV_MOVE_TILES_PER_SECOND: &str = "WYD_MOVE_TILES_PER_SECOND";
pub const DEFAULT_MOVE_TILES_PER_SECOND: u16 = 4;
/// Máximo de tiles num `_MSG_Action`: `Route[24]` do legado = 24 passos de 1 tile.
pub const MOVE_MAX_TILES: u32 = 24;

/// Velocidade configurada (zero ou lixo cai no padrão, nunca em "sem movimento").
fn move_tiles_per_second() -> u16 {
    std::env::var(ENV_MOVE_TILES_PER_SECOND)
        .ok()
        .and_then(|valor| valor.trim().parse().ok())
        .filter(|valor: &u16| *valor > 0)
        .unwrap_or(DEFAULT_MOVE_TILES_PER_SECOND)
}
/// Regras de combate do canal (constantes válidas por construção).
const ATTACK_RULES: AttackRules = AttackRules {
    range_tiles: ATTACK_RANGE_TILES,
    cooldown_ms: ATTACK_COOLDOWN_MS,
};

/// Variável do teto de pacotes de cena por varredura de sessão.
pub const ENV_SCENE_MAX_PER_REFRESH: &str = "WYD_SCENE_MAX_PER_REFRESH";
/// Teto default de pacotes de cena por varredura (= por segundo, `SCENE_POLL`).
///
/// Um `_MSG_CreateMob` custa 232 B e a varredura acontece a 1 Hz: 64 pacotes são 14 848 B/s
/// por sessão, muito abaixo do que a sessão já tráfega. O teto existe para uma recarga de
/// `npc_definition` com milhares de criaturas não virar uma rajada — e, quando ele corta, a
/// sessão **não** avança a geração e continua de onde parou na varredura seguinte.
///
/// Era 64 e isso deixava a cena incompleta por ~95 s (6 100 NPCs a 1 varredura/s): o cliente
/// que entrava e olhava a área recebia só os primeiros ids e nunca os NPCs criados por último
/// (caso do Campo, ids 6301..6310). O teto continua existindo para a recarga não virar rajada,
/// mas agora cabe o mapa inteiro em duas varreduras.
pub const DEFAULT_SCENE_MAX_PER_REFRESH: usize = 4_096;

/// Raio (tiles, Chebyshev) da cena que a sessão recebe em volta do próprio personagem.
///
/// O mapa tem 6 110 NPCs: enviar todos no login é uma rajada de ~1,4 MB em `_MSG_CreateMob`
/// de 232 B, e o cliente 7662 não desenha nenhum deles (buffer/limite do lado C). A sessão
/// passa a receber só o que está à vista; ao andar, o delta remove quem saiu e cria quem
/// entrou — a geração do motor não muda, então a varredura confere o conjunto a cada poll.
/// 56 tiles (Chebyshev) cobrem a câmera do 7662 **e** todos os serviços de Armia em volta do
/// ponto de save (2096/2096). Medido no banco: raio 24 = 32 NPCs na cena (mercadores de
/// `y >= 2121` fora), raio 40 = 45, raio 56 = **71** — inclui `ForeLearner` (2090, 2047) e
/// `9thMaster` (2136, 2115), que a cidade tem e o jogador não via. Continua muito abaixo da
/// rajada de 6 110 que o cliente 7662 não desenha. Ajustável por `WYD_SCENE_VIEW_TILES`.
pub const ENV_SCENE_VIEW_TILES: &str = "WYD_SCENE_VIEW_TILES";
pub const DEFAULT_SCENE_VIEW_TILES: u32 = 56;
/// Raio da cena pedido no ambiente (valor inválido cai no padrão).
pub fn scene_view_tiles_from_env() -> u32 {
    std::env::var(ENV_SCENE_VIEW_TILES)
        .ok()
        .and_then(|valor| valor.trim().parse().ok())
        .filter(|valor: &u32| (1..=128).contains(valor))
        .unwrap_or(DEFAULT_SCENE_VIEW_TILES)
}
/// Faixa aceita do teto (0 e negativo não fazem sentido; a sessão pararia de desenhar o mapa).
pub const MAX_SCENE_PER_REFRESH: usize = 4_096;

/// Teto de pacotes de cena por varredura pedido no ambiente (default 64).
pub fn scene_budget_from_env() -> Result<usize, String> {
    let valor = std::env::var(ENV_SCENE_MAX_PER_REFRESH).ok();
    scene_budget_from_value(valor.as_deref())
}

/// Interpreta o teto; `None`/vazio usa o default.
///
/// Separado do ambiente para o teste provar `0`, `4097` e `"abc"` sem mexer em variável de
/// processo. Valor inválido é erro fatal no boot — nunca correção silenciosa.
pub fn scene_budget_from_value(valor: Option<&str>) -> Result<usize, String> {
    let bruto = match valor.map(str::trim) {
        None | Some("") => return Ok(DEFAULT_SCENE_MAX_PER_REFRESH),
        Some(bruto) => bruto,
    };
    let teto = bruto.parse::<usize>().map_err(|_| {
        format!(
            "{ENV_SCENE_MAX_PER_REFRESH} precisa ser um número entre 1 e {MAX_SCENE_PER_REFRESH}, \
             recebido {bruto:?}"
        )
    })?;
    if !(1..=MAX_SCENE_PER_REFRESH).contains(&teto) {
        return Err(format!(
            "{ENV_SCENE_MAX_PER_REFRESH}={teto} fora da faixa 1..={MAX_SCENE_PER_REFRESH}"
        ));
    }
    Ok(teto)
}

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
    /// Sem id de cena livre no espaço de jogador (`< MAX_USER`): canal cheio.
    SceneFull,
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
            Self::SceneFull => write!(
                f,
                "sem id de cena livre no espaço de jogador (< {MAX_USER})"
            ),
            Self::Core(error) => write!(f, "core recusou o spawn: {error}"),
        }
    }
}

impl std::error::Error for EnterWorldError {}

/// Personagem encarnado: a entidade do armazém e o id que o cliente conhece.
///
/// `scene_id` é o `ClientID` do `CNFCharacterLogin` e o `MobID` do `_MSG_CreateMob` do
/// próprio jogador — sempre `< MAX_USER`, ao contrário do índice do armazém.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct Entered {
    pub entity: Entity,
    pub scene_id: u16,
}

/// Estado vivo serializável do personagem — o que o save grava e o teste inspeciona.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct LiveState {
    pub character_id: i64,
    pub level: u16,
    /// CP do personagem (`-75..=75`), persistido no save e usado na cor do nome.
    pub cp: i16,
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
            cp: i64::from(state.cp),
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

/// Aparência do personagem para a cena (`_MSG_CreateMob`), montada na borda a partir do
/// `STRUCT_MOB` do login — o motor não conhece inventário nem template de classe.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ActorSkin {
    /// `MobName[16]` do `STRUCT_MOB`.
    pub name: [u8; 16],
    /// `Equip[i].sIndex` — é por ela que o cliente desenha o personagem.
    pub equip: [u16; 16],
    /// `STRUCT_MOB.Guild` (0 = sem guilda).
    pub guild: u16,
    /// `CurrentScore.AttackRun` (velocidade da animação de andar).
    pub attack_run: u8,
    /// `CurrentScore.Direction`.
    pub direction: u8,
}

/// Uma entidade na vista do cliente: NPC/monstro **ou** personagem.
///
/// `id` é o **índice da entidade** no armazém, não o id de `npc_definition`: o cliente indexa
/// as criaturas por `MobID`, então NPC e jogador têm de viver no mesmo espaço de ids — usar o
/// id do banco para NPC e o índice para jogador faria duas entidades diferentes colidirem no
/// mesmo `MobID`. Sai do mundo vivo (posição autoritativa), nunca da foto do snapshot.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SceneMob {
    /// `MobID` do `_MSG_CreateMob` (índice da entidade).
    pub id: u16,
    /// `true` para personagem: muda o `RemoveType` na saída (logout) e o `Merchant`.
    pub actor: bool,
    /// Id em `npc_definition` (0 = personagem): é a chave do catálogo de loja.
    ///
    /// Não vai no fio — o cliente só conhece o `MobID` —, então a loja é resolvida
    /// aqui, no mundo vivo, nunca pelo que o cliente manda.
    pub npc_id: i32,
    pub name: [u8; 16],
    pub position: Position,
    pub level: u16,
    pub hp: u32,
    pub max_hp: u32,
    pub attack: u32,
    pub defense: u32,
    /// `STRUCT_SCORE.Merchant`: NPC de loja tem nome sempre visível; jogador é 0.
    pub merchant: u8,
    /// `STRUCT_SCORE.ChaosRate` = CP + 75: é o byte que o cliente lê para a cor do nome.
    pub chaos_rate: u8,
    pub attack_run: u8,
    pub direction: u8,
    pub equip: [u16; 16],
    /// `STRUCT_MOB.Guild` (0 = sem guilda).
    pub guild: u16,
}

impl SceneMob {
    /// Nome sem os NULs de preenchimento (log/teste).
    #[must_use]
    pub fn name_str(&self) -> String {
        let fim = self
            .name
            .iter()
            .position(|byte| *byte == 0)
            .unwrap_or(self.name.len());
        String::from_utf8_lossy(&self.name[..fim]).to_string()
    }
}

/// Foto da cena (NPCs + personagens) e a geração que a produziu.
///
/// A geração sobe a cada entidade que nasce ou sai do mundo: a sessão compara o número e só
/// busca a foto quando ela mudou (nada de varrer 6 100 NPCs por pacote).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SceneSnapshot {
    pub generation: u64,
    pub mobs: Vec<SceneMob>,
}

/// Delta de cena de **uma** sessão: quem entra, quem sai e se o delta coube no teto.
///
/// `complete = false` significa que o teto de pacotes por varredura cortou o resto: a sessão
/// **não** avança a geração e tenta de novo na próxima varredura (o que já foi enviado não é
/// reenviado, porque o conjunto enviado é atualizado).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct ScenePlan {
    pub enter: Vec<SceneMob>,
    /// `(id, RemoveType)`: `0` fora de vista, `2` logout (`ADR-0025`).
    pub leave: Vec<(u16, i32)>,
    pub complete: bool,
}

/// `RemoveType` do `_MSG_RemoveMob` (`ADR-0025`): saiu da vista.
pub const REMOVE_OUT_OF_VIEW: i32 = 0;
/// `RemoveType` do `_MSG_RemoveMob`: o personagem saiu do mundo (logout/queda).
pub const REMOVE_LOGOUT: i32 = 2;

/// Calcula o delta entre o que a sessão já viu (`sent`: id → é personagem) e a foto atual.
///
/// Só entrada e saída: entidade que **mudou** de posição/HP não é reenviada como `CreateMob`
/// (o cliente recusaria o id repetido e o pacote de movimento não está provado — resíduo
/// declarado na FASE-03-9). Saídas vêm primeiro: liberar o cliente de um fantasma vale mais
/// que desenhar um novo.
#[must_use]
pub fn plan_scene(
    sent: &std::collections::BTreeMap<u16, bool>,
    snapshot: &SceneSnapshot,
    budget: usize,
) -> ScenePlan {
    let atuais: std::collections::BTreeMap<u16, &SceneMob> =
        snapshot.mobs.iter().map(|mob| (mob.id, mob)).collect();
    let mut leave: Vec<(u16, i32)> = sent
        .iter()
        .filter(|(id, _)| !atuais.contains_key(id))
        .map(|(id, actor)| {
            (
                *id,
                if *actor {
                    REMOVE_LOGOUT
                } else {
                    REMOVE_OUT_OF_VIEW
                },
            )
        })
        .collect();
    leave.sort_unstable_by_key(|(id, _)| *id);
    let mut enter: Vec<SceneMob> = snapshot
        .mobs
        .iter()
        .filter(|mob| !sent.contains_key(&mob.id))
        .cloned()
        .collect();
    let total = leave.len() + enter.len();
    let complete = total <= budget;
    if !complete {
        // Saídas primeiro; o que sobrar do teto vai para entradas.
        leave.truncate(budget);
        let restante = budget.saturating_sub(leave.len());
        enter.truncate(restante);
    }
    ScenePlan {
        enter,
        leave,
        complete,
    }
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
    /// Id do alvo na cena (`MobID` = índice da entidade), usado no `_MSG_RemoveMob` de morte.
    pub target_scene: u16,
    /// Template do alvo (`npc_template.template_name`), usado para resolver o loot.
    pub template: String,
    pub damage: u32,
    pub remaining_hp: u32,
    pub killed: bool,
    /// EXP base do template (o multiplicador entra depois, na sessão).
    pub exp: i64,
    /// `true` = o alvo era outro jogador (PvP), não um NPC.
    pub target_actor: bool,
    /// CP do autor depois do golpe (só muda em abate PvP).
    pub cp_after: i16,
    /// Caos descontado do autor neste abate (0 fora de PvP ou para vítima já listada).
    pub cp_delta: i16,
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
        skin: Box<ActorSkin>,
        reply: SyncSender<Result<Entered, EnterWorldError>>,
    },
    /// Tira a entidade do mundo e devolve o estado para o save.
    Leave {
        entity: Entity,
        reply: SyncSender<Option<LiveState>>,
    },
    /// Foto de todos os vivos (save periódico/shutdown).
    Snapshots { reply: SyncSender<Vec<LiveState>> },
    /// CP vivo da entidade (comando `/cp` do chat).
    Cp {
        entity: Entity,
        reply: SyncSender<Option<i16>>,
    },
    /// Materializa a foto de `npc_definition` (boot e cada recarga de configuração).
    SyncNpcs {
        specs: Vec<NpcSpec>,
        reply: SyncSender<NpcSync>,
    },
    /// Foto da cena de NPCs (o que o cliente deve ver) + geração.
    Npcs { reply: SyncSender<SceneSnapshot> },
    /// Foto da cena **visível** em volta de `speaker` (raio em tiles).
    SceneAround {
        speaker: Entity,
        radius: u32,
        reply: SyncSender<Option<SceneSnapshot>>,
    },
    /// Golpe corpo a corpo contra o NPC/monstro mais próximo do ponto clicado.
    Attack {
        attacker: Entity,
        x: i32,
        y: i32,
        now_ms: u64,
        reply: SyncSender<Result<Option<AttackOutcome>, CoreError>>,
    },
    /// `_MSG_Action` de movimento: destino pedido, validado no tick (orçamento + mapa).
    Move {
        entity: Entity,
        x: i32,
        y: i32,
        now_ms: u64,
        reply: SyncSender<Result<Position, CoreError>>,
    },
    /// Clique de `_MSG_Action`: quem está no ponto, dentro do alcance do personagem.
    Talk {
        speaker: Entity,
        x: i32,
        y: i32,
        reply: SyncSender<Option<TalkTarget>>,
    },
    /// Uso de chave num portão: `keys` são os `EF_KEYID` que o personagem carrega.
    UseKey {
        speaker: Entity,
        x: i32,
        y: i32,
        keys: Vec<i32>,
        now_ms: u64,
        reply: SyncSender<Result<crate::gameplay::GateOutcome, crate::gameplay::GateError>>,
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
    /// Aparência dos personagens vivos, por índice de entidade (cena de jogador).
    skins: HashMap<u32, ActorSkin>,
    /// Índice de entidade → id de cena (`MobID`/`ClientID`), estável enquanto ela vive.
    ///
    /// Duas faixas, como no legado: jogador (`1..MAX_USER`) e criatura
    /// (`MAX_USER..MAX_MOB`). O id é liberado na saída e reusado pelo mesmo espaço.
    scene_ids: HashMap<u32, u16>,
    /// Ids de jogador devolvidos ao espaço (pilha: reuso imediato e previsível).
    free_player_ids: Vec<u16>,
    /// Ids de criatura devolvidos ao espaço.
    free_mob_ids: Vec<u16>,
    /// Próximo id de jogador nunca usado.
    next_player_scene: u16,
    /// Próximo id de criatura nunca usado.
    next_mob_scene: u16,
    /// Sobe a cada entidade que nasce ou sai do mundo: a borda difunde o delta.
    scene_generation: u64,
    /// Portões abertos por chave (`EF_KEYID`), com fechamento por tempo (FASE-03-18).
    gates: crate::gameplay::GateBook,
    /// PvP ligado neste canal/mapa (papel do canal × mapa, decidido na borda).
    pvp: bool,
    /// Regras do movimento autoritativo (velocidade + retângulo do mapa).
    move_rules: MoveRules,
    /// Último movimento aceito por entidade (ms do processo): base do orçamento.
    last_move: HashMap<u32, u64>,
    /// Destino de caminhada pendente por índice de entidade (`_MSG_Action` longe).
    ///
    /// O cliente manda **um** destino por clique; o personagem anda até lá nos ticks, um
    /// orçamento por vez — sem teleporte e sem depender de novos pacotes do cliente.
    destinations: HashMap<u32, (Entity, Position)>,
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
            skins: HashMap::new(),
            scene_ids: HashMap::new(),
            free_player_ids: Vec::new(),
            free_mob_ids: Vec::new(),
            next_player_scene: 1,
            next_mob_scene: MAX_USER,
            scene_generation: 0,
            gates: crate::gameplay::GateBook::default(),
            pvp: false,
            move_rules: MoveRules::new(
                move_tiles_per_second(),
                Position { x: 0, y: 0 },
                Position {
                    x: WORLD_WIDTH - 1,
                    y: WORLD_HEIGHT - 1,
                },
            )?,
            last_move: HashMap::new(),
            destinations: HashMap::new(),
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

    /// Liga/desliga o PvP deste mundo (decisão da borda: papel do canal × mapa).
    pub fn set_pvp(&mut self, pvp: bool) {
        self.pvp = pvp;
    }

    /// CP vivo da entidade (teste/observância).
    #[must_use]
    pub fn cp_of(&self, entity: Entity) -> Option<i16> {
        self.live.get(&entity.index).map(|state| state.cp)
    }

    /// Reserva o id de cena da entidade no espaço do legado.
    ///
    /// `actor = true` pega da faixa de jogador (`1..MAX_USER`), senão da faixa de
    /// criatura (`MAX_USER..MAX_MOB`). Id já reservado devolve o mesmo (idempotente).
    /// `None` = espaço esgotado; quem chamou desfaz o spawn.
    fn alloc_scene_id(&mut self, entity: Entity, actor: bool) -> Option<u16> {
        if let Some(id) = self.scene_ids.get(&entity.index) {
            return Some(*id);
        }
        let id = if actor {
            match self.free_player_ids.pop() {
                Some(id) => id,
                None => {
                    let id = self.next_player_scene;
                    if id >= MAX_USER {
                        return None;
                    }
                    self.next_player_scene += 1;
                    id
                }
            }
        } else {
            match self.free_mob_ids.pop() {
                Some(id) => id,
                None => {
                    let id = self.next_mob_scene;
                    if id >= MAX_MOB {
                        return None;
                    }
                    self.next_mob_scene += 1;
                    id
                }
            }
        };
        self.scene_ids.insert(entity.index, id);
        Some(id)
    }

    /// Devolve o id de cena ao espaço de onde ele veio.
    fn release_scene_id(&mut self, entity: Entity) {
        let Some(id) = self.scene_ids.remove(&entity.index) else {
            return;
        };
        if id < MAX_USER {
            self.free_player_ids.push(id);
        } else {
            self.free_mob_ids.push(id);
        }
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
        // Entidade viva = mudança de cena (o cliente precisa de `_MSG_CreateMob`).
        self.scene_generation = self.scene_generation.wrapping_add(1);
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
        self.skins.remove(&entity.index);
        self.release_scene_id(entity);
        let saiu = self.store.despawn(entity);
        if saiu {
            // Saída = mudança de cena (o cliente precisa de `_MSG_RemoveMob`).
            self.scene_generation = self.scene_generation.wrapping_add(1);
        }
        saiu
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
                    // Id de cena no espaço de criatura: jogadores ficam abaixo de
                    // `MAX_USER`. Sem id livre o NPC não entra (nunca com id de jogador).
                    if self.alloc_scene_id(entity, false).is_none() {
                        self.despawn(entity);
                        report.rejected += 1;
                        continue;
                    }
                    self.npcs.insert(id, NpcLive { entity, spec });
                    report.spawned += 1;
                }
                Err(_) => report.rejected += 1,
            }
        }
        report
    }

    /// Cena atual: NPCs e personagens vivos, com a posição que o motor conhece, ordenados por id.
    ///
    /// A ordem determinística importa para o delta da borda: duas leituras iguais devolvem a
    /// mesma lista, então a sessão só emite o que realmente mudou.
    #[must_use]
    /// Foto da cena **visível** a até `radius` tiles de `center` (Chebyshev).
    ///
    /// Inclui o próprio personagem (distância 0) e só as criaturas dentro do raio — é o
    /// conjunto que a sessão entrega, em vez dos 6 110 NPCs do mapa inteiro.
    pub fn scene_around(&self, center: Position, radius: u32) -> SceneSnapshot {
        let mut cena = self.scene_snapshot();
        cena.mobs.retain(|mob| {
            mob.position
                .x
                .abs_diff(center.x)
                .max(mob.position.y.abs_diff(center.y))
                <= radius
        });
        cena
    }

    pub fn scene_snapshot(&self) -> SceneSnapshot {
        let mut mobs: Vec<SceneMob> = Vec::with_capacity(self.npcs.len() + self.live.len());
        for npc in self.npcs.values() {
            let Some(position) = self.store.position_of(npc.entity) else {
                continue;
            };
            let Some(id) = self.scene_ids.get(&npc.entity.index).copied() else {
                continue;
            };
            mobs.push(SceneMob {
                id,
                actor: false,
                npc_id: npc.spec.id,
                name: fixed_name(&npc.spec.name),
                position,
                level: npc.spec.stats.level,
                hp: npc.spec.stats.hp,
                max_hp: npc.spec.stats.max_hp,
                attack: npc.spec.stats.attack,
                defense: npc.spec.stats.defense,
                merchant: u8::try_from(npc.spec.merchant).unwrap_or(0),
                chaos_rate: crate::personagens::CP_NEUTRO,
                attack_run: npc.spec.attack_run,
                direction: npc.spec.direction,
                equip: npc.spec.equip,
                guild: 0,
            });
        }
        for (indice, estado) in &self.live {
            let Some(id) = self.scene_ids.get(indice).copied() else {
                continue;
            };
            let skin = self.skins.get(indice).cloned().unwrap_or_default();
            // Stats vêm do armazém (fonte autoritativa); a entidade é achada pelo personagem
            // porque `LiveState` guarda o id do banco, não a entidade.
            let stats = self
                .by_character
                .get(&estado.character_id)
                .and_then(|entity| self.store.stats_of(*entity));
            mobs.push(SceneMob {
                id,
                actor: true,
                npc_id: 0,
                name: skin.name,
                position: Position {
                    x: estado.x,
                    y: estado.y,
                },
                level: estado.level,
                hp: estado.hp,
                max_hp: stats.map_or(estado.hp, |stats| stats.max_hp),
                attack: stats.map_or(0, |stats| stats.attack),
                defense: stats.map_or(0, |stats| stats.defense),
                merchant: 0,
                chaos_rate: wyd_sim::chaos_rate(estado.cp),
                attack_run: skin.attack_run,
                direction: skin.direction,
                equip: skin.equip,
                guild: skin.guild,
            });
        }
        mobs.sort_by_key(|mob| mob.id);
        SceneSnapshot {
            generation: self.scene_generation,
            mobs,
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
            // Sem criatura no ponto: com PvP ligado, o alvo pode ser outro jogador.
            return self.attack_player(attacker, x, y, now_ms);
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
        }
        Ok(Some(AttackOutcome {
            target_npc: npc_id,
            target_scene: self.scene_ids.get(&entity.index).copied().unwrap_or(0),
            template,
            damage: golpe.amount,
            remaining_hp: golpe.remaining_hp,
            killed,
            exp: if killed { exp } else { 0 },
            target_actor: false,
            cp_after: 0,
            cp_delta: 0,
        }))
    }

    /// Golpe em **outro jogador** (PvP) no ponto clicado.
    ///
    /// Só existe com o PvP do canal/mapa ligado e autor com CP acima do piso (`-75`). O
    /// abate desconta o caos do autor (0 para vítima já listada) e devolve a vítima com HP
    /// cheio — sem EXP/item, porque no original o CP é a consequência do kill. O save
    /// periódico leva o CP novo (`LiveState.cp`).
    fn attack_player(
        &mut self,
        attacker: Entity,
        x: i32,
        y: i32,
        now_ms: u64,
    ) -> Result<Option<AttackOutcome>, CoreError> {
        if !self.pvp {
            return Ok(None);
        }
        let atacante_cp = self.live.get(&attacker.index).map_or(0, |state| state.cp);
        if !wyd_sim::pode_iniciar_pk(atacante_cp) {
            return Ok(None);
        }
        // Menor distância e depois menor índice: mesma ordem determinística de `npc_near`.
        let alvo = self
            .by_character
            .values()
            .copied()
            .filter(|entity| *entity != attacker)
            .filter_map(|entity| {
                let posicao = self.store.position_of(entity)?;
                let hp = self.store.stats_of(entity)?.hp;
                let distancia = posicao.x.abs_diff(x).max(posicao.y.abs_diff(y));
                (hp > 0 && distancia <= ATTACK_RANGE_TILES).then_some((
                    distancia,
                    entity.index,
                    entity,
                ))
            })
            .min_by_key(|(distancia, indice, _)| (*distancia, *indice))
            .map(|(_, _, entity)| entity);
        let Some(alvo) = alvo else {
            return Ok(None);
        };
        let golpe = combat_system::attack(
            &mut self.store,
            attacker,
            alvo,
            now_ms,
            ATTACK_RULES,
            &TemplateDamage,
        )?;
        let killed = golpe.remaining_hp == 0;
        let mut cp_after = atacante_cp;
        let mut cp_delta = 0;
        if killed {
            let vitima_cp = self.live.get(&alvo.index).map_or(0, |state| state.cp);
            cp_delta = wyd_sim::penalidade_de_kill(vitima_cp);
            cp_after = wyd_sim::clamp_cp(atacante_cp - cp_delta);
            if let Some(state) = self.live.get_mut(&attacker.index) {
                state.cp = cp_after;
            }
            if let Some(stats) = self.store.stats_of(alvo) {
                self.store.restore_hp(alvo, stats.max_hp);
                if let Some(state) = self.live.get_mut(&alvo.index) {
                    state.hp = stats.max_hp;
                }
            }
        }
        Ok(Some(AttackOutcome {
            target_npc: 0,
            target_scene: self.scene_ids.get(&alvo.index).copied().unwrap_or(0),
            template: String::new(),
            damage: golpe.amount,
            remaining_hp: golpe.remaining_hp,
            killed,
            exp: 0,
            target_actor: true,
            cp_after,
            cp_delta,
        }))
    }

    /// Aplica um passo de movimento autoritativo (`_MSG_Action`, `Route[24]` do legado).
    ///
    /// O cliente manda **destino**; orçamento por tempo, retângulo do mapa e teto de 24 tiles
    /// por pacote são do servidor (`MoveRules`). Aceitar move a entidade no SoA e grava a nova
    /// posição no estado vivo, que é o que vai para o save.
    pub fn move_to(
        &mut self,
        entity: Entity,
        to: Position,
        now_ms: u64,
    ) -> Result<Position, CoreError> {
        let Some(atual) = self.store.position_of(entity) else {
            return Err(CoreError::EntityNotAlive {
                index: entity.index,
                generation: entity.generation,
            });
        };
        if !self.move_rules.contains(to) {
            return Err(CoreError::MoveTooFar {
                distance: movement_system::distance(atual, to),
                allowed: MOVE_MAX_TILES,
            });
        }
        // Primeiro passo da entidade usa 1 s de orçamento; depois, o tempo real desde o
        // último passo aceito (cap de 1 s: o que ficou parado não vira teleporte).
        let anterior = self
            .last_move
            .get(&entity.index)
            .copied()
            .unwrap_or_else(|| now_ms.saturating_sub(1_000));
        let elapsed_ms = now_ms.saturating_sub(anterior).min(1_000);
        // Um clique de 20 tiles é destino, não teleporte: anda o que o orçamento permite
        // agora e o resto nos ticks (`walk_pending`). Recusar o clique inteiro era o bug que
        // travava o personagem no spawn e deixava toda loja de Armia fora de alcance.
        let budget = u32::try_from(self.move_rules.budget(elapsed_ms))
            .unwrap_or(u32::MAX)
            .max(1);
        let alvo = step_toward(atual, to, budget);
        let posicao =
            movement_system::step(&mut self.store, entity, alvo, elapsed_ms, self.move_rules)?;
        if let Some(state) = self.live.get_mut(&entity.index) {
            state.x = posicao.x;
            state.y = posicao.y;
        }
        self.last_move.insert(entity.index, now_ms);
        if posicao == to {
            self.destinations.remove(&entity.index);
        } else {
            self.destinations.insert(entity.index, (entity, to));
        }
        Ok(posicao)
    }

    /// Anda um orçamento por tick até o destino pendente de cada entidade.
    ///
    /// O passo usa o mesmo `movement_system::step` (orçamento por tempo + retângulo do
    /// mapa): destino fora do mapa ou entidade morta cancela a caminhada, sem teleporte.
    fn walk_pending(&mut self, elapsed_ms: u64) {
        if self.destinations.is_empty() {
            return;
        }
        let pendentes: Vec<(u32, Entity, Position)> = self
            .destinations
            .iter()
            .map(|(indice, (entity, destino))| (*indice, *entity, *destino))
            .collect();
        let elapsed_ms = elapsed_ms.min(1_000);
        let budget = u32::try_from(self.move_rules.budget(elapsed_ms))
            .unwrap_or(u32::MAX)
            .max(1);
        for (indice, entity, destino) in pendentes {
            let Some(atual) = self.store.position_of(entity) else {
                self.destinations.remove(&indice);
                continue;
            };
            if atual == destino {
                self.destinations.remove(&indice);
                continue;
            }
            let alvo = step_toward(atual, destino, budget);
            match movement_system::step(&mut self.store, entity, alvo, elapsed_ms, self.move_rules)
            {
                Ok(posicao) => {
                    if let Some(state) = self.live.get_mut(&entity.index) {
                        state.x = posicao.x;
                        state.y = posicao.y;
                    }
                    if posicao == destino {
                        self.destinations.remove(&indice);
                    }
                }
                Err(_) => {
                    self.destinations.remove(&indice);
                }
            }
        }
    }

    /// Quem o personagem clicou em `_MSG_Action`, se houver NPC no ponto clicado.
    ///
    /// Clique no sprite (`CLICK_RANGE_TILES`) resolve o NPC mesmo com o personagem longe:
    /// o `_MSG_Action` também é o destino da caminhada, então a loja/diálogo abre quando ele
    /// chega. Criatura de combate continua exigindo `TALK_RANGE_TILES` — atacar longe é
    /// recusado como antes.
    pub fn talk_at(&self, speaker: Entity, x: i32, y: i32) -> Option<TalkTarget> {
        let posicao = self.store.position_of(speaker)?;
        let distancia = posicao.x.abs_diff(x).max(posicao.y.abs_diff(y));
        let npc = self.npc_near(x, y, CLICK_RANGE_TILES, false)?;
        if distancia > TALK_RANGE_TILES && npc.spec.kind == EntityKind::Monster {
            return None;
        }
        Some(TalkTarget {
            id: npc.spec.id,
            name: npc.spec.name.clone(),
            merchant: npc.spec.merchant,
            exp: npc.spec.exp,
        })
    }

    /// Uso de chave num portão (`_MSG_UpdateItem` do legado): ponto exato + alcance.
    ///
    /// `keys` são os `EF_KEYID` das provas que o personagem carrega (a borda lê o
    /// inventário); a chave só é apagada depois do aceite, pelo `SelectionStore`.
    pub fn use_key_at(
        &mut self,
        speaker: Entity,
        x: i32,
        y: i32,
        keys: &[i32],
        now_ms: u64,
    ) -> Result<crate::gameplay::GateOutcome, crate::gameplay::GateError> {
        use crate::gameplay::{gate_at, GateError, CAMP_GATE_RANGE};
        let portao = gate_at(x, y).ok_or(GateError::NoGate)?;
        let posicao = self
            .store
            .position_of(speaker)
            .ok_or(GateError::OutOfRange)?;
        if posicao.x.abs_diff(x).max(posicao.y.abs_diff(y)) > CAMP_GATE_RANGE as u32 {
            return Err(GateError::OutOfRange);
        }
        self.gates.open(portao, keys, now_ms)
    }

    /// Portão aberto agora (`now_ms` do processo).
    #[must_use]
    pub fn gate_open(&self, gate: i16, now_ms: u64) -> bool {
        self.gates.is_open(gate, now_ms)
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
    /// A `skin` é a aparência do `STRUCT_MOB` do login: sem ela o cliente desenharia um
    /// personagem sem nome nem equipamento.
    pub fn enter(
        &mut self,
        character: &Character,
        skin: ActorSkin,
    ) -> Result<Entered, EnterWorldError> {
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
        // Faixa de jogador (`< MAX_USER`): é o número que vai em `ClientID`/`MobID`.
        let Some(scene_id) = self.alloc_scene_id(entity, true) else {
            self.despawn(entity);
            return Err(EnterWorldError::SceneFull);
        };
        let state = LiveState {
            character_id: character.id,
            level: stats.level,
            cp: i16::try_from(character.cp).unwrap_or(0).clamp(-75, 75),
            exp: character.exp,
            hp: stats.hp,
            mp: stats.mp,
            x,
            y,
        };
        self.live.insert(entity.index, state);
        self.skins.insert(entity.index, skin);
        self.by_character.insert(character.id, entity);
        Ok(Entered { entity, scene_id })
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
            Command::Enter {
                character,
                skin,
                reply,
            } => {
                let _ = reply.send(self.enter(&character, *skin));
            }
            Command::Leave { entity, reply } => {
                let _ = reply.send(self.leave(entity));
            }
            Command::Cp { entity, reply } => {
                let _ = reply.send(self.cp_of(entity));
            }
            Command::Snapshots { reply } => {
                let _ = reply.send(self.snapshots());
            }
            Command::Npcs { reply } => {
                let _ = reply.send(self.scene_snapshot());
            }
            Command::SceneAround {
                speaker,
                radius,
                reply,
            } => {
                let cena = self
                    .store
                    .position_of(speaker)
                    .map(|centro| self.scene_around(centro, radius));
                let _ = reply.send(cena);
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
            Command::UseKey {
                speaker,
                x,
                y,
                keys,
                now_ms,
                reply,
            } => {
                let _ = reply.send(self.use_key_at(speaker, x, y, &keys, now_ms));
            }
            Command::Move {
                entity,
                x,
                y,
                now_ms,
                reply,
            } => {
                let _ = reply.send(self.move_to(entity, Position { x, y }, now_ms));
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
        self.walk_pending(elapsed_ms);
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
/// Nome do fio (`MobName[16]`): corta em 16 bytes sem quebrar UTF-8 no meio de um caractere.
fn fixed_name(nome: &str) -> [u8; 16] {
    let mut out = [0u8; 16];
    let mut usado = 0;
    for caractere in nome.chars() {
        let mut buf = [0u8; 4];
        let bytes = caractere.encode_utf8(&mut buf).as_bytes();
        if usado + bytes.len() > out.len() {
            break;
        }
        out[usado..usado + bytes.len()].copy_from_slice(bytes);
        usado += bytes.len();
    }
    out
}

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

    /// Encarna o personagem com a aparência do login; erro é recusa tipada (nada de entidade
    /// parcial).
    pub fn enter(
        &self,
        character: &Character,
        skin: &ActorSkin,
    ) -> Result<Entered, EnterWorldError> {
        self.call(
            |reply| Command::Enter {
                character: Box::new(character.clone()),
                skin: Box::new(skin.clone()),
                reply,
            },
            COMMAND_TIMEOUT,
        )
        .unwrap_or(Err(EnterWorldError::Unavailable))
    }

    /// CP vivo da entidade (`/cp` do chat).
    #[must_use]
    pub fn cp_of(&self, entity: Entity) -> Option<i16> {
        self.call(|reply| Command::Cp { entity, reply }, COMMAND_TIMEOUT)
            .flatten()
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

    /// Cena vigente do motor (NPCs e personagens com posição autoritativa + geração).
    #[must_use]
    pub fn scene(&self) -> SceneSnapshot {
        self.call(|reply| Command::Npcs { reply }, COMMAND_TIMEOUT)
            .unwrap_or_default()
    }

    /// Cena visível em volta de `speaker`; `None` = entidade fora do mundo.
    #[must_use]
    pub fn scene_around(&self, speaker: Entity, radius: u32) -> Option<SceneSnapshot> {
        self.call(
            |reply| Command::SceneAround {
                speaker,
                radius,
                reply,
            },
            COMMAND_TIMEOUT,
        )
        .flatten()
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

    /// Abre portão com a chave do inventário (posição autoritativa validada no tick).
    pub fn use_key(
        &self,
        speaker: Entity,
        x: i32,
        y: i32,
        keys: &[i32],
        now_ms: u64,
    ) -> Result<crate::gameplay::GateOutcome, crate::gameplay::GateError> {
        self.call(
            |reply| Command::UseKey {
                speaker,
                x,
                y,
                keys: keys.to_vec(),
                now_ms,
                reply,
            },
            COMMAND_TIMEOUT,
        )
        .unwrap_or(Err(crate::gameplay::GateError::NoGate))
    }

    /// Paso de movimento pedido por `_MSG_Action`: o tick confere orçamento e mapa.
    pub fn move_to(
        &self,
        entity: Entity,
        x: i32,
        y: i32,
        now_ms: u64,
    ) -> Result<Position, CoreError> {
        match self.call(
            |reply| Command::Move {
                entity,
                x,
                y,
                now_ms,
                reply,
            },
            COMMAND_TIMEOUT,
        ) {
            Some(resultado) => resultado,
            None => Err(CoreError::EntityNotAlive {
                index: entity.index,
                generation: entity.generation,
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

/// Ponto intermediário a até `tiles` de `from` na direção de `to` (Chebyshev).
fn step_toward(from: Position, to: Position, tiles: u32) -> Position {
    let limite = i32::try_from(tiles).unwrap_or(i32::MAX);
    Position {
        x: from.x + (to.x - from.x).clamp(-limite, limite),
        y: from.y + (to.y - from.y).clamp(-limite, limite),
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
    fn pvp_abaixa_o_cp_do_autor_e_restaura_a_vitima() {
        let mut engine = engine();
        engine.set_pvp(true);
        let autor = personagem(1, 100, 100);
        let vitima = Character {
            hp: 1,
            ..personagem(2, 101, 100)
        };
        let atacante = engine.enter(&autor, ActorSkin::default()).expect("entra");
        let alvo = engine.enter(&vitima, ActorSkin::default()).expect("entra");
        let golpe = engine
            .attack_at(atacante.entity, 101, 100, 1_000)
            .expect("consulta")
            .expect("jogador no alcance");
        assert!(golpe.target_actor, "o alvo era jogador, não NPC");
        assert!(golpe.killed, "HP 1 vira abate");
        assert_eq!(golpe.cp_delta, 2, "vítima neutra rende 2 de caos");
        assert_eq!(engine.cp_of(atacante.entity), Some(-2));
        assert_eq!(
            engine.store().stats_of(alvo.entity).map(|stats| stats.hp),
            Some(100),
            "vítima volta com HP cheio"
        );
    }

    #[test]
    fn pvp_no_piso_do_cp_nao_atinge_jogador() {
        let mut engine = engine();
        engine.set_pvp(true);
        let autor = Character {
            cp: -75,
            ..personagem(1, 100, 100)
        };
        let atacante = engine.enter(&autor, ActorSkin::default()).expect("entra");
        engine
            .enter(
                &Character {
                    hp: 1,
                    ..personagem(2, 101, 100)
                },
                ActorSkin::default(),
            )
            .expect("entra");
        assert!(
            engine
                .attack_at(atacante.entity, 101, 100, 1_000)
                .expect("consulta")
                .is_none(),
            "CP -75 não inicia PK"
        );
    }

    #[test]
    fn pvp_desligado_nao_golpeia_jogador() {
        let mut engine = engine();
        let atacante = engine
            .enter(&personagem(1, 100, 100), ActorSkin::default())
            .expect("entra");
        engine
            .enter(
                &Character {
                    hp: 1,
                    ..personagem(2, 101, 100)
                },
                ActorSkin::default(),
            )
            .expect("entra");
        assert!(
            engine
                .attack_at(atacante.entity, 101, 100, 1_000)
                .expect("consulta")
                .is_none(),
            "PvP desligado não golpeia jogador"
        );
    }

    #[test]
    fn clique_longe_e_destino_de_caminhada_e_nao_recusa() {
        let mut engine = engine();
        let personagem = personagem(1, 100, 100);
        let entered = engine
            .enter(&personagem, ActorSkin::default())
            .expect("entra no mundo");
        let destino = Position { x: 120, y: 100 };
        let primeiro = engine
            .move_to(entered.entity, destino, 1_000)
            .expect("destino longe é aceito (o bug era recusar)");
        assert_eq!(
            primeiro,
            Position { x: 104, y: 100 },
            "anda só o orçamento do tick, sem teleporte"
        );
        for _ in 0..10 {
            engine.advance(1_000);
        }
        assert_eq!(
            engine.store().position_of(entered.entity),
            Some(destino),
            "os ticks levam até o destino clicado"
        );
    }

    #[test]
    fn clique_no_sprite_do_npc_longe_devolve_alvo() {
        let mut engine = engine();
        let personagem = personagem(1, 100, 100);
        let entered = engine
            .enter(&personagem, ActorSkin::default())
            .expect("entra no mundo");
        engine.sync_npcs(vec![npc(7, "Aylin", 140, 100, 0)]);
        let alvo = engine
            .talk_at(entered.entity, 140, 100)
            .expect("clique no sprite resolve o NPC mesmo longe");
        assert_eq!(alvo.id, 7);
        assert!(
            engine.talk_at(entered.entity, 300, 300).is_none(),
            "clique no vazio não vira alvo"
        );
    }

    #[test]
    fn cena_por_proximidade_corta_o_que_esta_longe() {
        let mut engine = engine();
        engine.sync_npcs(vec![
            npc(1, "Perto", 2096, 2096, 0),
            npc(2, "Longe", 2000, 2000, 0),
            npc(3, "Borda", 2096 + 24, 2096 - 24, 0),
        ]);
        let cena = engine.scene_around(Position { x: 2096, y: 2096 }, 24);
        let nomes: Vec<String> = cena.mobs.iter().map(|mob| mob.name_str()).collect();
        assert!(nomes.iter().any(|nome| nome == "Perto"), "centro entra");
        assert!(
            nomes.iter().any(|nome| nome == "Borda"),
            "borda do raio entra"
        );
        assert!(
            !nomes.iter().any(|nome| nome == "Longe"),
            "fora do raio não entra"
        );
        assert_eq!(cena.mobs.len(), 2);
        // O snapshot completo continua disponível para o harness.
        assert_eq!(engine.scene_snapshot().mobs.len(), 3);
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
        let entrada = engine
            .enter(&personagem, ActorSkin::default())
            .expect("entrada válida");
        assert_eq!(engine.players(), 1);
        assert_eq!(engine.entities(), 1);
        let posicao = engine.store().position_of(entrada.entity).expect("vivo");
        assert_eq!((posicao.x, posicao.y), (300, 400));

        let estado = engine.leave(entrada.entity).expect("estado final");
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
        engine
            .enter(&vivo, ActorSkin::default())
            .expect("primeira entrada");
        assert_eq!(
            engine.enter(&vivo, ActorSkin::default()),
            Err(EnterWorldError::AlreadyInWorld { character_id: 9 })
        );

        let mut quebrado = personagem(10, 10, 10);
        quebrado.hp = 200; // acima do max_hp
        assert!(matches!(
            engine.enter(&quebrado, ActorSkin::default()),
            Err(EnterWorldError::InvalidState(_))
        ));
        let mut fora = personagem(11, -5, 10);
        fora.pos_x = -5;
        assert!(matches!(
            engine.enter(&fora, ActorSkin::default()),
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
        let jogador = engine
            .enter(&personagem(1, 42, 40), ActorSkin::default())
            .expect("entrada");

        let alvo = engine
            .talk_at(jogador.entity, 40, 40)
            .expect("NPC no alcance");
        assert_eq!(
            (alvo.id, alvo.name.as_str(), alvo.merchant),
            (7, "Aylin", 1)
        );

        // `TargetX/TargetY` longe do personagem não abre diálogo nenhum.
        assert_eq!(engine.talk_at(jogador.entity, 90, 90), None);
        // Ponto sem NPC no alcance também não.
        assert_eq!(engine.talk_at(jogador.entity, 50, 50), None);
    }

    #[test]
    fn abate_usa_a_exp_do_template_e_npc_sem_combate_nao_e_alvo() {
        let mut engine = engine();
        engine.sync_npcs(vec![
            npc(1_000, "Orc Cavaleiro", 41, 40, 5_000),
            npc(7, "Aylin", 70, 70, 0),
        ]);
        let jogador = engine
            .enter(&personagem(1, 40, 40), ActorSkin::default())
            .expect("entrada");

        // `Npc` de loja nunca é alvo de `_MSG_Attack`: clicar nele não acha criatura.
        assert_eq!(
            engine
                .attack_at(jogador.entity, 70, 70, 1_000)
                .expect("sem erro"),
            None,
            "NPC sem EXP não entra em combate"
        );

        // O monstro cai no golpe do servidor e libera a EXP que o template declara.
        let golpe = engine
            .attack_at(jogador.entity, 41, 40, 1_000)
            .expect("sem erro")
            .expect("monstro no alcance");
        assert!(golpe.killed);
        assert_eq!((golpe.target_npc, golpe.exp), (1_000, 5_000));
        assert_eq!(golpe.remaining_hp, 0);

        // Morte tira a criatura do mundo: não sobra cadáver no mapa nem alvo para um
        // segundo golpe, e a cena publica a remoção pela geração nova.
        assert_eq!(
            engine
                .attack_at(jogador.entity, 41, 40, 9_000)
                .expect("sem erro"),
            None,
            "o NPC abatido deixa de existir no mundo"
        );
        assert_eq!(engine.npc_actors(), 1, "só o NPC de loja continua vivo");
        let cena = engine.scene_snapshot();
        // Personagem + NPC de loja: o monstro abatido saiu da cena.
        assert_eq!(cena.mobs.len(), 2);
        assert!(cena.mobs.iter().any(|mob| mob.actor));
        assert_eq!(cena.mobs.iter().filter(|mob| !mob.actor).count(), 1);
        assert!(
            cena.mobs
                .iter()
                .all(|mob| mob.name_str() != "Orc Cavaleiro"),
            "o abate tira o monstro da vista"
        );
    }

    #[test]
    fn handle_da_borda_entra_sai_e_fotografa() {
        let (mut thread, handle) = EngineThread::spawn(engine(), 10).expect("thread do tick");
        let personagem = personagem(21, 100, 200);
        let entrada = handle
            .enter(&personagem, &ActorSkin::default())
            .expect("entrada pelo handle");
        let fotos = handle.snapshots();
        assert_eq!(fotos.len(), 1);
        assert_eq!(fotos[0].character_id, 21);
        let estado = handle.leave(entrada.entity).expect("saída pelo handle");
        assert_eq!(estado.character_id, 21);
        assert!(handle.snapshots().is_empty());
        thread.stop_and_join();
    }

    fn skin(nome: &[u8], guild: u16) -> ActorSkin {
        let mut name = [0u8; 16];
        name[..nome.len()].copy_from_slice(nome);
        let mut equip = [0u16; 16];
        equip[0] = 831;
        equip[1] = 1201;
        ActorSkin {
            name,
            equip,
            guild,
            attack_run: 82,
            direction: 3,
        }
    }

    #[test]
    fn personagem_entra_na_cena_com_nome_equip_e_id_proprio() {
        let mut engine = engine();
        engine.sync_npcs(vec![npc(7, "Aylin", 40, 40, 0)]);
        let jogador = engine
            .enter(&personagem(1, 42, 40), skin(b"Buffy", 4))
            .expect("entrada");

        let cena = engine.scene_snapshot();
        assert_eq!(cena.mobs.len(), 2, "NPC + personagem na mesma cena");
        let ator = cena
            .mobs
            .iter()
            .find(|mob| mob.actor)
            .expect("personagem na cena");
        assert_eq!(ator.id, jogador.scene_id);
        assert!(
            ator.id < MAX_USER,
            "jogador vive na faixa `< MAX_USER` (o cliente decide personagem por id)"
        );
        assert!(
            cena.mobs
                .iter()
                .filter(|mob| !mob.actor)
                .all(|mob| mob.id >= MAX_USER),
            "criatura vive na faixa `>= MAX_USER`"
        );
        assert_eq!(ator.name_str(), "Buffy");
        assert_eq!(ator.equip[0], 831, "aparência vem do STRUCT_MOB do login");
        assert_eq!(ator.guild, 4);
        assert_eq!((ator.level, ator.hp, ator.max_hp), (10, 90, 100));
        assert_eq!((ator.position.x, ator.position.y), (42, 40));
        assert_eq!(ator.merchant, 0, "jogador não é NPC de loja");

        // Espaço de id único: o cliente indexa por `MobID`, então NPC e jogador não podem
        // compartilhar número (o NPC usa o índice da entidade, não o id do banco).
        let ids: std::collections::BTreeSet<u16> = cena.mobs.iter().map(|mob| mob.id).collect();
        assert_eq!(ids.len(), cena.mobs.len());
        assert!(
            cena.mobs.iter().all(|mob| mob.id != 7),
            "id de npc_definition não vira MobID"
        );
    }

    #[test]
    fn saida_do_personagem_sai_da_cena_e_sobe_a_geracao() {
        let mut engine = engine();
        let jogador = engine
            .enter(&personagem(1, 10, 10), skin(b"Buffy", 0))
            .expect("entrada");
        let antes = engine.scene_snapshot();
        assert_eq!(antes.mobs.len(), 1);

        engine.leave(jogador.entity).expect("estado final");
        let depois = engine.scene_snapshot();
        assert!(depois.mobs.is_empty(), "quem saiu não fica na cena");
        assert!(
            depois.generation > antes.generation,
            "a saída precisa acordar o delta das outras sessões"
        );
    }

    #[test]
    fn plan_scene_emite_entrada_saida_e_respeita_o_teto() {
        let mut engine = engine();
        engine.sync_npcs(vec![npc(1, "Aylin", 40, 40, 0), npc(2, "Orc", 41, 40, 10)]);
        let cena = engine.scene_snapshot();
        let ids: Vec<u16> = cena.mobs.iter().map(|mob| mob.id).collect();

        // Sessão nova: tudo entra, nada sai.
        let vazio = std::collections::BTreeMap::new();
        let plano = plan_scene(&vazio, &cena, 64);
        assert_eq!(plano.enter.len(), 2);
        assert!(plano.leave.is_empty());
        assert!(plano.complete);

        // Sessão que já viu tudo: delta vazio (nada é reenviado).
        let vistos: std::collections::BTreeMap<u16, bool> =
            ids.iter().map(|id| (*id, false)).collect();
        let plano = plan_scene(&vistos, &cena, 64);
        assert!(plano.enter.is_empty() && plano.leave.is_empty());
        assert!(plano.complete);

        // Id que sumiu do mundo vira `RemoveMob` fora de vista; personagem vira logout.
        let mut antigos = vistos.clone();
        antigos.insert(9_999, false);
        antigos.insert(9_998, true);
        let plano = plan_scene(&antigos, &cena, 64);
        assert_eq!(
            plano.leave,
            vec![(9_998, REMOVE_LOGOUT), (9_999, REMOVE_OUT_OF_VIEW)]
        );
        assert!(plano.enter.is_empty());

        // Teto de 1: sai o que precisa sair, entra um, e o delta fica incompleto.
        let plano = plan_scene(&antigos, &cena, 1);
        assert_eq!(plano.leave.len(), 1);
        assert!(plano.enter.is_empty());
        assert!(!plano.complete, "teto atingido não avança a geração");
    }

    #[test]
    fn teto_de_cena_por_varredura_valida_o_ambiente() {
        assert_eq!(
            scene_budget_from_value(None),
            Ok(DEFAULT_SCENE_MAX_PER_REFRESH)
        );
        assert_eq!(
            scene_budget_from_value(Some("  ")),
            Ok(DEFAULT_SCENE_MAX_PER_REFRESH)
        );
        assert_eq!(scene_budget_from_value(Some("1")), Ok(1));
        assert_eq!(scene_budget_from_value(Some("4096")), Ok(4_096));
        for invalido in ["0", "4097", "abc", "-1"] {
            assert!(
                scene_budget_from_value(Some(invalido)).is_err(),
                "{invalido} deveria ser recusado"
            );
        }
    }

    #[test]
    fn nome_do_fio_corta_em_16_bytes_sem_quebrar_utf8() {
        assert_eq!(&fixed_name("Buffy")[..5], b"Buffy");
        assert_eq!(fixed_name("Buffy")[5], 0);
        // 'á' ocupa 2 bytes: 8 cabem em 16, o 9º não pode entrar partido.
        let bytes = fixed_name(&"á".repeat(9));
        assert_eq!(&bytes[..16], "á".repeat(8).as_bytes());
    }
}
