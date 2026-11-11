//! `STRUCT_SELCHAR` (840 B) e `STRUCT_MOB` (816 B) **exatamente** como o motor antigo,
//! montados a partir do estado persistido e do template `BaseMob` da classe.
//!
//! Layout conferido campo a campo contra `main:Source/Code/Basedef.h` (`STRUCT_SCORE`,
//! `STRUCT_ITEM`, `STRUCT_MOB`, `STRUCT_SELCHAR`, `MSG_DBCNFAccountLogin`,
//! `MSG_CNFNewCharacter`, `MSG_CNFClientCharacterLogin`) e contra o porte Go de
//! `main:tmserver/internal/protocol/{mob,selchar}.go`. Os tamanhos vêm do alinhamento
//! natural do MSVC x86 — **não** é `pack(1)`:
//!
//! - `STRUCT_SCORE` 48 B: `Level@0`, `Ac@4`, `Damage@8`, `Merchant@12`, `AttackRun@13`,
//!   `Direction@14`, `ChaosRate@15`, `MaxHp@16`, `MaxMp@20`, `Hp@24`, `Mp@28`, `Str@32`,
//!   `Int@34`, `Dex@36`, `Con@38`, `Special[4]@40`.
//! - `STRUCT_MOB` 816 B: `MobName@0` (16), `Clan@16`, `Merchant@17`, `Guild@18`, `Class@20`,
//!   `Rsv@22` (padding em 21), `Quest@24` (padding em 25–27), `Coin@28` (int), `Exp@32`
//!   (`long long`, 8 B), `SPX@40`, `SPY@42`, `BaseScore@44`, `CurrentScore@92`,
//!   `Equip[16]@140`, `Carry[64]@268`, `LearnedSkill@780`, `Magic@784`, `ScoreBonus@788`,
//!   `SpecialBonus@790`, `SkillBonus@792`, `Critical@794`, `SaveMana@795`, `SkillBar[4]@796`,
//!   `GuildLevel@800` (padding em 801), `RegenHP@802`, `RegenMP@804`, `Resist[4]@806`, cauda
//!   de alinhamento 810–815.
//! - `STRUCT_SELCHAR` 840 B: `SPX[4]@0`, `SPY[4]@8`, `Name[4][16]@16`, `Score[4]@80`,
//!   `Equip[4][16]@272`, `Guild[4]@784`, `Coin[4]@792`, `Exp[4]@808`.
//!
//! Corpos de pacote (o `HEADER` de 12 bytes fica fora):
//! `MSG_DBCNFAccountLogin` 1996 B (sel@20, Cargo[128]@860, Coin@1884, AccountName@1888);
//! `MSG_CNFNewCharacter` 844 B (sel@4, padding do `STRUCT_SELCHAR` no C);
//! `MSG_CNFClientCharacterLogin` 1820 B (PosX@0, PosY@2, mob@4, Slot@1028, ClientID@1030,
//! Weather@1032, ShortSkill[16]@1034, AccountName em `Unk2[448]`@1500).
//!
//! **Valores iniciais** não são inventados: a criação copia o `BaseMob` da classe, como o
//! `CFileDB.cpp` fazia (`memcpy(mob, &g_pBaseSet[cls], sizeof(STRUCT_MOB))`), e sobrescreve
//! nome, vitais (nível/HP/MP/atributos de `BaseSIDCHM` + incrementos) e pontos livres. A
//! origem de cada campo está em `docs/wyd/classes.md` §1.

use wyd_db::{Character, ItemRow, CARRY_FIRST_SLOT};
use wyd_sim::formula::refine::{encode_sanc_cvalue, EF_SANC};

/// `sizeof(STRUCT_SCORE)`.
pub const SCORE_LEN: usize = 48;
/// `sizeof(STRUCT_ITEM)`.
pub const ITEM_LEN: usize = 8;
/// `MAX_EQUIP`.
pub const MAX_EQUIP: usize = 16;
/// `MAX_CARRY`.
pub const MAX_CARRY: usize = 64;
/// `MOB_PER_ACCOUNT` — slots desenhados pelo seletor.
pub const SLOT_COUNT: usize = 4;
/// `NAME_LENGTH`.
pub const NAME_LEN: usize = 16;
/// `sizeof(STRUCT_SELCHAR)`.
pub const SELCHAR_LEN: usize = 840;
/// `sizeof(STRUCT_MOB)`.
pub const MOB_LEN: usize = 816;
/// `MAX_CARGO` (cofre da conta no `CNFAccountLogin`).
pub const MAX_CARGO: usize = 128;
/// Corpo de `MSG_DBCNFAccountLogin` (2008 − 12).
pub const ACCOUNT_LOGIN_LEN: usize = 1996;
/// Corpo de `MSG_CNFNewCharacter` (856 − 12).
pub const NEW_CHARACTER_LEN: usize = 844;
/// Corpo de `MSG_CNFClientCharacterLogin` (1832 − 12).
pub const CHARACTER_LOGIN_LEN: usize = 1820;
/// `sizeof(ShortSkill)`.
pub const SHORT_SKILL_LEN: usize = 16;
/// `memset(ShortSkill, -1, 16)` na criação (`CFileDB.cpp`).
pub const SHORT_SKILL_EMPTY: u8 = 0xFF;
/// `KILL_MARK` (`Basedef.h:226`): slot do `Carry` onde o legado repõe o item 547.
pub const KILL_MARK_SLOT: usize = 63;
/// Item do kill mark (`ProcessDBMessage.cpp:850`).
pub const KILL_MARK_ITEM: i16 = 547;
/// `EF_AMOUNT` (`ItemEffect.h:118`): quantidade da pilha em `stEffect[0]`.
pub const EF_AMOUNT: u8 = 61;
/// `EF_CURKILL`/`EF_LTOTKILL`/`EF_HTOTKILL` (`ItemEffect.h:135-137`), sem valor.
const KILL_MARK_EFFECTS: [u8; 3] = [75, 76, 77];

/// Rosto padrão da classe, transcrito do `Equip[0]` do `BaseMob` (índice + aparência).
///
/// É aparência de **classe**, não inventário: `STRUCT_SELCHAR` não tem campo `Class` e o
/// cliente descobre a classe pelo rosto (`DBGetSelChar`, `CFileDB.cpp:2636`). Só entra quando
/// o personagem não tem linha no slot 0 — a customização que altera `Equip[0].stEffect`
/// (hoje via `_MSG_UseItem`) ainda não existe e, quando existir, vira linha do banco.
const CLASS_FACE: [(i16, [[u8; 2]; 3]); 4] = [
    (1, [[43, 0], [98, 1], [0, 0]]),
    (11, [[43, 0], [98, 1], [0, 0]]),
    (21, [[43, 0], [99, 0], [98, 21]]),
    (31, [[43, 0], [98, 1], [0, 0]]),
];
/// `strncpy(&sm.Unk2[448], AccountName, 12)` no login: corpo 1052 + 448.
pub const ACCOUNT_NAME_OFFSET: usize = 1500;
/// `strncpy` do legado copia só 12 bytes do nome da conta.
pub const LOGIN_ACCOUNT_NAME_LEN: usize = 12;

/// `ChaosRate` de um personagem **sem PK** — CP 0 deslocado em `+75`.
///
/// Neste cliente `ChaosRate@15` não é stat de combate: é o CP (`−75..+75`) codificado em
/// `0..150`. A função de cor do nome (`WYD.exe` `0x4e2393`) lê o byte de `pMob+0x11e`,
/// limita a `10..150` e interpola a tabela em `0x6188a4` (vermelho → laranja → amarelo →
/// verde → azul → branco); abaixo de `10` cai no ramo de criminoso (nome escuro/vermelho).
/// O próprio cliente grava `0x4B` ao inicializar um mob (`WYD.exe` `0x4d9ae2`), e o `0xCC`
/// do `BaseMob` é lixo de dump — mandar `0` é o que pintava o personagem novo de vermelho.
pub const CP_NEUTRO: u8 = 0x4B;

/// Byte `ChaosRate` que o cliente usa para a cor do nome: `CP + 75` na faixa `0..=150`.
#[must_use]
fn chaos_rate_de(cp: i64) -> u8 {
    wyd_sim::chaos_rate(i16::try_from(cp).unwrap_or(0))
}

