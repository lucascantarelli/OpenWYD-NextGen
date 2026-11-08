/**
 * Helpers de Postgres efêmero (Docker + psql) usados pelos runners de conteúdo.
 *
 * Extraído de `migrate-cycle.mjs` para o verificador de schema usar exatamente a mesma espera:
 * a imagem oficial sobe um servidor temporário durante o init e reinicia; `pg_isready` fica
 * verde nesse meio-tempo e daria falso positivo. Por isso exigimos duas consultas reais
 * seguidas antes de qualquer coisa.
 */
import { spawnSync } from 'node:child_process';

export function docker(args, options = {}) {
  const result = spawnSync('docker', args, { encoding: 'utf8', ...options });
  if (result.error) throw result.error;
  return result;
}

export function psql(ctx, sql, label) {
  const result = docker(
    [
      'exec', '-i', ctx.container,
      'psql', '-U', ctx.user, '-d', ctx.db,
      '-v', 'ON_ERROR_STOP=1', '-q', '-t', '-A', '-F', '|', '-f', '-',
    ],
    { input: sql },
  );
  if (result.status !== 0) {
    const detalhe = `${result.stdout ?? ''}${result.stderr ?? ''}`.trim();
    throw new Error(`${label} falhou (exit ${result.status})\n${detalhe}`);
  }
  return (result.stdout ?? '').trim();
}

const dormir = (ms) => Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, ms);

export function waitForPostgres(ctx) {
  let seguidas = 0;
  for (let i = 0; i < 120; i += 1) {
    const probe = docker(['exec', ctx.container, 'psql', '-U', ctx.user, '-d', ctx.db, '-tAc', 'SELECT 1']);
    seguidas = probe.status === 0 && probe.stdout.trim() === '1' ? seguidas + 1 : 0;
    if (seguidas >= 2) return;
    dormir(250);
  }
  const logs = docker(['logs', '--tail', '20', ctx.container]);
  throw new Error(`${ctx.container}: sem duas consultas seguidas em 30s\n${logs.stderr ?? ''}`);
}

export function ensureContainer(ctx) {
  const rodando = docker([
    'ps', '--filter', `name=^${ctx.container}$`, '--filter', 'status=running', '--format', '{{.Names}}',
  ]);
  if (rodando.stdout.trim() === ctx.container) {
    console.log(`container: reutilizando ${ctx.container}`);
    return;
  }
  docker(['rm', '-f', ctx.container]);
  const run = docker([
    'run', '-d', '--name', ctx.container,
    '-e', `POSTGRES_PASSWORD=${ctx.password}`,
    '-e', `POSTGRES_DB=${ctx.db}`,
    '-e', 'POSTGRES_HOST_AUTH_METHOD=scram-sha-256',
    ctx.image,
  ]);
  if (run.status !== 0) throw new Error(`docker run falhou\n${run.stderr}`);
  console.log(`container: ${ctx.container} criado a partir de ${ctx.image}`);
  waitForPostgres(ctx);
}

export function resetSchema(ctx) {
  psql(ctx, 'DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;', 'reset do schema');
}

export function contexto(env = process.env) {
  return {
    container: env.WYD_MIGCHECK_CONTAINER ?? 'wyd-migcheck',
    image: env.WYD_MIGCHECK_IMAGE ?? 'postgres:17-alpine',
    db: env.WYD_MIGCHECK_DB ?? 'migcheck',
    user: env.WYD_MIGCHECK_USER ?? 'postgres',
    password: env.WYD_MIGCHECK_PASSWORD ?? 'migcheck',
  };
}
