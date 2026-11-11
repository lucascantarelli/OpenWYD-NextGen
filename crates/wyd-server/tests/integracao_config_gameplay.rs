//! Configuração viva no resolvedor de gameplay (refino e loot).
//!
//! O teste não reinicia processo: publica uma revisão nova no `ConfigSource`, chama
//! `LiveConfig::refresh` (o mesmo caminho da thread `wyd-config`) e repete a **mesma**
//! chamada de `Gameplay`. O resultado muda; reverter a revisão restaura o resultado.
//!
//! Não há banco externo: a fonte é um duplo com as mesmas linhas que `config_rates` /
//! `config_drops` devolveriam. Um carregamento inválido é `Err` na fonte, e o resolvedor
//! precisa continuar com o último snapshot bom — sem `panic` e sem taxa zero.

use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::{Arc, Mutex};

use wyd_core::{EntityKind, Metrics, Position, SpawnSpec, Stats};
use wyd_db::{ConfigRateRow, DropConfigRow, ItemRow};
use wyd_protocol::codec::checksum::{self, ChecksumPolicy};
use wyd_protocol::codec::keyword::{decode_in_place, encode_in_place};
use wyd_protocol::payload::combat::{AttackOne, TargetDamage};
use wyd_protocol::payload::WirePayload;
use wyd_protocol::wire::opcode::Opcode;
use wyd_protocol::{Header, UseItem};
use wyd_server::config::{ConfigSource, GameConfig, LiveConfig, ReloadOutcome};
use wyd_server::gameplay::{
    Gameplay, LootError, RefineError, RefineFailure, RefineRequest, UseItemRolls,
};
use wyd_server::personagens::{update_etc_body, Mob};
use wyd_server::world::{Engine, NpcSpec};
use wyd_sim::formula::drop::DROP_ROLL_MAX;
use wyd_sim::formula::refine::{powder_success_bps, Powder, RefinePolicy, REFINE_RATE_BPS};
use wyd_sim::roll::XorShift64;

struct Source {
    config: Mutex<GameConfig>,
    fail: AtomicBool,
}

impl Source {
    fn new(config: GameConfig) -> Self {
        Self {
            config: Mutex::new(config),
            fail: AtomicBool::new(false),
        }
    }

    fn publish(&self, config: GameConfig) {
        *self.config.lock().expect("config") = config;
    }

    fn snapshot(&self) -> GameConfig {
        self.config.lock().expect("config").clone()
    }

    fn fail_reads(&self, fail: bool) {
        self.fail.store(fail, Ordering::Relaxed);
    }
}

impl ConfigSource for Source {
    fn revision(&self) -> Result<i64, String> {
        if self.fail.load(Ordering::Relaxed) {
            return Err("pool fechado (simulado)".to_string());
        }
        Ok(self.snapshot().revision())
    }

    fn load(&self, _revision: i64) -> Result<GameConfig, String> {
        if self.fail.load(Ordering::Relaxed) {
            return Err("timeout de leitura (simulado)".to_string());
        }
        Ok(self.snapshot())
    }
}

fn rate(domain: &str, key: &str, value: i64) -> ConfigRateRow {
    ConfigRateRow {
        domain: domain.to_string(),
        key: key.to_string(),
        value,
        unit: if domain == "sanc" { "%" } else { "bps" }.to_string(),
        source: "seed".to_string(),
        overridden: false,
    }
}

fn drop_row(weight: i64) -> DropConfigRow {
    DropConfigRow {
        template_name: "wolf".to_string(),
        item_index: 413,
        item_name: "Poeira de Lactolerium".to_string(),
        weight,
        min_qty: 1,
        max_qty: 2,
        source: "portal".to_string(),
    }
}