/// `HEADER.ID` de `MSG_CNFAccountLogin` (`IDSelChar`).
pub const ID_SELCHAR: u16 = 30002;
/// `HEADER.ID` de `MSG_CNFNewCharacter` (`IDNewCharacter`).
pub const ID_NEW_CHARACTER: u16 = 30001;
/// `HEADER.ID` de `MSG_CNFDeleteCharacter`: mesmo id do seletor (`IDSelChar`), porque o
/// corpo é o `STRUCT_SELCHAR` redesenhado. O id exato do legado não está provado; o valor
/// sai do mesmo domínio dos outros pacotes de seleção, nunca inventado fora dele.
pub const ID_DELETE_CHARACTER: u16 = ID_SELCHAR;
/// `HEADER.ID` de `MSG_CNFCharacterLogin` (`ESCENE_FIELD`, invariante do projeto).
pub const ID_SCENE: u16 = 30000;

/// Offsets de `STRUCT_MOB` (conferidos por teste).
pub const MOB_BASE_SCORE_OFFSET: usize = 44;
pub const MOB_CURRENT_SCORE_OFFSET: usize = 92;
pub const MOB_EQUIP_OFFSET: usize = 140;
pub const MOB_CARRY_OFFSET: usize = 268;
pub const MOB_SKILL_BONUS_OFFSET: usize = 792;
pub const MOB_SKILL_BAR_OFFSET: usize = 796;
pub const MOB_RESIST_OFFSET: usize = 806;

/// Offsets de `STRUCT_SELCHAR`.
pub const SELCHAR_NAME_OFFSET: usize = 16;
pub const SELCHAR_SCORE_OFFSET: usize = 80;
pub const SELCHAR_EQUIP_OFFSET: usize = 272;
pub const SELCHAR_GUILD_OFFSET: usize = 784;
pub const SELCHAR_COIN_OFFSET: usize = 792;
pub const SELCHAR_EXP_OFFSET: usize = 808;

/// Bytes que o dump do motor antigo carrega como lixo de padding do MSVC (0xCC/0xF5).
/// O encoder escreve zero neles; por isso a comparação byte a byte ignora só estas
/// posições (ver o teste `serializacao_reproduz_o_template_fora_o_padding`).
pub const MOB_PADDING: [usize; 11] = [21, 25, 26, 27, 801, 810, 811, 812, 813, 814, 815];

/// Templates de classe reais, extraídos de
/// `main:tmserver/content/DBsrv/run/BaseMob/{TK,FM,BM,HT}` (SHA-256 em `docs/wyd/classes.md`).
const BASE_MOB_TK: &[u8] = include_bytes!("../assets/basemob/TK");
const BASE_MOB_FM: &[u8] = include_bytes!("../assets/basemob/FM");
const BASE_MOB_BM: &[u8] = include_bytes!("../assets/basemob/BM");
const BASE_MOB_HT: &[u8] = include_bytes!("../assets/basemob/HT");

/// Template cru da classe (`0`–`3`), ou `None` fora da faixa.
#[must_use]
pub fn base_mob(class: i64) -> Option<&'static [u8]> {
    match class {
        0 => Some(BASE_MOB_TK),
        1 => Some(BASE_MOB_FM),
        2 => Some(BASE_MOB_BM),
        3 => Some(BASE_MOB_HT),
        _ => None,
    }
}

/// `BASE_ClearMob` + `memcpy(mob, &g_pBaseSet[cls])`: o template da classe decodificado.
///
/// Classe fora da faixa cai no mob zerado — o mesmo resultado de `BASE_ClearMob` sem
/// `BaseSet` carregado. Nunca é valor inventado: é o fallback explícito do motor antigo.
#[must_use]
pub fn template_mob(class: i64) -> Mob {
    base_mob(class)
        .and_then(|cru| Mob::decode(cru).ok())
        .unwrap_or_default()
}

/// Rosto padrão da classe (`Equip[0]` do `BaseMob`): aparência de classe, não inventário.
#[must_use]
pub fn class_face(class: i64) -> Item {
    match usize::try_from(class)
        .ok()
        .and_then(|classe| CLASS_FACE.get(classe))
    {
        Some((index, effects)) => Item {
            index: *index,
            effects: *effects,
        },
        None => Item::default(),
    }
}

/// Linhas de `character_items` do inventário inicial, a partir do template da classe.
///
/// É o `memcpy(mob, &g_pBaseSet[cls], sizeof(STRUCT_MOB))` da criação traduzido para o
/// esquema normalizado: `Equip[i]` → slot `i`, `Carry[i]` → slot `16 + i`, e o `EF_AMOUNT`
/// do template (401/406 = 120) vira `qty`. Slot vazio do template não gera linha.
#[must_use]
pub fn template_items(class: i64) -> Vec<ItemRow> {
    let mob = template_mob(class);
    let mut rows = Vec::new();
    for (slot, item) in mob.equip.iter().enumerate() {
        if item.is_empty() {
            continue;
        }
        rows.push(ItemRow {
            slot: slot as i64,
            item_id: i64::from(item.index),
            qty: 1,
            ..ItemRow::default()
        });
    }
    for (slot, item) in mob.carry.iter().enumerate() {
        if item.is_empty() {
            continue;
        }
        let qty = if item.effects[0][0] == EF_AMOUNT {
            i64::from(item.effects[0][1])
        } else {
            1
        };
        rows.push(ItemRow {
            slot: CARRY_FIRST_SLOT + slot as i64,
            item_id: i64::from(item.index),
            qty,
            ..ItemRow::default()
        });
    }
    rows
}

/// Linha do kill mark: item 547 em `Carry[KILL_MARK]`, que o legado repõe no login.
#[must_use]
pub fn kill_mark_row() -> ItemRow {
    ItemRow {
        slot: CARRY_FIRST_SLOT + KILL_MARK_SLOT as i64,
        item_id: i64::from(KILL_MARK_ITEM),
        qty: 1,
        ..ItemRow::default()
    }
}

/// `Cargo[128]` do `MSG_DBCNFAccountLogin` a partir de `vault_items`.
#[must_use]
pub fn cargo_items(rows: &[ItemRow]) -> Vec<Item> {
    let mut cargo = vec![Item::default(); MAX_CARGO];
    for row in rows {
        if let Ok(slot) = usize::try_from(row.slot) {
            if slot < MAX_CARGO {
                cargo[slot] = Item::from_row(row);
            }
        }
    }
    cargo
}

/// Corpo de `MSG_UpdateEtc` (48 − 12 = 36 B): o `SendEtc` do legado.
pub const UPDATE_ETC_LEN: usize = 36;
/// Corpo de `MSG_UpdateScore` (152 − 12 = 140 B): o `SendScore` do legado.
pub const UPDATE_SCORE_LEN: usize = 140;
/// `MAX_AFFECT` (`Basedef.h:182`).
pub const MAX_AFFECT: usize = 32;
/// `Special[4]` do `MSG_UpdateScore` sai `0xCC` fixo no `SendScore` (`SendFunc.cpp:1274-1277`):
/// é sentinela do legado, **não** a maestria. O valor real de `Special[4]` viaja no
/// `STRUCT_MOB` (`BaseScore`/`CurrentScore`, login e relogin).
const SCORE_SPECIAL_SENTINELA: u8 = 0xCC;

/// `MSG_UpdateEtc` (44 B + alinhamento 8 = 48): pontos livres, EXP, `Learn` e `Coin`.
///
/// É o que o cliente usa para redesenhar os botões de bônus e o ouro. `Hold` fica zero:
/// carga (`extra.Hold`) não é persistida nesta entrega — valor explícito, não estimativa.
#[must_use]
pub fn update_etc_body(mob: &Mob) -> Vec<u8> {
    let mut out = vec![0u8; UPDATE_ETC_LEN];
    out[0..4].copy_from_slice(&0u32.to_le_bytes());
    out[4..12].copy_from_slice(&mob.exp.to_le_bytes());
    out[12..20].copy_from_slice(&i64::from(mob.learned_skill).to_le_bytes());
    out[20..22].copy_from_slice(&mob.score_bonus.to_le_bytes());
    out[22..24].copy_from_slice(&mob.special_bonus.to_le_bytes());
    out[24..26].copy_from_slice(&mob.skill_bonus.to_le_bytes());
    out[26..28].copy_from_slice(&u16::try_from(mob.magic).unwrap_or(u16::MAX).to_le_bytes());
    out[28..32].copy_from_slice(&mob.coin.to_le_bytes());
    out
}

