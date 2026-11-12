//! Custo da IA de NPC no orçamento do tick — critério de aceite da GAME-06.
//!
//! Roda com `cargo bench -p wyd-core --bench tick` (release, `harness = false`, sem
//! dependência nova: o projeto mantém build offline e `criterion` não está no lock).
//!
//! Cenário: [`NPCS`] NPCs vivos + [`PLAYERS`] jogadores no mesmo `ComponentStore`, no grid de
//! 25×20 tiles que o mundo usa nos testes. A cada tick o bench percorre a **cadeia real de
//! decisão** (`systems::npc::decide` → `npc::brain::decide` → `budget`), que é exatamente a que
//! roda no Hot Path; nada é simulado à parte. Mede o tempo de parede por tick com `Instant`,
//! reporta média/p50/p99 e compara com os 50 ms de um tick a 20 Hz.
//!
//! O bench **falha** (exit 1) se o custo passar do teto de 10% declarado no ADR-0016 — trocar a
//! régua por exceção escrita em doc não vale.

use std::hint::black_box;
use std::time::{Duration, Instant};

use wyd_core::npc::{DecisionBudget, NpcProfile, NpcStates, DECISION_EVERY_TICKS};
use wyd_core::systems::npc::decide;
use wyd_core::{
    tick_interval_ms, ComponentStore, Entity, EntityKind, Inventory, Position, SpawnSpec, Stats,
    TICK_HZ,
};

/// NPCs vivos no cenário (o número do critério de aceite).
const NPCS: u32 = 500;
/// Jogadores no mundo (candidatos a alvo; 1 a cada 10 NPCs).
const PLAYERS: u32 = 50;
/// Ticks descartados para estabilizar cache/ramp-up.
const WARMUP_TICKS: u32 = 50;
/// Ticks medidos.
const MEASURED_TICKS: u32 = 400;
/// Teto do critério de aceite: IA ≤ 10% do orçamento do tick.
const MAX_SHARE_PCT: f64 = 10.0;
/// Offset/base do FNV-1a 64.
const FNV_OFFSET: u64 = 0xcbf2_9ce4_8422_2325;
/// Primo do FNV-1a 64.
const FNV_PRIME: u64 = 0x0000_0100_0000_01B3;

fn perfil() -> NpcProfile {
    // Perfil de conteúdo (não constante de engine): o valor vem do servidor em produção.
    NpcProfile::new(6, 12, 2, 2_000, 3).expect("perfil do cenário é válido")
}

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

struct Mundo {
    store: ComponentStore,
    npcs: Vec<Entity>,
    alvos: Vec<Entity>,
    states: NpcStates,
    budget: DecisionBudget,
}

impl Mundo {
    fn novo(jogadores: u32) -> Self {
        let mut store = ComponentStore::default();
        let mut npcs = Vec::with_capacity(NPCS as usize);
        for i in 0..NPCS {
            let posicao = Position {
                x: i32::try_from((i % 25) * 2).expect("coordenada do grid"),
                y: i32::try_from((i / 25) * 2).expect("coordenada do grid"),
            };
            npcs.push(
                store
                    .spawn(EntityKind::Monster, posicao, stats(), Inventory::default())
                    .expect("NPC do cenário é válido"),
            );
        }
        let mut alvos = Vec::with_capacity(jogadores as usize);
        for i in 0..jogadores {
            let posicao = Position {
                x: i32::try_from((i % 10) * 5 + 2).expect("coordenada do grid"),
                y: i32::try_from((i / 10) * 5 + 2).expect("coordenada do grid"),
            };
            alvos.push(
                store
                    .spawn(EntityKind::Player, posicao, stats(), Inventory::default())
                    .expect("jogador do cenário é válido"),
            );
        }
        let teto = (NPCS / DECISION_EVERY_TICKS) + 8;
        Self {
            store,
            npcs,
            alvos,
            states: NpcStates::default(),
            budget: DecisionBudget::new(u16::try_from(teto).expect("teto cabe em u16")),
        }
    }

    /// Um tick: percorre os NPCs como o sistema faz em produção e devolve
    /// `(decisões emitidas, hash do fluxo)`.
    fn tick(&mut self, tick: u32) -> (usize, u64) {
        let Self {
            store,
            npcs,
            alvos,
            states,
            budget,
        } = self;
        budget.reset();
        let mut decisoes = 0usize;
        let mut hash = FNV_OFFSET;
        for npc in npcs.iter() {
            let decisao = decide(store, *npc, tick, perfil(), alvos, states, budget)
                .expect("NPC vivo do cenário decide sem erro");
            if let Some(decisao) = decisao {
                decisoes += 1;
                hash ^= u64::from(u32::try_from(decisao.intention.kind().len()).unwrap_or(0));
                hash = hash.wrapping_mul(FNV_PRIME);
            }
        }
        (decisoes, hash)
    }
}

