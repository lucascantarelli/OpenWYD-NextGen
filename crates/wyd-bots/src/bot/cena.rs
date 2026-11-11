//! Submódulo de `bot` (ADR-0027): leitura de `CreateMob`.

/// Criatura vista na cena (`_MSG_CreateMob`).
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct SceneMob {
    /// `MobID` do servidor (o mesmo id do abate).
    pub id: u16,
    /// Nome do template, sem o preenchimento NUL.
    pub name: String,
    pub x: i32,
    pub y: i32,
    /// `STRUCT_SCORE` do `_MSG_CreateMob`: o teste cobra que a criatura chegue desenhável.
    pub level: u16,
    pub hp: u32,
    pub max_hp: u32,
    /// `Merchant` do `STRUCT_SCORE` (0 = sem loja).
    pub merchant: u8,
}

/// Nome do `MobName[16]` sem a cauda NUL.
#[must_use]
pub fn nome_do_campo(bytes: &[u8; 16]) -> String {
    let fim = bytes.iter().position(|byte| *byte == 0).unwrap_or(16);
    String::from_utf8_lossy(&bytes[..fim]).to_string()
}
