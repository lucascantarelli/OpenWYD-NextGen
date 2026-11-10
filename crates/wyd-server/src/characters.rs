//! Seleção de personagem do canal (FASE-08-2): `_MSG_CreateCharacter`,
//! `_MSG_DeleteCharacter` e `_MSG_CharacterLogin` traduzidos em operações de `characters`.
//!
//! A conta **nunca** vem do corpo do pacote: ela é a que abriu a sessão no
//! `_MSG_AccountLogin` (`cpsock::handshake`), e slot/nome só escolhem **entre** os personagens
//! daquela conta. Criação sem dono e seleção de personagem alheio não têm caminho por aqui.
//!
//! O corpo dos pacotes de resposta (`_MSG_CNFNewCharacter`, `_MSG_CNFCharacterLogin`,
//! `_MSG_CNFDeleteCharacter`) é UNVERIFIED (`docs/arquitetura/protocolo.md` §6): o
//! `SELCHAR`/`STRUCT_MOB` provisório de `personagens.rs` é o mesmo já usado no
//! `CNFAccountLogin`; este módulo só decide o **resultado**. A criação responde com o roster
//! real (o antigo reenviava a seleção inteira), nunca com sucesso inventado.
//!
//! Valores iniciais **não** são default do schema: vêm de `BaseSIDCHM` e dos incrementos
//! por classe de `origin/main:Source/Code/Basedef.cpp` (porte Go em
//! `tmserver/internal/level/level.go`). No nível 1, MORTAL: atributos = base da classe,
//! HP/MP máximos = base + 1 × incremento, 5 pontos de atributo e 3 de skill.
//!
//! O resto da criação é cópia do `BaseMob` da classe, como o `CFileDB.cpp` fazia
//! (`memcpy(mob, &g_pBaseSet[cls], sizeof(STRUCT_MOB))`): ouro, ponto de save (spawn) e
//! aparência/equipamento inicial saem do template real, nunca de constante inventada.

use std::sync::Arc;

use wyd_core::Entity;
use wyd_db::{
    Accounts, Character, Characters, DbError, ItemRow, PgBackend, Progression, SkillRule,
    SkillRules, MAX_SLOTS, SCORE_BONUS_BULK, SHORT_SKILL_EMPTY, SHORT_SKILL_LEN, SKILL_TIER_COIN,
    SKILL_TIER_POSITIONS, SPECIAL_LINEAGE_SKILL, SPECIAL_MAX, SPECIAL_MAX_LINEAGE, SPECIAL_SLOTS,
};
use wyd_protocol::ApplyBonus;
use wyd_sim::{grant_exp, ExpTable, LevelState};

use crate::personagens::{kill_mark_row, template_items, template_mob, Mob};
use crate::world::{EngineHandle, SaveSink};

/// Tamanho do campo de nome no fio: `char[16]` em `_MSG_CreateCharacter`/`_MSG_DeleteCharacter`.
pub const MAX_NAME_BYTES: usize = 16;

/// Última classe do 7662: 0–3 (TransKnight, Foema, BeastMaster, Huntress, na ordem de
/// `docs/wyd/classes.md` §1). O **índice** numérico de cada classe é hipótese até a captura;
/// o que é fato é a faixa (quatro classes) — fora dela a criação é recusada.
pub const MAX_CLASS: i32 = 3;

/// Nível inicial de todo personagem novo (MORTAL).
const STARTING_LEVEL: i64 = 1;
/// `BaseSIDCHM[classe][Str, Int, Dex, Con, HP, MP]` — `Basedef.cpp:44`.
const BASE_SIDCHM: [[i64; 6]; 4] = [
    [8, 4, 7, 6, 80, 45],  // TransKnight
    [5, 8, 5, 5, 60, 65],  // Foema
    [6, 6, 9, 5, 70, 55],  // BeastMaster
    [8, 9, 13, 6, 75, 60], // Huntress
];
/// `g_pIncrementHp` / `g_pIncrementMp` — `Basedef.cpp:63-64`.
const INCREMENT_HP: [i64; 4] = [3, 1, 1, 2];
const INCREMENT_MP: [i64; 4] = [1, 3, 2, 1];
/// `BASE_GetBonusScorePoint` no MORTAL: 5 pontos por nível (`Basedef.cpp:920`).
const SCORE_POINTS_PER_LEVEL: i64 = 5;
/// `BASE_GetBonusSkillPoint`: 3 pontos de skill por nível (`Basedef.cpp:852`).
const SKILL_POINTS_PER_LEVEL: i64 = 3;
/// `SpecialBonus += 2` a cada nível (`CMob::CheckGetLevel`).
const SPECIAL_POINTS_PER_LEVEL: i64 = 2;
/// A partir deste nível **interno** (`Level@0` do `STRUCT_MOB`) o level-up dá 4 pontos de
/// skill em vez de 3: `if (MOB.BaseScore.Level >= 200) SkillBonus += 4`.
const SKILL_BONUS_FOUR_FROM: i64 = 200;