/// `MSG_UpdateScore` (152 B): `CurrentScore` inteiro + HP/MP correntes, como `SendScore`.
///
/// `Affect[32]` sai zerado: a lista de afetos vivos é FASE-10 e o legado a lê de
/// `pMob[conn].Affect`; `Special[4]` é `0xCC` fixo no próprio `SendScore`.
#[must_use]
pub fn update_score_body(mob: &Mob) -> Vec<u8> {
    let mut out = vec![0u8; UPDATE_SCORE_LEN];
    mob.current_score.encode_into(&mut out[0..SCORE_LEN]);
    out[48] = mob.critical;
    out[49] = mob.save_mana;
    out[114..116].copy_from_slice(&mob.guild.to_le_bytes());
    out[116..118].copy_from_slice(&u16::from(mob.guild_level).to_le_bytes());
    for (indice, valor) in mob.resist.iter().enumerate() {
        out[118 + indice] = *valor as u8;
    }
    out[122] = u8::try_from(mob.regen_hp).unwrap_or(u8::MAX);
    out[123] = u8::try_from(mob.regen_mp).unwrap_or(u8::MAX);
    out[124..128].copy_from_slice(&mob.current_score.hp.to_le_bytes());
    out[128..132].copy_from_slice(&mob.current_score.mp.to_le_bytes());
    out[132..136].copy_from_slice(&i32::try_from(mob.magic).unwrap_or(i32::MAX).to_le_bytes());
    out[136..140].copy_from_slice(&[SCORE_SPECIAL_SENTINELA; 4]);
    out
}

/// `STRUCT_ITEM`: `sIndex` + 3 pares efeito/valor.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Item {
    pub index: i16,
    pub effects: [[u8; 2]; 3],
}

impl Item {
    /// Slot vazio (`sIndex == 0`).
    #[must_use]
    pub fn is_empty(&self) -> bool {
        self.index == 0
    }

    /// `STRUCT_ITEM` a partir da linha persistida (`character_items`/`vault_items`).
    ///
    /// `sIndex` é o `item_id`. `stEffect[0]` carrega `EF_AMOUNT` quando a pilha tem mais de
    /// uma unidade — é o `3d 78` (amount 120) que o `BaseMob` traz no 401/406. O kill mark
    /// recebe os três efeitos fixos que o legado grava em `Carry[KILL_MARK]`. Equipamento
    /// sem refino/anct/gema não tem efeito no fio: no `BaseMob`, `Equip[1..6]` só tem
    /// `sIndex`; Ancient/gema continuam estado da linha (FASE-10 codifica no fio).
    #[must_use]
    pub fn from_row(row: &ItemRow) -> Self {
        Self::from_row_with_pity(row, 0)
    }

    /// `STRUCT_ITEM` com o par EF_SANC do refino (`FASE-03-16`).
    ///
    /// O legado empacota nível + piedade no próprio item
    /// (`cValue = nível + 10×piedade`, `Basedef.cpp:2282`); o par entra no primeiro slot
    /// livre ou que já seja sanc (regra do `Bootstrap`, `_MSG_UseItem.cpp:814-840`), sem
    /// deslocar `EF_AMOUNT`. Nível 0 sem piedade não escreve par — o fio fica idêntico ao
    /// de [`Self::from_row`]. `gem` é sempre 0 no caminho de pó (nunca se minta gema).
    #[must_use]
    pub fn from_row_with_pity(row: &ItemRow, pity: u8) -> Self {
        let index = i16::try_from(row.item_id).unwrap_or(0);
        if index == 0 {
            return Self::default();
        }
        if index == KILL_MARK_ITEM {
            return Self {
                index,
                effects: [
                    [KILL_MARK_EFFECTS[0], 0],
                    [KILL_MARK_EFFECTS[1], 0],
                    [KILL_MARK_EFFECTS[2], 0],
                ],
            };
        }
        let mut effects = if row.qty > 1 {
            [
                [EF_AMOUNT, u8::try_from(row.qty).unwrap_or(u8::MAX)],
                [0, 0],
                [0, 0],
            ]
        } else {
            [[0, 0], [0, 0], [0, 0]]
        };
        let level = row.refine.clamp(0, 15) as u8;
        if level > 0 || pity > 0 {
            let par = [EF_SANC, encode_sanc_cvalue(level, pity)];
            if let Some(livre) = effects
                .iter()
                .position(|slot| *slot == [0, 0] || slot[0] == EF_SANC)
            {
                effects[livre] = par;
            }
        }
        Self { index, effects }
    }

    fn decode(buf: &[u8]) -> Self {
        if buf.len() < ITEM_LEN {
            return Self::default();
        }
        let index = i16::from_le_bytes([buf[0], buf[1]]);
        Self {
            index,
            effects: [[buf[2], buf[3]], [buf[4], buf[5]], [buf[6], buf[7]]],
        }
    }

    fn encode_into(&self, out: &mut [u8]) {
        out[0..2].copy_from_slice(&self.index.to_le_bytes());
        for (indice, par) in self.effects.iter().enumerate() {
            let base = 2 + indice * 2;
            out[base] = par[0];
            out[base + 1] = par[1];
        }
    }

    /// Os 8 bytes do `STRUCT_ITEM` no fio (`sIndex` + 3 pares).
    #[must_use]
    pub fn to_bytes(self) -> [u8; ITEM_LEN] {
        let mut out = [0u8; ITEM_LEN];
        self.encode_into(&mut out);
        out
    }
}

/// `STRUCT_SCORE` (48 B).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Default)]
pub struct Score {
    pub level: i32,
    pub ac: i32,
    pub damage: i32,
    pub merchant: u8,
    pub attack_run: u8,
    pub direction: u8,
    pub chaos_rate: u8,
    pub max_hp: i32,
    pub max_mp: i32,
    pub hp: i32,
    pub mp: i32,
    pub strength: i16,
    pub intelligence: i16,
    pub dexterity: i16,
    pub constitution: i16,
    pub special: [i16; 4],
}

impl Score {
    fn decode(buf: &[u8]) -> Self {
        if buf.len() < SCORE_LEN {
            return Self::default();
        }
        Self {
            level: i32::from_le_bytes([buf[0], buf[1], buf[2], buf[3]]),
            ac: i32::from_le_bytes([buf[4], buf[5], buf[6], buf[7]]),
            damage: i32::from_le_bytes([buf[8], buf[9], buf[10], buf[11]]),
            merchant: buf[12],
            attack_run: buf[13],
            direction: buf[14],
            chaos_rate: buf[15],
            max_hp: i32::from_le_bytes([buf[16], buf[17], buf[18], buf[19]]),
            max_mp: i32::from_le_bytes([buf[20], buf[21], buf[22], buf[23]]),
            hp: i32::from_le_bytes([buf[24], buf[25], buf[26], buf[27]]),
            mp: i32::from_le_bytes([buf[28], buf[29], buf[30], buf[31]]),
            strength: i16::from_le_bytes([buf[32], buf[33]]),
            intelligence: i16::from_le_bytes([buf[34], buf[35]]),
            dexterity: i16::from_le_bytes([buf[36], buf[37]]),
            constitution: i16::from_le_bytes([buf[38], buf[39]]),
            special: [
                i16::from_le_bytes([buf[40], buf[41]]),
                i16::from_le_bytes([buf[42], buf[43]]),
                i16::from_le_bytes([buf[44], buf[45]]),
                i16::from_le_bytes([buf[46], buf[47]]),
            ],
        }
    }

    fn encode_into(&self, out: &mut [u8]) {
        out[0..4].copy_from_slice(&self.level.to_le_bytes());
        out[4..8].copy_from_slice(&self.ac.to_le_bytes());
        out[8..12].copy_from_slice(&self.damage.to_le_bytes());
        out[12] = self.merchant;
        out[13] = self.attack_run;
        out[14] = self.direction;
        out[15] = self.chaos_rate;
        out[16..20].copy_from_slice(&self.max_hp.to_le_bytes());
        out[20..24].copy_from_slice(&self.max_mp.to_le_bytes());
        out[24..28].copy_from_slice(&self.hp.to_le_bytes());
        out[28..32].copy_from_slice(&self.mp.to_le_bytes());
        out[32..34].copy_from_slice(&self.strength.to_le_bytes());
        out[34..36].copy_from_slice(&self.intelligence.to_le_bytes());
        out[36..38].copy_from_slice(&self.dexterity.to_le_bytes());
        out[38..40].copy_from_slice(&self.constitution.to_le_bytes());
        for (indice, valor) in self.special.iter().enumerate() {
            let base = 40 + indice * 2;
            out[base..base + 2].copy_from_slice(&valor.to_le_bytes());
        }
    }

