//! Provisiona a conta de teste do canal (conta + senha argon2id + personagem slot 0).
//!
//! Usado pelo teste vivo (`cena_mobs`/`handshake`/`selecao`), que precisa de credencial real:
//! `create_account` cria a conta sem senha (`NO_CREDENTIAL`) e a autenticação é fail-closed.
//!
//! Gated por `WYD_TEST_DATABASE_URL`/`DATABASE_URL`, como os demais testes de banco. Imprime
//! `CONTA=` e `SENHA=` com `--nocapture` para o operador exportar em `WYD_LIVE_ACCOUNT`/
//! `WYD_LIVE_PASSWORD`. Rodar de novo reaproveita a conta (senha reaplicada, personagem só
//! criado se o slot estiver vazio).

use std::sync::Arc;

use wyd_admin::auth::hash_secret;
use wyd_db::{Accounts, Character, Characters, PgBackend, PoolConfig, Role};

const SENHA: &str = "qa12345678";

fn dsn() -> Option<String> {
    let valor = ["WYD_TEST_DATABASE_URL", "DATABASE_URL"]
        .iter()
        .filter_map(|var| std::env::var(var).ok())
        .map(|valor| valor.trim().to_string())
        .find(|valor| !valor.is_empty());
    if valor.is_none() {
        eprintln!(
            "SKIP: WYD_TEST_DATABASE_URL/DATABASE_URL ausente — conta de teste não provisionada"
        );
    }
    valor
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

#[test]
fn provisiona_conta_e_personagem_do_teste_vivo() {
    let Some(dsn) = dsn() else { return };
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("banco"),
    );

    // Conta fixa: o teste vivo aponta para ela; rodar de novo não duplica nada.
    let login = "qavivo01";
    let hash = hash_secret(SENHA).expect("argon2id");
    match rt.block_on(backend.create_account(login, Role::Player)) {
        Ok(_) => {}
        Err(wyd_db::DbError::ConstraintViolation { .. }) => {}
        Err(error) => panic!("conta: {error}"),
    }
    rt.block_on(backend.set_credentials(login, &hash, Role::Player))
        .expect("senha da conta");

    let conta = rt.block_on(backend.find_by_login(login)).expect("conta");
    let personagens = rt
        .block_on(backend.list_by_account(conta.id))
        .expect("roster");
    // Personagem sempre recriado na posição do teste vivo: ao lado do `Guarda_Carga`
    // (`npc_definition` 1212, merchant 2, 2104,2050), o NPC de diálogo do snapshot.
    for personagem in personagens.iter().filter(|personagem| personagem.slot == 0) {
        rt.block_on(backend.soft_delete(personagem.id))
            .expect("limpa personagem antigo");
    }
    let personagem = Character {
        id: 0,
        account_id: conta.id,
        slot: 0,
        name: "QAVIVO01".to_string(),
        class: 0,
        level: 10,
        exp: 100,
        hp: 100,
        mp: 50,
        hp_max: 100,
        mp_max: 50,
        pos_x: 2103,
        pos_y: 2050,
        ..Character::default()
    };
    rt.block_on(backend.create_character(&personagem))
        .expect("personagem do teste");

    println!("CONTA={login}");
    println!("SENHA={SENHA}");
    println!("SLOT=0");
}
