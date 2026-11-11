//! Prova viva do **gold inicial** do personagem novo.
//!
//! A regra vem do legado, não de número inventado: a criação copia o `BaseMob` da classe
//! (`crates/wyd-server/assets/basemob/{TK,FM,BM,HT}`, SHA-256 em `docs/wyd/classes.md`) e o
//! `Coin@28` desse `STRUCT_MOB` é 5 000 000 nas quatro classes — já cobrado pelo teste
//! `templates_de_classe_tem_816_bytes_e_campos_nos_offsets_do_c`.
//!
//! O defeito não era o valor: o banco sempre recebeu 5 000 000. Faltava enviar
//! `_MSG_UpdateEtc`, o pacote que carrega `Coin@28` e desenha o ouro no cliente (o par
//! `SendEtc`/`SendScore` do legado). O teste falha se o cliente entrar no mundo sem esse
//! pacote ou com outro valor.
//!
//! Gated por `WYD_LIVE_CPSOCK` + `DATABASE_URL`.

use std::sync::Arc;

use wyd_admin::auth::hash_secret;
use wyd_bots::{Bot, BotConfig, Race};
use wyd_db::{Accounts, Characters, PgBackend, PoolConfig, Role};

const SENHA: &str = "qa12345678";
const CONTA: &str = "qagold01";
const NOME: &str = "QGOLD01";
/// `Coin@28` do `BaseMob` das quatro classes (TK/FM/BM/HT), em `assets/basemob`.
const COIN_BASE_MOB: i32 = 5_000_000;

fn dsn() -> Option<String> {
    let valor = ["WYD_TEST_DATABASE_URL", "DATABASE_URL"]
        .iter()
        .filter_map(|var| std::env::var(var).ok())
        .map(|valor| valor.trim().to_string())
        .find(|valor| !valor.is_empty());
    if valor.is_none() {
        eprintln!("SKIP: DATABASE_URL ausente — prova viva do gold não roda");
    }
    valor
}

#[test]
fn personagem_novo_entra_no_mundo_com_o_gold_do_basemob() {
    let Ok(endereco) = std::env::var("WYD_LIVE_CPSOCK") else {
        eprintln!("SKIP: WYD_LIVE_CPSOCK ausente");
        return;
    };
    let Some(dsn) = dsn() else { return };
    let rt = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(1)
        .enable_all()
        .build()
        .expect("runtime");
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("banco"),
    );

    let hash = hash_secret(SENHA).expect("argon2id");
    match rt.block_on(backend.create_account(CONTA, Role::Player)) {
        Ok(_) => {}
        Err(wyd_db::DbError::ConstraintViolation { .. }) => {}
        Err(error) => panic!("conta: {error}"),
    }
    rt.block_on(backend.set_credentials(CONTA, &hash, Role::Player))
        .expect("senha da conta");
    let conta = rt.block_on(backend.find_by_login(CONTA)).expect("conta");
    for personagem in rt
        .block_on(backend.list_by_account(conta.id))
        .expect("roster")
    {
        rt.block_on(backend.soft_delete(personagem.id))
            .expect("limpa personagem antigo");
    }

    let mut config = BotConfig::new(endereco.clone(), CONTA, SENHA);
    config.pin = "2468".to_string();
    config.slot = 0;
    // `connect_novo` para no `CNFCharacterLogin`; o `UpdateScore`/`UpdateEtc` do login ficam
    // no buffer e são lidos logo em seguida.
    let mut bot = Bot::connect_novo(&config, Race::TransKnight, NOME, Race::TransKnight.class())
        .expect("criação + entrada no mundo");

    let coin = bot.read_etc_coin(3_000).expect("_MSG_UpdateEtc");
    assert_eq!(
        coin,
        Some(COIN_BASE_MOB),
        "entrada no mundo sem o gold do `BaseMob` (Coin@28 = {COIN_BASE_MOB}) no `_MSG_UpdateEtc`"
    );

    // O mesmo valor no `STRUCT_MOB` do login (o blob do `CNFCharacterLogin`): se divergir, o
    // banco e o pacote contam histórias diferentes.
    let personagem = rt
        .block_on(backend.list_by_account(conta.id))
        .expect("roster")
        .into_iter()
        .find(|linha| linha.name.trim() == NOME)
        .expect("personagem criado");
    assert_eq!(
        personagem.gold,
        i64::from(COIN_BASE_MOB),
        "gold persistido diverge do `BaseMob`"
    );
    eprintln!(
        "PASS(gold): UpdateEtc coin={} e characters.gold={} (BaseMob Coin@28)",
        coin.unwrap_or_default(),
        personagem.gold
    );
}