    /// Vitais/atributos do personagem persistido por cima do template, preservando os
    /// bytes que o motor antigo mantém do `BaseMob` (`Ac`, `Damage`, `Merchant`,
    /// `AttackRun`, `Direction`, `ChaosRate`) e a maestria `Special`.
    ///
    /// O legado recalcula nível/HP/MP com `GetCurrentScore` no login
    /// (`ProcessDBMessage.cpp`); aqui o banco já guarda o valor recalculado, então a
    /// sobreposição deixa pacote e banco idênticos.
    fn apply_character(&mut self, personagem: &Character) {
        self.level = wire_level(personagem.level);
        self.max_hp = i32_of(personagem.hp_max);
        self.max_mp = i32_of(personagem.mp_max);
        self.hp = i32_of(personagem.hp);
        self.mp = i32_of(personagem.mp);
        self.strength = i16_of(personagem.strength);
        self.intelligence = i16_of(personagem.intelligence);
        self.dexterity = i16_of(personagem.dexterity);
        self.constitution = i16_of(personagem.constitution);
    }
}

impl Default for Mob {
    fn default() -> Self {
        Self {
            name: [0u8; NAME_LEN],
            clan: 0,
            merchant: 0,
            guild: 0,
            class: 0,
            rsv: 0,
            quest: 0,
            coin: 0,
            exp: 0,
            spx: 0,
            spy: 0,
            base_score: Score::default(),
            current_score: Score::default(),
            equip: [Item::default(); MAX_EQUIP],
            carry: [Item::default(); MAX_CARRY],
            learned_skill: 0,
            magic: 0,
            score_bonus: 0,
            special_bonus: 0,
            skill_bonus: 0,
            critical: 0,
            save_mana: 0,
            skill_bar: [0u8; 4],
            guild_level: 0,
            regen_hp: 0,
            regen_mp: 0,
            resist: [0i8; 4],
        }
    }
}

/// `STRUCT_MOB` (816 B) — o personagem encarnado.
///
/// `Default` reproduz `BASE_ClearMob`, não um personagem jogável.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Mob {
    pub name: [u8; NAME_LEN],
    pub clan: u8,
    pub merchant: u8,
    pub guild: u16,
    pub class: u8,
    pub rsv: u16,
    pub quest: u8,
    /// `int` no C (offset 28).
    pub coin: i32,
    /// `long long` no C (offset 32).
    pub exp: i64,
    pub spx: i16,
    pub spy: i16,
    pub base_score: Score,
    pub current_score: Score,
    pub equip: [Item; MAX_EQUIP],
    pub carry: [Item; MAX_CARRY],
    pub learned_skill: i32,
    pub magic: u32,
    pub score_bonus: u16,
    pub special_bonus: u16,
    pub skill_bonus: u16,
    pub critical: u8,
    pub save_mana: u8,
    pub skill_bar: [u8; 4],
    pub guild_level: u8,
    pub regen_hp: u16,
    pub regen_mp: u16,
    pub resist: [i8; 4],
}

impl Mob {
    /// Personagem no formato do motor antigo: `BaseMob` da classe + estado persistido.
    ///
    /// Aparência (`Equip`, incluindo o rosto em `Equip[0]`), `Carry`, `Ac`, `Damage`,
    /// `Merchant`, `AttackRun`, `Direction`, `Special` e a cauda vêm do template real da
    /// classe; nome, nível, HP/MP, atributos, pontos livres, ouro, EXP, ponto de save e
    /// mochila de skills vêm do banco. Classe fora da faixa cai no `BASE_ClearMob`.
    #[must_use]
    pub fn from_character(personagem: &Character) -> Self {
        let mut mob = template_mob(personagem.class);
        mob.name = nome_de_16(&personagem.name);
        mob.class = u8::try_from(personagem.class).unwrap_or(0);
        mob.coin = i32_of(personagem.gold);
        mob.exp = personagem.exp;
        // O ponto de save (`SPX/SPY`) é a posição persistida — o legado envia o mesmo
        // valor em `MSG_CNFCharacterLogin.PosX/PosY` (`ProcessDBMessage.cpp:831`).
        mob.spx = i16_of(personagem.pos_x);
        mob.spy = i16_of(personagem.pos_y);
        mob.base_score.apply_character(personagem);
        mob.current_score.apply_character(personagem);
        // `ChaosRate` é o CP codificado (`CP + 75`), não um stat. A cor do nome no cliente
        // interpola por esse byte (`WYD.exe 0x4e2393`); CP 0 = 75 = amarelo neutro. O `0xCC`
        // do template é lixo e nem chega a sair daqui.
        let cor = chaos_rate_de(personagem.cp);
        mob.base_score.chaos_rate = cor;
        mob.current_score.chaos_rate = cor;
        mob.score_bonus = u16_of(personagem.score_bonus);
        mob.skill_bonus = u16_of(personagem.skill_bonus);
        mob
    }

    /// Personagem com equipamento e mochila **do banco** (`character_items`).
    ///
    /// O template da classe só permanece no que não é inventário (aparência/`Merchant`/`
    /// Critical`/`SkillBar`/`Resist` e os scores base); `Equip`/`Carry` vêm exclusivamente
    /// das linhas persistidas. Slot 0 sem linha recebe o rosto padrão da classe, porque é
    /// ele que desenha a classe no seletor.
    #[must_use]
    pub fn from_persisted(personagem: &Character, items: &[ItemRow]) -> Self {
        let mut mob = Self::from_character(personagem);
        mob.equip = [Item::default(); MAX_EQUIP];
        mob.carry = [Item::default(); MAX_CARRY];
        for row in items {
            if let Some(slot) = row.equip_slot() {
                mob.equip[slot] = Item::from_row(row);
            } else if let Some(slot) = row.carry_slot() {
                mob.carry[slot] = Item::from_row(row);
            }
        }
        // Slot 0 é o rosto: a linha guarda a identidade (`item_id`) e a aparência padrão da
        // classe (cabelo/pele em `stEffect`) entra enquanto a customização de rosto não
        // existe. É por ele que o cliente desenha a classe (`STRUCT_SELCHAR` sem `Class`).
        let rosto = class_face(personagem.class);
        if mob.equip[0].is_empty() || mob.equip[0].index == rosto.index {
            mob.equip[0] = rosto;
        }
        mob
    }

    /// Corpo serializado (`816` bytes) — o mesmo buffer que vai no `STRUCT_MOB`.
    #[must_use]
    pub fn encode(&self) -> Vec<u8> {
        let mut out = vec![0u8; MOB_LEN];
        self.encode_into(&mut out);
        out
    }

    fn encode_into(&self, out: &mut [u8]) {
        out[0..NAME_LEN].copy_from_slice(&self.name);
        out[16] = self.clan;
        out[17] = self.merchant;
        out[18..20].copy_from_slice(&self.guild.to_le_bytes());
        out[20] = self.class;
        out[22..24].copy_from_slice(&self.rsv.to_le_bytes()); // padding em 21
        out[24] = self.quest; // padding em 25–27
        out[28..32].copy_from_slice(&self.coin.to_le_bytes());
        out[32..40].copy_from_slice(&self.exp.to_le_bytes());
        out[40..42].copy_from_slice(&self.spx.to_le_bytes());
        out[42..44].copy_from_slice(&self.spy.to_le_bytes());
        self.base_score
            .encode_into(&mut out[MOB_BASE_SCORE_OFFSET..]);
        self.current_score
            .encode_into(&mut out[MOB_CURRENT_SCORE_OFFSET..]);
        for (indice, item) in self.equip.iter().enumerate() {
            let base = MOB_EQUIP_OFFSET + indice * ITEM_LEN;
            item.encode_into(&mut out[base..base + ITEM_LEN]);
        }
        for (indice, item) in self.carry.iter().enumerate() {
            let base = MOB_CARRY_OFFSET + indice * ITEM_LEN;
            item.encode_into(&mut out[base..base + ITEM_LEN]);
        }
        out[780..784].copy_from_slice(&self.learned_skill.to_le_bytes());
        out[784..788].copy_from_slice(&self.magic.to_le_bytes());
        out[788..790].copy_from_slice(&self.score_bonus.to_le_bytes());
        out[790..792].copy_from_slice(&self.special_bonus.to_le_bytes());
        out[MOB_SKILL_BONUS_OFFSET..MOB_SKILL_BONUS_OFFSET + 2]
            .copy_from_slice(&self.skill_bonus.to_le_bytes());
        out[794] = self.critical;
        out[795] = self.save_mana;
        out[MOB_SKILL_BAR_OFFSET..MOB_SKILL_BAR_OFFSET + 4].copy_from_slice(&self.skill_bar);
        out[800] = self.guild_level; // padding em 801
        out[802..804].copy_from_slice(&self.regen_hp.to_le_bytes());
        out[804..806].copy_from_slice(&self.regen_mp.to_le_bytes());
        for (indice, valor) in self.resist.iter().enumerate() {
            out[MOB_RESIST_OFFSET + indice] = *valor as u8;
        }
    }

