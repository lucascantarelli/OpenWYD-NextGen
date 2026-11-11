//! Regressão do seed de staff em boot real.
//!
//! Sobe o binário `wyd-gateway` contra um Postgres descartável com
//! `WYD_SEED_STAFF=0` e prova duas coisas:
//!
//! 1. o hash antigo é reconciliado com a senha do ambiente mesmo com o flag
//!    desligado;
//! 2. um segundo boot com a mesma senha não regrava o hash (`changed=false`).
//!
//! Sem `WYD_TEST_DATABASE_URL`/`DATABASE_URL` o teste **pula** com mensagem
//! explícita, como os demais testes de Postgres do repositório.

#[path = "../common/mod.rs"]
mod common;

use std::io::{BufRead, BufReader};
use std::process::{Child, Command, Stdio};
use std::sync::mpsc;
use std::time::Duration;

use wyd_admin::{hash_password, verify_password};
use wyd_db::{Accounts, PgBackend, PoolConfig, Role};

const SENHA: &str = "senha-de-boot-123";

fn runtime() -> tokio::runtime::Runtime {
    tokio::runtime::Builder::new_multi_thread()
        .worker_threads(1)
        .enable_all()
        .build()
        .expect("runtime")
}

fn gateway(dsn: &str, login: &str) -> Child {
    Command::new(env!("CARGO_BIN_EXE_wyd-gateway"))
        // O teste prova o caminho de ambiente; um `WYD_ENV_FILE` herdado
        // apontaria para o `.env` real e mudaria a credencial sob teste.
        .env_remove("WYD_ENV_FILE")
        .env("DATABASE_URL", dsn)
        .env("WEB_API_TOKEN", "token-de-teste")
        .env("WYD_GATEWAY_BIND", "127.0.0.1:0")
        .env("WYD_SEED_STAFF", "0")
        .env("PORTAL_ADMIN_USER", login)
        .env("PORTAL_ADMIN_PASSWORD", SENHA)
        .stdin(Stdio::null())
        .stdout(Stdio::piped())
        .stderr(Stdio::null())
        .spawn()
        .expect("sobe o gateway de teste")
}

/// Espera a linha `staff_seed` esperada no stdout do gateway.
fn wait_for_seed_log(child: &mut Child, esperado: &str) {
    let stdout = child.stdout.take().expect("stdout do gateway");
    let (tx, rx) = mpsc::channel();
    let esperado = esperado.to_string();
    std::thread::spawn(move || {
        for line in BufReader::new(stdout).lines().map_while(Result::ok) {
            if line.contains("event=staff_seed") && line.contains(&esperado) {
                let _ = tx.send(());
                return;
            }
        }
    });
    rx.recv_timeout(Duration::from_secs(20))
        .expect("gateway não registrou o seed esperado");
}

fn stop(mut child: Child) {
    let _ = child.kill();
    let _ = child.wait();
}

/// Remove a conta descartável mesmo se uma asserção falhar (Drop).
struct Cleanup {
    dsn: String,
    login: String,
}

impl Drop for Cleanup {
    fn drop(&mut self) {
        let dsn = self.dsn.clone();
        let login = self.login.clone();
        let _ = std::thread::spawn(move || {
            let Ok(rt) = tokio::runtime::Builder::new_current_thread()
                .enable_all()
                .build()
            else {
                return;
            };
            rt.block_on(async {
                let Ok(pool) = sqlx::PgPool::connect(&dsn).await else {
                    return;
                };
                let _ = sqlx::query("DELETE FROM accounts WHERE login = $1")
                    .bind(&login)
                    .execute(&pool)
                    .await;
            });
        })
        .join();
    }
}

#[test]
fn boot_com_flag_zero_reconcilia_hash_e_nao_regrava_quando_confere() {
    let Some(dsn) = common::dsn_ou_skip("seed de boot") else {
        return;
    };
    let rt = runtime();
    let backend = rt
        .block_on(PgBackend::connect(&dsn, PoolConfig::local()))
        .expect("conecta no Postgres de teste");
    let login = format!(
        "seed{}",
        std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map(|d| d.as_nanos())
            .unwrap_or(0)
    );

    let _cleanup = Cleanup {
        dsn: dsn.clone(),
        login: login.clone(),
    };

    // Conta com senha antiga: o primeiro boot precisa trocar o hash.
    let antiga = hash_password("senha-antiga-123").expect("hash antigo");
    rt.block_on(backend.set_credentials(&login, &antiga, Role::Admin))
        .expect("conta de teste");
    let antes = rt
        .block_on(backend.credentials(&login))
        .expect("credencial antes")
        .password_hash;

    let mut primeiro = gateway(&dsn, &login);
    wait_for_seed_log(&mut primeiro, "changed=true");
    let reconciliado = rt
        .block_on(backend.credentials(&login))
        .expect("credencial reconciliada")
        .password_hash;
    stop(primeiro);

    assert!(
        verify_password(SENHA, &reconciliado),
        "hash do banco não autentica a senha do ambiente"
    );
    assert_ne!(antes, reconciliado, "seed não regravou o hash antigo");

    // Segundo boot: a senha já confere, então o hash não pode mudar.
    let mut segundo = gateway(&dsn, &login);
    wait_for_seed_log(&mut segundo, "changed=false");
    let repetido = rt
        .block_on(backend.credentials(&login))
        .expect("credencial repetida")
        .password_hash;
    stop(segundo);

    assert_eq!(reconciliado, repetido, "seed regravou hash que já conferia");
}
