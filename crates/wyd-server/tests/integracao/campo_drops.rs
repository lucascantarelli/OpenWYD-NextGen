//! GAME-18 — drops dos guardiões e do boss pelo pipeline real de morte/loot.
//!
//! O bot de fio não consegue matar `GiantCrill`/`OrcSniper`/`OrcWarrior` no nível 1: o dano
//! do template é 1 e os alvos têm 1 400–2 500 HP. Em vez de abrir um caminho paralelo de drop,
//! este teste usa as **mesmas peças do servidor** no caminho real:
//!
//! `Engine::attack_at` (morte no mundo) → `Gameplay::loot_at_death` (tabela do snapshot) →
//! `gameplay::add_item` (entrega no inventário) → `quest_engage`/`apply_quest_reward`.
//!
//! Só o HP do alvo é reduzido para 1 no fixture; template, nome, posição e tabela de drop vêm
//! do seed `0046` no Postgres. O teste também cobre o NPC `Sephira +9` (migration `0049`) e
//! prova que o `ReqLvl` real do catálogo é usado no bônus e na validação de equipagem.

#[path = "../common/mod.rs"]
mod common;

use std::sync::Arc;

use wyd_core::{
    equip, ComponentStore, CoreError, EntityKind, EquipmentRules, Inventory, Item, ItemRules,
    Position, Stats,
};
use wyd_db::{ItemRow, PgBackend, PoolConfig};
use wyd_server::character::EquipmentCatalog;
use wyd_server::config::{GameConfig, LiveConfig, PgConfigSource};
use wyd_server::gameplay::{
    add_item, apply_quest_reward, quest_engage, Gameplay, QuestEngage, QuestReward, RefineRequest,
    MERCHANT_TRAINER_CHIEF, MERCHANT_TRAINER_FIRST, MERCHANT_TRAINER_SECOND,
    MERCHANT_TRAINER_THIRD, QUEST_PROOF_EMBLEM, QUEST_PROOF_FIRST, QUEST_PROOF_LAST,
    QUEST_PROOF_SECOND,
};
use wyd_server::personagens::template_items;
use wyd_server::world::{npc_specs, ActorSkin, Engine};
use wyd_sim::formula::refine::Powder;
use wyd_sim::roll::XorShift64;

struct Db {
    backend: Arc<PgBackend>,
    runtime: Arc<tokio::runtime::Runtime>,
    config: Arc<GameConfig>,
}

/// Aplica a série no banco de teste (idempotente, com ledger): o seed `0049` do NPC
/// Sephira +9 não pode depender de um banco migrado à mão antes do teste.
fn migrar(dsn: &str, rt: &tokio::runtime::Runtime) {
    let dir = std::path::PathBuf::from(env!("CARGO_MANIFEST_DIR"))
        .join("..")
        .join("wyd-db")
        .join("migrations");
    rt.block_on(wyd_db::migrate::run_dir(dsn, &dir, false))
        .expect("migrations aplicadas no banco de teste");
}

fn banco(contexto: &str) -> Option<Db> {
    let dsn = common::dsn_ou_skip(contexto)?;
    let runtime = Arc::new(
        tokio::runtime::Builder::new_multi_thread()
            .worker_threads(1)
            .enable_all()
            .build()
            .expect("runtime"),
    );
    migrar(&dsn, &runtime);
    let backend = Arc::new(
        runtime
            .block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("Postgres da stack"),
    );
    let source = PgConfigSource::new((*backend).clone(), Arc::clone(&runtime));
    let config = Arc::new(source.load_current().expect("snapshot de configuração"));
    Some(Db {
        backend,
        runtime,
        config,
    })
}

fn personagem(x: i32, y: i32) -> wyd_db::Character {
    wyd_db::Character {
        id: 1,
        account_id: 1,
        slot: 0,
        name: "QADROP".to_string(),
        class: 0,
        level: 1,
        exp: 0,
        hp: 80,
        mp: 45,
        hp_max: 83,
        mp_max: 46,
        pos_x: i64::from(x),
        pos_y: i64::from(y),
        ..wyd_db::Character::default()
    }
}

fn matar(
    engine: &mut Engine,
    spec: &wyd_server::world::NpcSpec,
    x: i32,
    y: i32,
) -> wyd_server::world::AttackOutcome {
    let mut alvo = spec.clone();
    // Fixture de velocidade: os templates reais têm 1 400–2 500 HP e o dano do nível 1 é 1.
    // Nome, template, posição e tabela de drop continuam vindo do seed.
    alvo.stats.hp = 1;
    alvo.stats.max_hp = 1;
    engine.sync_npcs(vec![alvo]);
    let jogador = engine
        .enter(&personagem(x, y), ActorSkin::default())
        .expect("entra no mundo");
    engine
        .attack_at(jogador.entity, spec.position.x, spec.position.y, 1_000)
        .expect("golpe aceito")
        .expect("NPC no alcance")
}

