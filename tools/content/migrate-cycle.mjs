#!/usr/bin/env node
/**
 * Roda a série de migrations num Postgres REAL (container efêmero) — prova de ponta a ponta.
 *
 * Ciclo: `up` de todas as migrations → contagens → `down` de todas em ordem inversa → base
 * vazia → `up` de novo, comparando as contagens dos dois ciclos. Ele não "conserta" nada
 * sozinho: qualquer erro do psql interrompe com a mensagem original, porque a série precisa
 * falhar alto, não ser mascarada.
 *
 * Uso:
 *   node tools/content/migrate-cycle.mjs            # container efêmero padrão (wyd-migcheck)
 *   node tools/content/migrate-cycle.mjs --keep     # não remove o container ao terminar
 *
 * Requer Docker com a imagem `postgres:17-alpine` (a mesma do compose).
 */
import { readdirSync, readFileSync } from 'node:fs';
import path from 'node:path';

import { blocos } from './banco-parse.mjs';
import {
  contexto,
  docker,
  ensureContainer as pgEnsureContainer,
  psql as pgPsql,
  resetSchema as pgResetSchema,
} from './pg.mjs';

const ROOT = path.resolve(import.meta.dirname, '..', '..');
const DIR = path.join(ROOT, 'crates', 'wyd-db', 'migrations');
const ctx = contexto();
const keep = process.argv.includes('--keep');

const started = Date.now();
const files = readdirSync(DIR)
  .filter((f) => f.endsWith('.sql'))
  .sort();

const psql = (sql, label) => pgPsql(ctx, sql, label);
const ensureContainer = () => pgEnsureContainer(ctx);
const resetSchema = () => pgResetSchema(ctx);

function apply(sequence, direction) {
  for (const file of sequence) {
    const sql = blocos(readFileSync(path.join(DIR, file), 'utf8'), file)[direction];
    const t0 = Date.now();
    psql(sql, `${direction} ${file}`);
    console.log(`  ${direction === 'up' ? '↑' : '↓'} ${file}  ${Date.now() - t0} ms`);
  }
}

function counts(label) {
  const query = `
    SELECT 'tabelas', count(*)::text FROM information_schema.tables
      WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
    UNION ALL SELECT 'item_catalog', count(*)::text FROM item_catalog
    UNION ALL SELECT 'skill_catalog', count(*)::text FROM skill_catalog
    UNION ALL SELECT 'skills_com_linhagem', count(*)::text FROM skill_catalog WHERE class_code IS NOT NULL
    UNION ALL SELECT 'item_effect', count(*)::text FROM item_effect
    UNION ALL SELECT 'game_system', count(*)::text FROM game_system
    UNION ALL SELECT 'region', count(*)::text FROM region
    UNION ALL SELECT 'npc_definition', count(*)::text FROM npc_definition
    UNION ALL SELECT 'npc_shop_item', count(*)::text FROM npc_shop_item
    UNION ALL SELECT 'level_item', count(*)::text FROM level_item
    UNION ALL SELECT 'city', count(*)::text FROM city
    UNION ALL SELECT 'mob_template_stat', count(*)::text FROM mob_template_stat
    UNION ALL SELECT 'refine_rate', count(*)::text FROM refine_rate
    UNION ALL SELECT 'refine_gem_effect', count(*)::text FROM refine_gem_effect
    UNION ALL SELECT 'quest', count(*)::text FROM quest
    UNION ALL SELECT 'world_event', count(*)::text FROM world_event
    UNION ALL SELECT 'arena', count(*)::text FROM arena
    UNION ALL SELECT 'character_skills', count(*)::text FROM character_skills
    UNION ALL SELECT 'content_version_max', coalesce(max(version), 0)::text FROM content_version`;
  const rows = psql(query, `contagens (${label})`)
    .split('\n')
    .map((line) => line.split('|'))
    .filter((cells) => cells.length === 2);
  console.log(`\n${label}`);
  for (const [name, value] of rows) console.log(`  ${name.padEnd(22)} ${value}`);
  return Object.fromEntries(rows);
}

try {
  ensureContainer();

  console.log(`\n=== ciclo 1: up das ${files.length} migrations em base vazia`);
  resetSchema();
  apply(files, 'up');
  const first = counts('contagens após o primeiro up');

  console.log('\n=== ciclo 1: down em ordem inversa');
  apply([...files].reverse(), 'down');
  const afterDown = psql(
    "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE';",
    'contagem após o down',
  );
  console.log(`  tabelas restantes: ${afterDown}`);
  if (afterDown !== '0') throw new Error(`down deixou ${afterDown} tabela(s) na base`);

  console.log('\n=== ciclo 2: up de novo (base vazia, sem reset)');
  apply(files, 'up');
  const second = counts('contagens após o segundo up');

  const divergent = Object.keys(first).filter((key) => first[key] !== second[key]);
  if (divergent.length) {
    throw new Error(
      `ciclo 2 divergiu em: ${divergent.map((k) => `${k} ${first[k]}→${second[k]}`).join(', ')}`,
    );
  }
  console.log(
    `\nok: ciclo up/down/up reproduz as mesmas contagens (${files.length} migrations, ${((Date.now() - started) / 1000).toFixed(1)} s)`,
  );
} catch (error) {
  console.error(`\nFALHOU: ${error.message}`);
  process.exitCode = 1;
} finally {
  if (!keep) {
    const removed = docker(['rm', '-f', ctx.container]);
    if (removed.status === 0) console.log(`container: ${ctx.container} removido`);
  } else {
    console.log(`container: ${ctx.container} mantido (--keep)`);
  }
}
