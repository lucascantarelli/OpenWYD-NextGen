#!/usr/bin/env node
/**
 * Provas de contrato do deploy local (DEVOPS-09) — o deploy só é "feito" se as quatro passarem.
 *
 * As mesmas provas de `make local-rebuild`, mas com o schema DERIVADO de `SCHEMA_VERSION`
 * (DEVOPS-04/B9): fixar `"schema":49` reprovava deploy saudável com 50 migrations.
 *
 * Provas:
 *   1. portal `/login` responde 200;
 *   2. portal `/entregas` responde 307 (rota protegida redireciona, não 500);
 *   3. gateway `/health` responde 200 e o `schema` é o `SCHEMA_VERSION` do código;
 *   4. portal `/api/health` responde 200 (BFF server-side falando com o gateway).
 *
 * Uso: `node tools/dev/deploy-provas.mjs` (host) ou dentro do runner `self-hosted,wyd-local`.
 */
import { readFileSync } from "node:fs";

const PORTAL = process.env.WYD_PORTAL_URL ?? "http://127.0.0.1:3000";
const GATEWAY = process.env.WYD_GATEWAY_URL ?? "http://127.0.0.1:7600";
const TENTATIVAS = Number(process.env.WYD_DEPLOY_TENTATIVAS ?? 20);
const ESPERA_MS = Number(process.env.WYD_DEPLOY_ESPERA_MS ?? 3000);

/** `SCHEMA_VERSION` do código — a fonte, não um número copiado no script. */
function schemaVersion() {
  const fonte = readFileSync("crates/wyd-db/src/lib.rs", "utf8");
  const valor = /pub const SCHEMA_VERSION:\s*i(?:32|64)\s*=\s*(\d+)/.exec(fonte)?.[1];
  if (!valor) throw new Error("SCHEMA_VERSION não encontrado em crates/wyd-db/src/lib.rs");
  return Number(valor);
}

const espera = (ms) => new Promise((r) => setTimeout(r, ms));

async function pegar(url, tentativas = TENTATIVAS) {
  let ultimo = "sem resposta";
  for (let i = 0; i < tentativas; i += 1) {
    try {
      const resposta = await fetch(url, { redirect: "manual" });
      return { status: resposta.status, corpo: await resposta.text() };
    } catch (erro) {
      ultimo = erro.message;
      await espera(ESPERA_MS);
    }
  }
  throw new Error(`${url} não respondeu em ${tentativas} tentativas (${ultimo})`);
}

function exigir(condicao, mensagem) {
  if (!condicao) {
    console.error(`FALHOU: ${mensagem}`);
    process.exit(1);
  }
  console.log(`OK ${mensagem}`);
}

const schema = schemaVersion();
console.log(`schema esperado (SCHEMA_VERSION): ${schema}`);

const login = await pegar(`${PORTAL}/login`);
exigir(login.status === 200, `portal /login → ${login.status}`);

const entregas = await pegar(`${PORTAL}/entregas`);
exigir(entregas.status === 307, `portal /entregas → ${entregas.status} (esperado 307 sem sessão)`);

const health = await pegar(`${GATEWAY}/health`);
exigir(health.status === 200, `gateway /health → ${health.status}`);
let corpo;
try {
  corpo = JSON.parse(health.corpo);
} catch {
  console.error(`FALHOU: gateway /health não devolveu JSON: ${health.corpo.slice(0, 120)}`);
  process.exit(1);
}
exigir(
  Number(corpo.schema) === schema,
  `gateway /health schema=${corpo.schema} == SCHEMA_VERSION=${schema}`,
);

const apiHealth = await pegar(`${PORTAL}/api/health`);
exigir(apiHealth.status === 200, `portal /api/health → ${apiHealth.status} (BFF → gateway)`);

console.log("deploy local provado: quatro provas verdes");