/// `g_pNextLevel[MAX_LEVEL+202]` (`main:Source/Code/Basedef.cpp:625`) — a curva de EXP do
/// MORTAL, transcrita do porte Go (`main:tmserver/internal/level/nextlevel.go`), que a
/// confere com a captura `captura-wyd-levelup.md`.
///
/// Índice = nível **interno** do legado (`0` = personagem recém-criado, o `Level@44` do
/// `BaseMob`). O banco guarda nível 1-based, então `exp_for_level(banco.level)` lê
/// `LEVEL_CURVE[banco.level - 1]` — a mesma conta de `MOB.Exp >= g_pNextLevel[cur + 1]` no
/// `CMob::CheckGetLevel`.
const LEVEL_CURVE: [i64; 401] = [
    0, 500, 1124, 1826, 2610, 3480, 4440, 5494, 6646, 7900, 9260, 10893, 12817, 15050, 17610,
    20515, 23783, 27432, 31480, 35945, 40845, 46251, 52187, 58677, 65745, 73415, 81711, 90657,
    100277, 110595, 121635, 133647, 146671, 160747, 175915, 192215, 209687, 228371, 248307, 269535,
    292095, 316151, 341751, 368943, 397775, 428295, 460551, 494591, 530463, 568215, 607895, 649715,
    693731, 739999, 788575, 839515, 892875, 948711, 1007079, 1068035, 1131635, 1198670, 1269230,
    1343405, 1421285, 1502960, 1588520, 1678055, 1771655, 1869410, 1971410, 2078255, 2190055,
    2306920, 2428960, 2556285, 2689005, 2827230, 2971070, 3120635, 3276035, 3438521, 3608249,
    3785375, 3970055, 4162445, 4362701, 4570979, 4787435, 5012225, 5245505, 5488163, 5740379,
    6002333, 6274205, 6556175, 6848423, 7151129, 7464473, 7788635, 8123795, 8460174, 8797774,
    9136597, 9476645, 9817920, 10160424, 10504159, 10849127, 11195330, 11542770, 11892311,
    12243959, 12597720, 12953600, 13311605, 13671741, 14034014, 14398430, 14764995, 15133715,
    15508850, 15890450, 16278565, 16673245, 17074540, 17482500, 17897175, 18318615, 18746870,
    19181990, 19625811, 20078403, 20539836, 21010180, 21489505, 21977881, 22475378, 22982066,
    23498015, 24023295, 24559110, 25105558, 25662737, 26230745, 26809680, 27399640, 28000723,
    28613027, 29236650, 29871690, 30517485, 31174125, 31841700, 32520300, 33210015, 33910935,
    34623150, 35346750, 36081825, 36828465, 37587867, 38360139, 39145389, 39943725, 40755255,
    41580087, 42418329, 43270089, 44135475, 45014595, 45904870, 46806370, 47719165, 48643325,
    49578920, 50526020, 51484695, 52455015, 53437050, 54430870, 55439542, 56463162, 57501826,
    58555630, 59624670, 60709042, 61808842, 62924166, 64055110, 65201770, 66366010, 67547930,
    68747630, 69965210, 71200770, 72454410, 73726230, 75016330, 76324810, 77651770, 78985354,
    80325578, 81672458, 83026010, 84386250, 85753194, 87126858, 88507258, 89894410, 91288330,
    92693002, 94108458, 95534730, 96971850, 98419850, 99878762, 101348618, 102829450, 104321290,
    105824170, 107352234, 108905674, 110484682, 112089450, 113720170, 115377034, 117060234,
    118769962, 120506410, 122269770, 124065890, 125895058, 127757562, 129653690, 131583730,
    133547970, 135546698, 137580202, 139648770, 141752690, 143928178, 146176386, 148498466,
    150895570, 153368850, 155919458, 158548546, 161257266, 164046770, 166918210, 169956978,
    173167682, 176554930, 180123330, 205345890, 209100050, 212902550, 216753470, 220652890,
    224600890, 228597550, 232642950, 236737170, 240880290, 245072390, 249313550, 253603850,
    257943370, 262332190, 266770390, 271258050, 275795250, 280382070, 285018590, 289904810,
    295042730, 300434350, 306081670, 311986690, 318151410, 324577830, 331267950, 338223770,
    345447290, 354039310, 364049830, 375528850, 388526370, 403092390, 419276910, 437129930,
    456701450, 476272970, 495844490, 515416010, 534987530, 554559050, 574130570, 593702090,
    613273610, 632845130, 652416650, 671988170, 691559690, 711131210, 730702730, 750274250,
    769845770, 789417290, 808988810, 828560330, 848131850, 867703370, 887274890, 906846410,
    926417930, 945989450, 965560970, 985132490, 1004704010, 1024275530, 1043847050, 1063418570,
    1082990090, 1102561610, 1122133130, 1141704650, 1161276170, 1180847690, 1200419210, 1222705731,
    1244995262, 1267288477, 1289622601, 1311966887, 1334333102, 1356724650, 1379151914, 1401651370,
    1424151231, 1448674779, 1473220997, 1497782544, 1522364697, 1546957043, 1571581919, 1596243411,
    1620925875, 1645647464, 1670373305, 1710373305, 1770373305, 1870373305, 2000000000, 2039000000,
    2078000000, 2117000000, 2156000000, 2195000000, 2234000000, 2273000000, 2312000000, 2351000000,
    2390000000, 2429000000, 2468000000, 2507000000, 2546000000, 2585000000, 2624000000, 2663000000,
    2702000000, 2741000000, 2780000000, 2819000000, 2858000000, 2897000000, 2936000000, 3000000000,
    3043000000, 3086000000, 3129000000, 3172000000, 3215000000, 3258000000, 3301000000, 3344000000,
    3387000000, 3430000000, 3473000000, 3516000000, 3559000000, 3602000000, 3645000000, 3688000000,
    3731000000, 3774000000, 3817000000, 4000000000, 4100000000,
];

/// Teto do banco: `MAX_LEVEL` + 1 (o índice `399` do legado), porque o banco é 1-based.
const MAX_DB_LEVEL: u16 = 400;

/// A curva do legado como [`ExpTable`] — sem cópia em memória nem validação em runtime:
/// a tabela é constante e o teste `curva_de_exp_do_legado_e_crescente` a valida.
struct LegacyExp;

impl ExpTable for LegacyExp {
    fn exp_for_level(&self, level: u16) -> Option<u64> {
        LEVEL_CURVE
            .get(usize::from(level).checked_sub(1)?)
            .copied()
            .and_then(|exp| u64::try_from(exp).ok())
    }

    fn max_level(&self) -> u16 {
        MAX_DB_LEVEL
    }
}

/// Ficha inicial derivada da classe, sem estado persistido ainda.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct StartingScores {
    strength: i64,
    intelligence: i64,
    dexterity: i64,
    constitution: i64,
    hp_max: i64,
    mp_max: i64,
    score_bonus: i64,
    skill_bonus: i64,
}

