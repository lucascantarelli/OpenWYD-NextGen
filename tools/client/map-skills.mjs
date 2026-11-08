#!/usr/bin/env node
/**
 * Fecha o mapeamento skill → classe/linhagem a partir do próprio cliente.
 *
 * Descoberta que viabiliza isto: `ItemList.bin` (6.500 registros de 140 B, XOR 0x5A)
 * contém os **nomes das skills**, além dos itens. Como o nome fica no registro, dá para
 * alinhar `skill_index` → registro → campos, sem depender de trace dinâmico.
 *
 * Método (o que faz o resultado ser evidência e não chute):
 *   1. casa o nome de cada skill do seed com o run ASCII dentro de um registro;
 *   2. descobre o offset do nome (é o mesmo em todos?);
 *   3. procura o campo (u8/u16 em qualquer offset) cuja partição REPRODUZ os rótulos
 *      de classe/linhagem já confirmados por regra (`rule:name_confirmed`);
 *   4. só grava o mapa se a pureza for alta — senão reporta `inconclusivo`.
 *
 * Uso: node tools/client/map-skills.mjs
 * Saída: docs/cliente/gerado/skills-map.json
 */
import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "..", "..");
const BIN = path.join(ROOT, "client", "ItemList.bin");
const SEED = path.join(ROOT, "crates", "wyd-db", "migrations", "0018_seed_skill_catalog.sql");
const OUT = path.join(ROOT, "docs", "cliente", "gerado", "skills-map.json");

const KEY = 0x5a;
const HEADER = 4;
const SIZE = 140;

const raw = readFileSync(BIN);
const decoded = Buffer.from(raw.subarray(HEADER).map((b) => b ^ KEY));
if (decoded.length % SIZE !== 0) {
  throw new Error(`ItemList: ${decoded.length} bytes não é múltiplo de ${SIZE}`);
}
const records = [];
for (let i = 0; i < decoded.length / SIZE; i += 1) {
  records.push(decoded.subarray(i * SIZE, (i + 1) * SIZE));
}

/** Runs ASCII imprimíveis (>= 3 chars) de um registro. */
function runs(record) {
  const out = [];
  let cur = "";
  for (const byte of record) {
    if (byte >= 0x20 && byte < 0x7f) {
      cur += String.fromCharCode(byte);
    } else {
      if (cur.length >= 3) out.push(cur);
      cur = "";
    }
  }
  if (cur.length >= 3) out.push(cur);
  return out;
}

const seed = readFileSync(SEED, "utf8");
const skills = [...seed.matchAll(/\((\d+), '([^']+)'/g)].map((m) => ({
  index: Number(m[1]),
  name: m[2],
  ascii: m[2].replace(/_/g, " "),
}));