    /// Decodifica um `STRUCT_MOB` de 816 bytes (template `BaseMob`, teste e correlação).
    pub fn decode(buf: &[u8]) -> Result<Self, String> {
        if buf.len() < MOB_LEN {
            return Err(format!(
                "struct mob curto: {} B (esperado {MOB_LEN})",
                buf.len()
            ));
        }
        let mut name = [0u8; NAME_LEN];
        name.copy_from_slice(&buf[0..NAME_LEN]);
        let mut equip = [Item::default(); MAX_EQUIP];
        for (indice, item) in equip.iter_mut().enumerate() {
            let base = MOB_EQUIP_OFFSET + indice * ITEM_LEN;
            *item = Item::decode(&buf[base..base + ITEM_LEN]);
        }
        let mut carry = [Item::default(); MAX_CARRY];
        for (indice, item) in carry.iter_mut().enumerate() {
            let base = MOB_CARRY_OFFSET + indice * ITEM_LEN;
            *item = Item::decode(&buf[base..base + ITEM_LEN]);
        }
        let mut skill_bar = [0u8; 4];
        skill_bar.copy_from_slice(&buf[MOB_SKILL_BAR_OFFSET..MOB_SKILL_BAR_OFFSET + 4]);
        Ok(Self {
            name,
            clan: buf[16],
            merchant: buf[17],
            guild: u16::from_le_bytes([buf[18], buf[19]]),
            class: buf[20],
            rsv: u16::from_le_bytes([buf[22], buf[23]]),
            quest: buf[24],
            coin: i32::from_le_bytes([buf[28], buf[29], buf[30], buf[31]]),
            exp: i64::from_le_bytes(buf[32..40].try_into().map_err(|_| "exp")?),
            spx: i16::from_le_bytes([buf[40], buf[41]]),
            spy: i16::from_le_bytes([buf[42], buf[43]]),
            base_score: Score::decode(&buf[MOB_BASE_SCORE_OFFSET..]),
            current_score: Score::decode(&buf[MOB_CURRENT_SCORE_OFFSET..]),
            equip,
            carry,
            learned_skill: i32::from_le_bytes([buf[780], buf[781], buf[782], buf[783]]),
            magic: u32::from_le_bytes([buf[784], buf[785], buf[786], buf[787]]),
            score_bonus: u16::from_le_bytes([buf[788], buf[789]]),
            special_bonus: u16::from_le_bytes([buf[790], buf[791]]),
            skill_bonus: u16::from_le_bytes([buf[792], buf[793]]),
            critical: buf[794],
            save_mana: buf[795],
            skill_bar,
            guild_level: buf[800],
            regen_hp: u16::from_le_bytes([buf[802], buf[803]]),
            regen_mp: u16::from_le_bytes([buf[804], buf[805]]),
            resist: [
                buf[806] as i8,
                buf[807] as i8,
                buf[808] as i8,
                buf[809] as i8,
            ],
        })
    }
}

/// `STRUCT_SELCHAR` (840 B) — os quatro slots da tela de seleção.
///
/// Não tem campo de classe: o cliente desenha a classe pelo rosto em `Equip[slot][0]`,
/// exatamente como `DBGetSelChar` (`CFileDB.cpp:2636`) monta o pacote.
#[derive(Debug, Clone, PartialEq, Eq, Default)]
pub struct Selchar {
    pub spx: [i16; SLOT_COUNT],
    pub spy: [i16; SLOT_COUNT],
    pub name: [[u8; NAME_LEN]; SLOT_COUNT],
    pub score: [Score; SLOT_COUNT],
    pub equip: [[Item; MAX_EQUIP]; SLOT_COUNT],
    pub guild: [u16; SLOT_COUNT],
    pub coin: [i32; SLOT_COUNT],
    pub exp: [i64; SLOT_COUNT],
}

impl Selchar {
    /// Monta o seletor a partir do roster persistido (slots vazios ficam zerados).
    ///
    /// Cada item é `(personagem, linhas de character_items)`: o equipamento desenhado na
    /// seleção é o do banco, não o do template da classe.
    #[must_use]
    pub fn from_roster(roster: &[(Character, Vec<ItemRow>)]) -> Self {
        let mut sel = Self::default();
        for (personagem, items) in roster {
            let Ok(slot) = usize::try_from(personagem.slot) else {
                continue;
            };
            if slot >= SLOT_COUNT {
                continue;
            }
            let mob = Mob::from_persisted(personagem, items);
            sel.spx[slot] = mob.spx;
            sel.spy[slot] = mob.spy;
            sel.name[slot] = mob.name;
            // `DBGetSelChar` copia `CurrentScore` para `sel->Score[i]`.
            sel.score[slot] = mob.current_score;
            sel.equip[slot] = mob.equip;
            sel.guild[slot] = mob.guild;
            sel.coin[slot] = mob.coin;
            sel.exp[slot] = mob.exp;
        }
        sel
    }

    /// Corpo serializado (`840` bytes) do `STRUCT_SELCHAR`.
    #[must_use]
    pub fn encode(&self) -> Vec<u8> {
        let mut out = vec![0u8; SELCHAR_LEN];
        for slot in 0..SLOT_COUNT {
            out[slot * 2..slot * 2 + 2].copy_from_slice(&self.spx[slot].to_le_bytes());
            out[8 + slot * 2..8 + slot * 2 + 2].copy_from_slice(&self.spy[slot].to_le_bytes());
            let nome = SELCHAR_NAME_OFFSET + slot * NAME_LEN;
            out[nome..nome + NAME_LEN].copy_from_slice(&self.name[slot]);
            let score = SELCHAR_SCORE_OFFSET + slot * SCORE_LEN;
            self.score[slot].encode_into(&mut out[score..score + SCORE_LEN]);
            let equip = SELCHAR_EQUIP_OFFSET + slot * MAX_EQUIP * ITEM_LEN;
            for (indice, item) in self.equip[slot].iter().enumerate() {
                let base = equip + indice * ITEM_LEN;
                item.encode_into(&mut out[base..base + ITEM_LEN]);
            }
            out[SELCHAR_GUILD_OFFSET + slot * 2..SELCHAR_GUILD_OFFSET + slot * 2 + 2]
                .copy_from_slice(&self.guild[slot].to_le_bytes());
            out[SELCHAR_COIN_OFFSET + slot * 4..SELCHAR_COIN_OFFSET + slot * 4 + 4]
                .copy_from_slice(&self.coin[slot].to_le_bytes());
            out[SELCHAR_EXP_OFFSET + slot * 8..SELCHAR_EXP_OFFSET + slot * 8 + 8]
                .copy_from_slice(&self.exp[slot].to_le_bytes());
        }
        out
    }

    /// Decodifica os `840` bytes do `STRUCT_SELCHAR`.
    pub fn decode(buf: &[u8]) -> Result<Self, String> {
        if buf.len() < SELCHAR_LEN {
            return Err(format!(
                "struct selchar curto: {} B (esperado {SELCHAR_LEN})",
                buf.len()
            ));
        }
        let mut sel = Self::default();
        for slot in 0..SLOT_COUNT {
            sel.spx[slot] = i16::from_le_bytes([buf[slot * 2], buf[slot * 2 + 1]]);
            sel.spy[slot] = i16::from_le_bytes([buf[8 + slot * 2], buf[8 + slot * 2 + 1]]);
            let nome = SELCHAR_NAME_OFFSET + slot * NAME_LEN;
            sel.name[slot].copy_from_slice(&buf[nome..nome + NAME_LEN]);
            let score = SELCHAR_SCORE_OFFSET + slot * SCORE_LEN;
            sel.score[slot] = Score::decode(&buf[score..score + SCORE_LEN]);
            let equip = SELCHAR_EQUIP_OFFSET + slot * MAX_EQUIP * ITEM_LEN;
            for indice in 0..MAX_EQUIP {
                let base = equip + indice * ITEM_LEN;
                sel.equip[slot][indice] = Item::decode(&buf[base..base + ITEM_LEN]);
            }
            sel.guild[slot] = u16::from_le_bytes([
                buf[SELCHAR_GUILD_OFFSET + slot * 2],
                buf[SELCHAR_GUILD_OFFSET + slot * 2 + 1],
            ]);
            sel.coin[slot] = i32::from_le_bytes(
                buf[SELCHAR_COIN_OFFSET + slot * 4..SELCHAR_COIN_OFFSET + slot * 4 + 4]
                    .try_into()
                    .map_err(|_| "coin")?,
            );
            sel.exp[slot] = i64::from_le_bytes(
                buf[SELCHAR_EXP_OFFSET + slot * 8..SELCHAR_EXP_OFFSET + slot * 8 + 8]
                    .try_into()
                    .map_err(|_| "exp")?,
            );
        }
        Ok(sel)
    }
}

