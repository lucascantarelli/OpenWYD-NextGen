//! Verificação da senha de login (`_MSG_AccountLogin`) com Argon2id.
//!
//! Recebe o pool e o runtime do processo; não guarda estado. O bloqueio de
//! moderação vale no jogo: a recusa acontece **antes** de comparar a senha.

use wyd_admin::verify_password;
use wyd_db::{Accounts, PgBackend};

use super::{mapear_db, AuthError};

/// Confere a senha do login. Nunca loga senha nem hash.
pub(crate) fn verify(
    backend: &PgBackend,
    runtime: &tokio::runtime::Runtime,
    login: &str,
    senha: &[u8],
) -> Result<(), AuthError> {
    let senha = String::from_utf8_lossy(wyd_protocol::nul_trimmed(senha)).to_string();
    if senha.is_empty() {
        return Err(AuthError::BadPassword);
    }
    let registro = runtime
        .block_on(backend.credentials(login))
        .map_err(mapear_db)?;
    // Bloqueio de moderação vale no jogo: recusa antes de comparar a senha.
    if registro.blocked {
        return Err(AuthError::Blocked);
    }
    if verify_password(&senha, &registro.password_hash) {
        Ok(())
    } else {
        Err(AuthError::BadPassword)
    }
}