fn rows(sanc_po5: i64, drop_multiplier: i64) -> Vec<ConfigRateRow> {
    let mut rates: Vec<ConfigRateRow> = (1..=15)
        .map(|step| {
            rate(
                "refine",
                &step.to_string(),
                i64::from(REFINE_RATE_BPS[step - 1]),
            )
        })
        .collect();
    // A tabela efetiva tem as células que `powder_rate_bps` lê em `nível + 1`; o teste
    // sobrescreve PO:5 para provar que a edição do portal muda a taxa.
    for index in 0..=15 {
        rates.push(rate("sanc", &format!("PO:{index}"), 10));
        rates.push(rate("sanc", &format!("PL:{index}"), 5));
        rates.push(rate("sanc", &format!("AMAGO:{index}"), 100));
    }
    rates.push(rate("sanc", "PO:5", sanc_po5));
    rates.push(rate("combine", "Odin.PLUS12_BASE", 100));
    rates.push(rate("multiplier", "drop", drop_multiplier));
    rates.push(rate("setting", "event.exp_bonus", 0));
    rates
}

fn config(revision: i64, sanc_po5: i64, drop_weight: i64, drop_multiplier: i64) -> GameConfig {
    let drops = if drop_weight == 0 {
        Vec::new()
    } else {
        vec![drop_row(drop_weight)]
    };
    GameConfig::from_rows(
        revision,
        &rows(sanc_po5, drop_multiplier),
        Vec::new(),
        drops,
        Vec::new(),
        Vec::new(),
    )
    .expect("configuração de teste válida")
}

/// Mesma base de `config`, com o multiplicador `exp` explícito.
fn config_exp(revision: i64, exp_bps: i64) -> GameConfig {
    let mut rates = rows(10, 10_000);
    rates.retain(|row| !(row.domain == "multiplier" && row.key == "exp"));
    rates.push(rate("multiplier", "exp", exp_bps));
    GameConfig::from_rows(
        revision,
        &rates,
        Vec::new(),
        Vec::new(),
        Vec::new(),
        Vec::new(),
    )
    .expect("configuração de teste válida")
}

fn boot(config: GameConfig) -> (Arc<LiveConfig>, Arc<Gameplay>, Source) {
    let source = Source::new(config);
    let live = Arc::new(LiveConfig::new(source.snapshot()));
    let gameplay = Arc::new(Gameplay::new(Arc::clone(&live)));
    (live, gameplay, source)
}

fn item(slot: i64, item_id: i64, qty: i64, refine: i64) -> ItemRow {
    ItemRow {
        slot,
        item_id,
        qty,
        refine,
        ..ItemRow::default()
    }
}

fn use_packet() -> UseItem {
    UseItem {
        sour_type: 1,
        sour_pos: 0,
        dest_type: 0,
        dest_pos: 1,
        grid_x: 0,
        grid_y: 0,
        warp_id: 0,
    }
}

fn refine_items() -> Vec<ItemRow> {
    // Pó de Ori no primeiro slot da mochila (`Carry[0]` = slot 16) e arma +4 equipada.
    vec![item(16, 412, 2, 0), item(1, 1_900, 1, 4)]
}

fn request(level: u8, roll: u8) -> RefineRequest {
    RefineRequest {
        item_index: 1_900,
        level,
        powder: Powder::Ori,
        sealed: false,
        pity: 0,
        roll,
        pity_roll: 3,
    }
}

#[test]
fn refino_le_a_sancrate_do_snapshot_em_cada_chamada() {
    let (live, gameplay, source) = boot(config(1, 10, 500_000, 10_000));

    let primeira = gameplay.refine(&request(4, 0)).expect("refino válido");
    assert!(primeira.success);
    assert_eq!(primeira.rate_bps, 1_000, "10% da célula PO:5");
    assert_eq!((primeira.from_level, primeira.to_level), (4, 5));

    // Mesma chamada, revisão nova: a célula vira 0 e o mesmo roll 0 falha.
    source.publish(config(2, 0, 500_000, 10_000));
    assert_eq!(
        live.refresh(&source),
        ReloadOutcome::Applied { from: 1, to: 2 }
    );
    let segunda = gameplay.refine(&request(4, 0)).expect("refino válido");
    assert!(!segunda.success, "taxa 0 nunca passa, nem com roll 0");
    assert_eq!(segunda.rate_bps, 0);
    assert_eq!((segunda.from_level, segunda.to_level), (4, 4));
    assert_eq!(segunda.failure, Some(RefineFailure::Kept));

    // Reverter a edição restaura o comportamento na mesma chamada.
    source.publish(config(3, 10, 500_000, 10_000));
    assert_eq!(
        live.refresh(&source),
        ReloadOutcome::Applied { from: 2, to: 3 }
    );
    let terceira = gameplay.refine(&request(4, 0)).expect("refino válido");
    assert!(terceira.success);
    assert_eq!(terceira.rate_bps, 1_000);
    assert_eq!(terceira.to_level, 5);
    assert_eq!(live.revision(), 3);
}

