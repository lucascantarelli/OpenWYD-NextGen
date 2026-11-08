#!/usr/bin/env node
/**
 * Aplica a série de migrations no Postgres DA STACK (`wyd-db` do compose).
 *
 * Não é o mesmo runner de `tools/content/migrate-cycle.mjs`: aquele prova o ciclo completo
 * num container efêmero, este prepara o banco que o gateway e o portal usam. Sem ele o
 * `/health` do gateway não tem `content_version` e o catálogo não existe — saúde sem dados.
 *
 * Uso:
 *   node tools/dev/db-migrate.mjs            # aplica o que ainda não está aplicado (falha alto)
 *   node tools/dev/db-migrate.mjs --reset    # DROP SCHEMA public CASCADE antes (dev)
 *
 * Requer o container `wyd-db` de pé (`make stack-up`).
 */
import { readdirSync, readFileSync } from 'node:fs';
import path from 'node:path';

import { blocos } from '../content/banco-parse.mjs';
import { docker, psql, waitForPostgres } from '../content/pg.mjs';

const ROOT = path.resolve(import.meta.dirname, '..', '..');
const DIR = path.join(ROOT, 'crates', 'wyd-db', 'migrations');
const reset = process.argv.includes('--reset');

const ctx = {
  container: process.env.WYD_STACK_CONTAINER ?? 'wyd-db',
  user: process.env.POSTGRES_USER ?? 'wyd',
  db: process.env.POSTGRES_DB ?? 'wyd',
};

waitForPostgres(ctx);

const files = readdirSync(DIR)
  .filter((f) => f.endsWith('.sql'))
  .sort();

if (reset) {
  psql(ctx, 'DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;', 'reset do schema');
  console.log(`reset: schema public recriado em ${ctx.container}/${ctx.db}`);
}

// `CREATE TABLE` sem IF NOT EXISTS é proposital: aplicar duas vezes precisa estourar, não
// fingir sucesso. Para reaplicar existe --reset.
let aplicadas = 0;
for (const file of files) {
  const sql = blocos(readFileSync(path.join(DIR, file), 'utf8'), file).up;
  psql(ctx, sql, `up ${file}`);
  aplicadas += 1;
  console.log(`  ↑ ${file}`);
}

const contagens = psql(
  ctx,
  `SELECT 'tabelas=' || count(*)::text FROM information_schema.tables
     WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
   UNION ALL SELECT 'item_catalog=' || count(*)::text FROM item_catalog
   UNION ALL SELECT 'content_version=' || count(*)::text FROM content_version`,
  'contagens',
);
console.log(aplicadas === files.length ? `migrations aplicadas: ${aplicadas}` : 'fila incompleta');
console.log(contagens.split('\n').join(' '));

const sobra = docker([
  'exec', ctx.container, 'psql', '-U', ctx.user, '-d', ctx.db, '-tAc',
  'SELECT count(*) FROM information_schema.tables WHERE table_schema = \'public\' AND table_type = \'BASE TABLE\'',
]);
if (Number(sobra.stdout.trim()) === 0) {
  throw new Error('banco sem tabelas depois de aplicar: migrations não surtiram efeito');
}
