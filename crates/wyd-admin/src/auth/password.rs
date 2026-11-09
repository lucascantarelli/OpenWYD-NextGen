//! Hash e verificação de senha de login (`argon2id`, AGENTS §3.6).
//!
//! Única borda que toca credencial: o hash nunca sai daqui, o portal só recebe
//! `ok`/`role`. Texto plano não é logado, comparado nem guardado.
//!
//! Parâmetros ficam no default da crate (`Argon2::default()` = argon2id, m=19 MiB,
//! t=2, p=1) — forte o bastante para login humano e barato no gateway.

use argon2::password_hash::rand_core::OsRng;
use argon2::password_hash::{PasswordHash, PasswordHasher, PasswordVerifier, SaltString};
use argon2::Argon2;

/// Tamanho mínimo de senha aceito no seed/hash (não é política de jogo).
pub const MIN_PASSWORD_LEN: usize = 8;

/// Senha curta demais ou hash inválido (nunca carrega a senha na mensagem).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PasswordError {
    /// `password.len() < MIN_PASSWORD_LEN`.
    TooShort,
    /// Falha ao gerar o hash (entropia/parâmetros).
    HashFailed,
}

impl core::fmt::Display for PasswordError {
    fn fmt(&self, f: &mut core::fmt::Formatter<'_>) -> core::fmt::Result {
        match self {
            Self::TooShort => write!(f, "senha com menos de {MIN_PASSWORD_LEN} caracteres"),
            Self::HashFailed => f.write_str("não foi possível gerar o hash da senha"),
        }
    }
}

impl std::error::Error for PasswordError {}

/// Gera o hash argon2id com sal aleatório (`$argon2id$v=19$...`).
pub fn hash_password(password: &str) -> Result<String, PasswordError> {
    if password.len() < MIN_PASSWORD_LEN {
        return Err(PasswordError::TooShort);
    }
    let salt = SaltString::generate(&mut OsRng);
    Argon2::default()
        .hash_password(password.as_bytes(), &salt)
        .map(|hash| hash.to_string())
        .map_err(|_| PasswordError::HashFailed)
}

/// Hash de **segredo curto** (PIN de 4..6 dígitos): mesmo Argon2id, mas sem a política de
/// tamanho da senha de login — PIN tem 4 a 6 caracteres por contrato do cliente.
pub fn hash_secret(segredo: &str) -> Result<String, PasswordError> {
    if segredo.is_empty() {
        return Err(PasswordError::TooShort);
    }
    let salt = SaltString::generate(&mut OsRng);
    Argon2::default()
        .hash_password(segredo.as_bytes(), &salt)
        .map(|hash| hash.to_string())
        .map_err(|_| PasswordError::HashFailed)
}

/// `true` só quando a senha confere com um hash argon2 válido.
///
/// Hash ausente/marcador (`NO_CREDENTIAL`, que não começa com `$argon2`) ou
/// corrompido retorna `false`: conta sem credencial não autentica.
#[must_use]
pub fn verify_password(password: &str, stored_hash: &str) -> bool {
    if password.is_empty() || !stored_hash.starts_with("$argon2") {
        return false;
    }
    let Ok(parsed) = PasswordHash::new(stored_hash) else {
        return false;
    };
    Argon2::default()
        .verify_password(password.as_bytes(), &parsed)
        .is_ok()
}

#[cfg(test)]
mod tests {
    use super::*;
    use wyd_db::NO_CREDENTIAL;

    #[test]
    fn hash_roundtrips_and_is_argon2id() {
        let hash = hash_password("senha-de-dev-123").expect("hash");
        assert!(hash.starts_with("$argon2id$"), "hash: {hash}");
        assert!(verify_password("senha-de-dev-123", &hash));
        assert!(!verify_password("senha-de-dev-124", &hash));
        assert!(!verify_password("", &hash));
    }

    #[test]
    fn short_password_is_rejected() {
        assert_eq!(hash_password("curta"), Err(PasswordError::TooShort));
    }

    #[test]
    fn placeholder_and_garbage_never_authenticate() {
        assert!(!verify_password("qualquer", NO_CREDENTIAL));
        assert!(!verify_password("qualquer", "$argon2id$lixo"));
        assert!(!verify_password("qualquer", ""));
    }

    #[test]
    fn same_password_gets_different_salt() {
        let a = hash_password("mesma-senha-123").expect("a");
        let b = hash_password("mesma-senha-123").expect("b");
        assert_ne!(a, b);
        assert!(verify_password("mesma-senha-123", &a));
        assert!(verify_password("mesma-senha-123", &b));
    }
}
