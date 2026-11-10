#!/usr/bin/env node
/**
 * Gate de deriva de status (FASE-02-6): o status de cada issue tem de bater entre o índice
 * (`roadmap-rust/index.md`, tabela de execução) e o painel (`docs/rust/03-progresso.md`).
 *
 * Compara só os IDs presentes nas duas fontes. Issue que está só no índice é reportada como
 * aviso, não falha (painel ainda não registra toda issue em andamento).
 *
 * Uso: node tools/content/check-issue-status.mjs
 * Saída: uma linha por divergência; exit 1 se houver divergência.
 */
import { readFileSync } from "node:fs";

const STATUS = ["✅", "🚧", "⬜", "⛔"];
const statusOf = (cell) => STATUS.find((s) => cell.includes(s)) ?? null;

const indice = new Map();
for (const linha of readFileSync("roadmap-rust/index.md", "utf8").split("\n")) {
  const m = linha.match(/^\| `issues\/(FASE-\d\d-\d+)-[^`]*` \|.*\|\s*(\S+)\s*\|$/);
  if (m) indice.set(m[1], statusOf(m[2]));
}

const painel = new Map();
for (const linha of readFileSync("docs/rust/03-progresso.md", "utf8").split("\n")) {
  const m = linha.match(/^\| (FASE-\d\d-\d+)\s[^|]*\|\s*([^|]*)\|/);
  if (m) painel.set(m[1], statusOf(m[2]));
}

const problemas = [];
let comparados = 0;
for (const [id, st] of indice) {
  if (!painel.has(id)) continue;
  comparados++;
  if (painel.get(id) !== st) {
    problemas.push(`${id}: índice ${st} × painel ${painel.get(id)}`);
  }
}

const soIndice = [...indice.keys()].filter((id) => !painel.has(id));
console.log(`status: ${comparados} issues comparadas | só no índice: ${soIndice.length}`);
for (const p of problemas) console.log(`  ✗ ${p}`);
if (problemas.length) {
  console.log(`\n${problemas.length} divergência(s) de status`);
  process.exit(1);
}
console.log("ok: status do índice e do painel batem");
