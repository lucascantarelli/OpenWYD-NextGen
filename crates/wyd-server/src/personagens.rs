//! Corpo de `_MSG_CNFAccountLogin` (`STRUCT_SELCHAR`) e de `_MSG_CNFCharacterLogin`
//! (`STRUCT_MOB`), montado a partir do estado persistido.
//!
//! **Layout provisório (`rust-v1`)**: os offsets do cliente 7662 continuam UNVERIFIED
//! (`docs/cliente/captura-selchar.md`). O que este módulo garante é o que o projeto pode
//! garantir hoje: o corpo sai do banco, é decodificado de volta pelo mesmo código e o
//! `diff-captura.mjs` enxerga a faixa que muda quando um campo muda. Quando o diff do
//! cliente revelar os offsets reais, só as constantes daqui mudam — o resto do fluxo fica.
//!
//! Campos sem coluna no banco (`hp_max`, `mp_max`, `gold`, `guild_id`, `pvp`) vão zerados
//! ou espelhados, sempre marcados: sem estado real, não se inventa valor.

use wyd_db::Character;

/// Registro de personagem no `CNFAccountLogin` (layout provisório).
pub const SELCHAR_LEN: usize = 144;
/// Corpo de `STRUCT_MOB` (layout provisório; tamanho do 7662 desconhecido).
pub const MOB_LEN: usize = 256;
/// Slots desenhados pelo seletor.
pub const SLOT_COUNT: usize = 4;
/// Versão do layout provisório, no corpo da lista.
pub const LAYOUT_VERSION: u16 = 1;

/// Um personagem no seletor.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Selchar {
    pub name: [u8; 16],
    pub class: u8,
    pub citizen: u8,
    pub level: u16,
    pub exp: u32,
    pub hp: u32,
    pub hp_max: u32,
    pub mp: u32,
    pub mp_max: u32,
    pub gold: u32,
    pub guild_id: u16,
    pub slot: u8,
    pub tier: u8,
    pub pvp: u32,
    pub pos_x: u16,
    pub pos_y: u16,
    pub flags: u32,
    pub char_id: i64,
    pub account_id: i64,
    pub guild_name: [u8; 16],
}

impl Selchar {
    /// Converte o personagem persistido; `citizen` vem do canal (`ServerIndex + 1`).
    #[must_use]
    pub fn from_character(personagem: &Character, citizen: u8) -> Self {
        let mut name = [0u8; 16];
        let bytes = personagem.name.as_bytes();
        let n = bytes.len().min(name.len());
        name[..n].copy_from_slice(&bytes[..n]);
        Self {
            name,
            class: u8::try_from(personagem.class).unwrap_or(0),
            citizen,
            level: u16::try_from(personagem.level).unwrap_or(u16::MAX),
            exp: u32::try_from(personagem.exp).unwrap_or(u32::MAX),
            hp: u32::try_from(personagem.hp).unwrap_or(0),
            hp_max: u32::try_from(personagem.hp_max).unwrap_or(0),
            mp: u32::try_from(personagem.mp).unwrap_or(0),
            mp_max: u32::try_from(personagem.mp_max).unwrap_or(0),
            gold: u32::try_from(personagem.gold).unwrap_or(u32::MAX),
            guild_id: u16::try_from(personagem.guild_id.unwrap_or(0)).unwrap_or(0),
            slot: u8::try_from(personagem.slot).unwrap_or(0),
            tier: 0, // `tier` é TEXT no banco; índice numérico do 7662 é UNVERIFIED
            pvp: u32::try_from(personagem.pvp_points).unwrap_or(0),
            pos_x: u16::try_from(personagem.pos_x).unwrap_or(0),
            pos_y: u16::try_from(personagem.pos_y).unwrap_or(0),
            flags: 0,
            char_id: personagem.id,
            account_id: personagem.account_id,
            guild_name: nome_de_16(&personagem.guild_name),
        }
    }

    fn encode_into(&self, out: &mut [u8]) {
        out[0..16].copy_from_slice(&self.name);
        out[16] = self.class;
        out[17] = self.citizen;
        out[18..20].copy_from_slice(&self.level.to_le_bytes());
        out[20..24].copy_from_slice(&self.exp.to_le_bytes());
        out[24..28].copy_from_slice(&self.hp.to_le_bytes());
        out[28..32].copy_from_slice(&self.hp_max.to_le_bytes());
        out[32..36].copy_from_slice(&self.mp.to_le_bytes());
        out[36..40].copy_from_slice(&self.mp_max.to_le_bytes());
        out[40..44].copy_from_slice(&self.gold.to_le_bytes());
        out[44..46].copy_from_slice(&self.guild_id.to_le_bytes());
        out[46] = self.slot;
        out[47] = self.tier;
        out[48..52].copy_from_slice(&self.pvp.to_le_bytes());
        out[52..54].copy_from_slice(&self.pos_x.to_le_bytes());
        out[54..56].copy_from_slice(&self.pos_y.to_le_bytes());
        out[56..60].copy_from_slice(&self.flags.to_le_bytes());
        out[60..68].copy_from_slice(&self.char_id.to_le_bytes());
        out[68..76].copy_from_slice(&self.account_id.to_le_bytes());
        out[76..92].copy_from_slice(&self.guild_name);
    }

