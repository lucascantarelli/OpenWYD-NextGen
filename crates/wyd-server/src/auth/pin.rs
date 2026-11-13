//! Senha numérica (PIN): política, cadastro, troca e bloqueio persistido.
//!
//! O bloqueio vive em `accounts` (`pin_failures`/`pin_locked_until`): reiniciar o
//! canal não devolve tentativas ao atacante. PIN nunca vai para log nem para
//! resposta de erro.

use wyd_admin::auth::hash_secret;
use wyd_db::{Accounts, PgBackend};

use super::{
    mapear_db, AuthError, PinOutcome, LOCKOUT, MAX_PIN_DIGITS, MAX_PIN_FAILURES, MIN_PIN_DIGITS,
};

/// A conta já tem PIN numérico cadastrado?
pub(crate) fn registered(
    backend: &PgBackend,
    runtime: &tokio::runtime::Runtime,
    login: &str,
) -> Result<bool, AuthError> {
    let atual = runtime
        .block_on(backend.pin_hash(login))
        .map_err(mapear_db)?;
    Ok(atual.is_some())
}

/// Confere o PIN existente ou cadastra o primeiro.
pub(crate) fn secure(
    backend: &PgBackend,
    runtime: &tokio::runtime::Runtime,
    login: &str,
    token: &[u8],
) -> Result<PinOutcome, AuthError> {
    let pin = normalizar(token)?;
    check_lockout(backend, runtime, login)?;
    let atual = runtime
        .block_on(backend.pin_hash(login))
        .map_err(mapear_db)?;
    match atual {
        Some(hash) => {
            if wyd_admin::verify_password(&pin, &hash) {
                reset(backend, runtime, login)?;
                Ok(PinOutcome::Verified)
            } else {
                Err(registrar_falha(backend, runtime, login))
            }
        }
        None => {
            let hash = hash_secret(&pin).map_err(|_| AuthError::PinFormat)?;
            runtime
                .block_on(backend.set_pin_hash(login, &hash))
                .map_err(|error| AuthError::Db(error.to_string()))?;
            reset(backend, runtime, login)?;
            Ok(PinOutcome::Registered)
        }
    }
}

/// Grava um PIN novo. A borda **só** chama depois de `secure` aceitar o PIN atual.
pub(crate) fn change(
    backend: &PgBackend,
    runtime: &tokio::runtime::Runtime,
    login: &str,
    novo: &[u8],
) -> Result<PinOutcome, AuthError> {
    let pin = normalizar(novo)?;
    let hash = hash_secret(&pin).map_err(|_| AuthError::PinFormat)?;
    runtime
        .block_on(backend.set_pin_hash(login, &hash))
        .map_err(|error| AuthError::Db(error.to_string()))?;
    reset(backend, runtime, login)?;
    Ok(PinOutcome::Changed)
}

/// PIN do fio: até 6 bytes, terminação NUL, só dígitos, 4..=6 posições.
pub(crate) fn normalizar(token: &[u8]) -> Result<String, AuthError> {
    let bytes = wyd_protocol::nul_trimmed(token);
    if bytes.len() < MIN_PIN_DIGITS || bytes.len() > MAX_PIN_DIGITS {
        return Err(AuthError::PinFormat);
    }
    if !bytes.iter().all(u8::is_ascii_digit) {
        return Err(AuthError::PinFormat);
    }
    Ok(String::from_utf8_lossy(bytes).to_string())
}

fn check_lockout(
    backend: &PgBackend,
    runtime: &tokio::runtime::Runtime,
    login: &str,
) -> Result<(), AuthError> {
    let (_, segundos) = runtime
        .block_on(backend.pin_guard(login))
        .map_err(mapear_db)?;
    if segundos > 0 {
        return Err(AuthError::Locked {
            segundos: u64::try_from(segundos).unwrap_or(u64::MAX),
        });
    }
    Ok(())
}

fn registrar_falha(
    backend: &PgBackend,
    runtime: &tokio::runtime::Runtime,
    login: &str,
) -> AuthError {
    let limite = i64::from(MAX_PIN_FAILURES);
    let segundos = i64::try_from(LOCKOUT.as_secs()).unwrap_or(i64::MAX);
    match runtime.block_on(backend.pin_failure(login, limite, segundos)) {
        Ok((_, restante)) if restante > 0 => AuthError::Locked {
            segundos: u64::try_from(restante).unwrap_or(u64::MAX),
        },
        Ok(_) => AuthError::BadPin,
        Err(error) => AuthError::Db(error.to_string()),
    }
}

fn reset(
    backend: &PgBackend,
    runtime: &tokio::runtime::Runtime,
    login: &str,
) -> Result<(), AuthError> {
    runtime
        .block_on(backend.pin_success(login))
        .map_err(|error| AuthError::Db(error.to_string()))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn pin_exige_digitos_e_faixa() {
        assert_eq!(normalizar(b"1234\0\0"), Ok("1234".to_string()));
        assert_eq!(normalizar(b"123456"), Ok("123456".to_string()));
        assert_eq!(normalizar(b"123\0\0\0"), Err(AuthError::PinFormat));
        assert_eq!(normalizar(b"12a4\0\0"), Err(AuthError::PinFormat));
        assert_eq!(normalizar(b"\0\0\0\0\0\0"), Err(AuthError::PinFormat));
    }
}