#[test]
fn refino_respeita_tetos_do_pano_e_passo_especial() {
    let (_live, gameplay, _source) = boot(config(1, 10, 500_000, 10_000));

    assert_eq!(
        gameplay.refine(&request(6, 0)),
        Err(RefineError::NotAllowed {
            powder: Powder::Ori,
            level: 6,
            item_index: 1_900,
        })
    );
    assert!(matches!(
        gameplay.refine(&RefineRequest {
            powder: Powder::Lac,
            level: 9,
            ..request(9, 0)
        }),
        Err(RefineError::NotAllowed { .. })
    ));
    assert!(matches!(
        gameplay.refine(&RefineRequest {
            item_index: 769,
            powder: Powder::Lac,
            level: 9,
            ..request(9, 0)
        }),
        Err(RefineError::NotAllowed { .. })
    ));
    assert_eq!(
        gameplay.refine(&request(16, 0)),
        Err(RefineError::LevelOutOfRange { level: 16 })
    );
    assert_eq!(
        gameplay.refine(&request(4, 100)),
        Err(RefineError::RollOutOfRange { roll: 100 })
    );

    // Âmago é 100% e o passo +10→+11 é fixo em 15%, fora da célula.
    let amago = gameplay
        .refine(&RefineRequest {
            powder: Powder::Amago,
            level: 4,
            roll: 99,
            ..request(4, 99)
        })
        .expect("âmago válido");
    assert!(amago.success);
    assert_eq!(amago.rate_bps, 10_000);

    let especial = gameplay
        .refine(&RefineRequest {
            powder: Powder::Lac,
            level: 10,
            roll: 15,
            ..request(10, 15)
        })
        .expect("passo especial válido");
    assert!(especial.success);
    assert_eq!(especial.rate_bps, 1_500);
    assert_eq!(especial.to_level, 11);
}

#[test]
fn use_item_e_o_caminho_do_pacote_leem_a_sancrate_viva() {
    let (live, gameplay, source) = boot(config(1, 10, 500_000, 10_000));
    let packet = use_packet();

    let mut items = refine_items();
    let primeira = gameplay
        .use_item(
            &mut items,
            &packet,
            UseItemRolls {
                roll: 0,
                pity_roll: 3,
            },
            0,
        )
        .expect("pó e alvo válidos");
    assert!(primeira.resolution.success);
    assert_eq!(primeira.resolution.rate_bps, 1_000);
    assert_eq!(
        (primeira.resolution.from_level, primeira.resolution.to_level),
        (4, 5)
    );
    assert_eq!(primeira.powder_left, 1);
    assert_eq!(
        items
            .iter()
            .find(|item| item.slot == 1)
            .map(|item| item.refine),
        Some(5),
        "o alvo da sessão subiu para +5"
    );

    // Mesma chamada de `Gameplay::use_item`, revisão nova com SancRate PO:5 = 0: falha.
    source.publish(config(2, 0, 500_000, 10_000));
    assert_eq!(
        live.refresh(&source),
        ReloadOutcome::Applied { from: 1, to: 2 }
    );
    let mut items = refine_items();
    let segunda = gameplay
        .use_item(
            &mut items,
            &packet,
            UseItemRolls {
                roll: 0,
                pity_roll: 3,
            },
            0,
        )
        .expect("pó e alvo válidos");
    assert!(!segunda.resolution.success);
    assert_eq!(segunda.resolution.rate_bps, 0);
    assert_eq!(segunda.resolution.failure, Some(RefineFailure::Kept));
    assert_eq!(segunda.powder_left, 1, "falha também consome uma unidade");
    assert_eq!(
        items
            .iter()
            .find(|item| item.slot == 1)
            .map(|item| item.refine),
        Some(4),
        "falha comum mantém o item no nível anterior"
    );

    // Reverter restaura o sucesso na mesma chamada, sem reiniciar.
    source.publish(config(3, 10, 500_000, 10_000));
    assert_eq!(
        live.refresh(&source),
        ReloadOutcome::Applied { from: 2, to: 3 }
    );
    let mut items = refine_items();
    let terceira = gameplay
        .use_item(
            &mut items,
            &packet,
            UseItemRolls {
                roll: 0,
                pity_roll: 3,
            },
            0,
        )
        .expect("pó e alvo válidos");
    assert!(terceira.resolution.success);
    assert_eq!(terceira.resolution.rate_bps, 1_000);
    assert_eq!(terceira.resolution.to_level, 5);

    // Valor fora da faixa (SancRate 101) é recusado na construção do snapshot: não há
    // `GameConfig` inválido para publicar, então a revisão 3 continua valendo.
    assert!(
        GameConfig::from_rows(
            4,
            &rows(101, 10_000),
            Vec::new(),
            vec![drop_row(500_000)],
            Vec::new(),
            Vec::new(),
        )
        .is_err(),
        "SancRate acima de 100 não pode virar snapshot"
    );
    // Falha de leitura mantém a revisão 3 e o mesmo caminho continua servindo o snapshot bom.
    source.fail_reads(true);
    assert!(matches!(
        live.refresh(&source),
        ReloadOutcome::Failed { .. }
    ));
    assert_eq!(live.revision(), 3);
    let mut items = refine_items();
    let ultima = gameplay
        .use_item(
            &mut items,
            &packet,
            UseItemRolls {
                roll: 0,
                pity_roll: 3,
            },
            0,
        )
        .expect("snapshot bom continua valendo");
    assert!(ultima.resolution.success);
    assert_eq!(ultima.resolution.rate_bps, 1_000);
}

