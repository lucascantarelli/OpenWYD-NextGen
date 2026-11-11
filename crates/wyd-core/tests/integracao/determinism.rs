//! Determinismo ponta a ponta: mesmo seed + mesmos inputs = mesmo estado.
//!
//! O RNG aqui é um LCG local de teste (ordem de chamadas fixa); o contrato de
//! RNG do gameplay vive na FASE-10. O que se prova: acumulador de tick,
//! armazém SoA e grade convergem bit a bit entre execuções independentes.

use wyd_core::{
    elapsed_ms, Clock, ComponentStore, Entity, EntityKind, Grid, GridConfig, Inventory,
    ManualClock, Position, Scheduler, SchedulerConfig, Stats,
};

/// LCG de teste (constantes de Knuth/MMIX): só para gerar o mesmo script.
fn lcg(state: &mut u64) -> u64 {
    *state = state
        .wrapping_mul(6364136223846793005)
        .wrapping_add(1442695040888963407);
    *state >> 33
}

/// FNV-1a 64 bits sobre o estado observável (ordem de iteração fixa).
fn hash_store(store: &ComponentStore, tick: u64) -> u64 {
    const OFFSET: u64 = 0xcbf29ce484222325;
    const PRIME: u64 = 0x100000001b3;
    let mut h = OFFSET;
    let mut mix = |b: u8| {
        h ^= u64::from(b);
        h = h.wrapping_mul(PRIME);
    };
    for b in tick.to_le_bytes() {
        mix(b);
    }
    for p in store.positions() {
        for b in p.x.to_le_bytes().into_iter().chain(p.y.to_le_bytes()) {
            mix(b);
        }
    }
    for s in store.stats() {
        for w in [
            s.level as u32,
            s.hp,
            s.max_hp,
            s.mp,
            s.max_mp,
            s.attack,
            s.defense,
        ] {
            for b in w.to_le_bytes() {
                mix(b);
            }
        }
    }
    h
}

fn stats_for(seed: u64) -> Stats {
    Stats {
        level: 1 + (seed % 200) as u16,
        hp: 100,
        max_hp: 100,
        mp: 10,
        max_mp: 10,
        attack: 5,
        defense: 3,
    }
}

/// Roteiro fixo: N spawns com posições do LCG + despawn dos pares.
fn script(seed: u64, n: usize) -> (ComponentStore, Vec<Entity>) {
    let mut store = ComponentStore::default();
    store.reserve(n);
    let mut state = seed;
    let mut ids = Vec::with_capacity(n);
    for i in 0..n {
        let x = (lcg(&mut state) % 4096) as i32;
        let y = (lcg(&mut state) % 4096) as i32;
        let e = store
            .spawn(
                if i % 4 == 0 {
                    EntityKind::Player
                } else {
                    EntityKind::Monster
                },
                Position { x, y },
                stats_for(lcg(&mut state)),
                Inventory::default(),
            )
            .expect("script usa stats válidos");
        ids.push(e);
    }
    for (i, e) in ids.iter().enumerate() {
        if i % 2 == 0 {
            assert!(store.despawn(*e));
        }
    }
    (store, ids)
}

#[test]
fn same_seed_same_state_hash() {
    let (a, _) = script(0x1BAD5EED, 500);
    let (b, _) = script(0x1BAD5EED, 500);
    assert_eq!(a.live_count(), 250);
    assert_eq!(hash_store(&a, 0), hash_store(&b, 0));

    let (c, _) = script(0xDECAFBAD, 500);
    assert_ne!(
        hash_store(&a, 0),
        hash_store(&c, 0),
        "hash é sensível ao seed"
    );
}

#[test]
fn tick_chunking_does_not_change_state() {
    // Mesmo tempo total em fatias diferentes → mesmos ticks → mesmo hash.
    let run = |chunks: &[u64]| -> (u64, u64) {
        let clock = ManualClock::at(0);
        let mut sched = Scheduler::new(SchedulerConfig::validated(20).expect("válido"));
        let mut before = 0u64;
        for &ms in chunks {
            clock.advance(ms);
            let elapsed = elapsed_ms(&clock, before).expect("relógio avança");
            sched.advance(elapsed);
            before = clock.now_ms();
        }
        let (store, _) = script(0x1BAD5EED, 200);
        (sched.tick().get(), hash_store(&store, sched.tick().get()))
    };
    let fatias_finas: Vec<u64> = vec![7; 286]; // 2002 ms em passos de 7 ms
    let fatias_grossas: Vec<u64> = vec![50; 40]; // 2000 ms em passos de 50 ms
    assert_eq!(run(&fatias_finas), run(&fatias_grossas));
    assert_eq!(run(&fatias_grossas).0, 40);
}

#[test]
fn grid_density_war_scenario() {
    // 5000 entidades na mesma região (guerra): inserção + consulta com números.
    let mut store = ComponentStore::default();
    store.reserve(5000);
    let mut grid = Grid::new(GridConfig::validated(4096, 4096, 128).expect("grade válida"));
    grid.reserve_per_cell(256);
    let mut state = 0xC0FFEE;
    let t0 = std::time::Instant::now();
    for _ in 0..5000 {
        let x = 1900 + (lcg(&mut state) % 200) as i32;
        let y = 1900 + (lcg(&mut state) % 200) as i32;
        let e = store
            .spawn(
                EntityKind::Monster,
                Position { x, y },
                stats_for(1),
                Inventory::default(),
            )
            .expect("spawn válido");
        grid.insert(e, x, y).expect("dentro do mundo");
    }
    let insert_ms = t0.elapsed();
    let mut found = 0usize;
    let t1 = std::time::Instant::now();
    grid.for_each_in_radius(2000, 2000, 256, |_| found += 1);
    let query_ms = t1.elapsed();
    eprintln!(
        "guerra: 5000 inserts em {insert_ms:?}, consulta raio-256 achou {found} em {query_ms:?}"
    );
    assert_eq!(found, 5000);
    assert!(
        query_ms.as_millis() < 50,
        "consulta de guerra abaixo de 50 ms"
    );
}
