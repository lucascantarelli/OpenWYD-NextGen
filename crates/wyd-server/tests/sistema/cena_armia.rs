//! Prova viva da cena **por proximidade**: um personagem novo entrando em Armia recebe os
//! NPCs da cidade em volta — e só eles.
//!
//! Sem a entrega por raio o servidor manda os 6 110 NPCs do mapa no login (~1,4 MB em
//! `_MSG_CreateMob` de 232 B) e o cliente 7662 não desenha nenhum. O teste falha se a cena
//! voltar a vir inteira, se a criatura chegar sem nome, ou se não houver NPC perto do spawn.
//!
//! Gated por `WYD_LIVE_CPSOCK` + `DATABASE_URL`; cria a conta/personagem pelo protocolo.

#[path = "../common/mod.rs"]
mod common;

use std::sync::Arc;

use wyd_admin::auth::hash_secret;
use wyd_bots::{Bot, BotConfig, Race};
use wyd_db::{Accounts, Characters, PgBackend, PoolConfig, Role};

const SENHA: &str = "qa12345678";
const CONTA: &str = "qaarmia01";
const NOME: &str = "QARMIA01";
/// Raio da cena (`world::DEFAULT_SCENE_VIEW_TILES`); o teste cobra o teto com folga.
/// Subiu de 24 para 56 em 2026-11-11: com 24 os mercadores de Armia em `y >= 2121`
/// (montarias, sets, armas) e os skillmasters (ForeLearner em 2090,2047) ficavam fora da
/// cena e a cidade parecia incompleta. Raio 56 entrega 71 NPCs em volta do spawn.
const RAIO: i32 = 56;
/// Ponto de save do personagem criado (`point_of_save` do template da classe).
const INICIO: (i32, i32) = (2096, 2096);

fn distancia(a: (i32, i32), b: (i32, i32)) -> i32 {
    (a.0 - b.0).abs().max((a.1 - b.1).abs())
}

#[test]
fn personagem_novo_em_armia_recebe_os_npcs_da_cidade_e_so_os_proximos() {
    let Ok(endereco) = std::env::var("WYD_LIVE_CPSOCK") else {
        eprintln!("SKIP: WYD_LIVE_CPSOCK ausente");
        return;
    };
    let Some(dsn) = common::dsn_ou_skip("prova viva da cena") else {
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
    // Uma segunda leitura pega o que chegou no poll seguinte; a cena por proximidade é
    // pequena e fecha em uma varredura, então o conjunto tem de estabilizar.
    cena.extend(bot.read_scene(1500).expect("cena 2"));
    let total = cena.len();
    eprintln!("PASS(cena): {total} mobs na cena de Armia");

    // A entrega é por raio: o mapa inteiro (6 110) nunca pode vir de uma vez.
    assert!(
        total < 500,
        "a cena veio com {total} mobs — a entrega por proximidade não está ativa"
    );
    assert!(total >= 2, "cena vazia: só o próprio personagem ({total})");

    // Todos dentro do raio (folga de 1 tile pela posição do próprio char).
    let fora: Vec<_> = cena
        .iter()
        .filter(|mob| distancia((mob.x, mob.y), INICIO) > RAIO + 1)
        .collect();
    assert!(
        fora.is_empty(),
        "{} mobs fora do raio de {RAIO} tiles: {:?}",
        fora.len(),
        fora.iter()
            .take(5)
            .map(|mob| (mob.name.clone(), mob.x, mob.y))
            .collect::<Vec<_>>()
    );

    // Todo mob chega desenhável: nome não vazio e sem lixo de controle.
    for mob in &cena {
        assert!(
            !mob.name.trim().is_empty(),
            "mob {} em ({},{}) sem nome",
            mob.id,
            mob.x,
            mob.y
        );
        assert!(
            !mob.name.chars().any(char::is_control),
            "mob {} com nome inválido: {:?}",
            mob.id,
            mob.name
        );
    }

    let npcs: Vec<_> = cena
        .iter()
        .filter(|mob| mob.name != NOME && !(mob.x == INICIO.0 && mob.y == INICIO.1))
        .collect();
    assert!(
        npcs.len() >= 5,
        "Armia deveria ter NPCs em volta do spawn: {} encontrados",
        npcs.len()
    );
    assert!(
        npcs.iter().any(|npc| npc.merchant != 0),
        "nenhum NPC de loja perto do spawn: {:?}",
        npcs.iter()
            .take(8)
            .map(|npc| (npc.name.clone(), npc.merchant))
            .collect::<Vec<_>>()
    );
    // Amostra lida dos pacotes decodificados: id, nome, merchant, posição e score.
    for npc in npcs.iter().take(8) {
        eprintln!(
            "AMOSTRA id={} nome={} merchant={} pos=({},{}) level={} hp={}/{}",
            npc.id, npc.name, npc.merchant, npc.x, npc.y, npc.level, npc.hp, npc.max_hp
        );
    }
    eprintln!(
        "PASS(armia): {} NPCs em volta de {:?}, {} com merchant, menor/maior id {}..{}",
        npcs.len(),
        INICIO,
        npcs.iter().filter(|npc| npc.merchant != 0).count(),
        npcs.iter().map(|npc| npc.id).min().unwrap_or(0),
        npcs.iter().map(|npc| npc.id).max().unwrap_or(0)
    );
}