fn drop_unico(gameplay: &Gameplay, template: &str, item: i32) -> i16 {
    let mut die = XorShift64::from_seed(0xC0FFEE);
    let loot = gameplay
        .loot_at_death(template, &mut die, false)
        .expect("tabela de drop do template");
    let drop = loot
        .drops
        .iter()
        .find(|drop| drop.item_index == item)
        .unwrap_or_else(|| panic!("{template} não dropou {item}: {:?}", loot.drops));
    drop.qty
}

#[test]
fn guardioes_e_boss_entregam_as_provas_pelo_combate_e_a_quest_consome_uma_vez() {
    let Some(db) = banco("drops do Campo no combate") else {
        return;
    };
    let gameplay = Gameplay::new(Arc::new(LiveConfig::new((*db.config).clone())));
    let specs = npc_specs(&db.config, 0);

    let guardioes = [
        ("GremlinDecoy", QUEST_PROOF_FIRST as i32, (2076, 2022)),
        ("GiantCrill", QUEST_PROOF_SECOND as i32, (2146, 1997)),
        ("OrcSniper", QUEST_PROOF_LAST as i32, (2078, 1971)),
        ("OrcWarrior", QUEST_PROOF_EMBLEM as i32, (2135, 1948)),
    ];

    // Arma real do catálogo com `ReqLvl` > 0. O template da classe traz a Adaga (861,
    // `ReqLvl` 0); a troca aqui é fixture de bônus, não caminho de jogo.
    let mut items = template_items(0);
    items.retain(|linha| linha.slot != 6);
    items.push(ItemRow {
        slot: 6,
        item_id: 862,
        qty: 1,
        ..ItemRow::default()
    });
    let arma = items
        .iter()
        .find(|linha| linha.slot == 6 && linha.item_id != 0)
        .expect("arma equipada");
    let req_level = db
        .runtime
        .block_on(db.backend.item_req_level(arma.item_id))
        .expect("ReqLvl do catálogo")
        .unwrap_or(0);
    assert!(
        req_level > 0,
        "o equipamento {} precisa de ReqLvl no catálogo",
        arma.item_id
    );

    let treinadores = [
        (MERCHANT_TRAINER_FIRST, 0),
        (MERCHANT_TRAINER_SECOND, 1),
        (MERCHANT_TRAINER_THIRD, 2),
        (MERCHANT_TRAINER_CHIEF, 3),
    ];
    let mut mask = 0i32;

    for (indice, (template, prova, (x, y))) in guardioes.into_iter().enumerate() {
        let spec = specs
            .iter()
            .find(|npc| npc.template == template)
            .unwrap_or_else(|| panic!("{template} ausente do seed"));
        assert_eq!(
            (spec.position.x, spec.position.y),
            (x, y),
            "{template} mudou de coordenada no seed"
        );
        assert!(
            spec.exp > 0 && spec.kind == EntityKind::Monster,
            "{template} precisa ser criatura de combate no snapshot"
        );

        let mut engine = Engine::new(20, Arc::new(wyd_core::Metrics::new())).expect("motor");
        let golpe = matar(&mut engine, spec, x + 1, y);
        assert!(golpe.killed, "{template} não morreu no golpe");
        assert_eq!(golpe.template, template, "template do abate");
        eprintln!(
            "PASS(abate): {template} morreu em ({x},{y}) e liberou {}",
            golpe.template
        );

        let qty = drop_unico(&gameplay, &golpe.template, prova);
        assert_eq!(qty, 1, "a prova {prova} cai uma vez");
        add_item(&mut items, prova, i32::from(qty)).expect("prova entra na mochila");
        assert!(
            items.iter().any(|linha| linha.item_id == i64::from(prova)),
            "prova {prova} não entrou no inventário"
        );

        let (merchant, etapa) = treinadores[indice];
        let ids: Vec<i64> = items.iter().map(|linha| linha.item_id).collect();
        let passo = quest_engage(&QuestEngage {
            merchant,
            stage: etapa,
            reward_mask: mask,
            level: 1,
            inventory: &ids,
            weapon_req_level: Some(i64::from(req_level)),
            roll: 1,
        })
        .expect("etapa aceita com a prova do combate");
        assert_eq!(passo.consumed, i64::from(prova), "prova consumida");
        if etapa == 1 {
            assert_eq!(
                passo.reward,
                QuestReward::EnchantWeapon {
                    bonus: 50 + i64::from(req_level)
                },
                "bônus usa o ReqLvl do catálogo"
            );
        }
        apply_quest_reward(&mut items, &passo).expect("recompensa aplicada");
        assert!(
            !items.iter().any(|linha| linha.item_id == i64::from(prova)),
            "a prova {prova} foi consumida na entrega"
        );
        mask |= passo.reward_bit;

        // Reentrega da mesma etapa: o estado avançado recusa antes de consumir qualquer coisa.
        let ids_depois: Vec<i64> = items.iter().map(|linha| linha.item_id).collect();
        let recusa = quest_engage(&QuestEngage {
            merchant,
            stage: passo.stage,
            reward_mask: mask,
            level: 1,
            inventory: &ids_depois,
            weapon_req_level: Some(i64::from(req_level)),
            roll: 1,
        })
        .expect_err("a mesma etapa não pode pagar duas vezes");
        assert!(
            matches!(
                recusa,
                wyd_server::gameplay::QuestRefusal::OutOfOrder { .. }
            ),
            "recusa inesperada: {recusa:?}"
        );
        eprintln!(
            "PASS(quest): etapa {} entregue por {template}, prova={prova} consumida",
            passo.stage
        );
        assert_eq!(
            items
                .iter()
                .filter(|linha| linha.item_id == i64::from(prova))
                .count(),
            0,
            "prova {prova} não sobra no inventário"
        );
    }

    assert_eq!(mask, 0b1111, "as quatro recompensas foram pagas");
}