#[test]
fn loot_le_a_tabela_do_snapshot_e_cada_item_tem_seu_roll() {
    let (live, gameplay, source) = boot(config(1, 10, 500_000, 10_000));

    let caiu = gameplay
        .loot("wolf", &[(413, 1)], false)
        .expect("tabela existe");
    assert_eq!(caiu.len(), 1);
    assert_eq!((caiu[0].item_index, caiu[0].qty), (413, 1));

    // Peso 500.000: roll 500.001 não cai; roll 500.000 cai (comparação inclusiva).
    assert!(gameplay
        .loot("wolf", &[(413, 500_001)], false)
        .expect("tabela existe")
        .is_empty());
    assert_eq!(
        gameplay
            .loot("wolf", &[(413, 500_000)], false)
            .expect("tabela")
            .len(),
        1
    );

    // Edição no portal: peso 1 muda o resultado da mesma chamada.
    source.publish(config(2, 10, 1, 10_000));
    assert_eq!(
        live.refresh(&source),
        ReloadOutcome::Applied { from: 1, to: 2 }
    );
    assert_eq!(
        gameplay
            .loot("wolf", &[(413, 1)], false)
            .expect("tabela")
            .len(),
        1
    );
    assert!(gameplay
        .loot("wolf", &[(413, 2)], false)
        .expect("tabela")
        .is_empty());

    // Reverter o peso volta a dropar no roll que antes falhava.
    source.publish(config(3, 10, 500_000, 10_000));
    assert_eq!(
        live.refresh(&source),
        ReloadOutcome::Applied { from: 2, to: 3 }
    );
    assert_eq!(
        gameplay
            .loot("wolf", &[(413, 500_000)], false)
            .expect("tabela")
            .len(),
        1
    );

    // Cada linha exige seu próprio sorteio; faltar é erro, não "não dropou".
    assert_eq!(
        gameplay.loot("wolf", &[], false),
        Err(LootError::MissingRoll { item_index: 413 })
    );
    assert_eq!(
        gameplay.loot("wolf", &[(413, 0)], false),
        Err(LootError::InvalidRoll {
            item_index: 413,
            roll: 0
        })
    );
    assert_eq!(
        gameplay.loot("troll", &[(413, 1)], false),
        Err(LootError::NoTable {
            template: "troll".to_string()
        })
    );
}

