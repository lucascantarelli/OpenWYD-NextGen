//! Custo do tick **do serviço** (FASE-03-7), não só da lib.
//!
//! Mede `Engine::advance(50)` — o mesmo caminho que a thread `wyd-tick` executa a 20 Hz —
//! com mundo vazio e com 1 000 entidades. Sem dependência nova: `std::time::Instant`, igual
//! ao bench de IA do core (`crates/wyd-core/benches/tick.rs`).
//!
//! Roda com `cargo bench -p wyd-server --bench tick`; em debug sai verde sem medir.

use std::hint::black_box;
use std::sync::Arc;
use std::time::{Duration, Instant};

use wyd_core::{EntityKind, Metrics, Position, SpawnSpec, Stats};
use wyd_server::world::Engine;

/// Entidades do cenário cheio.
const ENTIDADES: u32 = 1_000;
/// Ticks descartados (cache/ramp-up).
const AQUECIMENTO: u32 = 100;
/// Ticks medidos.
const MEDIDOS: u32 = 1_000;
/// Orçamento do tick a 20 Hz, em µs.
const ORCAMENTO_US: f64 = 50_000.0;

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

fn micros(d: Duration) -> f64 {
    d.as_nanos() as f64 / 1_000.0
}

fn medir(id: &str, rotulo: &str, engine: &mut Engine) {
    for _ in 0..AQUECIMENTO {
        engine.advance(50);
    }
    let inicio = Instant::now();
    for _ in 0..MEDIDOS {
        black_box(engine.advance(50));
    }
    let media = micros(inicio.elapsed()) / f64::from(MEDIDOS);
    let share = media / ORCAMENTO_US * 100.0;
    println!(
        "--- tick do serviço: {rotulo} ---\nticks medidos: {MEDIDOS} | média/tick: {media:.2} µs | % do orçamento: {share:.3}%"
    );
    println!(
        "BENCH_PONTO {{\"cenario\":\"{id}\",\"media_us\":{media:.2},\"share_pct\":{share:.3}}}"
    );
}

fn main() {
    if cfg!(debug_assertions) {
        println!("bench em debug: sem medição — rode `cargo bench -p wyd-server --bench tick`");
        return;
    }

    let vazio_metrics = Arc::new(Metrics::new());
    let mut vazio = Engine::new(20, Arc::clone(&vazio_metrics)).expect("motor com 20 Hz");
    medir("servico_vazio", "mundo vazio", &mut vazio);

    let cheio_metrics = Arc::new(Metrics::new());
    let mut cheio = Engine::new(20, Arc::clone(&cheio_metrics)).expect("motor com 20 Hz");
    for i in 0..ENTIDADES {
        let posicao = Position {
            x: i32::try_from((i % 100) * 4).expect("coordenada"),
            y: i32::try_from((i / 100) * 4 + 1).expect("coordenada"),
        };
        cheio
            .spawn(SpawnSpec {
                kind: EntityKind::Monster,
                position: posicao,
                stats: stats(),
            })
            .expect("spawn dentro do mapa");
    }
    medir(
        "servico_1000_entidades",
        &format!("{ENTIDADES} entidades"),
        &mut cheio,
    );
}
