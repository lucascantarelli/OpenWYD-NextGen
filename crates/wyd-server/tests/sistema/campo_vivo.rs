//! Primeiro gameplay no servidor **vivo**: personagem novo → mundo → quest do Campo aceita.
//!
//! Gated por `WYD_LIVE_CPSOCK` e `DATABASE_URL`/`WYD_TEST_DATABASE_URL`. O personagem é criado
//! pelo protocolo (`_MSG_CreateCharacter` do `wyd-bots`), não por SQL: é o mesmo caminho da tela
//! de criação do 9662/7662. A prova é o estado real — `character_quest` com `step=1`,
//! `progress=0` — e o segundo clique, sem a chave 451, recusando o objetivo pendente.

#[path = "../common/mod.rs"]
mod common;

use std::sync::Arc;

use wyd_admin::auth::hash_secret;
use wyd_bots::{Bot, BotConfig, Race};
use wyd_db::{Accounts, Characters, PgBackend, PoolConfig, Role};

const SENHA: &str = "qa12345678";
const CONTA: &str = "qacampo01";
const NOME: &str = "QACAMPO01";
const QUEST: &str = "campo_de_treinamento";
/// `MobID` não: a âncora é a coordenada do seed `0046` do Primeiro Treinador.
const TREINADOR: (i32, i32) = (2077, 2017);
/// Chave da primeira porta (`ItemDropList.txt` do `GremlinDecoy`) e kit pago na etapa 1.
const CHAVE_451: i64 = 451;
const QUEST_HEAL_KIT: i64 = 682;
/// Ponto de save do personagem criado (`point_of_save` do template da classe).
const INICIO: (i32, i32) = (2096, 2096);
/// Coordenadas das fixtures removidas pela `0047`.
const FIXTURES: [(i32, i32); 8] = [
    (2108, 2046),
    (2112, 2046),
    (2116, 2046),
    (2120, 2046),
    (2115, 2042),
    (2120, 2048),
    (2125, 2052),
    (2130, 2055),
];

#[test]
fn personagem_novo_entra_e_aceita_a_quest_do_campo() {
    let Ok(endereco) = std::env::var("WYD_LIVE_CPSOCK") else {
        eprintln!("SKIP: WYD_LIVE_CPSOCK ausente");
        return;
    };
    let Some(dsn) = common::dsn_ou_skip("quest do campo") else {
        return;
    };
    let rt = tokio::runtime::Builder::new_multi_thread()
        .worker_threads(1)
        .enable_all()
        .build()
        .expect("runtime");
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("banco"),
    );

    // Conta fixa sem personagem: o roster é limpo a cada execução para o char ser "novo".
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

    let mut cena = bot.read_scene(1500).expect("cena");
    cena.extend(bot.read_scene(1500).expect("cena 2"));
    for (x, y) in FIXTURES {
        let sobra = cena.iter().filter(|mob| mob.x == x && mob.y == y).count();
        assert_eq!(sobra, 0, "fixture removida ainda na cena em ({x},{y})");
    }
    let treinador = cena
        .iter()
        .filter(|mob| mob.x == TREINADOR.0 && mob.y == TREINADOR.1)
        .collect::<Vec<_>>();
    assert_eq!(
        treinador.len(),
        1,
        "Primeiro Treinador deve ter uma única instância na cena"
    );
    let treinador = treinador[0].clone();

    // Aceite: primeiro clique no Primeiro Treinador.
    let painel = bot.quest(i32::from(treinador.id)).expect("aceite da quest");
    assert!(
        painel.contains("aceita"),
        "painel do aceite inesperado: {painel}"
    );

    let personagem = rt
        .block_on(backend.list_by_account(conta.id))
        .expect("roster")
        .into_iter()
        .find(|personagem| personagem.slot == 0)
        .expect("personagem criado pelo protocolo");
    let estado = rt
        .block_on(backend.quest_state(personagem.id, QUEST))
        .expect("estado da quest")
        .expect("linha em character_quest depois do aceite");
    assert_eq!(estado.stage, 0, "Newbie 0 = etapa 1 pendente");
    assert_eq!(estado.reward_mask, 0, "nenhuma recompensa paga no aceite");
    assert!(!estado.completed);

    // Objetivo é real: sem a chave 451 o mesmo treinador recusa e nada muda.
    let painel = bot
        .quest(i32::from(treinador.id))
        .expect("recusa sem prova");
    assert!(
        painel.contains("451"),
        "recusa deveria citar a prova 451: {painel}"
    );
    let depois = rt
        .block_on(backend.quest_state(personagem.id, QUEST))
        .expect("estado da quest")
        .expect("estado preservado");
    assert_eq!(depois.stage, 0, "sem prova o objetivo não avança");
    assert_eq!(depois.reward_mask, 0, "sem prova nada é pago");

    // Passo 1 real: abater o GremlinDecoy (`ItemDropList.txt`: dropa a chave 451).
    // `_MSG_Attack` leva o tile do alvo; o servidor resolve o NPC, rola a tabela do abate e
    // entrega o loot no inventário — nada é escrito à mão no banco.
    // O ataque valida a distância **do personagem** (2 tiles): o primeiro objetivo exige
    // andar até o guarda. Cada passo é um `_MSG_Action` validado no tick.
    let guarda = cena
        .iter()
        .find(|mob| mob.name == "Gremlin Decoy")
        .expect("GremlinDecoy na cena");
    let perto_do_guarda = (guarda.x, guarda.y + 1);
    bot.walk_to(INICIO, perto_do_guarda)
        .expect("caminhada até o guarda");
    bot.kill_at(guarda.id, guarda.x, guarda.y)
        .expect("abate do GremlinDecoy");
    eprintln!(
        "PASS(passo 1): GremlinDecoy abatido em ({},{})",
        guarda.x, guarda.y
    );
    let apos_abate = rt
        .block_on(backend.items(personagem.id))
        .expect("inventário depois do abate");
    assert!(
        apos_abate.iter().any(|linha| linha.item_id == CHAVE_451),
        "a chave 451 precisa estar no inventário: {} linhas",
        apos_abate.len()
    );

    // Anda até o Primeiro Treinador (alcance de conversa) e entrega pelo `_MSG_Quest`.
    bot.walk_to(perto_do_guarda, (TREINADOR.0, TREINADOR.1 + 1))
        .expect("caminhada até o treinador");
    let painel = bot
        .quest(i32::from(treinador.id))
        .expect("entrega da chave 451");
    assert!(
        painel.contains("etapa 1/4"),
        "painel da entrega inesperado: {painel}"
    );
    let avancado = rt
        .block_on(backend.quest_state(personagem.id, QUEST))
        .expect("estado da quest")
        .expect("linha em character_quest");
    assert_eq!(avancado.stage, 1, "etapa 1 entregue (Newbie 1)");
    assert_eq!(avancado.reward_mask, 1, "bit da etapa 1 pago");
    let final_ = rt
        .block_on(backend.items(personagem.id))
        .expect("inventário final");
    assert!(
        !final_.iter().any(|linha| linha.item_id == CHAVE_451),
        "a chave 451 é consumida na entrega"
    );
    assert!(
        final_.iter().any(|linha| linha.item_id == QUEST_HEAL_KIT),
        "o kit 682 da etapa 1 entra no inventário"
    );
    eprintln!(
        "PASS(campo vivo): conta={CONTA} personagem={} id={} quest={QUEST} step=2 progress=1 npc={}",
        personagem.name, personagem.id, treinador.id
    );
}
