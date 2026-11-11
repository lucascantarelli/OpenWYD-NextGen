//! Entrada no mundo e save de saída (FASE-03-8), contra banco real.
//!
//! Prova o caminho que o `_MSG_CharacterLogin` usa: personagem persistido → entidade no
//! motor → saída → snapshot na fila de save → `UPDATE` no banco. Sem `DATABASE_URL` o teste
//! **pula com aviso** (mesmo contrato dos demais testes de banco).

use std::sync::Arc;

use wyd_core::Metrics;
use wyd_db::{Accounts, Character, Characters, PgBackend, PoolConfig, Role, SavePolicy, SaveQueue};
use wyd_server::world::{ActorSkin, Engine, EngineThread, SaveQueueSink, SaveSink};

fn dsn() -> Option<String> {
    let valor = ["WYD_TEST_DATABASE_URL", "DATABASE_URL"]
        .iter()
        .filter_map(|var| std::env::var(var).ok())
        .map(|valor| valor.trim().to_string())
        .find(|valor| !valor.is_empty());
    if valor.is_none() {
        eprintln!(
            "SKIP: WYD_TEST_DATABASE_URL/DATABASE_URL ausente — teste de entrada no mundo não executado"
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

fn conta_unica() -> String {
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_nanos())
        .unwrap_or(0);
    format!("em{:013}", nanos % 10_000_000_000_000)
}

#[test]
fn entrar_e_sair_salva_posicao_e_vitais_no_banco() {
    let Some(dsn) = dsn() else { return };
    let rt = runtime();
    let backend = Arc::new(
        rt.block_on(PgBackend::connect(&dsn, PoolConfig::local()))
            .expect("banco"),
    );

    let login = conta_unica();
    rt.block_on(backend.create_account(&login, Role::Player))
        .expect("conta");
    let conta = rt
        .block_on(backend.find_by_login(&login))
        .expect("conta criada");

    let personagem = Character {
        id: 0,
        account_id: conta.id,
        slot: 0,
        name: format!("E{}", &login[2..8]),
        class: 0,
        level: 10,
        exp: 100,
        hp: 90,
        mp: 40,
        pos_x: 640,
        pos_y: 480,
        hp_max: 100,
        mp_max: 50,
        ..Character::default()
    };
    let personagem = rt
        .block_on(backend.create_character(&personagem))
        .expect("personagem");

    let (queue, mut drain) = SaveQueue::bounded(SavePolicy::local()).expect("fila de save");
    let saves = Arc::new(SaveQueueSink::new(Arc::new(queue)));
    let metrics = Arc::new(Metrics::new());
    let engine = Engine::new(20, Arc::clone(&metrics)).expect("motor");
    let (mut tick, handle) = EngineThread::spawn(engine, 10).expect("thread do tick");

    let skin = ActorSkin {
        name: {
            let mut nome = [0u8; 16];
            nome[..7].copy_from_slice(b"Teste01");
            nome
        },
        equip: [0; 16],
        guild: 0,
        attack_run: 82,
        direction: 0,
    };
    let entrada = handle.enter(&personagem, &skin).expect("entrada no mundo");
    assert!(
        entrada.scene_id < 1_000,
        "ClientID do jogador na faixa `< MAX_USER`"
    );
    let vivos = handle.snapshots();
    assert_eq!(vivos.len(), 1, "uma entidade viva");
    assert_eq!(vivos[0].character_id, personagem.id);
    assert_eq!((vivos[0].x, vivos[0].y), (640, 480), "posição do banco");

    // Duplicado no mesmo processo é recusa tipada, sem segunda entidade.
    assert!(handle.enter(&personagem, &skin).is_err());
    assert_eq!(handle.snapshots().len(), 1);

    let estado = handle.leave(entrada.entity).expect("estado final");
    assert_eq!((estado.hp, estado.mp), (90, 40));
    saves.save(estado);

    let report = rt.block_on(drain.flush(&backend)).expect("flush do save");
    assert_eq!(report.written, 1, "um personagem gravado");
    assert_eq!(report.missing, 0);

    let lidos = rt
        .block_on(backend.list_by_account(conta.id))
        .expect("leitura");
    let relido = lidos
        .iter()
        .find(|candidate| candidate.id == personagem.id)
        .expect("personagem recarregado");
    assert_eq!(
        (relido.pos_x, relido.pos_y, relido.hp, relido.mp),
        (640, 480, 90, 40)
    );

    tick.stop_and_join();
}
