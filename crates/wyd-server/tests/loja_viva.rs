//! Prova viva da **loja de NPC**: mercador de Armia abre catálogo e vende pelo preço do
//! servidor.
//!
//! O teste usa o caminho real do cliente, sem atalho de banco: `_MSG_REQShopList` no
//! `MobID` da criatura vista na cena (o servidor resolve o `npc_definition` no mundo vivo) e
//! `_MSG_Buy` com o preço do catálogo. Falha se o catálogo vier vazio, se o `_MSG_ShopList`
//! não chegar, ou se nenhum mercador entregar `_MSG_SendItem`.
//!
//! O personagem novo nasce com gold 0, então a compra que fecha é a de uma posição cujo
//! preço em `item_catalog` é 0 (existem 2 083 linhas assim na base). Compra com preço
//! divergente é recusada pelo servidor — o `Coin` do pacote nunca decide o preço.
//!
//! Gated por `WYD_LIVE_CPSOCK` + `DATABASE_URL`.

use std::sync::Arc;

use wyd_admin::auth::hash_secret;
use wyd_bots::{Bot, BotConfig, Race};
use wyd_db::{Accounts, Characters, PgBackend, PoolConfig, Role};

const SENHA: &str = "qa12345678";
const CONTA: &str = "qaloja01";
const NOME: &str = "QALOJA01";

fn dsn() -> Option<String> {
    let valor = ["WYD_TEST_DATABASE_URL", "DATABASE_URL"]
        .iter()
        .filter_map(|var| std::env::var(var).ok())
        .map(|valor| valor.trim().to_string())
        .find(|valor| !valor.is_empty());
    if valor.is_none() {
        eprintln!("SKIP: DATABASE_URL ausente — prova viva da loja não roda");
    }
    valor
}

#[test]
fn mercador_de_armia_abre_catalogo_e_entrega_item_comprado() {
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
    let mut bot = Bot::connect_novo(&config, Race::TransKnight, NOME, Race::TransKnight.class())
        .expect("criação + entrada no mundo");

    let cena = bot.read_scene(1_500).expect("cena");
    let mercadores: Vec<_> = cena
        .iter()
        .filter(|mob| mob.merchant != 0 && mob.name != NOME)
        .collect();
    assert!(
        !mercadores.is_empty(),
        "nenhum mercador na cena de Armia: {:?}",
        cena.iter()
            .map(|mob| (mob.name.clone(), mob.merchant))
            .collect::<Vec<_>>()
    );
    eprintln!("PASS(cena): {} mercadores em Armia", mercadores.len());

    let mut comprado = None;
    for mercador in mercadores.iter().take(16) {
        let itens = bot.open_shop(mercador.id).expect("abre loja");
        assert!(
            !itens.is_empty(),
            "catálogo vazio em {} (MobID {})",
            mercador.name,
            mercador.id
        );
        eprintln!(
            "LOJA nome={} mob={} itens={} primeiro={:?}",
            mercador.name,
            mercador.id,
            itens.len(),
            itens.first().copied()
        );
        // Preço é do servidor: com gold 0, só fecha a posição cujo preço de catálogo é 0.
        bot.buy(mercador.id, 0, 0).expect("_MSG_Buy");
        if let Some((tipo, slot, item)) = bot.read_send_item(1_200).expect("send item") {
            comprado = Some((mercador.name.clone(), tipo, slot, item));
            break;
        }
    }

    let (nome, tipo, slot, item) = comprado.expect(
        "nenhum mercador entregou item de preço 0 com gold 0 — catálogo ou preço quebrados",
    );
    assert_eq!(
        tipo, 1,
        "item comprado tem de entrar na mochila (invType 1)"
    );
    assert_ne!(
        i16::from_le_bytes([item[0], item[1]]),
        0,
        "item comprado chegou vazio"
    );
    eprintln!(
        "PASS(loja): {nome} entregou item {} no slot {slot} da mochila",
        i16::from_le_bytes([item[0], item[1]])
    );

    // O servidor é a autoridade do preço: posição com preço > 0 recusada com `Coin` 0.
    bot.buy(mercadores[0].id, 1, 0)
        .expect("_MSG_Buy divergente");
    assert!(
        bot.read_send_item(800).expect("send item").is_none(),
        "compra com preço divergente não pode entregar item"
    );
    eprintln!("PASS(preco): compra com Coin fora do catálogo foi recusada");
}