#[test]
fn multiplicador_de_drop_entra_no_peso_efetivo() {
    let (_live, gameplay, source) = boot(config(1, 10, 500_000, 20_000));

    // 500.000 × 200% = 1.000.000: o topo da escala passa a cair.
    assert_eq!(
        gameplay
            .loot("wolf", &[(413, 1_000_000)], false)
            .expect("tabela")
            .len(),
        1
    );

    source.publish(config(2, 10, 500_000, 0));
    assert_eq!(
        _live.refresh(&source),
        ReloadOutcome::Applied { from: 1, to: 2 }
    );
    assert!(gameplay
        .loot("wolf", &[(413, 1)], false)
        .expect("tabela")
        .is_empty());
}

#[test]
fn refino_segue_os_goldens_do_wyd_sim_em_cada_chamada() {
    // A taxa efetiva do handler é a célula de SancRate do snapshot aplicada pela fórmula
    // portada (`powder_success_bps`), sem cópia local da tabela.
    let (live, gameplay, source) = boot(config(1, 10, 500_000, 10_000));
    for level in 0..=5u8 {
        let esperado = powder_success_bps(RefinePolicy::LEGACY, Powder::Ori, level, 0, 10)
            .expect("fórmula de refino do wyd-sim");
        let resolucao = gameplay.refine(&request(level, 0)).expect("refino válido");
        assert_eq!(resolucao.rate_bps, esperado, "golden do passo +{level}");
        assert_eq!(resolucao.success, esperado > 0, "roll 0 vs taxa +{level}");
    }

    // Editar a célula no portal muda a taxa do mesmo passo na chamada seguinte.
    source.publish(config(2, 0, 500_000, 10_000));
    assert_eq!(
        live.refresh(&source),
        ReloadOutcome::Applied { from: 1, to: 2 }
    );
    // A tabela é lida em `nível + 1`: o passo +4→+5 usa a célula PO:5, zerada no portal.
    assert_eq!(
        gameplay
            .refine(&request(4, 0))
            .expect("refino válido")
            .rate_bps,
        0
    );
    // Piedade entra na fórmula portada, não em constante local: com a célula de volta a 10%
    // o desconto do `powder_success_bps` é o mesmo que o golden calcula.
    source.publish(config(3, 10, 500_000, 10_000));
    assert_eq!(
        live.refresh(&source),
        ReloadOutcome::Applied { from: 2, to: 3 }
    );
    let com_piedade = gameplay
        .refine(&RefineRequest {
            pity: 2,
            ..request(5, 0)
        })
        .expect("refino válido");
    assert_eq!(
        com_piedade.rate_bps,
        powder_success_bps(RefinePolicy::LEGACY, Powder::Ori, 5, 2, 10).expect("fórmula")
    );
}

