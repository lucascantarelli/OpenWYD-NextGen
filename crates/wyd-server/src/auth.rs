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

use std::sync::Arc;
use std::time::Duration;

use wyd_admin::auth::hash_secret;
use wyd_admin::verify_password;
use wyd_db::{Accounts, DbError, PgBackend};

/// Falhas de autenticação — cada uma tem um caminho de recusa próprio no log.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum AuthError {
    /// Conta inexistente (mesma recusa de senha errada para o cliente, sem vazar existência).
    UnknownAccount,
    /// Senha de login não confere.
    BadPassword,
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
        let senha = String::from_utf8_lossy(wyd_protocol::nul_trimmed(senha)).to_string();
        if senha.is_empty() {
            return Err(AuthError::BadPassword);
        }
        let registro = self
            .runtime
            .block_on(self.backend.credentials(login))
            .map_err(|error| match error {
                DbError::NotFound { .. } | DbError::NotFoundKey { .. } => AuthError::UnknownAccount,
                outro => AuthError::Db(outro.to_string()),
            })?;
        if verify_password(&senha, &registro.password_hash) {
            Ok(())
        } else {
            Err(AuthError::BadPassword)
        }
    }

    /// Valida o `_MSG_AccountSecure`: confere o PIN existente ou cadastra o primeiro.
    ///
    /// O bloqueio é por conta e vive na memória do processo (o canal é o dono da conexão);
    /// um PIN correto zera a contagem.
    pub fn secure_pin(&self, login: &str, token: &[u8]) -> Result<PinOutcome, AuthError> {
        let pin = normalizar_pin(token)?;
        self.check_lockout(login)?;
        let atual = self
            .runtime
            .block_on(self.backend.pin_hash(login))
            .map_err(|error| match error {
                DbError::NotFound { .. } | DbError::NotFoundKey { .. } => AuthError::UnknownAccount,
                outro => AuthError::Db(outro.to_string()),
            })?;
        match atual {
            Some(hash) => {
                if verify_password(&pin, &hash) {
                    self.reset(login)?;
                    Ok(PinOutcome::Verified)
                } else {
                    Err(self.registrar_falha(login))
                }
            }
            None => {
                let hash = hash_secret(&pin).map_err(|_| AuthError::PinFormat)?;
                self.runtime
                    .block_on(self.backend.set_pin_hash(login, &hash))
                    .map_err(|error| AuthError::Db(error.to_string()))?;
                self.reset(login)?;
                Ok(PinOutcome::Registered)
            }
        }
    }

    /// Grava um PIN novo. A borda **só** chama depois de `secure_pin` aceitar o PIN atual
    /// (`0x0FDE` com `change_numeric = 1`); aqui não há como pular a prova do PIN antigo.
    pub fn change_pin(&self, login: &str, novo: &[u8]) -> Result<PinOutcome, AuthError> {
        let pin = normalizar_pin(novo)?;
        let hash = hash_secret(&pin).map_err(|_| AuthError::PinFormat)?;
        self.runtime
            .block_on(self.backend.set_pin_hash(login, &hash))
            .map_err(|error| AuthError::Db(error.to_string()))?;
        self.reset(login)?;
        Ok(PinOutcome::Changed)
    }

    fn check_lockout(&self, login: &str) -> Result<(), AuthError> {
        let (_, segundos) = self
            .runtime
            .block_on(self.backend.pin_guard(login))
            .map_err(|error| match error {
                DbError::NotFound { .. } | DbError::NotFoundKey { .. } => AuthError::UnknownAccount,
                outro => AuthError::Db(outro.to_string()),
            })?;
        if segundos > 0 {
            return Err(AuthError::Locked {
                segundos: u64::try_from(segundos).unwrap_or(u64::MAX),
            });
        }
        Ok(())
    }

    fn registrar_falha(&self, login: &str) -> AuthError {
        let limite = i64::from(MAX_PIN_FAILURES);
        let segundos = i64::try_from(LOCKOUT.as_secs()).unwrap_or(i64::MAX);
        match self
            .runtime
            .block_on(self.backend.pin_failure(login, limite, segundos))
        {
            Ok((_, restante)) if restante > 0 => AuthError::Locked {
                segundos: u64::try_from(restante).unwrap_or(u64::MAX),
            },
            Ok(_) => AuthError::BadPin,
            Err(error) => AuthError::Db(error.to_string()),
        }
    }

    fn reset(&self, login: &str) -> Result<(), AuthError> {
        self.runtime
            .block_on(self.backend.pin_success(login))
            .map_err(|error| AuthError::Db(error.to_string()))
    }
}

/// PIN do fio: até 6 bytes, terminação NUL, só dígitos, 4..=6 posições.
fn normalizar_pin(token: &[u8]) -> Result<String, AuthError> {
    let bytes = wyd_protocol::nul_trimmed(token);
    if bytes.len() < MIN_PIN_DIGITS || bytes.len() > MAX_PIN_DIGITS {
        return Err(AuthError::PinFormat);
    }
    if !bytes.iter().all(u8::is_ascii_digit) {
        return Err(AuthError::PinFormat);
    }
    Ok(String::from_utf8_lossy(bytes).to_string())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn pin_exige_digitos_e_faixa() {
        assert_eq!(normalizar_pin(b"1234\0\0"), Ok("1234".to_string()));
        assert_eq!(normalizar_pin(b"123456"), Ok("123456".to_string()));
        assert_eq!(normalizar_pin(b"123\0\0\0"), Err(AuthError::PinFormat));
        assert_eq!(normalizar_pin(b"12a4\0\0"), Err(AuthError::PinFormat));
        assert_eq!(normalizar_pin(b"\0\0\0\0\0\0"), Err(AuthError::PinFormat));
    }
}