/// Corpo do `_MSG_CNFAccountLogin` (`MSG_DBCNFAccountLogin`, 1996 B).
///
/// `Unknow_28 = 1` é o "não recriar as poções iniciais" do porte Go. `cargo` é o cofre da
/// conta (`vault_items`); sem linhas, slots vazios. `Coin` e `AccountName` vêm da conta.
#[must_use]
pub fn account_login_body(
    roster: &[(Character, Vec<ItemRow>)],
    cargo: &[Item],
    coin: i32,
    account_name: &str,
) -> Vec<u8> {
    let mut out = vec![0u8; ACCOUNT_LOGIN_LEN];
    out[16..20].copy_from_slice(&1u32.to_le_bytes());
    let sel = Selchar::from_roster(roster).encode();
    out[20..20 + SELCHAR_LEN].copy_from_slice(&sel);
    for (indice, item) in cargo.iter().take(MAX_CARGO).enumerate() {
        let base = 860 + indice * ITEM_LEN;
        item.encode_into(&mut out[base..base + ITEM_LEN]);
    }
    out[1884..1888].copy_from_slice(&coin.to_le_bytes());
    out[1888..1888 + NAME_LEN].copy_from_slice(&nome_de_16(account_name));
    out
}

/// Corpo do `_MSG_CNFNewCharacter` (844 B): o roster inteiro depois da criação.
#[must_use]
pub fn new_character_body(roster: &[(Character, Vec<ItemRow>)]) -> Vec<u8> {
    let mut out = vec![0u8; NEW_CHARACTER_LEN];
    out[4..4 + SELCHAR_LEN].copy_from_slice(&Selchar::from_roster(roster).encode());
    out
}

/// Corpo do `_MSG_CNFDeleteCharacter` (844 B): o roster **depois** da remoção.
///
/// Mesmo layout do `CNFNewCharacter` (`sel@4`, padding do `STRUCT_SELCHAR` no C), porque o
/// legado devolve a seleção inteira para o cliente redesenhar os slots — confirmar a remoção
/// com `Size = 12` deixava a tela com o personagem apagado ainda desenhado.
#[must_use]
pub fn delete_character_body(roster: &[(Character, Vec<ItemRow>)]) -> Vec<u8> {
    new_character_body(roster)
}

/// Nível no fio: o `Level` do `STRUCT_SCORE` é o do banco **menos um**.
///
/// Evidência: o `BaseMob` das quatro classes traz `Level@44 = 0` e a criação do legado só
/// copia o template (`CFileDB.cpp`, `memcpy(mob, &g_pBaseSet[cls])`), nunca escreve o nível —
/// e o cliente 7662 desenha `wire + 1` (personagem nível 1 aparecia como 2). A curva de EXP
/// confirma a mesma base: `g_pNextLevel[0] = 0` é o personagem recém-criado.
#[must_use]
pub fn wire_level(db_level: i64) -> i32 {
    i32_of(db_level - WIRE_LEVEL_OFFSET)
}

/// Distância entre o nível do banco (1-based) e o do fio (0-based).
pub const WIRE_LEVEL_OFFSET: i64 = 1;

/// Corpo do `_MSG_CNFClientCharacterLogin` (1820 B) — o snapshot do mundo.
///
/// `PosX/PosY` é a posição de entrada; `mob.SPX/SPY` continua sendo o ponto de save
/// (iguais no login, como no legado). `short_skill` nasce `0xFF` (`CFileDB.cpp`),
/// `account_name` entra em `Unk2[448]` como o `strncpy` de 12 bytes do
/// `ProcessDBMessage.cpp:840`.
#[must_use]
pub fn character_login_body(
    mob: &Mob,
    slot: u16,
    client_id: u16,
    weather: u16,
    short_skill: &[u8; SHORT_SKILL_LEN],
    account_name: &str,
) -> Vec<u8> {
    let mut out = vec![0u8; CHARACTER_LOGIN_LEN];
    out[0..2].copy_from_slice(&mob.spx.to_le_bytes());
    out[2..4].copy_from_slice(&mob.spy.to_le_bytes());
    mob.encode_into(&mut out[4..]);
    out[1028..1030].copy_from_slice(&slot.to_le_bytes());
    out[1030..1032].copy_from_slice(&client_id.to_le_bytes());
    out[1032..1034].copy_from_slice(&weather.to_le_bytes());
    out[1034..1034 + SHORT_SKILL_LEN].copy_from_slice(short_skill);
    let nome = account_name.as_bytes();
    let limite = nome.len().min(LOGIN_ACCOUNT_NAME_LEN);
    out[ACCOUNT_NAME_OFFSET..ACCOUNT_NAME_OFFSET + limite].copy_from_slice(&nome[..limite]);
    out
}

/// Nome no campo fixo de 16 bytes (UTF-8 truncado no limite do campo).
fn nome_de_16(texto: &str) -> [u8; NAME_LEN] {
    let mut destino = [0u8; NAME_LEN];
    let bytes = texto.as_bytes();
    let limite = bytes.len().min(destino.len());
    destino[..limite].copy_from_slice(&bytes[..limite]);
    destino
}

fn i32_of(valor: i64) -> i32 {
    i32::try_from(valor).unwrap_or(if valor < 0 { i32::MIN } else { i32::MAX })
}

fn i16_of(valor: i64) -> i16 {
    i16::try_from(valor).unwrap_or(if valor < 0 { i16::MIN } else { i16::MAX })
}

fn u16_of(valor: i64) -> u16 {
    u16::try_from(valor).unwrap_or(u16::MAX)
}

#[cfg(test)]
mod tests {
    use super::*;
    use wyd_db::Character;

    /// Roster no formato do login: cada personagem com o inventário persistido.
    fn com_itens(roster: &[Character]) -> Vec<(Character, Vec<ItemRow>)> {
        roster
            .iter()
            .map(|personagem| {
                let mut rows = template_items(personagem.class);
                rows.push(kill_mark_row());
                (personagem.clone(), rows)
            })
            .collect()
    }

    fn personagem(slot: i64, nome: &str, class: i64) -> Character {
        Character {
            id: 7 + slot,
            account_id: 3,
            slot,
            name: nome.to_string(),
            class,
            level: 1,
            exp: 0,
            hp: 83,
            hp_max: 83,
            mp: 46,
            mp_max: 46,
            pos_x: 2096,
            pos_y: 2096,
            gold: 5_000_000,
            strength: 8,
            intelligence: 4,
            dexterity: 7,
            constitution: 6,
            score_bonus: 5,
            skill_bonus: 3,
            ..Character::default()
        }
    }

    #[test]
    fn personagem_nasce_com_cp_neutro_e_nao_com_o_lixo_do_template() {
        let mob = Mob::from_character(&personagem(0, "Novato", 0));
        assert_eq!(mob.base_score.chaos_rate, CP_NEUTRO, "BaseScore.ChaosRate");
        assert_eq!(
            mob.current_score.chaos_rate, CP_NEUTRO,
            "CurrentScore.ChaosRate"
        );
        assert_eq!(
            update_score_body(&mob)[15],
            CP_NEUTRO,
            "MSG_UpdateScore.ChaosRate@15 é a cor do nome"
        );
        // CP persistido vira cor: -75 = 0 (ramo vermelho do cliente), +25 = 100.
        let mut pk = personagem(0, "Pkzinho", 0);
        pk.cp = -75;
        assert_eq!(
            Mob::from_character(&pk).current_score.chaos_rate,
            0,
            "CP -75 no piso da tabela"
        );
        pk.cp = 25;
        assert_eq!(
            Mob::from_character(&pk).current_score.chaos_rate,
            100,
            "CP 25 deslocado em 75"
        );
    }

