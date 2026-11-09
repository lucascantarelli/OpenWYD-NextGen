#!/usr/bin/env node
/**
 * E2E (FASE-07-4): provisiona banco e containers efêmeros, roda o Playwright e
 * derruba tudo no `finally`.
 *
 * Regras:
 *
 * 1. **Nunca toca o banco `wyd`** (dev/produção). O alvo é `wyd_e2e`, recriado
 *    do zero a cada execução e migrado pela série real do `crates/wyd-db`.
 * 2. O staff nasce no seed do gateway (`WYD_SEED_STAFF=1`); o jogador e os dois
 *    personagens do roteiro são semeados aqui.
 * 3. Gateway e canal saem de imagens já construídas (`wyd/gateway:dev`,
 *    `wyd/server:dev`) na rede `wyd`; o canal é quem drena o ouro.
 * 4. Sai com o código do Playwright: spec que falha derruba o script.
 *
 * Uso (na pasta `wyd-portal`):
 *   npm run test:e2e            # suíte completa
 *   npm run test:e2e -- --grep login
 */
import { spawnSync } from "node:child_process";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "..", "..");
const PORTAL = path.join(ROOT, "wyd-portal");
const DB_NAME = "wyd_e2e";
const GATEWAY = "wyd-gateway-e2e";
const GAME = "wyd-server-e2e";
const PORTAL_CONTAINER = "wyd-portal-e2e";
const GATEWAY_PORT = 7601;
// 3110: a 3000 é do compose (teste manual). O E2E usa a MESMA imagem do portal.
const PORTAL_PORT = 3110;
const TOKEN = "e2e-token-do-gateway";
const ADMIN_USER = "admin";
const ADMIN_PASS = "e2e-admin-pass-123";

/**
 * DSN do E2E: role e banco **dedicados**, criados pelo superusuário da stack.
 *
 * Derivar credencial do `.env`/container divergiu do pgdata já inicializado e o
 * gateway E2E não autenticava; as imagens Rust são distroless (sem `printenv`).
 * O role `wyd_e2e` tem senha fixa de teste e é dono do banco `wyd_e2e` — nunca
 * enxerga o banco `wyd`.
 */
const E2E_ROLE = "wyd_e2e";
const E2E_PASSWORD = "wyd_e2e_local";
const E2E_DSN = `postgres://${E2E_ROLE}:${E2E_PASSWORD}@db:5432/${DB_NAME}`;

/** psql por TCP contra o Postgres da stack, de dentro da rede `wyd`. */
function psqlTcp(sql) {
  return run(
    "docker",
    [
      "run", "--rm", "-i", "--network", "wyd", "postgres:17-alpine",
      "psql", E2E_DSN,
      "-v", "ON_ERROR_STOP=1", "-q", "-t", "-A", "-f", "-",
    ],
    { input: sql },
  ).stdout.trim();
}

function run(cmd, args, options = {}) {
  const result = spawnSync(cmd, args, { encoding: "utf8", ...options });
  if (result.status !== 0) {
    const detail = `${result.stdout ?? ""}${result.stderr ?? ""}`.trim();
    throw new Error(`${cmd} ${args.join(" ")} falhou (exit ${result.status})\n${detail}`);
  }
  return result;
}

/** Usuário do Postgres da stack, lido do container (sem `container_name` fixo). */
let cachedStackUser;
function stackUser() {
  if (cachedStackUser) return cachedStackUser;
  const probe = spawnSync(
    "docker",
    ["compose", "exec", "-T", "db", "printenv", "POSTGRES_USER"],
    { cwd: ROOT, encoding: "utf8" },
  );
  cachedStackUser =
    probe.status === 0 && probe.stdout.trim()
      ? probe.stdout.trim()
      : (process.env.POSTGRES_USER ?? "wyd");
  return cachedStackUser;
}

/** psql na stack via compose (o container agora tem nome do projeto, não fixo). */
function psql(database, sql) {
  return run(
    "docker",
    [
      "compose",
      "exec",
      "-T",
      "db",
      "psql",
      "-U",
      stackUser(),
      "-d",
      database,
      "-v",
      "ON_ERROR_STOP=1",
      "-q",
      "-t",
      "-A",
      "-f",
      "-",
    ],
    { cwd: ROOT, input: sql },
  ).stdout.trim();
}

function removeContainer(name) {
  spawnSync("docker", ["rm", "-f", name], { encoding: "utf8" });
}

