#!/usr/bin/env node
/**
 * Gate de deriva de status (PROC-13): o status de cada issue tem de bater entre o índice
 * (`roadmap-rust/index.md`, tabela de execução) e o painel (`docs/rust/03-progresso.md`).
 *
 * A tabela do índice pode ter uma coluna de fila (`| 07 | \`ID\` | ...`); o gate ignora essa
 * coluna e continua exigindo cobertura total e status idêntico ao painel.
 *
 * Uso: node tools/content/check-issue-status.mjs
 * Saída: uma linha por divergência; exit 1 se houver divergência.
 */
import { existsSync, readFileSync, readdirSync } from "node:fs";

const STATUS = ["✅", "🚧", "⬜", "⛔", "🟡", "🟢"];
const statusOf = (cell) => STATUS.find((s) => cell.includes(s)) ?? null;

const indice = new Map();
for (const linha of readFileSync("roadmap-rust/index.md", "utf8").split("\n")) {
  const m = linha.match(/^\| (?:\d+ \| )?`([A-Z]+-\d+)` \| \[[^\]]*\]\(issues\/([A-Z]+-\d+)-[^)]*\) \| ([^|]+) \|/);
  if (m) indice.set(m[1], statusOf(m[3]));
}

const painel = new Map();
const conflitosPainel = [];
for (const linha of readFileSync("docs/rust/03-progresso.md", "utf8").split("\n")) {
  const m = linha.match(/^\| ([A-Z]+-\d+)\s[^|]*\|\s*([^|]*)\|/);
  if (!m) continue;
  const st = statusOf(m[2]);
  if (painel.has(m[1]) && painel.get(m[1]) !== st) {
    conflitosPainel.push(`${m[1]}: linhas do painel divergem (${painel.get(m[1])} × ${st})`);
  }
  painel.set(m[1], st);
}

const problemas = [...conflitosPainel];
const problemasEstrutura = [];

// Estrutura canônica: arquivo, título e índice usam o mesmo CATEGORIA-NN.
const arquivos = readdirSync("roadmap-rust/issues").filter((f) => f.endsWith(".md"));
const ativos = arquivos.filter((f) => !/^FASE-\d\d\.md$/.test(f));
for (const f of arquivos) {
  if (!/^(?:FASE-\d\d|[A-Z]+-\d+)(?:-.+)?\.md$/.test(f)) problemasEstrutura.push(`arquivo fora do padrão: ${f}`);
}
const idsArquivos = new Set();
for (const f of ativos) {
  const m = f.match(/^([A-Z]+-\d+)-/);
  if (!m) {
    problemasEstrutura.push(`arquivo ativo sem ID canônico: ${f}`);
    continue;
  }
  if (idsArquivos.has(m[1])) problemasEstrutura.push(`ID canônico repetido: ${m[1]}`);
  idsArquivos.add(m[1]);
  const texto = readFileSync(`roadmap-rust/issues/${f}`, "utf8");
  const h1 = texto.match(/^#\s+([A-Z]+-\d+)\s+—/m);
  if (!h1 || h1[1] !== m[1]) problemasEstrutura.push(`${f}: título não usa ${m[1]}`);
  if (!texto.includes("**ID histórico:**")) problemasEstrutura.push(`${f}: falta o metadado ID histórico`);
  const stArquivo = statusOf(texto.match(/^> \*\*Estado 2026-11-12:\*\*\s*([^\n]+)/m)?.[1] ?? "");
  if (!stArquivo) problemasEstrutura.push(`${f}: falta status no cabeçalho`);
  else if (indice.has(m[1]) && indice.get(m[1]) !== stArquivo) {
    problemasEstrutura.push(`${m[1]}: arquivo ${stArquivo} × índice ${indice.get(m[1])}`);
  }
}
for (const id of idsArquivos) if (!indice.has(id)) problemasEstrutura.push(`${id}: arquivo sem linha no índice`);
for (const id of indice.keys()) if (!idsArquivos.has(id)) problemasEstrutura.push(`${id}: índice sem arquivo ativo`);
const linhasIndice = readFileSync("roadmap-rust/index.md", "utf8").split("\n");
for (const id of indice.keys()) {
  const linha = linhasIndice.find((l) => l.includes(`\`${id}\` | [`));
  const link = linha?.match(/\(issues\/([^)]+)\)/)?.[1];
  if (link && !existsSync(`roadmap-rust/issues/${link}`)) problemasEstrutura.push(`${id}: link do índice não existe (${link})`);
}

let comparados = 0;
for (const [id, st] of indice) {
  if (!painel.has(id)) continue;
  comparados++;
  if (painel.get(id) !== st) {
    problemas.push(`${id}: índice ${st} × painel ${painel.get(id)}`);
  }
}

const soIndice = [...indice.keys()].filter((id) => !painel.has(id));
for (const id of soIndice) problemas.push(`${id}: sem linha no painel de progresso`);
console.log(`status: ${comparados}/${indice.size} issues ativas comparadas | cobertura incompleta: ${soIndice.length}`);
for (const p of problemas) console.log(`  ✗ ${p}`);
if (problemas.length || problemasEstrutura.length) {
  for (const p of problemasEstrutura) console.log(`  ✗ ${p}`);
  console.log(`\n${problemas.length} divergência(s) de status | ${problemasEstrutura.length} problema(s) de estrutura`);
  process.exit(1);
}
console.log(`ok: status do índice e do painel batem | ${ativos.length} issues canônicas`);
