//! Bloqueio do PIN persistido entre processos: as tentativas vivem em `accounts`, então um
//! novo `Authenticator` (o que um reinício do canal faz) encontra o mesmo estado.

use std::sync::Arc;
use std::time::Duration;

use wyd_admin::hash_password;
use wyd_db::{Accounts, PgBackend, PoolConfig, Role};
use wyd_server::auth::{AuthError, Authenticator, PinOutcome, MAX_PIN_FAILURES};

const SENHA: &str = "SenhaForte1";
const PIN: &str = "1357";

fn dsn() -> String {
    ["WYD_TEST_DATABASE_URL", "DATABASE_URL"]
        .iter()
        .filter_map(|var| std::env::var(var).ok())
        .map(|valor| valor.trim().to_string())
        .find(|valor| !valor.is_empty())
        .expect("WYD_TEST_DATABASE_URL/DATABASE_URL para o teste de PIN")
}

fn runtime() -> Arc<tokio::runtime::Runtime> {
    Arc::new(
        tokio::runtime::Builder::new_multi_thread()
            .worker_threads(1)
            .enable_all()
            .build()
            .expect("runtime"),
    )
}

fn conta(backend: &PgBackend, rt: &tokio::runtime::Runtime, prefixo: &str) -> String {
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    let login = format!("{prefixo}{:011}", nanos % 100_000_000_000);
    rt.block_on(backend.create_account(&login, Role::Player))
        .expect("conta");
    let hash = hash_password(SENHA).expect("hash");
    rt.block_on(backend.set_credentials(&login, &hash, Role::Player))
        .expect("credencial");
    login
}

fn falhar(auth: &Authenticator, login: &str, pin: &str) -> AuthError {
    auth.secure_pin(login, pin.as_bytes())
        .expect_err("PIN errado não passa")
}

#[test]
fn bloqueio_sobrevive_a_novo_processo() {
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn(), PoolConfig::local()))
            .expect("banco"),
    );
    let login = conta(&backend, &rt, "pp");
    // Primeiro cadastro (conta sem PIN).
    let primeiro = Authenticator::new(Arc::clone(&backend), Arc::clone(&rt));
    assert_eq!(
        primeiro
            .secure_pin(&login, PIN.as_bytes())
            .expect("cadastro"),
        PinOutcome::Registered
    );
    // 4 falhas: ainda é PIN errado, não bloqueio.
    for _ in 0..(MAX_PIN_FAILURES - 1) {
        assert_eq!(falhar(&primeiro, &login, "0000"), AuthError::BadPin);
    }
    // A 5ª falha arma o bloqueio.
    assert!(matches!(
        falhar(&primeiro, &login, "0000"),
        AuthError::Locked { .. }
    ));
    drop(primeiro);
    // "Reinício": outra instância, mesmo banco.
    let depois = Authenticator::new(Arc::clone(&backend), Arc::clone(&rt));
    assert!(
        matches!(
            depois.secure_pin(&login, PIN.as_bytes()),
            Err(AuthError::Locked { .. })
        ),
        "bloqueio tem de sobreviver ao reinício do processo"
    );
    assert!(matches!(
        falhar(&depois, &login, "0000"),
        AuthError::Locked { .. }
    ));
}

#[test]
fn pin_correto_zera_falhas_para_o_proximo_processo() {
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn(), PoolConfig::local()))
            .expect("banco"),
    );
    let login = conta(&backend, &rt, "pz");
    let antes = Authenticator::new(Arc::clone(&backend), Arc::clone(&rt));
    antes.secure_pin(&login, PIN.as_bytes()).expect("cadastro");
    assert_eq!(falhar(&antes, &login, "0000"), AuthError::BadPin);
    assert_eq!(falhar(&antes, &login, "9999"), AuthError::BadPin);
    assert_eq!(
        antes.secure_pin(&login, PIN.as_bytes()).expect("PIN certo"),
        PinOutcome::Verified
    );
    drop(antes);
    // Duas falhas antigas não podem voltar depois do reinício.
    let depois = Authenticator::new(Arc::clone(&backend), Arc::clone(&rt));
    assert_eq!(
        falhar(&depois, &login, "0000"),
        AuthError::BadPin,
        "contador zerado pelo PIN correto"
    );
    let (falhas, _) = rt.block_on(backend.pin_guard(&login)).expect("guard");
    assert!(falhas <= 1, "contador não pode continuar em 2");
}

#[test]
fn pin_gravado_nao_e_texto_plano() {
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn(), PoolConfig::local()))
            .expect("banco"),
    );
    let login = conta(&backend, &rt, "ph");
    let auth = Authenticator::new(Arc::clone(&backend), Arc::clone(&rt));
    auth.secure_pin(&login, PIN.as_bytes()).expect("cadastro");
    let hash = rt
        .block_on(backend.pin_hash(&login))
        .expect("hash")
        .expect("PIN cadastrado");
    assert_ne!(hash, PIN, "PIN nunca em claro no banco");
    assert!(hash.starts_with("$argon2"), "hash Argon2id: {hash:.12}…");
    assert!(!hash.contains(PIN));
    let _ = Duration::from_secs(0);
}