    #[test]
    fn templates_de_classe_tem_816_bytes_e_campos_nos_offsets_do_c() {
        // Valores lidos dos arquivos do main: nome, classe, HP/MP base, atributos,
        // rosto (`Equip[0]`) e `Merchant@12` (0 nas classes que não vendem).
        let esperado = [
            ("TK", 0, 80, 45, 8, 4, 7, 6, 1, 0),
            ("FM", 1, 60, 65, 5, 8, 5, 5, 11, 16),
            ("BM", 2, 70, 55, 6, 6, 9, 5, 21, 16),
            ("HT", 3, 75, 60, 8, 9, 13, 6, 31, 0),
        ];
        for (indice, (nome, class, hp, mp, forca, int, dex, con, rosto, merchant)) in
            esperado.into_iter().enumerate()
        {
            let cru = base_mob(indice as i64).expect("template da classe");
            assert_eq!(cru.len(), MOB_LEN, "{nome}: sizeof(STRUCT_MOB)");
            let mob = Mob::decode(cru).expect("template decodifica");
            assert_eq!(&mob.name[..2], nome.as_bytes(), "{nome}: MobName@0");
            assert_eq!(mob.class, class, "{nome}: Class@20");
            assert_eq!(mob.clan, 0, "{nome}: Clan@16");
            assert_eq!(mob.guild, 0, "{nome}: Guild@18");
            assert_eq!(mob.coin, 5_000_000, "{nome}: Coin@28");
            assert_eq!(mob.exp, 0, "{nome}: Exp@32");
            assert_eq!((mob.spx, mob.spy), (2096, 2096), "{nome}: SPX@40/SPY@42");
            for score in [&mob.base_score, &mob.current_score] {
                assert_eq!(score.level, 0, "{nome}: Level@0 do score");
                assert_eq!((score.max_hp, score.hp), (hp, hp), "{nome}: MaxHp/Hp");
                assert_eq!((score.max_mp, score.mp), (mp, mp), "{nome}: MaxMp/Mp");
                assert_eq!(
                    (
                        score.strength,
                        score.intelligence,
                        score.dexterity,
                        score.constitution
                    ),
                    (forca, int, dex, con),
                    "{nome}: Str/Int/Dex/Con"
                );
                assert_eq!(score.ac, 4, "{nome}: Ac@4");
                assert_eq!(score.damage, 5, "{nome}: Damage@8");
                assert_eq!(score.merchant, merchant, "{nome}: Merchant@12");
                assert_eq!(score.attack_run, 82, "{nome}: AttackRun@13");
                // 0xCC é lixo do dump do MSVC; o porte Go também o mantém no login.
                assert_eq!(score.direction, 0xCC, "{nome}: Direction@14");
                assert_eq!(score.chaos_rate, 0xCC, "{nome}: ChaosRate@15");
            }
            assert_eq!(mob.equip[0].index, rosto, "{nome}: rosto em Equip[0]");
            assert_eq!(mob.critical, 0xFF, "{nome}: Critical@794");
            assert_eq!(mob.save_mana, 0xFF, "{nome}: SaveMana@795");
            assert_eq!(mob.skill_bar, [0xFF; 4], "{nome}: SkillBar@796");
            assert_eq!(mob.resist, [-11, -11, -11, -11], "{nome}: Resist@806");
        }
    }

    #[test]
    fn serializacao_reproduz_o_template_fora_o_padding() {
        // O dump do motor antigo carrega lixo de padding do MSVC; tudo mais tem de voltar
        // byte a byte do template real — a prova de que os offsets do encoder são os do C.
        for class in 0..4i64 {
            let cru = base_mob(class).expect("template");
            let escrito = Mob::decode(cru).expect("decodifica").encode();
            assert_eq!(escrito.len(), MOB_LEN);
            for (posicao, (novo, original)) in escrito.iter().zip(cru.iter()).enumerate() {
                if MOB_PADDING.contains(&posicao) {
                    continue;
                }
                assert_eq!(
                    novo, original,
                    "classe {class}: byte {posicao} divergiu do template"
                );
            }
            for posicao in MOB_PADDING {
                assert_eq!(
                    escrito[posicao], 0,
                    "padding {posicao} sai zerado, não lixo"
                );
            }
        }
    }

    #[test]
    fn mob_do_personagem_sobrepoe_vitais_e_preserva_aparencia() {
        let personagem = personagem(0, "Kael", 0);
        let mob = Mob::from_character(&personagem);
        let cru = Mob::decode(&mob.encode()).expect("mob");
        assert_eq!(cru, mob, "round-trip do STRUCT_MOB");
        assert_eq!(&cru.name[..4], b"Kael", "MobName@0");
        assert_eq!(cru.class, 0, "Class@20");
        assert_eq!(cru.coin, 5_000_000, "Coin@28 vem do BaseMob");
        assert_eq!((cru.spx, cru.spy), (2096, 2096), "ponto de save@40/42");
        // O fio é 0-based: banco nível 1 → `Level@44/92 = 0` (o cliente soma 1).
        assert_eq!(cru.base_score.level, 0, "BaseScore.Level@44");
        assert_eq!(cru.current_score.level, 0, "CurrentScore.Level@92");
        assert_eq!(
            (cru.current_score.max_hp, cru.current_score.hp),
            (83, 83),
            "MaxHp@108/Hp@116"
        );
        assert_eq!(
            (cru.current_score.max_mp, cru.current_score.mp),
            (46, 46),
            "MaxMp@112/Mp@120"
        );
        assert_eq!(
            (
                cru.current_score.strength,
                cru.current_score.intelligence,
                cru.current_score.dexterity,
                cru.current_score.constitution
            ),
            (8, 4, 7, 6),
            "Str@124..Con@130"
        );
        assert_eq!(
            (cru.score_bonus, cru.skill_bonus),
            (5, 3),
            "pontos@788/@792"
        );
        // Aparência/equipamento inicial continuam os do motor antigo.
        let template = template_mob(0);
        assert_eq!(cru.equip, template.equip, "Equip@140 do BaseMob");
        assert_eq!(cru.carry, template.carry, "Carry@268 do BaseMob");
        assert_eq!(cru.equip[0].index, 1, "rosto TransKnight");
        assert!(!cru.equip[1].is_empty(), "arma inicial presente");
    }

    #[test]
    fn update_etc_e_update_score_tem_os_offsets_do_sendfunc() {
        let mob = Mob::from_character(&personagem(0, "Kael", 0));
        let etc = update_etc_body(&mob);
        assert_eq!(etc.len(), UPDATE_ETC_LEN, "MSG_UpdateEtc − HEADER");
        assert_eq!(
            i64::from_le_bytes(etc[4..12].try_into().expect("exp")),
            mob.exp,
            "Exp@4"
        );
        assert_eq!(
            i64::from_le_bytes(etc[12..20].try_into().expect("learn")),
            i64::from(mob.learned_skill),
            "Learn@12"
        );
        assert_eq!(
            u16::from_le_bytes([etc[20], etc[21]]),
            mob.score_bonus,
            "ScoreBonus@20"
        );
        assert_eq!(
            u16::from_le_bytes([etc[22], etc[23]]),
            mob.special_bonus,
            "SpecialBonus@22"
        );
        assert_eq!(
            u16::from_le_bytes([etc[24], etc[25]]),
            mob.skill_bonus,
            "SkillBonus@24"
        );
        assert_eq!(
            i32::from_le_bytes(etc[28..32].try_into().expect("coin")),
            mob.coin,
            "Coin@28"
        );

        let score = update_score_body(&mob);
        assert_eq!(score.len(), UPDATE_SCORE_LEN, "MSG_UpdateScore − HEADER");
        assert_eq!(
            i32::from_le_bytes(score[0..4].try_into().expect("level")),
            mob.current_score.level,
            "Score.Level@0"
        );
        assert_eq!(
            i32::from_le_bytes(score[16..20].try_into().expect("maxhp")),
            mob.current_score.max_hp,
            "Score.MaxHp@16"
        );
        assert_eq!(
            i32::from_le_bytes(score[24..28].try_into().expect("hp")),
            mob.current_score.hp,
            "Score.Hp@24"
        );
        assert_eq!(
            i16::from_le_bytes([score[38], score[39]]),
            mob.current_score.constitution,
            "Score.Con@38"
        );
        assert_eq!(
            (score[48], score[49]),
            (mob.critical, mob.save_mana),
            "Critical/SaveMana"
        );
        assert_eq!(
            i32::from_le_bytes(score[124..128].try_into().expect("currhp")),
            mob.current_score.hp,
            "CurrHp@124"
        );
        assert_eq!(score[136..140], [0xCC; 4], "Special[4]@136");
    }