/// `BASE_GetHpMp` no nível 1: base da classe + incremento do nível + 2 por ponto de
/// `Con`/`Int` acima da base (que é zero num personagem recém-criado).
fn starting_scores(class: i64) -> StartingScores {
    let cls = usize::try_from(class)
        .ok()
        .filter(|c| *c < BASE_SIDCHM.len())
        .unwrap_or(0);
    let base = BASE_SIDCHM[cls];
    StartingScores {
        strength: base[0],
        intelligence: base[1],
        dexterity: base[2],
        constitution: base[3],
        hp_max: base[4] + STARTING_LEVEL * INCREMENT_HP[cls],
        mp_max: base[5] + STARTING_LEVEL * INCREMENT_MP[cls],
        score_bonus: STARTING_LEVEL * SCORE_POINTS_PER_LEVEL,
        skill_bonus: STARTING_LEVEL * SKILL_POINTS_PER_LEVEL,
    }
}

/// Recusas de seleção — todas viram `*Fail` no fio e uma linha de log com motivo.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum SelectionError {
    /// Processo respondeu sem porta de personagem (teste sem banco): nunca inventa sucesso.
    Unavailable,
    /// Slot fora de `0..=MAX_SLOTS`.
    Slot { slot: i32 },
    /// Nome vazio, com byte fora de ASCII imprimível ou acima de `MAX_NAME_BYTES`.
    Name,
    /// Classe fora de `0..=MAX_CLASS`.
    Class { class: i32 },
    /// A conta do login não tem linha em `accounts` (criação não inventa dono).
    Account(String),
    /// Personagem inexistente/removido para essa conta+slot, ou nome divergente no slot.
    NotFound,
    /// O mundo recusou a entrada (duplicado, estado inválido, motor indisponível).
    World(String),
    /// Nome já usado por outro personagem vivo (UNIQUE parcial do banco).
    Conflict,
    /// Banco indisponível ou dado corrompido.
    Db(String),
    /// Pacote com tamanho diferente do layout portado.
    Layout(String),
    /// `_MSG_ApplyBonus` recusado (sem ponto, classe errada, pré-requisito, morto).
    Bonus(String),
    /// Ganho de EXP recusado (valor não positivo).
    Exp(String),
    /// Conta com os quatro slots ocupados (criação não sobrescreve personagem).
    Full,
}

impl core::fmt::Display for SelectionError {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::Unavailable => write!(f, "processo sem porta de personagem"),
            Self::Slot { slot } => write!(f, "slot {slot} fora de 0..={MAX_SLOTS}"),
            Self::Name => write!(f, "nome inválido (1..={MAX_NAME_BYTES} ASCII imprimível)"),
            Self::Class { class } => write!(f, "classe {class} fora de 0..={MAX_CLASS}"),
            Self::Account(motivo) => write!(f, "conta sem cadastro: {motivo}"),
            Self::NotFound => write!(f, "personagem inexistente para esta conta"),
            Self::World(motivo) => write!(f, "mundo recusou a entrada: {motivo}"),
            Self::Conflict => write!(f, "nome de personagem já usado"),
            Self::Db(motivo) => write!(f, "personagem sem lastro no banco: {motivo}"),
            Self::Layout(motivo) => write!(f, "payload inválido: {motivo}"),
            Self::Bonus(motivo) => write!(f, "bônus recusado: {motivo}"),
            Self::Exp(motivo) => write!(f, "EXP recusada: {motivo}"),
            Self::Full => write!(f, "conta sem slot livre em 0..={MAX_SLOTS}"),
        }
    }
}

impl std::error::Error for SelectionError {}

/// Porta de seleção de **um** canal: mesmo pool e mesmo runtime do handshake (ADR-0021/0022).
pub struct SelectionStore {
    backend: Arc<PgBackend>,
    runtime: Arc<tokio::runtime::Runtime>,
    world: Option<WorldBridge>,
}

/// Ponte para o mundo vivo (FASE-03-8): entrada/saída por comando e save pela fila.
struct WorldBridge {
    handle: EngineHandle,
    saves: Arc<dyn SaveSink>,
}

/// Roster com o inventário persistido de cada personagem (o `STRUCT_SELCHAR` do login).
pub type RosterItems = Vec<(Character, Vec<ItemRow>)>;

/// Foto do `_MSG_AccountLogin`: personagens, cofre (`vault_items`) e `Coin` da conta.
#[derive(Debug, Clone, Default)]
pub struct LoginSnapshot {
    pub roster: RosterItems,
    pub vault: Vec<ItemRow>,
    pub coin: i64,
}

/// Personagem escolhido no `_MSG_CharacterLogin`: estado persistido completo.
///
/// `progression` traz atributos/pontos/`LearnedSkill` do banco (tabelas `character_scores`
/// e `character_learned_skill`): o `STRUCT_MOB` do login e o `_MSG_ApplyBonus` leem daqui,
/// nunca do template da classe.
#[derive(Debug, Clone)]
pub struct Selected {
    pub character: Character,
    pub items: Vec<ItemRow>,
    pub short_skill: [u8; SHORT_SKILL_LEN],
    pub progression: Progression,
}

impl SelectionStore {
    /// Monta a porta sobre o pool já conectado pelo processo.
    #[must_use]
    pub fn new(backend: Arc<PgBackend>, runtime: Arc<tokio::runtime::Runtime>) -> Self {
        Self {
            backend,
            runtime,
            world: None,
        }
    }

    /// Liga a seleção ao mundo do canal (sem porta, a seleção só responde como antes).
    #[must_use]
    pub fn with_world(mut self, handle: EngineHandle, saves: Arc<dyn SaveSink>) -> Self {
        self.world = Some(WorldBridge { handle, saves });
        self
    }

    /// Encarna o personagem no mundo. `Ok(None)` = processo sem mundo (teste/legado).
    pub fn enter_world(&self, character: &Character) -> Result<Option<Entity>, SelectionError> {
        match &self.world {
            None => Ok(None),
            Some(bridge) => bridge
                .handle
                .enter(character)
                .map(Some)
                .map_err(|error| SelectionError::World(error.to_string())),
        }
    }