#[test]
fn exp_do_abate_no_protocolo_aplica_o_multiplicador_do_snapshot() {
    // 1. Pacote real do cliente (`_MSG_AttackOne`, 68 B) montado, ofuscado e lido de volta
    //    pelo mesmo caminho do servidor: ofuscação + checksum estrito + layout.
    let ataque = AttackOne {
        unk_1: [0; 4],
        current_hp: 500,
        unk_2: [0; 4],
        // Valor do cliente: o servidor ignora e credita o do template.
        current_exp: 0,
        unk_0: 0,
        pos_x: 1_689,
        pos_y: 1_840,
        target_x: 1_689,
        target_y: 1_841,
        attacker_id: 1,
        progress: 0,
        motion: 1,
        skill_parm: 0,
        double_critical: 0,
        flag_local: 0,
        rsv: 0,
        current_mp: 10,
        skill_index: 0,
        req_mp: 0,
        dam: [TargetDamage {
            target_id: 1_001,
            damage: 9_999,
        }],
    };
    let mut frame = vec![0u8; AttackOne::WIRE_LEN];
    Header {
        size: AttackOne::WIRE_LEN as u16,
        key_word: 7,
        checksum: 0,
        message_type: Opcode::AttackOne.wire(),
        id: 1,
        client_tick: 1,
    }
    .encode(&mut frame)
    .expect("header");
    ataque.encode_into(&mut frame).expect("corpo");
    encode_in_place(&mut frame, 7).expect("ofuscação");
    let computado = decode_in_place(&mut frame).expect("desofuscação");
    checksum::verify(computado, frame[3], ChecksumPolicy::Strict).expect("checksum");
    let lido = AttackOne::decode(&frame).expect("layout do cliente");
    assert_eq!(lido.dam[0].target_id, 1_001);
    assert_eq!((lido.target_x, lido.target_y), (1_689, 1_841));

    // 2. Servidor: o alvo é o NPC do template (EXP 5.000), o dano vem do `Stats` dele.
    let mut engine = Engine::new(20, Arc::new(Metrics::new())).expect("motor 20 Hz");
    let spawn = engine.sync_npcs(vec![NpcSpec {
        id: 7,
        name: "Orc Cavaleiro".to_string(),
        template: "orc_cavaleiro".to_string(),
        position: Position { x: 1_689, y: 1_841 },
        stats: Stats {
            level: 50,
            hp: 40,
            max_hp: 40,
            mp: 0,
            max_mp: 0,
            attack: 10,
            defense: 0,
        },
        kind: EntityKind::Monster,
        exp: 5_000,
        merchant: 0,
        equip: [0; 16],
        attack_run: 82,
        direction: 3,
    }]);
    assert_eq!((spawn.spawned, spawn.rejected), (1, 0));
    let jogador = engine
        .spawn(SpawnSpec {
            kind: EntityKind::Player,
            position: Position {
                x: i32::from(lido.pos_x),
                y: i32::from(lido.pos_y),
            },
            stats: Stats {
                level: 30,
                hp: 500,
                max_hp: 500,
                mp: 10,
                max_mp: 10,
                attack: 100,
                defense: 10,
            },
        })
        .expect("jogador");
    let golpe = engine
        .attack_at(
            jogador,
            i32::from(lido.target_x),
            i32::from(lido.target_y),
            1_000,
        )
        .expect("sem erro de motor")
        .expect("alvo no alcance");
    assert!(golpe.killed, "100 de ataque contra 40 de HP");
    assert_eq!(golpe.exp, 5_000, "EXP do template, não do cliente");

    // 3. A EXP entra com o multiplicador do snapshot vigente e aparece no `UpdateEtc`.
    let (live, gameplay, source) = boot(config_exp(1, 10_000));
    assert_eq!(gameplay.exp_reward(golpe.exp), 5_000);
    source.publish(config_exp(2, 25_000));
    assert_eq!(
        live.refresh(&source),
        ReloadOutcome::Applied { from: 1, to: 2 }
    );
    let creditado = gameplay.exp_reward(golpe.exp);
    assert_eq!(creditado, 12_500, "250% da EXP do template");
    let mob = Mob {
        exp: creditado,
        ..Mob::default()
    };
    let etc = update_etc_body(&mob);
    assert_eq!(
        i64::from_le_bytes(etc[4..12].try_into().expect("exp no UpdateEtc")),
        12_500,
        "o pacote que o cliente recebe carrega a EXP multiplicada"
    );

    // 4. Reverter no portal restaura a recompensa na mesma chamada, sem reiniciar.
    source.publish(config_exp(3, 10_000));
    assert_eq!(
        live.refresh(&source),
        ReloadOutcome::Applied { from: 2, to: 3 }
    );
    assert_eq!(gameplay.exp_reward(golpe.exp), 5_000);
}