    #[test]
    fn selchar_tem_840_bytes_e_campos_nos_offsets_do_c() {
        let roster = [personagem(0, "Kael", 0), personagem(3, "Nyx", 3)];
        let sel = Selchar::from_roster(&com_itens(&roster));
        let corpo = sel.encode();
        assert_eq!(corpo.len(), SELCHAR_LEN, "sizeof(STRUCT_SELCHAR)");
        assert_eq!(Selchar::decode(&corpo).expect("selchar"), sel, "round-trip");

        // Slot 0 e slot 3 nos offsets do C, com os slots 1 e 2 zerados.
        for (slot, nome) in [(0usize, &b"Kael"[..]), (3usize, &b"Nyx"[..])] {
            let inicio = SELCHAR_NAME_OFFSET + slot * NAME_LEN;
            assert_eq!(&corpo[inicio..inicio + nome.len()], nome, "Name@{}", inicio);
            let score = SELCHAR_SCORE_OFFSET + slot * SCORE_LEN;
            assert_eq!(
                i32::from_le_bytes(corpo[score..score + 4].try_into().expect("level")),
                0,
                "Score.Level@{} (banco 1 → fio 0)",
                score
            );
            let equip = SELCHAR_EQUIP_OFFSET + slot * MAX_EQUIP * ITEM_LEN;
            assert_eq!(
                i16::from_le_bytes(corpo[equip..equip + 2].try_into().expect("índice")),
                if slot == 0 { 1 } else { 31 },
                "Equip[0]@{} carrega o rosto da classe",
                equip
            );
            assert_eq!(
                i32::from_le_bytes(
                    corpo[SELCHAR_COIN_OFFSET + slot * 4..SELCHAR_COIN_OFFSET + slot * 4 + 4]
                        .try_into()
                        .expect("coin")
                ),
                5_000_000,
                "Coin@{}",
                SELCHAR_COIN_OFFSET + slot * 4
            );
            assert_eq!(
                i64::from_le_bytes(
                    corpo[SELCHAR_EXP_OFFSET + slot * 8..SELCHAR_EXP_OFFSET + slot * 8 + 8]
                        .try_into()
                        .expect("exp")
                ),
                0,
                "Exp@{}",
                SELCHAR_EXP_OFFSET + slot * 8
            );
            assert_eq!(
                u16::from_le_bytes(
                    corpo[SELCHAR_GUILD_OFFSET + slot * 2..SELCHAR_GUILD_OFFSET + slot * 2 + 2]
                        .try_into()
                        .expect("guild")
                ),
                0,
                "Guild@{}",
                SELCHAR_GUILD_OFFSET + slot * 2
            );
        }
        for slot in [1usize, 2] {
            let inicio = SELCHAR_NAME_OFFSET + slot * NAME_LEN;
            assert_eq!(&corpo[inicio..inicio + NAME_LEN], &[0u8; NAME_LEN]);
        }
    }

    #[test]
    fn corpos_de_login_e_criacao_tem_o_tamanho_e_os_offsets_do_c() {
        let roster = [personagem(1, "Kael", 0)];
        let corpo = account_login_body(&com_itens(&roster), &[], 0, "admin");
        assert_eq!(corpo.len(), ACCOUNT_LOGIN_LEN, "MSG_DBCNFAccountLogin");
        assert_eq!(
            u32::from_le_bytes(corpo[16..20].try_into().expect("unk")),
            1,
            "Unknow_28 = 1"
        );
        let sel = Selchar::decode(&corpo[20..20 + SELCHAR_LEN]).expect("sel@20");
        assert_eq!(sel.name[1], *b"Kael\0\0\0\0\0\0\0\0\0\0\0\0");
        assert_eq!(&corpo[1888..1892], b"admi", "AccountName@1888");

        let novo = new_character_body(&com_itens(&roster));
        assert_eq!(novo.len(), NEW_CHARACTER_LEN, "MSG_CNFNewCharacter");
        let sel = Selchar::decode(&novo[4..4 + SELCHAR_LEN]).expect("sel@4");
        assert_eq!(sel.name[1][..4], *b"Kael");

        let mob = Mob::from_character(&roster[0]);
        let login = character_login_body(
            &mob,
            1,
            9,
            0,
            &[SHORT_SKILL_EMPTY; SHORT_SKILL_LEN],
            "admin",
        );
        assert_eq!(
            login.len(),
            CHARACTER_LOGIN_LEN,
            "MSG_CNFClientCharacterLogin"
        );
        assert_eq!(
            i16::from_le_bytes(login[0..2].try_into().expect("posx")),
            2096,
            "PosX@0"
        );
        assert_eq!(
            i16::from_le_bytes(login[2..4].try_into().expect("posy")),
            2096,
            "PosY@2"
        );
        assert_eq!(Mob::decode(&login[4..]).expect("mob@4"), mob, "mob@4");
        assert_eq!(
            u16::from_le_bytes(login[1028..1030].try_into().expect("slot")),
            1,
            "Slot@1028"
        );
        assert_eq!(
            u16::from_le_bytes(login[1030..1032].try_into().expect("client")),
            9,
            "ClientID@1030"
        );
        assert_eq!(
            &login[1034..1050],
            &[SHORT_SKILL_EMPTY; 16],
            "ShortSkill@1034"
        );
        assert_eq!(
            &login[ACCOUNT_NAME_OFFSET..ACCOUNT_NAME_OFFSET + 5],
            b"admin",
            "AccountName em Unk2[448]@1500"
        );
    }

    #[test]
    fn inventario_do_banco_reproduz_o_template_e_o_kill_mark() {
        let personagem = personagem(0, "Kael", 0);
        let mut rows = template_items(0);
        rows.push(kill_mark_row());
        let persistido = Mob::from_persisted(&personagem, &rows);
        let esperado = Mob::from_character(&personagem);
        // `Equip`/`Carry` vêm das linhas: o resultado bate com o template da criação.
        assert_eq!(persistido.equip, esperado.equip, "Equip@140");
        for slot in 0..MAX_CARRY {
            if slot == KILL_MARK_SLOT {
                continue;
            }
            assert_eq!(
                persistido.carry[slot], esperado.carry[slot],
                "Carry[{slot}]@268"
            );
        }
        // Kill mark do login: item 547 com os três efeitos do legado.
        assert_eq!(persistido.carry[KILL_MARK_SLOT].index, KILL_MARK_ITEM);
        assert_eq!(
            persistido.carry[KILL_MARK_SLOT].effects,
            [[75, 0], [76, 0], [77, 0]],
            "EF_CURKILL/EF_LTOTKILL/EF_HTOTKILL"
        );
        assert_eq!(
            Mob::decode(&persistido.encode()).expect("mob"),
            persistido,
            "round-trip do STRUCT_MOB"
        );
        // Sem linha não há equipamento derivado do template: só o rosto da classe.
        let vazio = Mob::from_persisted(&personagem, &[]);
        assert_eq!(vazio.equip[0].index, 1, "rosto TransKnight");
        for slot in 1..MAX_EQUIP {
            assert!(vazio.equip[slot].is_empty(), "Equip[{slot}] sem linha");
        }
        assert_eq!(vazio.carry, [Item::default(); MAX_CARRY]);
    }

    #[test]
    fn classe_fora_da_faixa_cai_no_clear_mob_do_legado() {
        assert!(base_mob(4).is_none());
        let mob = template_mob(9);
        assert_eq!(mob, Mob::default(), "BASE_ClearMob, nunca valor inventado");
    }

    fn linha(item_id: i64, refine: i64, qty: i64) -> ItemRow {
        ItemRow {
            slot: 16,
            item_id,
            qty,
            refine,
            ..ItemRow::default()
        }
    }

    #[test]
    fn item_sem_refino_nem_piedade_nao_tem_par_sanc() {
        let item = Item::from_row_with_pity(&linha(3_367, 0, 1), 0);
        assert_eq!(item.effects, [[0, 0], [0, 0], [0, 0]]);
        assert_eq!(
            item,
            Item::from_row(&linha(3_367, 0, 1)),
            "fio idêntico ao antigo"
        );
    }

    #[test]
    fn item_refinado_carrega_sanc_com_piedade() {
        // +7 com 3 de piedade: cValue = 7 + 30 = 37 no primeiro slot.
        let item = Item::from_row_with_pity(&linha(3_367, 7, 1), 3);
        assert_eq!(item.effects[0], [EF_SANC, 37]);
        assert_eq!(item.to_bytes()[2..4], [EF_SANC, 37]);
    }

    #[test]
    fn sanc_nao_desloca_amount_e_item_mais_dez_usa_gema_zero() {
        // Pilha com EF_AMOUNT: o par sanc vai para o primeiro slot livre.
        let item = Item::from_row_with_pity(&linha(3_367, 4, 5), 2);
        assert_eq!(item.effects[0][0], EF_AMOUNT, "amount fica no slot 0");
        assert_eq!(item.effects[1], [EF_SANC, 24], "+4 com 2 de piedade");
        // +11 do pó: 230 + 4 + gema 0.
        let onze = Item::from_row_with_pity(&linha(3_367, 11, 1), 0);
        assert_eq!(onze.effects[0], [EF_SANC, 234]);
    }
}
