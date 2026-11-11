//! Autenticação de conta do canal: senha de login + **senha numérica** (PIN).
//!
//! Ordem do fluxo (ver `docs/wyd/senha-numerica.md`): `_MSG_AccountLogin` (senha) →
//! `_MSG_CNFAccountLogin` → `_MSG_AccountSecure` (`0x0FDE`, PIN) → **só então** a cerca de
//! sessão abre (`session.rs`). Recusa de senha não recebe `CNF`; recusa de PIN recebe
//! `_MSG_AccountSecureFail` (`0x0FDF`) e a conexão fecha sem sessão.
//!
//! Regras que não podem quebrar: hashes só Argon2id (`AGENTS.md` §3.6), PIN nunca em log nem
//! em resposta de erro, e o limite de 5 falhas por 10 minutos do plano. O PIN é um campo de
//! até 6 dígitos no fio (`numeric_token[6]` em +12); a política aqui aceita 4 a 6 dígitos.
//!
//! `password` cobre a senha de login; `pin` cobre cadastro/troca/bloqueio do PIN.

mod password;
mod pin;

use std::sync::Arc;
use std::time::Duration;

use wyd_db::{DbError, PgBackend};

/// Falhas de autenticação — cada uma tem um caminho de recusa próprio no log.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum AuthError {
    /// Conta inexistente (mesma recusa de senha errada para o cliente, sem vazar existência).
    UnknownAccount,
    /// Senha de login não confere.
    BadPassword,
    /// Conta bloqueada pela moderação (motivo vive no painel, não vai ao cliente).
    Blocked,
    /// PIN fora da política (não numérico ou fora de 4..=6 dígitos).
    PinFormat,
    /// PIN não confere.
    BadPin,
    /// Falhas demais: bloqueio temporário.
    Locked {
        /// Segundos restantes do bloqueio (vai para o log, nunca para o cliente).
        segundos: u64,
    },
    /// Banco indisponível — fail-closed, nunca "deixa passar".
    Db(String),
}

impl core::fmt::Display for AuthError {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::UnknownAccount => write!(f, "conta inexistente"),
            Self::BadPassword => write!(f, "senha de login não confere"),
            Self::Blocked => write!(f, "conta bloqueada pela moderação"),
            Self::PinFormat => write!(f, "PIN fora da política de dígitos"),
            Self::BadPin => write!(f, "senha numérica não confere"),
            Self::Locked { segundos } => {
                write!(f, "bloqueado por falhas de PIN ({segundos}s restantes)")
            }
            Self::Db(motivo) => write!(f, "banco indisponível: {motivo}"),
        }
    }
}

impl std::error::Error for AuthError {}

/// Falhas de PIN antes do bloqueio e a janela em que elas contam.
pub const MAX_PIN_FAILURES: u32 = 5;
/// Janela e duração do bloqueio (as duas de 10 min, como no plano).
pub const LOCKOUT: Duration = Duration::from_secs(600);
/// PIN mínimo/máximo: o campo do fio tem 6 bytes.
pub const MIN_PIN_DIGITS: usize = 4;
/// Teto de dígitos do campo `numeric_token[6]`.
pub const MAX_PIN_DIGITS: usize = 6;

/// Resultado de um PIN aceito.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PinOutcome {
    /// Conta já tinha PIN e ele confere.
    Verified,
    /// Primeira entrada: o PIN informado passou a ser o da conta.
    Registered,
    /// Troca concluída: o PIN novo substituiu o antigo.
    Changed,
}

/// Converte falha de banco na recusa de autenticação correspondente.
pub(crate) fn mapear_db(error: DbError) -> AuthError {
    match error {
        DbError::NotFound { .. } | DbError::NotFoundKey { .. } => AuthError::UnknownAccount,
        outro => AuthError::Db(outro.to_string()),
    }
}

/// Porta de autenticação do processo: mesmo pool e runtime do handshake.
///
/// O bloqueio do PIN é **persistido** em `accounts` (`pin_failures`/`pin_locked_until`):
/// reiniciar o canal não devolve tentativas ao atacante.
pub struct Authenticator {
    backend: Arc<PgBackend>,
    runtime: Arc<tokio::runtime::Runtime>,
}

impl Authenticator {
    /// Monta a porta sobre o pool já conectado pelo processo.
    #[must_use]
    pub fn new(backend: Arc<PgBackend>, runtime: Arc<tokio::runtime::Runtime>) -> Self {
        Self { backend, runtime }
    }

    /// Confere a senha do `_MSG_AccountLogin`. Nunca loga senha nem hash.
    pub fn verify_password(&self, login: &str, senha: &[u8]) -> Result<(), AuthError> {
        password::verify(self.backend.as_ref(), self.runtime.as_ref(), login, senha)
    }

    /// A conta já tem PIN numérico cadastrado?
    ///
    /// O cliente 7662 só envia `_MSG_AccountSecure` para conta que já tem PIN; conta
    /// nova manda `_MSG_CharacterLogin` direto. Exigir PIN de quem nunca cadastrou
    /// recusava a entrada com `seleção antes da senha numérica` (o "Falha ao conectar").
    pub fn pin_registered(&self, login: &str) -> Result<bool, AuthError> {
        pin::registered(self.backend.as_ref(), self.runtime.as_ref(), login)
    }

    /// Valida o `_MSG_AccountSecure`: confere o PIN existente ou cadastra o primeiro.
    pub fn secure_pin(&self, login: &str, token: &[u8]) -> Result<PinOutcome, AuthError> {
        pin::secure(self.backend.as_ref(), self.runtime.as_ref(), login, token)
    }

    /// Grava um PIN novo. A borda **só** chama depois de `secure_pin` aceitar o PIN atual
    /// (`0x0FDE` com `change_numeric = 1`); aqui não há como pular a prova do PIN antigo.
    pub fn change_pin(&self, login: &str, novo: &[u8]) -> Result<PinOutcome, AuthError> {
        pin::change(self.backend.as_ref(), self.runtime.as_ref(), login, novo)
    }
}