    /// Retira a entidade do mundo e enfileira o save do estado final.
    pub fn leave_world(&self, entity: Option<Entity>) {
        let (Some(bridge), Some(entity)) = (&self.world, entity) else {
            return;
        };
        if let Some(state) = bridge.handle.leave(entity) {
            bridge.saves.save(state);
        }
    }

    /// Cria o personagem no slot pedido, com o dono vindo do login.
    pub fn create(
        &self,
        login: &str,
        slot: i32,
        name: &[u8],
        class: i32,
    ) -> Result<Character, SelectionError> {
        let pedido = check_slot(slot)?;
        let class = check_class(class)?;
        let name = check_name(name)?;
        let account_id = self.account_id(login)?;
        let slot = self.free_slot(account_id, pedido)?;
        let inicio = starting_scores(class);
        // Template real da classe: ouro, ponto de save e equipamento inicial.
        let modelo = template_mob(class);
        let character = Character {
            // `id` é do banco (`BIGSERIAL`); o INSERT não o envia.
            id: 0,
            account_id,
            slot,
            name,
            class,
            level: STARTING_LEVEL,
            exp: 0,
            hp: inicio.hp_max,
            mp: inicio.mp_max,
            hp_max: inicio.hp_max,
            mp_max: inicio.mp_max,
            strength: inicio.strength,
            intelligence: inicio.intelligence,
            dexterity: inicio.dexterity,
            constitution: inicio.constitution,
            score_bonus: inicio.score_bonus,
            skill_bonus: inicio.skill_bonus,
            gold: i64::from(modelo.coin),
            pos_x: i64::from(modelo.spx),
            pos_y: i64::from(modelo.spy),
            ..Character::default()
        };
        let criado = self
            .runtime
            .block_on(self.backend.create_character(&character))
            .map_err(map_db)?;
        // Inventário inicial (`BaseMob` + kill mark) e `ShortSkill`: daqui em diante o banco
        // é a fonte da verdade e o template só é copiado nesta criação.
        self.seed_inventory(&criado);
        let _ = self.short_skill(&criado);
        Ok(criado)
    }

    /// Remove (soft-delete) o personagem do slot, exigindo o nome que veio no pacote.
    ///
    /// A senha de 12 bytes do `_MSG_DeleteCharacter` **não** é conferida: ela pertence à
    /// autenticação de conta, que ainda não existe nesta entrega (o próprio
    /// `_MSG_AccountLogin` só tem a versão validada). Quem remove é o dono da sessão; a
    /// conferência da senha entra junto com a autenticação, nunca antes por palpite.
    pub fn delete(&self, login: &str, slot: i32, name: &[u8]) -> Result<Character, SelectionError> {
        let slot = check_slot(slot)?;
        let name = check_name(name)?;
        let account_id = self.account_id(login)?;
        let character = self.find(account_id, slot)?;
        if character.name != name {
            // O cliente manda nome + slot: divergência é recusa, não "apaga o que estiver lá".
            return Err(SelectionError::NotFound);
        }
        self.runtime
            .block_on(self.backend.soft_delete(character.id))
            .map_err(map_db)?;
        Ok(character)
    }

    /// Personagem vivo da conta naquele slot (o que o `_MSG_CharacterLogin` vai encarnar).
    pub fn select(&self, login: &str, slot: i32) -> Result<Character, SelectionError> {
        let slot = check_slot(slot)?;
        let account_id = self.account_id(login)?;
        self.find(account_id, slot)
    }

    /// Personagens vivos da conta, por slot (foto que o seletor do cliente desenha).
    pub fn roster(&self, login: &str) -> Result<Vec<Character>, SelectionError> {
        let account_id = self.account_id(login)?;
        self.runtime
            .block_on(self.backend.list_by_account(account_id))
            .map_err(map_db)
    }

    /// Roster com o inventário de cada personagem (o que o seletor desenha).
    pub fn roster_inventory(&self, login: &str) -> Result<RosterItems, SelectionError> {
        Ok(self
            .roster(login)?
            .into_iter()
            .map(|character| {
                let items = self.inventory(&character);
                (character, items)
            })
            .collect())
    }

    /// Foto do login da conta: roster com inventário, cofre e `Coin`.
    pub fn login_snapshot(&self, login: &str) -> Result<LoginSnapshot, SelectionError> {
        let account_id = self.account_id(login)?;
        let roster = self.roster_inventory(login)?;
        let vault = self
            .runtime
            .block_on(self.backend.vault_items(account_id))
            .map_err(map_db)?;
        // Conta sem linha em `account_wallet` responde `Coin = 0`, nunca valor inventado.
        let coin = self
            .runtime
            .block_on(self.backend.account_coin(account_id))
            .map_err(map_db)?
            .unwrap_or(0);
        Ok(LoginSnapshot {
            roster,
            vault,
            coin,
        })
    }

    /// Personagem + inventário + `ShortSkill` + progressão do banco (o corpo do login).
    pub fn select_snapshot(&self, login: &str, slot: i32) -> Result<Selected, SelectionError> {
        let character = self.select(login, slot)?;
        let items = self.inventory(&character);
        let short_skill = self.short_skill(&character);
        let progression = self
            .runtime
            .block_on(self.backend.progression(character.id))
            .map_err(map_db)?;
        Ok(Selected {
            character,
            items,
            short_skill,
            progression,
        })
    }

    /// Inventário persistido do personagem.
    ///
    /// Personagem anterior a esta entrega (sem linha em `character_items`) materializa o
    /// inventário inicial **uma vez**, do template, e passa a ser lido do banco como os
    /// demais — o caminho de leitura nunca deriva equipamento do template.
    fn inventory(&self, character: &Character) -> Vec<ItemRow> {
        let rows = self
            .runtime
            .block_on(self.backend.items(character.id))
            .unwrap_or_default();
        if !rows.is_empty() {
            return rows;
        }
        self.seed_inventory(character);
        self.runtime
            .block_on(self.backend.items(character.id))
            .unwrap_or_default()
    }