// rótulos já confirmados por regra — são o gabarito do passo 3
const known = [...seed.matchAll(/\((\d+), '([^']+)'[\s\S]*?'([a-z]+)', '([a-z_]+)', '[a-z]+', '[a-z]+', 'rule:name_confirmed'/g)];
const labels = new Map();
for (const m of known) labels.set(Number(m[1]), { classCode: m[3], lineageCode: m[4] });

/**
 * Casamento ESTRITO: o run precisa ser exatamente o nome (sem normalizar).
 * Normalizar gerou falsos positivos ("Titã" casando com "Tit"), e um mapeamento
 * errado é pior do que nenhum.
 */
const byExact = new Map(skills.map((s) => [s.ascii, s]));
const byExactUnderscore = new Map(skills.map((s) => [s.name, s]));

const matches = [];
const all = [];
for (let pos = 0; pos < decoded.length; pos += 1) {
  // nomes começam em offset múltiplo de 4 no arquivo decodificado
  if (pos % 4 !== 0) continue;
  for (const [text, hit] of byExact) {
    if (hit && decoded.subarray(pos, pos + text.length).toString("latin1") === text) {
      all.push({ offset: pos, index: hit.index, name: hit.name, length: text.length });
      break;
    }
  }
  for (const [text, hit] of byExactUnderscore) {
    if (hit && decoded.subarray(pos, pos + text.length).toString("latin1") === text) {
      all.push({ offset: pos, index: hit.index, name: hit.name, length: text.length });
      break;
    }
  }
}
for (const hit of all) {
  matches.push({ ...hit, record: Math.floor(hit.offset / SIZE), offsetInRecord: hit.offset % SIZE });
}

const offsetCounts = new Map();
for (const m of matches) offsetCounts.set(m.offsetInRecord, (offsetCounts.get(m.offsetInRecord) ?? 0) + 1);

console.log(`registros (grade de ${SIZE} B): ${records.length}`);
console.log(`skills no seed: ${skills.length} | nomes exactos achados: ${new Set(matches.map((m) => m.index)).size}`);
console.log(
  `offset dentro do registro: ${[...offsetCounts.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, 6)
    .map(([offset, count]) => `${offset}×${count}`)
    .join(", ")}`,
);

// O passo entre nomes consecutivos revela se a grade de 140 B vale na região de nomes.
const sorted = [...matches].sort((a, b) => a.offset - b.offset);
const deltas = [];
for (let i = 1; i < sorted.length; i += 1) deltas.push(sorted[i].offset - sorted[i - 1].offset);
const uniqueDeltas = [...new Set(deltas)].sort((a, b) => a - b);
console.log(
  `passo entre nomes: ${uniqueDeltas.slice(0, 10).join(", ")}${uniqueDeltas.length > 10 ? "…" : ""}`,
  uniqueDeltas.every((d) => d === SIZE)
    ? "→ grade fixa de 140 B (mapeável)"
    : "→ passo VARIÁVEL: o arquivo não é array fixo de 140 B nesta região",
);

/** pureza: um campo serve se cada valor aponta para um único rótulo */
function purity(field, pick) {
  const groups = new Map();
  for (const m of matches) {
    const label = labels.get(m.index);
    if (!label) continue;
    const value = pick(records[m.record], field);
    const key = value;
    const group = groups.get(key) ?? new Map();
    group.set(label.classCode, (group.get(label.classCode) ?? 0) + 1);
    groups.set(key, group);
  }
  let total = 0;
  let correct = 0;
  for (const group of groups.values()) {
    const counts = [...group.values()];
    total += counts.reduce((a, b) => a + b, 0);
    correct += Math.max(...counts);
  }
  return { field, purity: total ? correct / total : 0, total, classes: groups.size };
}

const candidates = [];
for (let field = 0; field < SIZE; field += 1) {
  candidates.push(purity(field, (record, f) => record.readUInt8(f)));
  if (field < SIZE - 1) candidates.push(purity(field, (record, f) => record.readUInt16LE(f)));
}
candidates.sort((a, b) => b.purity - a.purity || b.total - a.total);

console.log(
  `melhores campos (pureza sobre ${candidates[0]?.total ?? 0} skills rotuladas):`,
  candidates
    .slice(0, 6)
    .map((c) => `off ${c.field} purity ${(c.purity * 100).toFixed(1)}% (${c.classes} valores)`)
    .join(" | "),
);

const result = {
  generatedBy: "tools/client/map-skills.mjs",
  binary: "client/ItemList.bin",
  // honestidade: `diagnostico` = nomes localizados mas estrutura não fechada
  status:
    matches.length >= 20 && candidates[0]?.purity >= 0.9 && uniqueDeltas.every((d) => d === SIZE)
      ? "mapeavel"
      : "diagnostico",
  recordSize: SIZE,
  matches: matches.length,
  distinctSkills: new Set(matches.map((m) => m.index)).size,
  nameOffsets: [...offsetCounts.entries()].sort((a, b) => b[1] - a[1]).map(([offset, count]) => ({ offset, count })),
  candidates: candidates.slice(0, 6).map((c) => ({ ...c, purity: Number(c.purity.toFixed(4)) })),
  mapped: matches
    .filter((m, i, all) => all.findIndex((x) => x.index === m.index) === i)
    .map((m) => ({
      index: m.index,
      name: m.name,
      record: m.record,
      known: labels.get(m.index) ?? null,
    }))
    .sort((a, b) => a.index - b.index),
};

mkdirSync(path.dirname(OUT), { recursive: true });
writeFileSync(OUT, `${JSON.stringify(result, null, 2)}\n`);
console.log(`→ ${path.relative(ROOT, OUT)}`);

// amostra legível: 3 registros casados
for (const m of matches.slice(0, 3)) {
  const r = records[m.record];
  console.log(`\n#${m.record} skill ${m.index} "${m.name}"`);
  for (let line = 0; line < SIZE; line += 16) {
    console.log(`  ${line.toString().padStart(3)} ${r.subarray(line, line + 16).toString("hex").match(/.{2}/g).join(" ")}`);
  }
}