#[test]
fn sephira_mais_nove_esta_no_campo_e_o_refino_do_mais_nove_funciona() {
    let Some(db) = banco("Sephira +9 no Campo") else {
        return;
    };
    let gameplay = Gameplay::new(Arc::new(LiveConfig::new((*db.config).clone())));
    let specs = npc_specs(&db.config, 0);
    let sephira = specs
        .iter()
        .find(|npc| npc.name == "Sephira +9")
        .expect("NPC Sephira +9 ausente do seed 0049");
    assert_eq!(
        (sephira.position.x, sephira.position.y),
        (2133, 2040),
        "Sephira +9 fora da coordenada decidida (ao lado do Unicorn)"
    );

    let mut engine = Engine::new(20, Arc::new(wyd_core::Metrics::new())).expect("motor");
    engine.sync_npcs(vec![sephira.clone()]);
    let jogador = engine
        .enter(&personagem(2134, 2040), ActorSkin::default())
        .expect("entra no mundo");
    let alvo = engine
        .talk_at(jogador.entity, 2133, 2040)
        .expect("Sephira +9 no alcance do clique");
    assert_eq!(alvo.name, "Sephira +9");
    assert_eq!(alvo.id, 6311);

    let passo = gameplay
        .refine(&RefineRequest {
            item_index: 831,
            level: 8,
            powder: Powder::Lac,
            sealed: false,
            pity: 0,
            roll: 0,
            pity_roll: 0,
        })
        .expect("refino +9 com Lac");
    assert!(passo.success, "roll 0 deve passar na taxa do +8 → +9");
    assert_eq!(passo.to_level, 9, "+8 → +9");
    assert_eq!(
        db.config.refine_bps(9),
        Some(5_000),
        "taxa do +9 no snapshot"
    );
}

#[test]
fn equipagem_usa_o_req_level_do_catalogo_real() {
    let Some(db) = banco("ReqLvl na equipagem") else {
        return;
    };
    let snapshot = db
        .runtime
        .block_on(db.backend.catalog_snapshot())
        .expect("catálogo");
    let equipamentos = EquipmentCatalog::from_catalog(&snapshot.items);
    assert_eq!(equipamentos.equip_slot(1006), Some(6), "Arco é arma");
    assert_eq!(equipamentos.req_level(1006), 319, "ReqLvl do catálogo");

    let mut inv = Inventory::default();
    inv.bag[0] = Item {
        id: 1006,
        qty: 1,
        ..Item::default()
    };
    let mut store = ComponentStore::default();
    let baixo = store
        .spawn(
            EntityKind::Player,
            Position::default(),
            Stats {
                level: 1,
                hp: 100,
                max_hp: 100,
                mp: 0,
                max_mp: 0,
                attack: 1,
                defense: 0,
            },
            inv,
        )
        .expect("entidade");
    assert_eq!(
        equip(
            &mut store,
            baixo,
            0,
            &equipamentos,
            EquipmentRules::default()
        ),
        Err(CoreError::LevelTooLow {
            required: 319,
            current: 1
        })
    );

    let mut inv = Inventory::default();
    inv.bag[0] = Item {
        id: 1006,
        qty: 1,
        ..Item::default()
    };
    let mut store = ComponentStore::default();
    let alto = store
        .spawn(
            EntityKind::Player,
            Position::default(),
            Stats {
                level: 319,
                hp: 100,
                max_hp: 100,
                mp: 0,
                max_mp: 0,
                attack: 1,
                defense: 0,
            },
            inv,
        )
        .expect("entidade");
    let equipado = equip(
        &mut store,
        alto,
        0,
        &equipamentos,
        EquipmentRules::default(),
    )
    .expect("nível 319 equipa");
    assert_eq!(equipado.slot, 6);
    assert_eq!(
        store.inventory(alto).expect("vivo").equip[6].id,
        1006,
        "item equipado no slot do catálogo"
    );
}
