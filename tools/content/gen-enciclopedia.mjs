#!/usr/bin/env node
// Gera `wyd-portal/src/content/itens.json` a partir de
// `crates/wyd-db/migrations/0009_seed_item_catalog.sql`.
//
// POR QUE JSON VERSIONADO: a imagem do portal copia só `wyd-portal/` — o seed SQL não
// existe dentro do container, e a rota pública não deve depender do web-api (que exige
// sessão de staff). Uma fonte (o seed), um artefato (o JSON), um CI que reprova divergência.
//
// Formato compacto de propósito: 3.220 itens em array posicional ocupam ~40% do tamanho
// de um array de objetos com chave repetida por linha.
//
// Uso: node tools/content/gen-enciclopedia.mjs
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';

const SEED = 'crates/wyd-db/migrations/0009_seed_item_catalog.sql';
const OUT = 'wyd-portal/src/content/itens.json';

/** Divisão por vírgula respeitando string SQL, parênteses e colchetes do jsonb. */
function splitTop(value) {
  const parts = [];
  let cur = '';
  let depth = 0;
  let inString = false;
  for (let i = 0; i < value.length; i += 1) {
    const ch = value[i];
    if (inString) {
      if (ch === "'" && value[i + 1] === "'") {
        cur += "''";
        i += 1;
        continue;
      }
      if (ch === "'") inString = false;
      cur += ch;
      continue;
    }
    if (ch === "'") {
      inString = true;
      cur += ch;
      continue;
    }
    if (ch === '(' || ch === '[') depth += 1;
    if (ch === ')' || ch === ']') depth -= 1;
    if (ch === ',' && depth === 0) {
      parts.push(cur.trim());
      cur = '';
      continue;
    }
    cur += ch;
  }
  parts.push(cur.trim());
  return parts;
}

const unquote = (value) => value.replace(/^'/, '').replace(/'(::[a-z]+)?$/, '').replace(/''/g, "'");
const num = (value) => {
  if (value === 'NULL' || value === '') return null;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
};

/** `'[{"eff":3,"val":150}]'::jsonb` → `[{e:3,v:150}]` (par compactado). */
function parseEffects(raw) {
  const text = unquote(raw);
  if (!text || text === '[]') return [];
  try {
    return JSON.parse(text).map((pair) => [pair.eff, pair.val]);
  } catch {
    return [];
  }
}

const sql = readFileSync(SEED, 'utf8');
const lines = sql.split('\n');
const start = lines.findIndex((line) => line.startsWith('INSERT INTO item_catalog'));
if (start < 0) throw new Error('gen-enciclopedia: INSERT INTO item_catalog não encontrado');

const items = [];
for (let i = start + 1; i < lines.length; i += 1) {
  const line = lines[i].trim();
  // Fim do bloco: a instrução fecha em `ON CONFLICT` (ou começa o `down`).
  if (line.startsWith('ON CONFLICT') || line.startsWith('-- migrate:down')) break;
  // Lista de colunas, comentário e linha vazia não são tupla — e não encerram o bloco.
  if (!line.startsWith('(')) continue;
  const body = line.replace(/^\(/, '').replace(/\);?$/, '');
  const cols = splitTop(body);
  // Ordem do INSERT em 0009: 0 índice, 1 nome, 2 preço, 3 grade, 7 nível,
  // 14 efeitos, 17 kind, 18 sistema, 19 só evento, 20 negociável.
  const index = num(cols[0]);
  const name = unquote(cols[1]);
  if (!index || !name) continue;
  items.push([
    index,
    name,
    num(cols[2]) ?? 0,
    num(cols[3]) ?? 0,
    num(cols[7]) ?? 0,
    cols[17] === 'NULL' ? null : unquote(cols[17]),
    cols[18] === 'NULL' ? null : unquote(cols[18]),
    cols[19] === 'true',
    cols[20] === 'false' ? false : true,
    parseEffects(cols[14]),
  ]);
}

items.sort((a, b) => a[0] - b[0]);

mkdirSync(path.dirname(OUT), { recursive: true });
const payload = { version: 1, source: '0009_seed_item_catalog', count: items.length, items };
writeFileSync(OUT, `${JSON.stringify(payload)}\n`, 'utf8');

const withSystem = items.filter((item) => item[6]).length;
const withEffects = items.filter((item) => item[9].length > 0).length;
console.log(`gen-enciclopedia: ${items.length} itens (${withEffects} com efeito, ${withSystem} com sistema) → ${OUT}`);