    /// Grava o inventário inicial (`BaseMob` + kill mark) sem sobrescrever slot ocupado.
    fn seed_inventory(&self, character: &Character) {
        let mut rows = template_items(character.class);
        rows.push(kill_mark_row());
        if let Err(error) = self
            .runtime
            .block_on(self.backend.insert_items(character.id, &rows))
        {
            println!(
                "level=warn event=inventory result=seed_failed character={} reason={error}",
                character.id
            );
        }
    }

    /// `ShortSkill` persistido; personagem sem linha recebe `0xFF` (o `memset` da criação).
    fn short_skill(&self, character: &Character) -> [u8; SHORT_SKILL_LEN] {
        let vazio = [SHORT_SKILL_EMPTY; SHORT_SKILL_LEN];
        match self
            .runtime
            .block_on(self.backend.short_skill(character.id))
        {
            Ok(Some(skill)) => skill,
            Ok(None) => {
                if let Err(error) = self
                    .runtime
                    .block_on(self.backend.set_short_skill(character.id, &vazio))
                {
                    println!(
                        "level=warn event=short_skill result=seed_failed character={} reason={error}",
                        character.id
                    );
                }
                vazio
            }
            Err(error) => {
                println!(
                    "level=warn event=short_skill result=read_failed character={} reason={error}",
                    character.id
                );
                vazio
            }
        }
    }

    /// Slot da criação: honra o pedido se estiver livre, senão o primeiro livre de `0..=3`.
    ///
    /// O cliente manda o slot que **ele** desenha como vazio, e a tela dele pode estar
    /// dessincronizada (delete antigo respondia sem corpo). Nunca reutiliza slot ocupado: pedido
    /// em slot cheio desloca para o primeiro livre; conta com os quatro ocupados é recusa
    /// explícita. O índice `characters_account_slot_unique` continua sendo a palavra final
    /// contra corrida entre dois processos.
    fn free_slot(&self, account_id: i64, pedido: i64) -> Result<i64, SelectionError> {
        let ocupados: Vec<i64> = self
            .runtime
            .block_on(self.backend.list_by_account(account_id))
            .map_err(map_db)?
            .into_iter()
            .map(|character| character.slot)
            .collect();
        if !ocupados.contains(&pedido) {
            return Ok(pedido);
        }
        (0..=MAX_SLOTS)
            .find(|slot| !ocupados.contains(slot))
            .ok_or(SelectionError::Full)
    }

    fn account_id(&self, login: &str) -> Result<i64, SelectionError> {
        let login = login.trim();
        if login.is_empty() {
            return Err(SelectionError::Account("login vazio".to_string()));
        }
        let conta = self
            .runtime
            .block_on(self.backend.find_by_login(login))
            .map_err(|error| SelectionError::Account(error.to_string()))?;
        Ok(conta.id)
    }

    fn find(&self, account_id: i64, slot: i64) -> Result<Character, SelectionError> {
        let personagens = self
            .runtime
            .block_on(self.backend.list_by_account(account_id))
            .map_err(map_db)?;
        personagens
            .into_iter()
            .find(|character| character.slot == slot)
            .ok_or(SelectionError::NotFound)
    }
}

/// `MAX_USER` (`Basedef.h:116`): primeiro índice de NPC/mob na faixa de ids do `TargetID`.
const MAX_USER: i32 = 1000;
/// `MAX_MOB` (`Basedef.h:167`).
const MAX_MOB: i32 = 25_000;
/// `_MSG_ApplyBonus` tipo 2 cobre `Detail` 5000..=5095 (`g_pItemList` das skills).
const SKILL_ITEM_FIRST: i32 = 5000;
const SKILL_ITEM_LAST: i32 = 5095;

/// Resultado aceito de `_MSG_ApplyBonus`: o `Mob` já com a progressão gravada.
#[derive(Debug, Clone)]
pub struct BonusApplied {
    pub mob: Mob,
    pub progression: Progression,
    pub detail: i32,
}

/// Resultado de um ganho de EXP: o estado gravado, o `Mob` já atualizado e quantos níveis
/// foram ganhos de uma vez (zero quando a EXP só acumulou).
#[derive(Debug, Clone)]
pub struct ExpGain {
    pub mob: Mob,
    pub progression: Progression,
    pub levels: u32,
}

impl SelectionStore {
    /// `_MSG_ApplyBonus`: distribui ponto de atributo (tipo 0) ou aprende skill (tipo 2).
    ///
    /// O cliente só escolhe **onde** gastar: a quantidade vem do banco (`Progression`), a
    /// validação de skill do catálogo (`skill_catalog` + `item_catalog`) e a gravação é uma
    /// transação. `Con`/`Int` somam 2 de máximo cada, como `_MSG_ApplyBonus.cpp`; o
    /// `CurrentScore` reenviado ao cliente é o do `Mob` atualizado.
    pub fn apply_bonus(
        &self,
        selected: &Selected,
        packet: &ApplyBonus,
    ) -> Result<BonusApplied, SelectionError> {
        let personagem = &selected.character;
        let mut progression = self
            .runtime
            .block_on(self.backend.progression(personagem.id))
            .map_err(map_db)?;
        let mut mob = Mob::from_persisted(personagem, &selected.items);
        apply_progression(&mut mob, &progression);
        // Guarda do legado: só gasta ponto com o personagem vivo.
        if mob.current_score.hp <= 0 {
            return Err(SelectionError::Bonus("personagem sem HP".to_string()));
        }
        match packet.bonus_type {
            0 => score_points(&mut progression, i32::from(packet.detail))?,
            1 => special_points(&mut progression, i32::from(packet.detail))?,
            2 => {
                let regra = self.skill_rule(i64::from(packet.detail))?;
                learn_skill(&mob, &mut progression, packet, &regra)?;
            }
            outro => {
                return Err(SelectionError::Bonus(format!(
                    "BonusType {outro} desconhecido"
                )))
            }
        }
        self.runtime
            .block_on(self.backend.store_progression(personagem.id, &progression))
            .map_err(map_db)?;
        apply_progression(&mut mob, &progression);
        Ok(BonusApplied {
            mob,
            progression,
            detail: i32::from(packet.detail),
        })
    }