struct Medicao {
    amostras: Vec<Duration>,
    decisoes: usize,
    hash: u64,
}

fn medir(mundo: &mut Mundo, ticks: u32) -> Medicao {
    let mut amostras = Vec::with_capacity(ticks as usize);
    let mut decisoes = 0usize;
    let mut hash = FNV_OFFSET;
    for tick in 0..ticks {
        let inicio = Instant::now();
        let (n, h) = mundo.tick(tick);
        let decorrido = inicio.elapsed();
        black_box(h);
        if tick >= WARMUP_TICKS {
            amostras.push(decorrido);
            decisoes += n;
            hash = hash.rotate_left(1) ^ h;
        }
    }
    Medicao {
        amostras,
        decisoes,
        hash,
    }
}

fn micros(d: Duration) -> f64 {
    d.as_nanos() as f64 / 1_000.0
}

fn percentil(amostras: &[Duration], p: f64) -> f64 {
    if amostras.is_empty() {
        return 0.0;
    }
    let mut ordenadas = amostras.to_vec();
    ordenadas.sort_unstable();
    let indice = ((ordenadas.len() as f64 - 1.0) * p / 100.0).round() as usize;
    micros(ordenadas[indice.min(ordenadas.len() - 1)])
}

fn resumo(id: &str, nome: &str, medicao: &Medicao, orcamento_us: f64) {
    let total: Duration = medicao.amostras.iter().sum();
    let media = micros(total) / medicao.amostras.len() as f64;
    let p50 = percentil(&medicao.amostras, 50.0);
    let p99 = percentil(&medicao.amostras, 99.0);
    let pior = medicao.amostras.iter().copied().max().map_or(0.0, micros);
    let por_decisao = if medicao.decisoes == 0 {
        0.0
    } else {
        micros(total) / medicao.decisoes as f64
    };
    let share_media = media / orcamento_us * 100.0;
    let share_p99 = p99 / orcamento_us * 100.0;
    println!("--- {nome} ---");
    println!(
        "ticks medidos: {} | decisões: {} ({:.0}/tick)",
        medicao.amostras.len(),
        medicao.decisoes,
        medicao.decisoes as f64 / medicao.amostras.len() as f64
    );
    println!(
        "tempo total: {:.2} ms | média/tick: {media:.2} µs | p50: {p50:.2} µs | p99: {p99:.2} µs | pior: {pior:.2} µs",
        micros(total) / 1_000.0
    );
    println!("custo médio por decisão: {por_decisao:.3} µs");
    println!(
        "% do orçamento do tick: média {share_media:.3}% | p99 {share_p99:.3}% (teto {MAX_SHARE_PCT:.0}%)"
    );
    println!("hash FNV-1a do fluxo: 0x{:016x}", medicao.hash);
    // Linha de contrato com `tools/dev/bench-gate.mjs` (regressão no CI):
    // só chaves fixas, números já arredondados — o gate não faz parsing de prosa.
    println!(
        "BENCH_PONTO {{\"cenario\":\"{id}\",\"p50_us\":{p50:.2},\"p99_us\":{p99:.2},\"media_us\":{media:.2},\"share_p99_pct\":{share_p99:.3}}}"
    );
    if share_media > MAX_SHARE_PCT || share_p99 > MAX_SHARE_PCT {
        eprintln!(
            "ESTOURO: IA em {share_media:.3}% (p99 {share_p99:.3}%) do orçamento — ajustar cadência/orçamento antes de fechar o critério"
        );
        std::process::exit(1);
    }
}

/// Entidades por rodada do cenário ECS (GAME-02).
const ECS_BATCH: u32 = 10_000;
/// Rodadas medidas do cenário ECS.
const ECS_ROUNDS: u32 = 40;
/// Teto do cenário ECS: spawn + iteração + despawn de um lote ≤ 50% de um tick.
const ECS_MAX_SHARE_PCT: f64 = 50.0;