#[test]
fn valor_invalido_nao_publica_e_o_resolvedor_continua_no_ultimo_bom() {
    // A validação recusa antes de publicar: SancRate > 100 e peso <= 0.
    assert!(GameConfig::from_rows(
        1,
        &rows(101, 10_000),
        Vec::new(),
        vec![drop_row(500_000)],
        Vec::new(),
        Vec::new(),
    )
    .is_err());
    assert!(GameConfig::from_rows(
        1,
        &rows(10, 10_000),
        Vec::new(),
        vec![drop_row(0)],
        Vec::new(),
        Vec::new(),
    )
    .is_err());

    let (live, gameplay, source) = boot(config(1, 10, 500_000, 10_000));
    source.fail_reads(true);
    assert!(matches!(
        live.refresh(&source),
        ReloadOutcome::Failed { .. }
    ));
    assert_eq!(live.revision(), 1);
    assert!(live.last_error().is_some());

    // O caminho de gameplay continua usando o snapshot bom — sem panic, sem zero.
    let refino = gameplay.refine(&request(4, 0)).expect("snapshot bom");
    assert!(refino.success);
    assert_eq!(refino.rate_bps, 1_000);
    assert_eq!(
        gameplay
            .loot("wolf", &[(413, 1)], false)
            .expect("snapshot bom")
            .len(),
        1
    );

    // Publicar de novo e recuperar a leitura volta a aplicar revisões.
    source.fail_reads(false);
    source.publish(config(2, 10, 1, 10_000));
    assert_eq!(
        live.refresh(&source),
        ReloadOutcome::Applied { from: 1, to: 2 }
    );
    assert!(gameplay
        .loot("wolf", &[(413, 2)], false)
        .expect("tabela")
        .is_empty());
}

/// Linha de drop de qualquer item no template `wolf` (o resolvedor é indexado pelo template).
fn drop_de(item_index: i32, weight: i64) -> DropConfigRow {
    DropConfigRow {
        template_name: "wolf".to_string(),
        item_index,
        item_name: format!("item-{item_index}"),
        weight,
        min_qty: 1,
        max_qty: 1,
        source: "portal".to_string(),
    }
}

/// Configuração com a tabela de drop dada e multiplicador de drop neutro (10.000 bps = 1×).
fn config_loot(revision: i64, drops: Vec<DropConfigRow>) -> GameConfig {
    GameConfig::from_rows(
        revision,
        &rows(10, 10_000),
        Vec::new(),
        drops,
        Vec::new(),
        Vec::new(),
    )
    .expect("configuração de teste válida")
}

#[test]
fn morte_sorteia_um_roll_por_item_da_tabela_do_snapshot() {
    let (live, gameplay, source) = boot(config_loot(
        1,
        vec![drop_de(413, 500_000), drop_de(547, 1_000)],
    ));

    let mut die = XorShift64::from_seed(2_026);
    let resolucao = gameplay
        .loot_at_death("wolf", &mut die, false)
        .expect("tabela do snapshot");
    assert_eq!(resolucao.template, "wolf");
    assert_eq!(resolucao.rolls.len(), 2, "um roll por linha da tabela");
    for (item, roll) in &resolucao.rolls {
        assert!(
            (1..=DROP_ROLL_MAX).contains(roll),
            "roll na escala do original: {roll}"
        );
        let peso = if *item == 413 { 500_000 } else { 1_000 };
        let caiu = resolucao.drops.iter().any(|drop| drop.item_index == *item);
        assert_eq!(
            caiu,
            *roll <= peso,
            "roll {roll} × peso {peso} do item {item}"
        );
    }

    // Mesma seed, mesma decisão: o `seed=` do log do abate reproduz o kill.
    let mut outra = XorShift64::from_seed(2_026);
    assert_eq!(
        gameplay
            .loot_at_death("wolf", &mut outra, false)
            .expect("tabela"),
        resolucao
    );

    // Recarga do portal muda o peso: a mesma morte passa a decidir diferente.
    source.publish(config_loot(2, vec![drop_de(413, 1), drop_de(547, 1_000)]));
    assert_eq!(
        live.refresh(&source),
        ReloadOutcome::Applied { from: 1, to: 2 }
    );
    let mut depois = XorShift64::from_seed(2_026);
    let segunda = gameplay
        .loot_at_death("wolf", &mut depois, false)
        .expect("tabela");
    let caiu_413 = segunda.drops.iter().any(|drop| drop.item_index == 413);
    assert_eq!(
        caiu_413,
        segunda
            .rolls
            .iter()
            .any(|(item, roll)| *item == 413 && *roll <= 1),
        "peso 1 só cai no roll 1"
    );

    // Template sem tabela é erro explícito, nunca \"drop vazio\".
    assert!(matches!(
        gameplay.loot_at_death("troll", &mut XorShift64::from_seed(1), false),
        Err(LootError::NoTable { .. })
    ));
}