    /// Ganho de EXP (`CMob::CheckGetLevel`): acumula, sobe quantos níveis couberem e grava
    /// tudo numa transação (`characters` + `character_scores` + `character_special`).
    ///
    /// Por nível ganho, como no MORTAL do legado: `SkillBonus += 3` (4 a partir do nível
    /// interno 200), `SpecialBonus += 2`, `ScoreBonus += 5` e `MaxHp/MaxMp += g_pIncrementHp/Mp`
    /// da classe, com HP/MP cheios. `Ac` não entra: não existe coluna para ele ainda (o
    /// `BASE_GetBonusScorePoint` do legado também reequilibra atributos já gastos; aqui o
    /// ponto é somado — o teto por nível continua sendo validado em `score_points`).
    ///
    /// A EXP do banco é **acumulada** (limiar do nível + excedente), como `MOB.Exp`; o
    /// excedente no teto é retido, nunca descartado.
    pub fn gain_exp(&self, selected: &Selected, amount: i64) -> Result<ExpGain, SelectionError> {
        if amount <= 0 {
            return Err(SelectionError::Exp(format!("valor não positivo: {amount}")));
        }
        let personagem = &selected.character;
        let mut progression = self
            .runtime
            .block_on(self.backend.progression(personagem.id))
            .map_err(map_db)?;
        let curva = LegacyExp;
        let nivel = u16::try_from(progression.level).unwrap_or(1).max(1);
        let limiar = curva.exp_for_level(nivel).unwrap_or(0);
        let dentro = u64::try_from(
            progression
                .exp
                .saturating_sub(i64::try_from(limiar).unwrap_or(i64::MAX)),
        )
        .unwrap_or(0);
        let antes = LevelState {
            level: nivel,
            exp_into_level: dentro,
        };
        let depois = grant_exp(antes, u64::try_from(amount).unwrap_or(u64::MAX), &curva);
        let classe = usize::try_from(personagem.class)
            .ok()
            .filter(|classe| *classe < INCREMENT_HP.len())
            .unwrap_or(0);
        for novo in (antes.level + 1)..=depois.level {
            let interno = i64::from(novo) - 1;
            progression.skill_bonus += if interno >= SKILL_BONUS_FOUR_FROM {
                4
            } else {
                SKILL_POINTS_PER_LEVEL
            };
            progression.special_bonus += SPECIAL_POINTS_PER_LEVEL;
            progression.score_bonus += SCORE_POINTS_PER_LEVEL;
            progression.hp_max =
                (progression.hp_max + INCREMENT_HP[classe]).min(i64::from(i32::MAX));
            progression.mp_max =
                (progression.mp_max + INCREMENT_MP[classe]).min(i64::from(i32::MAX));
        }
        if depois.level > antes.level {
            progression.hp = progression.hp_max;
            progression.mp = progression.mp_max;
        }
        progression.level = i64::from(depois.level);
        progression.exp = i64::try_from(
            curva
                .exp_for_level(depois.level)
                .unwrap_or(0)
                .saturating_add(depois.exp_into_level),
        )
        .unwrap_or(i64::MAX);
        self.runtime
            .block_on(self.backend.store_progression(personagem.id, &progression))
            .map_err(map_db)?;
        let mut mob = Mob::from_persisted(personagem, &selected.items);
        apply_progression(&mut mob, &progression);
        mob.exp = progression.exp;
        Ok(ExpGain {
            mob,
            levels: u32::from(depois.level.saturating_sub(antes.level)),
            progression,
        })
    }

    fn skill_rule(&self, item_index: i64) -> Result<SkillRule, SelectionError> {
        let regra = self
            .runtime
            .block_on(self.backend.skill_rule(item_index))
            .map_err(map_db)?;
        regra.ok_or_else(|| SelectionError::Bonus(format!("skill {item_index} fora do catálogo")))
    }
}

/// `BonusType 0`: gasta ponto de atributo e soma 2 de máximo em `Con`/`Int`.
///
/// `ScoreBonus >= 300` gasta 100 pontos de uma vez (quirk do legado para distribuir em
/// bloco). O teto é o `short` do `STRUCT_SCORE`; estourar é recusa, nunca truncamento.
fn score_points(progression: &mut Progression, detail: i32) -> Result<(), SelectionError> {
    if !(0..=3).contains(&detail) {
        return Err(SelectionError::Bonus(format!(
            "atributo {detail} fora de 0..=3"
        )));
    }
    if progression.score_bonus <= 0 {
        return Err(SelectionError::Bonus(
            "sem ponto de atributo para gastar".to_string(),
        ));
    }
    let pontos = if progression.score_bonus >= SCORE_BONUS_BULK {
        100
    } else {
        1
    };
    let teto = i64::from(i16::MAX);
    let atributo = match detail {
        0 => progression.strength,
        1 => progression.intelligence,
        2 => progression.dexterity,
        _ => progression.constitution,
    };
    if atributo + pontos > teto {
        return Err(SelectionError::Bonus(format!(
            "atributo {detail} passaria de {teto}"
        )));
    }
    progression.score_bonus -= pontos;
    match detail {
        0 => progression.strength += pontos,
        1 => {
            progression.intelligence += pontos;
            progression.mp_max += 2 * pontos;
        }
        2 => progression.dexterity += pontos,
        _ => {
            progression.constitution += pontos;
            progression.hp_max += 2 * pontos;
        }
    }
    Ok(())
}