    /// Decodifica o registro (usado pelos testes e por ferramentas de correlação).
    pub fn decode(buf: &[u8]) -> Result<Self, String> {
        if buf.len() < SELCHAR_LEN {
            return Err(format!("selchar curto: {} B", buf.len()));
        }
        let mut name = [0u8; 16];
        name.copy_from_slice(&buf[0..16]);
        let mut guild_name = [0u8; 16];
        guild_name.copy_from_slice(&buf[76..92]);
        Ok(Self {
            name,
            class: buf[16],
            citizen: buf[17],
            level: u16::from_le_bytes([buf[18], buf[19]]),
            exp: u32::from_le_bytes([buf[20], buf[21], buf[22], buf[23]]),
            hp: u32::from_le_bytes([buf[24], buf[25], buf[26], buf[27]]),
            hp_max: u32::from_le_bytes([buf[28], buf[29], buf[30], buf[31]]),
            mp: u32::from_le_bytes([buf[32], buf[33], buf[34], buf[35]]),
            mp_max: u32::from_le_bytes([buf[36], buf[37], buf[38], buf[39]]),
            gold: u32::from_le_bytes([buf[40], buf[41], buf[42], buf[43]]),
            guild_id: u16::from_le_bytes([buf[44], buf[45]]),
            slot: buf[46],
            tier: buf[47],
            pvp: u32::from_le_bytes([buf[48], buf[49], buf[50], buf[51]]),
            pos_x: u16::from_le_bytes([buf[52], buf[53]]),
            pos_y: u16::from_le_bytes([buf[54], buf[55]]),
            flags: u32::from_le_bytes([buf[56], buf[57], buf[58], buf[59]]),
            char_id: i64::from_le_bytes(buf[60..68].try_into().map_err(|_| "char_id")?),
            account_id: i64::from_le_bytes(buf[68..76].try_into().map_err(|_| "account_id")?),
            guild_name,
        })
    }
}

/// `STRUCT_MOB` — o personagem encarnado.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct StructMob {
    pub char_id: i64,
    pub account_id: i64,
    pub slot: u8,
    pub class: u8,
    pub citizen: u8,
    pub dir: u8,
    pub level: u16,
    pub exp: u32,
    pub hp: u32,
    pub hp_max: u32,
    pub mp: u32,
    pub mp_max: u32,
    pub gold: u32,
    pub guild_id: u16,
    pub pos_x: u16,
    pub pos_y: u16,
    pub name: [u8; 16],
}

impl StructMob {
    /// Monta o mob do personagem escolhido (mesmos campos do selchar, sem duplicar regra).
    #[must_use]
    pub fn from_character(personagem: &Character, citizen: u8) -> Self {
        let selchar = Selchar::from_character(personagem, citizen);
        Self {
            char_id: selchar.char_id,
            account_id: selchar.account_id,
            slot: selchar.slot,
            class: selchar.class,
            citizen: selchar.citizen,
            dir: 0,
            level: selchar.level,
            exp: selchar.exp,
            hp: selchar.hp,
            hp_max: selchar.hp_max,
            mp: selchar.mp,
            mp_max: selchar.mp_max,
            gold: selchar.gold,
            guild_id: selchar.guild_id,
            pos_x: selchar.pos_x,
            pos_y: selchar.pos_y,
            name: selchar.name,
        }
    }

    /// Corpo do `_MSG_CNFCharacterLogin`.
    #[must_use]
    pub fn encode(&self) -> Vec<u8> {
        let mut out = vec![0u8; MOB_LEN];
        out[0..8].copy_from_slice(&self.char_id.to_le_bytes());
        out[8..16].copy_from_slice(&self.account_id.to_le_bytes());
        out[16] = self.slot;
        out[17] = self.class;
        out[18] = self.citizen;
        out[19] = self.dir;
        out[20..22].copy_from_slice(&self.level.to_le_bytes());
        out[24..28].copy_from_slice(&self.exp.to_le_bytes());
        out[28..32].copy_from_slice(&self.hp.to_le_bytes());
        out[32..36].copy_from_slice(&self.hp_max.to_le_bytes());
        out[36..40].copy_from_slice(&self.mp.to_le_bytes());
        out[40..44].copy_from_slice(&self.mp_max.to_le_bytes());
        out[44..48].copy_from_slice(&self.gold.to_le_bytes());
        out[48..50].copy_from_slice(&self.guild_id.to_le_bytes());
        out[50..52].copy_from_slice(&self.pos_x.to_le_bytes());
        out[52..54].copy_from_slice(&self.pos_y.to_le_bytes());
        out[54..70].copy_from_slice(&self.name);
        out
    }

