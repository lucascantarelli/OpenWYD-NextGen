//! Tick do motor dentro do processo do canal (FASE-03-7), sem banco.
//!
//! Prova o contrato que o `/metrics` publica: a thread `wyd-tick` roda de verdade, conta
//! ticks, mede o passo e morre quando pedida — sem tick órfão depois do `stop_and_join`.
//! O teste não depende de `DATABASE_URL` porque não sobe listeners nem toca persistência.

#[path = "../common/mod.rs"]
mod common;

use std::sync::Arc;
use std::time::Duration;

use wyd_core::{EntityKind, Metrics, Position, SpawnSpec, Stats};
use wyd_server::metrics::ServiceMetrics;
use wyd_server::world::{tick_hz_from_value, Engine, EngineThread};

fn stats() -> Stats {
    Stats {
        level: 10,
        hp: 1_000,
        max_hp: 1_000,
        mp: 100,
        max_mp: 100,
        attack: 10,
        defense: 1,
    }
}

fn spec(x: i32, y: i32) -> SpawnSpec {
    SpawnSpec {
        kind: EntityKind::Monster,
        position: Position { x, y },
        stats: stats(),
    }
}

#[test]
fn thread_do_tick_roda_conta_e_para_sem_tick_orfao() {
    let metrics = Arc::new(Metrics::new());
    let mut engine = Engine::new(20, Arc::clone(&metrics)).expect("motor com 20 Hz");
    engine.spawn(spec(10, 10)).expect("spawn dentro do mapa");

    let (mut thread, _handle) = EngineThread::spawn(engine, 10).expect("thread do tick");
    assert!(
        common::esperar_ate(Duration::from_secs(2), || metrics.ticks() > 0),
        "o processo não executou tick dentro do prazo"
    );
    thread.stop_and_join();

    let ticks = metrics.ticks();
    assert!(ticks > 0, "o processo executou ao menos um tick");

    let snapshot = metrics.tick_snapshot();
    let orcamento_us = 50_000;
    assert!(
        snapshot.p99 <= orcamento_us / 10,
        "p99 de {} µs acima de 10% do orçamento de tick",
        snapshot.p99
    );

    let texto = metrics.render("");
    assert!(texto.contains("wyd_ticks_total"));
    assert!(texto.contains("wyd_entities_active 1"));

    // Parada limpa: durante 150 ms nenhum tick novo pode aparecer.
    assert!(
        common::esperar_ate(Duration::from_millis(150), || metrics.ticks() == ticks),
        "sem tick órfão após a parada"
    );
}

#[test]
fn metricas_do_servico_publicam_o_registro_do_motor() {
    let metrics = Arc::new(Metrics::new());
    let mut engine = Engine::new(20, Arc::clone(&metrics)).expect("motor com 20 Hz");
    engine.spawn(spec(200, 200)).expect("spawn dentro do mapa");

    let servico = ServiceMetrics::new(2);
    servico.attach_engine(Arc::clone(&metrics));

    // 50 ms = exatamente um tick a 20 Hz.
    assert_eq!(engine.advance(50), 1);
    assert_eq!(metrics.ticks(), 1);

    let texto = servico.render();
    assert!(texto.contains("wyd_ticks_total 1"), "tick real no scrape");
    assert!(texto.contains("wyd_entities_active 1"), "entidade real");
    assert!(texto.contains("wyd_server_channels_published 2"));
    assert!(
        texto.contains("wyd_handshakes_total"),
        "fragmento de rede intacto"
    );
}

#[test]
fn spawn_fora_do_mapa_nao_deixa_entidade_no_motor() {
    let metrics = Arc::new(Metrics::new());
    let mut engine = Engine::new(20, Arc::clone(&metrics)).expect("motor com 20 Hz");
    assert!(engine.spawn(spec(-1, 0)).is_err());
    assert!(engine.spawn(spec(0, 100_000)).is_err());
    assert_eq!(engine.entities(), 0, "rollback explícito, sem meia-entrada");
}

#[test]
fn frequencia_invalida_e_erro_de_boot() {
    assert_eq!(tick_hz_from_value(Some("20")), Ok(20));
    assert_eq!(tick_hz_from_value(Some("50")), Ok(50));
    for invalido in ["19", "51", "0", "vinte"] {
        assert!(
            tick_hz_from_value(Some(invalido)).is_err(),
            "{invalido} deveria falhar no boot"
        );
    }
}