async function waitHttp(url, label) {
  for (let i = 0; i < 120; i += 1) {
    try {
      const res = await fetch(url);
      if (res.ok) return;
    } catch {
      // ainda subindo
    }
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  throw new Error(`${label}: sem resposta saudável em ${url} após 60 s`);
}

async function provision() {
  removeContainer(PORTAL_CONTAINER);
  removeContainer(GATEWAY);
  removeContainer(GAME);

  psql(
    "postgres",
    `SELECT pg_terminate_backend(pid) FROM pg_stat_activity
       WHERE datname = '${DB_NAME}' AND pid <> pg_backend_pid();`,
  );
  psql("postgres", `DROP DATABASE IF EXISTS ${DB_NAME};`);
  psql("postgres", `DROP ROLE IF EXISTS ${E2E_ROLE};`);
  psql("postgres", `CREATE ROLE ${E2E_ROLE} LOGIN PASSWORD '${E2E_PASSWORD}';`);
  psql("postgres", `CREATE DATABASE ${DB_NAME} OWNER ${E2E_ROLE};`);

  // Sem passo de migration no teste: o gateway abaixo aplica a série no próprio
  // boot (ADR-0024), no banco efêmero do role dedicado. A seed vem depois.

  const dsn = E2E_DSN;
  run("docker", [
    "run", "-d", "--rm", "--name", GATEWAY, "--network", "wyd",
    "-p", `127.0.0.1:${GATEWAY_PORT}:7600`,
    "-e", `DATABASE_URL=${dsn}`,
    "-e", "WYD_GATEWAY_BIND=0.0.0.0:7600",
    "-e", `WEB_API_TOKEN=${TOKEN}`,
    "-e", "WYD_SEED_STAFF=1",
    "-e", `PORTAL_ADMIN_USER=${ADMIN_USER}`,
    "-e", `PORTAL_ADMIN_PASSWORD=${ADMIN_PASS}`,
    "wyd/gateway:dev",
  ]);
  await waitHttp(`http://127.0.0.1:${GATEWAY_PORT}/health`, GATEWAY);
  console.log(`gateway E2E: saudável em 127.0.0.1:${GATEWAY_PORT}`);

  // Só depois do boot: o gateway é quem migra o banco efêmero (ADR-0024), então
  // `accounts`/`characters` já existem quando a seed do roteiro roda.
  psqlTcp(
    `INSERT INTO accounts (login, password_hash, role) VALUES ('e2e-player', 'x', 'player');
     INSERT INTO characters (account_id, slot, name, class, level, hp, mp, hp_max, mp_max, gold)
       SELECT id, 0, 'E2E Hero', 0, 50, 500, 50, 500, 50, 0 FROM accounts WHERE login = 'e2e-player';
     INSERT INTO characters (account_id, slot, name, class, level, hp, mp, hp_max, mp_max, gold)
       SELECT id, 1, 'E2E Gêmeo', 0, 50, 500, 50, 500, 50, 0 FROM accounts WHERE login = 'e2e-player';`,
  );

  run("docker", [
    "run", "-d", "--rm", "--name", GAME, "--network", "wyd",
    "-e", `DATABASE_URL=${dsn}`,
    "-e", "WYD_SERVER_CHANNEL=9",
    "-e", "WYD_SERVER_CPSOCK_BIND=0.0.0.0:8281",
    "-e", "WYD_SERVER_STATUS_BIND=0.0.0.0:8080",
    "-e", "WYD_SERVER_METRICS_BIND=0.0.0.0:9898",
    "wyd/server:dev",
  ]);
  // O dreno do canal roda a cada 1 s; dois segundos cobrem boot + primeira volta.
  await new Promise((resolve) => setTimeout(resolve, 2000));

  // Portal do compose (imagem `wyd/portal:dev`), apontando ao gateway efêmero
  // pelo DNS interno da rede `wyd`. Nada de `next dev`: sem lock e sem tocar a 3000.
  run("docker", [
    "run", "-d", "--rm", "--name", PORTAL_CONTAINER, "--network", "wyd",
    "-p", `127.0.0.1:${PORTAL_PORT}:3000`,
    "--tmpfs", "/var/log/wyd-portal",
    "-e", "NODE_ENV=production",
    "-e", "PORT=3000",
    "-e", "SESSION_SECRET=e2e-session-secret-com-32-caracteres-ok!!",
    "-e", `WEB_API_ADDR=http://${GATEWAY}:7600`,
    "-e", `WEB_API_TOKEN=${TOKEN}`,
    "-e", "PROMETHEUS_URL=http://prometheus:9090",
    "-e", "LOKI_URL=http://loki:3100",
    "wyd/portal:dev",
  ]);
  await waitHttp(`http://127.0.0.1:${PORTAL_PORT}/login`, PORTAL_CONTAINER);
  console.log(`portal E2E: saudável em 127.0.0.1:${PORTAL_PORT}`);
}

/** Remove banco/role efêmeros: E2E encerrado não deixa schema nem credencial no volume. */
function dropE2eDatabase() {
  try {
    psql(
      "postgres",
      `SELECT pg_terminate_backend(pid) FROM pg_stat_activity
         WHERE datname = '${DB_NAME}' AND pid <> pg_backend_pid();`,
    );
    psql("postgres", `DROP DATABASE IF EXISTS ${DB_NAME};`);
    psql("postgres", `DROP ROLE IF EXISTS ${E2E_ROLE};`);
  } catch (error) {
    console.error(
      `limpeza do banco E2E: ${error instanceof Error ? error.message : String(error)}`,
    );
  }
}

function teardown() {
  removeContainer(PORTAL_CONTAINER);
  removeContainer(GATEWAY);
  removeContainer(GAME);
  dropE2eDatabase();
}

async function main() {
  await provision();
  const result = spawnSync(
    "npx",
    ["playwright", "test", ...process.argv.slice(2)],
    {
      cwd: PORTAL,
      stdio: "inherit",
      // Windows: `npx` é um .cmd e só resolve via shell.
      shell: process.platform === "win32",
      env: {
        ...process.env,
        E2E_GATEWAY_URL: `http://127.0.0.1:${GATEWAY_PORT}`,
        E2E_GATEWAY_TOKEN: TOKEN,
        E2E_ADMIN_USER: ADMIN_USER,
        E2E_ADMIN_PASS: ADMIN_PASS,
      },
    },
  );
  if (result.error) {
    console.error(`playwright não iniciou: ${result.error.message}`);
  } else if (result.status !== 0) {
    console.error(`playwright: exit ${result.status ?? "desconhecido"}`);
  }
  return result.status ?? 1;
}

let code = 1;
try {
  code = await main();
} catch (error) {
  console.error(error instanceof Error ? error.message : error);
} finally {
  teardown();
}
process.exitCode = code;