/// `BonusType 1`: gasta ponto de maestria em `Special[detail]` (`_MSG_ApplyBonus.cpp`).
///
/// Teto duplo do legado: `3 × (nível + 1)` dividido por 2 (o jogador só alcança metade do
/// “nível de maestria”) e 200 por atributo — 255 quando a skill de linha correspondente
/// (`Detail` 1/2/3 → bits 7/15/23 do `LearnedSkill`) já foi aprendida. O bônus celestial de
/// `+3 × 400` não se aplica: `ClassMaster` CELESTIAL/SCELESTIAL ainda não existe na stack.
fn special_points(progression: &mut Progression, detail: i32) -> Result<(), SelectionError> {
    if !(0..i32::try_from(SPECIAL_SLOTS).unwrap_or(4)).contains(&detail) {
        return Err(SelectionError::Bonus(format!(
            "maestria {detail} fora de 0..{SPECIAL_SLOTS}"
        )));
    }
    if progression.special_bonus <= 0 {
        return Err(SelectionError::Bonus(
            "sem ponto de maestria para gastar".to_string(),
        ));
    }
    let indice = usize::try_from(detail).unwrap_or(0);
    let nivel = progression.level;
    let limite_nivel = 3 * (nivel + 1);
    let mut teto = SPECIAL_MAX;
    let linhagem = SPECIAL_LINEAGE_SKILL[indice];
    if linhagem != usize::MAX && progression.learned_skill & (1i64 << linhagem) != 0 {
        teto = SPECIAL_MAX_LINEAGE;
    }
    let atual = progression.special[indice];
    if atual >= (limite_nivel >> 1) {
        return Err(SelectionError::Bonus(format!(
            "maestria {detail} no teto do nível: {atual} de {} (3 × (nível + 1) ÷ 2)",
            limite_nivel >> 1
        )));
    }
    if atual >= teto {
        return Err(SelectionError::Bonus(format!(
            "maestria {detail} no teto de {teto}"
        )));
    }
    progression.special[indice] = atual + 1;
    progression.special_bonus -= 1;
    Ok(())
}

/// `BonusType 2`: aprende a skill do `Detail` (5000 + `classe * 24 + pos`).
///
/// Porte fiel de `_MSG_ApplyBonus.cpp`: classe da skill, alvo NPC (`TargetID` na faixa
/// `MAX_USER..MAX_MOB`), custo de `SkillPoint`, skill de linha (uma por classe, exige as 7
/// anteriores e cobra 50 000 000 de ouro) e pré-requisitos de nível/maestria do
/// `g_pItemList` (aqui `item_catalog`).
fn learn_skill(
    mob: &Mob,
    progression: &mut Progression,
    packet: &ApplyBonus,
    regra: &SkillRule,
) -> Result<(), SelectionError> {
    let detail = i32::from(packet.detail);
    if !(SKILL_ITEM_FIRST..=SKILL_ITEM_LAST).contains(&detail) {
        return Err(SelectionError::Bonus(format!(
            "skill {detail} fora de {SKILL_ITEM_FIRST}..={SKILL_ITEM_LAST}"
        )));
    }
    let indice = detail - SKILL_ITEM_FIRST;
    let slots = i32::try_from(wyd_db::SKILL_SLOTS).unwrap_or(24);
    let classe = i64::from(indice / slots);
    let posicao = usize::try_from(indice % slots).unwrap_or(0);
    if classe != i64::from(mob.class) {
        return Err(SelectionError::Bonus(format!(
            "skill da classe {classe}, personagem da classe {}",
            mob.class
        )));
    }
    let alvo = i32::from(packet.target_id);
    if !(MAX_USER..MAX_MOB).contains(&alvo) {
        return Err(SelectionError::Bonus(format!(
            "alvo {alvo} fora da faixa de NPC"
        )));
    }
    if regra.skill_point <= 0 {
        return Err(SelectionError::Bonus(format!(
            "skill {detail} sem custo no catálogo"
        )));
    }
    if progression.skill_bonus < regra.skill_point {
        return Err(SelectionError::Bonus(format!(
            "pontos de skill insuficientes: {} < {}",
            progression.skill_bonus, regra.skill_point
        )));
    }
    let bit = 1i64 << posicao;
    if progression.learned_skill & bit != 0 {
        return Err(SelectionError::Bonus("skill já aprendida".to_string()));
    }
    let tier = SKILL_TIER_POSITIONS.contains(&posicao);
    if tier {
        if SKILL_TIER_POSITIONS
            .iter()
            .any(|posicao| progression.learned_skill & (1i64 << posicao) != 0)
        {
            return Err(SelectionError::Bonus(
                "já há skill de linha aprendida".to_string(),
            ));
        }
        let anteriores = (1..=7)
            .filter(|salto| progression.learned_skill & (1i64 << (posicao - salto)) != 0)
            .count();
        if anteriores != 7 {
            return Err(SelectionError::Bonus(
                "aprenda as 7 skills anteriores antes da skill de linha".to_string(),
            ));
        }
        if progression.gold < SKILL_TIER_COIN {
            return Err(SelectionError::Bonus(format!(
                "ouro insuficiente: {} < {SKILL_TIER_COIN}",
                progression.gold
            )));
        }
    }
    if progression.level < regra.req_level {
        return Err(SelectionError::Bonus(format!(
            "nível {} abaixo do exigido {}",
            progression.level, regra.req_level
        )));
    }
    let maestria = mob.current_score.special;
    if i64::from(maestria[1]) < regra.req_int
        || i64::from(maestria[2]) < regra.req_dex
        || i64::from(maestria[3]) < regra.req_con
    {
        return Err(SelectionError::Bonus(
            "maestria insuficiente para a skill".to_string(),
        ));
    }
    progression.learned_skill |= bit;
    progression.skill_bonus -= regra.skill_point;
    if tier {
        progression.gold -= SKILL_TIER_COIN;
    }
    Ok(())
}