    /// Decodifica o mob (round-trip dos testes e correlação de captura).
    pub fn decode(buf: &[u8]) -> Result<Self, String> {
        if buf.len() < MOB_LEN {
            return Err(format!("struct mob curto: {} B", buf.len()));
        }
        let mut name = [0u8; 16];
        name.copy_from_slice(&buf[54..70]);
        Ok(Self {
            char_id: i64::from_le_bytes(buf[0..8].try_into().map_err(|_| "char_id")?),
            account_id: i64::from_le_bytes(buf[8..16].try_into().map_err(|_| "account_id")?),
            slot: buf[16],
            class: buf[17],
            citizen: buf[18],
            dir: buf[19],
            level: u16::from_le_bytes([buf[20], buf[21]]),
            exp: u32::from_le_bytes([buf[24], buf[25], buf[26], buf[27]]),
            hp: u32::from_le_bytes([buf[28], buf[29], buf[30], buf[31]]),
            hp_max: u32::from_le_bytes([buf[32], buf[33], buf[34], buf[35]]),
            mp: u32::from_le_bytes([buf[36], buf[37], buf[38], buf[39]]),
            mp_max: u32::from_le_bytes([buf[40], buf[41], buf[42], buf[43]]),
            gold: u32::from_le_bytes([buf[44], buf[45], buf[46], buf[47]]),
            guild_id: u16::from_le_bytes([buf[48], buf[49]]),
            pos_x: u16::from_le_bytes([buf[50], buf[51]]),
            pos_y: u16::from_le_bytes([buf[52], buf[53]]),
            name,
        })
    }
}

/// Nome de guilda no campo fixo de 16 bytes (UTF-8 truncado no limite do campo).
fn nome_de_16(texto: &str) -> [u8; 16] {
    let mut destino = [0u8; 16];
    let bytes = texto.as_bytes();
    let limite = bytes.len().min(destino.len());
    destino[..limite].copy_from_slice(&bytes[..limite]);
    destino
}

/// Corpo do `_MSG_CNFAccountLogin`: `count u16`, `versão u16` e 4 registros fixos.
#[must_use]
pub fn character_body(personagens: &[Character], citizen: u8) -> Vec<u8> {
    let mut out = vec![0u8; 4 + SLOT_COUNT * SELCHAR_LEN];
    let visiveis: Vec<_> = personagens.iter().take(SLOT_COUNT).collect();
    let count = u16::try_from(visiveis.len()).unwrap_or(0);
    out[0..2].copy_from_slice(&count.to_le_bytes());
    out[2..4].copy_from_slice(&LAYOUT_VERSION.to_le_bytes());
    for (indice, personagem) in visiveis.iter().enumerate() {
        let inicio = 4 + indice * SELCHAR_LEN;
        Selchar::from_character(personagem, citizen)
            .encode_into(&mut out[inicio..inicio + SELCHAR_LEN]);
    }
    out
}

/// Decodifica o corpo em `(count, registros preenchidos)`.
pub fn decode_character_body(buf: &[u8]) -> Result<(u16, Vec<Selchar>), String> {
    if buf.len() < 4 {
        return Err("corpo sem cabeçalho".to_string());
    }
    let count = u16::from_le_bytes([buf[0], buf[1]]);
    let mut registros = Vec::new();
    for indice in 0..usize::from(count).min(SLOT_COUNT) {
        let inicio = 4 + indice * SELCHAR_LEN;
        let fim = inicio + SELCHAR_LEN;
        if buf.len() < fim {
            return Err(format!("corpo curto para {} registros", indice + 1));
        }
        registros.push(Selchar::decode(&buf[inicio..fim])?);
    }
    Ok((count, registros))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn personagem() -> Character {
        Character {
            id: 7,
            account_id: 3,
            slot: 2,
            name: "Kael".to_string(),
            class: 1,
            level: 41,
            exp: 123456,
            hp: 900,
            mp: 300,
            pos_x: 120,
            pos_y: 88,
            ..Character::default()
        }
    }

    #[test]
    fn selchar_vai_e_volta_do_banco() {
        let corpo = character_body(&[personagem()], 2);
        assert_eq!(corpo.len(), 4 + SLOT_COUNT * SELCHAR_LEN);
        let (count, registros) = decode_character_body(&corpo).expect("corpo");
        assert_eq!(count, 1);
        let r = &registros[0];
        assert_eq!(r.name, *b"Kael\0\0\0\0\0\0\0\0\0\0\0\0");
        assert_eq!((r.level, r.exp), (41, 123456));
        assert_eq!((r.hp, r.mp, r.pos_x, r.pos_y), (900, 300, 120, 88));
        assert_eq!((r.slot, r.class, r.citizen), (2, 1, 2));
    }

    #[test]
    fn mob_vai_e_volta() {
        let mob = StructMob::from_character(&personagem(), 2);
        let voltou = StructMob::decode(&mob.encode()).expect("mob");
        assert_eq!(voltou, mob);
        assert_eq!((voltou.level, voltou.exp, voltou.hp), (41, 123456, 900));
    }
}
