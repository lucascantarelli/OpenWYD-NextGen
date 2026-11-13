#!/usr/bin/env node
/**
 * Anotação de deploy no Grafana (DEVOPS-09) — correlacionar mudança e métrica.
 *
 * Sem isso, um pico de `WydTickP99Alto` às 15h não diz se foi carga ou deploy das 14h58. A anotação
 * põe o commit e o run na mesma linha do tempo dos dashboards, com tag `deploy` para filtrar.
 *
 * Credenciais vêm do `.env` do host (o mesmo que o compose usa), não de argumento nem de log:
 * o script só lê `GRAFANA_USER`/`GRAFANA_PASSWORD` e nunca imprime o valor.
 *
 * Falha aqui **não** derruba o deploy: anotação é observabilidade, deploy é contrato. O run fica
 * com `::warning::` quando a API recusa ou está fora — o que já é sinal suficiente.
 */
import { readFileSync } from "node:fs";

const GRAFANA = process.env.GRAFANA_URL ?? "http://127.0.0.1:3001";
const ARQUIVO_ENV = process.env.ARQUIVO_ENV ?? ".env";
const RESULTADO = process.env.RESULTADO ?? "unknown";
const COMMIT = (process.env.COMMIT ?? "").slice(0, 7);
const RUN_URL = process.env.RUN_URL ?? "";

/** Lê só as duas chaves que interessam; não exporta o resto do `.env` para o processo. */
function credenciais() {
  let texto;
  try {
    texto = readFileSync(ARQUIVO_ENV, "utf8");
  } catch {
    return null;
  }
  const mapa = {};
  for (const linha of texto.split(/\r?\n/)) {
    const m = /^\s*([A-Z_]+)\s*=\s*(.*)$/.exec(linha);
    if (!m) continue;
    mapa[m[1]] = m[2].trim().replace(/^["']|["']$/g, "");
  }
  const usuario = mapa.GRAFANA_USER || "admin";
  const senha = mapa.GRAFANA_PASSWORD;
  return senha ? { usuario, senha } : null;
}

const cred = credenciais();
if (!cred) {
  console.log(`::warning::GRAFANA_PASSWORD ausente em ${ARQUIVO_ENV} — deploy não anotado no Grafana`);
  process.exit(0);
}

const icone = RESULTADO === "success" ? "✅" : "❌";
const corpo = {
  text: `${icone} deploy local · \`${COMMIT}\` · ${RESULTADO}${RUN_URL ? `\n${RUN_URL}` : ""}`,
  tags: ["deploy", "ci", RESULTADO],
  time: Date.now(),
};

try {
  const resposta = await fetch(`${GRAFANA}/api/annotations`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Basic ${Buffer.from(`${cred.usuario}:${cred.senha}`).toString("base64")}`,
    },
    body: JSON.stringify(corpo),
    signal: AbortSignal.timeout(10_000),
  });
  const texto = await resposta.text();
  if (!resposta.ok) throw new Error(`HTTP ${resposta.status}: ${texto.slice(0, 160)}`);
  const id = JSON.parse(texto).id;
  console.log(`OK anotação criada no Grafana (id=${id ?? "?"}) para o commit ${COMMIT}`);
} catch (erro) {
  console.log(`::warning::Grafana não aceitou a anotação (${erro.message}) — deploy segue`);
}