/// Sobrepõe o `STRUCT_MOB` com a progressão persistida (fonte da verdade do login e do
/// `_MSG_ApplyBonus`).
pub(crate) fn apply_progression(mob: &mut Mob, progressao: &Progression) {
    let atributo = |valor: i64| i16::try_from(valor).unwrap_or(i16::MAX);
    let vital = |valor: i64| i32::try_from(valor).unwrap_or(i32::MAX);
    let nivel = crate::personagens::wire_level(progressao.level);
    for score in [&mut mob.base_score, &mut mob.current_score] {
        score.level = nivel;
        score.strength = atributo(progressao.strength);
        score.intelligence = atributo(progressao.intelligence);
        score.dexterity = atributo(progressao.dexterity);
        score.constitution = atributo(progressao.constitution);
        score.max_hp = vital(progressao.hp_max);
        score.max_mp = vital(progressao.mp_max);
        for (indice, valor) in progressao.special.iter().enumerate() {
            score.special[indice] = atributo(*valor);
        }
    }
    mob.current_score.hp = vital(progressao.hp);
    mob.current_score.mp = vital(progressao.mp);
    mob.coin = i32::try_from(progressao.gold).unwrap_or(i32::MAX);
    mob.score_bonus = u16::try_from(progressao.score_bonus).unwrap_or(u16::MAX);
    mob.skill_bonus = u16::try_from(progressao.skill_bonus).unwrap_or(u16::MAX);
    mob.special_bonus = u16::try_from(progressao.special_bonus).unwrap_or(u16::MAX);
    mob.learned_skill = i32::try_from(progressao.learned_skill).unwrap_or(0);
}

fn check_slot(slot: i32) -> Result<i64, SelectionError> {
    if (0..=MAX_SLOTS).contains(&i64::from(slot)) {
        Ok(i64::from(slot))
    } else {
        Err(SelectionError::Slot { slot })
    }
}

fn check_class(class: i32) -> Result<i64, SelectionError> {
    if (0..=MAX_CLASS).contains(&class) {
        Ok(i64::from(class))
    } else {
        Err(SelectionError::Class { class })
    }
}

fn check_name(name: &[u8]) -> Result<String, SelectionError> {
    let visivel = wyd_protocol::nul_trimmed(name);
    if visivel.is_empty() || visivel.len() > MAX_NAME_BYTES {
        return Err(SelectionError::Name);
    }
    if !visivel
        .iter()
        .all(|byte| (0x21..=0x7e).contains(byte) && *byte != b' ')
    {
        return Err(SelectionError::Name);
    }
    // ASCII imprimível é UTF-8 válido por construção.
    String::from_utf8(visivel.to_vec()).map_err(|_| SelectionError::Name)
}

fn map_db(error: DbError) -> SelectionError {
    match &error {
        DbError::NotFound { .. } | DbError::NotFoundKey { .. } => SelectionError::NotFound,
        DbError::ConstraintViolation { .. } => SelectionError::Conflict,
        DbError::Transport { .. } => SelectionError::Db(error.to_string()),
        _ => SelectionError::Db(error.to_string()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn slot_e_classe_fora_da_faixa_sao_recusados() {
        assert_eq!(check_slot(0), Ok(0));
        assert_eq!(check_slot(3), Ok(3));
        assert_eq!(check_slot(4), Err(SelectionError::Slot { slot: 4 }));
        assert_eq!(check_slot(-1), Err(SelectionError::Slot { slot: -1 }));
        assert_eq!(check_class(0), Ok(0));
        assert_eq!(check_class(3), Ok(3));
        assert_eq!(check_class(9), Err(SelectionError::Class { class: 9 }));
    }

    #[test]
    fn nome_vem_do_char16_ate_o_nul() {
        let mut campo = [0u8; MAX_NAME_BYTES];
        campo[..4].copy_from_slice(b"Kael");
        assert_eq!(check_name(&campo), Ok("Kael".to_string()));
        // 16 caracteres sem NUL ainda cabem no campo.
        assert_eq!(
            check_name(b"Abcdefghijklmnop"),
            Ok("Abcdefghijklmnop".to_string())
        );
    }

    #[test]
    fn nome_vazio_com_espaco_ou_nao_ascii_e_recusado() {
        assert_eq!(check_name(&[0u8; 16]), Err(SelectionError::Name));
        assert_eq!(check_name(b"Ka el"), Err(SelectionError::Name));
        assert_eq!(check_name(&[0xFF, 0xFE]), Err(SelectionError::Name));
    }

    #[test]
    fn ficha_inicial_vem_da_classe_e_dos_incrementos_do_legado() {
        // BaseSIDCHM do main: TK, FM, BM, HT.
        let esperado = [
            (8, 4, 7, 6, 83, 46),
            (5, 8, 5, 5, 61, 68),
            (6, 6, 9, 5, 71, 57),
            (8, 9, 13, 6, 77, 61),
        ];
        for (classe, (forca, int, dex, con, hp, mp)) in esperado.into_iter().enumerate() {
            let ficha = starting_scores(classe as i64);
            assert_eq!(
                (
                    ficha.strength,
                    ficha.intelligence,
                    ficha.dexterity,
                    ficha.constitution,
                    ficha.hp_max,
                    ficha.mp_max
                ),
                (forca, int, dex, con, hp, mp),
                "classe {classe}"
            );
            assert_eq!(ficha.score_bonus, 5, "5 pontos de atributo no nível 1");
            assert_eq!(ficha.skill_bonus, 3, "3 pontos de skill no nível 1");
        }
    }

    #[test]
    fn curva_de_exp_do_legado_e_crescente_e_cobre_o_teto() {
        // Âncoras de `g_pNextLevel` (Basedef.cpp:625): 0 no recém-criado, 500 no nível 2
        // (banco) e o teto do Mortal em 4 000 000 000.
        assert_eq!(LegacyExp.exp_for_level(1), Some(0));
        assert_eq!(LegacyExp.exp_for_level(2), Some(500));
        assert_eq!(LegacyExp.exp_for_level(40), Some(269_535));
        assert_eq!(LegacyExp.exp_for_level(400), Some(4_000_000_000));
        assert_eq!(LegacyExp.exp_for_level(402), None);
        assert_eq!(LegacyExp.max_level(), MAX_DB_LEVEL);
        assert_eq!(LegacyExp.max_level(), 400, "MAX_LEVEL + 1 no banco 1-based");

        let mut anterior = 0;
        for nivel in 1..=MAX_DB_LEVEL {
            let exp = LegacyExp.exp_for_level(nivel).expect("nível na curva");
            assert!(exp >= anterior, "curva retrocedeu no nível {nivel}");
            anterior = exp;
        }
    }
}
