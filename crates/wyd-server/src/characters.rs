//! Seleção de personagem do canal (FASE-08-2): `_MSG_CreateCharacter`,
//! `_MSG_DeleteCharacter` e `_MSG_CharacterLogin` traduzidos em operações de `characters`.
//!
//! A conta **nunca** vem do corpo do pacote: ela é a que abriu a sessão no
//! `_MSG_AccountLogin` (`cpsock::handshake`), e slot/nome só escolhem **entre** os personagens
//! daquela conta. Criação sem dono e seleção de personagem alheio não têm caminho por aqui.
//!
//! O corpo dos pacotes de resposta (`_MSG_CNFNewCharacter`, `_MSG_CNFCharacterLogin`,
//! `_MSG_CNFDeleteCharacter`) é UNVERIFIED (`docs/arquitetura/protocolo.md` §6): este módulo
//! decide o **resultado** e quem responde manda só o cabeçalho com o opcode dele. Inventar
//! `STRUCT_MOB`/`STRUCT_SELCHAR` seria pior que confirmar a operação (invariante 5 do
//! `AGENTS.md`). O corpo entra quando a captura da FASE-08-6 trouxer os offsets.
//!
//! Valores iniciais de um personagem novo são os *defaults* do schema (`0002_personagens.sql`:
//! nível 1, exp 0, hp 1, mp 0, posição 0,0). Fórmula de criação é FASE-10 — nada é inventado
//! aqui.

use std::sync::Arc;

use wyd_db::{Accounts, Character, Characters, DbError, PgBackend, MAX_SLOTS};

/// Tamanho do campo de nome no fio: `char[16]` em `_MSG_CreateCharacter`/`_MSG_DeleteCharacter`.
pub const MAX_NAME_BYTES: usize = 16;

/// Última classe do 7662: 0–3 (TransKnight, Foema, BeastMaster, Huntress, na ordem de
/// `docs/wyd/classes.md` §1). O **índice** numérico de cada classe é hipótese até a captura;
/// o que é fato é a faixa (quatro classes) — fora dela a criação é recusada.
pub const MAX_CLASS: i32 = 3;

/// Nível inicial (default do schema; progressão é FASE-10).
const STARTING_LEVEL: i64 = 1;
/// HP/MP iniciais (defaults do schema).
const STARTING_HP: i64 = 1;
const STARTING_MP: i64 = 0;

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
    /// Nome já usado por outro personagem vivo (UNIQUE parcial do banco).
    Conflict,
    /// Banco indisponível ou dado corrompido.
    Db(String),
    /// Pacote com tamanho diferente do layout portado.
    Layout(String),
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
            Self::Conflict => write!(f, "nome de personagem já usado"),
            Self::Db(motivo) => write!(f, "personagem sem lastro no banco: {motivo}"),
            Self::Layout(motivo) => write!(f, "payload inválido: {motivo}"),
        }
    }
}

impl std::error::Error for SelectionError {}

/// Porta de seleção de **um** canal: mesmo pool e mesmo runtime do handshake (ADR-0021/0022).
pub struct SelectionStore {
    backend: Arc<PgBackend>,
    runtime: Arc<tokio::runtime::Runtime>,
}

impl SelectionStore {
    /// Monta a porta sobre o pool já conectado pelo processo.
    #[must_use]
    pub fn new(backend: Arc<PgBackend>, runtime: Arc<tokio::runtime::Runtime>) -> Self {
        Self { backend, runtime }
    }

    /// Cria o personagem no slot pedido, com o dono vindo do login.
    pub fn create(
        &self,
        login: &str,
        slot: i32,
        name: &[u8],
        class: i32,
    ) -> Result<Character, SelectionError> {
        let slot = check_slot(slot)?;
        let class = check_class(class)?;
        let name = check_name(name)?;
        let account_id = self.account_id(login)?;
        let character = Character {
            // `id` é do banco (`BIGSERIAL`); o INSERT não o envia.
            id: 0,
            account_id,
            slot,
            name,
            class,
            level: STARTING_LEVEL,
            exp: 0,
            hp: STARTING_HP,
            mp: STARTING_MP,
            pos_x: 0,
            pos_y: 0,
            ..Character::default()
        };
        self.runtime
            .block_on(self.backend.create_character(&character))
            .map_err(map_db)
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
}
