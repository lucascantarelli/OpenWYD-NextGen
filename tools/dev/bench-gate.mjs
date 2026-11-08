#!/usr/bin/env node
/**
 * Régua de regressão dos baselines medidos (FASE-03-2 ECS, FASE-03-6 IA de NPC).
 *
 * O bench `crates/wyd-core/benches/tick.rs` imprime, por cenário, uma linha
 * `BENCH_PONTO {json}` com p50/p99 em µs e (no ECS) ns por entidade. Este gate
 * roda o bench em release, lê essas linhas e falha (exit 1) quando um cenário
 * estoura o limite declarado abaixo, ou quando um cenário **deixa de reportar**
 * (proteção contra bench que some ou que passa a sair verde sem medir).
 *
 * Por que limite por cenário e não um único número: IA de NPC e estrutura SoA têm
 * custos e réguas diferentes. IA é fração do orçamento do tick (teto de 10% do
 * ADR-0016); ECS é spawn+iteração+despawn de um lote (teto de 50%, FASE-03-2).
 *
 * Por que 2,5–3× o baseline: o baseline foi medido nesta máquina (Windows,
 * 2026-11-08) e o runner do CI é outro hardware. Fator folgado absorve a
 * diferença de máquina **e continua pegando regressão real** (perder o layout
 * SoA, alocar por entidade ou varrer o mundo inteiro custa ordem de grandeza,
 * não 20%).
 *
 * Uso: `node tools/dev/bench-gate.mjs` (ou `make bench-gate`).
 */
import { spawnSync } from "node:child_process";

/** Medições de referência (2026-11-08, release, esta máquina). */
const REFERENCIA = "medido em 2026-11-08 (release, 400 ticks / 40 rodadas)";

/**
 * Limite por cenário; `null` = métrica não se aplica ao cenário.
 *
 * A régua usa **p50 e custo por entidade**, não p99: p99 varia 4× entre execuções
 * na mesma máquina (ruído de agendador/GC do SO) e viraria teste instável; o teto de
 * p99 em % do orçamento do tick continua sendo cobrado pelo próprio bench, que sai
 * com status 1 quando estoura (IA > 10%, ECS > 50%).
 */
const LIMITES = {
  ia_500_npcs_com_alvos: {
    p50_us: 300, // referência 79,70 µs → 3,8×
    ns_por_entidade: null,
  },
  ia_500_npcs_sem_alvos: {
    p50_us: 60, // referência 11,30 µs → 5,3×
    ns_por_entidade: null,
  },
  ecs_lote_10000: {
    p50_us: 9_000, // referências 1 825,50–2 860,10 µs → 3,1× sobre a pior
    ns_por_entidade: 700, // referências 207,322–315,477 ns → 2,2× sobre a pior
  },
};

const MARCADOR = "BENCH_PONTO ";

function cargo() {
  return process.env.CARGO ?? (process.platform === "win32" ? "cargo.exe" : "cargo");
}

function medir() {
  const run = spawnSync(cargo(), ["bench", "-p", "wyd-core", "--bench", "tick"], {
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  });
  if (run.error) {
    console.error(`bench-gate: não consegui executar cargo: ${run.error.message}`);
    process.exit(2);
  }
  // Saída do bench vai para o log do CI: o humano lê a medição completa.
  process.stdout.write(run.stdout ?? "");
  process.stderr.write(run.stderr ?? "");
  if (run.status !== 0) {
    console.error(
      `bench-gate: o bench saiu com status ${run.status} — teto do próprio bench estourado ` +
        "(IA > 10% ou ECS > 50% do tick). Corrija antes de olhar regressão de baseline.",
    );
    process.exit(1);
  }
  return (run.stdout ?? "")
    .split(/\r?\n/)
    .filter((linha) => linha.startsWith(MARCADOR))
    .map((linha) => {
      const texto = linha.slice(MARCADOR.length);
      try {
        return JSON.parse(texto);
      } catch {
        console.error(`bench-gate: linha de contrato ilegível: ${texto}`);
        process.exit(2);
      }
    });
}

function fmt(valor) {
  return typeof valor === "number" ? valor.toFixed(2) : "—";
}

const pontos = medir();
if (pontos.length === 0) {
  console.error(
    "bench-gate: nenhuma linha BENCH_PONTO — o bench mudou de formato ou não mediu (debug?).",
  );
  process.exit(1);
}

const falhas = [];
const vistos = new Set();
for (const ponto of pontos) {
  const limite = LIMITES[ponto.cenario];
  if (!limite) {
    falhas.push(`cenário desconhecido no gate: ${ponto.cenario} (declare o limite aqui)`);
    continue;
  }
  vistos.add(ponto.cenario);
  const p50 = Number(ponto.p50_us);
  const ns = Number(ponto.ns_por_entidade);
  const estourouP50 = Number.isFinite(p50) && p50 > limite.p50_us;
  const estourouNs =
    limite.ns_por_entidade !== null && Number.isFinite(ns) && ns > limite.ns_por_entidade;
  console.log(
    `[${estourouP50 || estourouNs ? "FALHA" : "ok"}] ${ponto.cenario}: ` +
      `p50 ${fmt(p50)} µs (limite ${limite.p50_us})` +
      (limite.ns_por_entidade === null
        ? ""
        : ` | ${fmt(ns)} ns/entidade (limite ${limite.ns_por_entidade})`) +
      ` | p99 ${fmt(Number(ponto.p99_us))} µs (informativo)`,
  );
  if (estourouP50) {
    falhas.push(`${ponto.cenario}: p50 ${fmt(p50)} µs > limite ${limite.p50_us} µs`);
  }
  if (estourouNs) {
    falhas.push(
      `${ponto.cenario}: ${fmt(ns)} ns/entidade > limite ${limite.ns_por_entidade} ns/entidade`,
    );
  }
}

for (const cenario of Object.keys(LIMITES)) {
  if (!vistos.has(cenario)) {
    falhas.push(`cenário ${cenario} não reportou medição (bench incompleto)`);
  }
}

if (falhas.length > 0) {
  console.error(`\nbench-gate: ${falhas.length} regressão(ões) de baseline (${REFERENCIA}):`);
  for (const falha of falhas) console.error(`  - ${falha}`);
  process.exit(1);
}

console.log(`\nbench-gate: ${pontos.length} cenário(s) dentro dos limites (${REFERENCIA}).`);