/// Baseline de estrutura de dados (GAME-02): spawn em lote, uma passada contígua nas
/// posições (como o tick faz) e despawn em lote, `ECS_ROUNDS` vezes com o **mesmo** lote.
///
/// Não mede regra de jogo nenhuma — mede o custo de manter `Vec<Position>`/`Vec<Stats>`
/// contíguos, que é o que o ADR-0004 troca por um crate de ECS.
fn cenario_ecs(orcamento_us: f64) {
    let specs: Vec<SpawnSpec> = (0..ECS_BATCH)
        .map(|i| SpawnSpec {
            kind: EntityKind::Monster,
            position: Position {
                x: i32::try_from(i % 100).expect("coordenada do grid"),
                y: i32::try_from(i / 100).expect("coordenada do grid"),
            },
            stats: stats(),
        })
        .collect();
    let mut store = ComponentStore::default();
    let mut amostras = Vec::with_capacity(ECS_ROUNDS as usize);
    for _ in 0..ECS_ROUNDS {
        let inicio = Instant::now();
        let ids = store.spawn_batch(&specs).expect("lote do cenário é válido");
        let mut soma = 0i64;
        for posicao in store.positions() {
            soma += i64::from(posicao.x) + i64::from(posicao.y);
        }
        black_box(soma);
        let removidos = store.despawn_batch(&ids);
        amostras.push(inicio.elapsed());
        assert_eq!(removidos, ECS_BATCH as usize, "lote inteiro despawnado");
    }
    assert_eq!(store.live_count(), 0, "nenhuma entidade viva ao fim");

    let total: Duration = amostras.iter().sum();
    let media = micros(total) / amostras.len() as f64;
    let p50 = percentil(&amostras, 50.0);
    let p99 = percentil(&amostras, 99.0);
    let por_entidade = media * 1_000.0 / f64::from(ECS_BATCH);
    let share_p99 = p99 / orcamento_us * 100.0;
    println!("--- ECS: {ECS_BATCH} entidades por rodada (spawn + iteração contígua + despawn) ---");
    println!(
        "rodadas medidas: {} | p50: {p50:.2} µs | p99: {p99:.2} µs",
        amostras.len()
    );
    println!(
        "custo por entidade: {por_entidade:.3} ns | % do orçamento do tick (p99): {share_p99:.3}% (teto {ECS_MAX_SHARE_PCT:.0}%)"
    );
    println!(
        "BENCH_PONTO {{\"cenario\":\"ecs_lote_10000\",\"p50_us\":{p50:.2},\"p99_us\":{p99:.2},\"ns_por_entidade\":{por_entidade:.3},\"share_p99_pct\":{share_p99:.3}}}"
    );
    println!(
        "estado final: {} entidades vivas, {} índices no armazém",
        store.live_count(),
        store.len()
    );
    if share_p99 > ECS_MAX_SHARE_PCT {
        eprintln!(
            "ESTOURO: cenário ECS em {share_p99:.3}% do orçamento — rever o layout SoA antes de fechar o critério"
        );
        std::process::exit(1);
    }
}

fn main() {
    // `cargo test --all-targets` também executa este alvo: em debug a medição não vale, então
    // ele sai verde sem medir (a régua de 10% roda em release, via `cargo bench`).
    if cfg!(debug_assertions) {
        println!("bench em debug: sem medição — rode `cargo bench -p wyd-core --bench tick`");
        return;
    }
    let intervalo = tick_interval_ms();
    let orcamento_us = intervalo as f64 * 1_000.0;
    println!("=== IA de NPC — custo no orçamento do tick (GAME-06) ===");
    println!(
        "cenário: {NPCS} NPCs vivos + {PLAYERS} jogadores | {TICK_HZ} Hz ({intervalo} ms/tick, orçamento {orcamento_us:.0} µs) | decisões/tick esperadas: {}",
        NPCS / DECISION_EVERY_TICKS
    );
    println!("aquecimento: {WARMUP_TICKS} ticks | medidos: {MEASURED_TICKS} ticks");

    let mut com_alvos = Mundo::novo(PLAYERS);
    let medicao = medir(&mut com_alvos, WARMUP_TICKS + MEASURED_TICKS);
    resumo(
        "ia_500_npcs_com_alvos",
        &format!("{NPCS} NPCs + {PLAYERS} jogadores (percepção com alvos)"),
        &medicao,
        orcamento_us,
    );

    // Isola o custo de percepção: mesmo mundo sem jogador nenhum.
    let mut sem_alvos = Mundo::novo(0);
    let medicao_vazia = medir(&mut sem_alvos, WARMUP_TICKS + MEASURED_TICKS);
    resumo(
        "ia_500_npcs_sem_alvos",
        &format!("{NPCS} NPCs sem jogador (piso de custo)"),
        &medicao_vazia,
        orcamento_us,
    );

    println!("veredito: DENTRO do teto de {MAX_SHARE_PCT:.0}% — custo da IA medido, não estimado");

    println!();
    println!("=== ECS — spawn/despawn em lote e iteração contígua (GAME-02) ===");
    cenario_ecs(orcamento_us);
}
