#!/usr/bin/env node
/**
 * Anúncio de release no canal dedicado (DEVOPS-10).
 *
 * Release é anúncio, não build: entra em `#wyd-releases`, com o que interessa a quem consome —
 * quais binários saíram, os checksums e onde estão as notas. Falha de CI continua no `#wyd-ci`;
 * misturar os dois faz release passar batido no meio de run vermelho.
 *
 * Segredo ausente = run verde com aviso (o artefato no GitHub continua sendo a fonte de verdade).
 * Segredo presente e Slack recusando = job falha, porque aí a promessa de anunciar foi quebrada.
 */
import { readdirSync } from "node:fs";

const WEBHOOK = process.env.SLACK_RELEASES_WEBHOOK ?? "";
const REPO = process.env.REPO ?? "";
const TAG = process.env.TAG ?? "";
const RUN_URL = process.env.RUN_URL ?? "";
const DIR = process.env.DIR_DIST ?? "dist";

function assets() {
  try {
    return readdirSync(DIR).sort();
  } catch {
    return [];
  }
}

const lista = assets();
const binarios = lista.filter((a) => !a.startsWith("SHA256"));
const temChecksums = lista.some((a) => a.startsWith("SHA256SUMS"));

const mensagem = [
  `*:package: Release ${TAG} publicada* — ${REPO}`,
  `${binarios.length} binário(s)${temChecksums ? " + `SHA256SUMS`" : ""} · SBOM por componente (CycloneDX)`,
  binarios.length ? `\`${binarios.join("`, `")}\`` : null,
  `<https://github.com/${REPO}/releases/tag/${TAG}|notas da release> · <${RUN_URL}|ver run>`,
  "_Rollback e procedimento: docs/rust/02-execucao.md §6_",
]
  .filter(Boolean)
  .join("\n");

console.log(mensagem.replace(/\*/g, "").replace(/<([^|>]+)\|([^>]+)>/g, "$2 ($1)"));

if (!WEBHOOK) {
  console.log("::warning::SLACK_RELEASES_WEBHOOK ausente — release não anunciada em #wyd-releases");
  process.exit(0);
}

const resposta = await fetch(WEBHOOK, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ text: mensagem, unfurl_links: false }),
  signal: AbortSignal.timeout(15_000),
});
if (!resposta.ok) {
  console.log(`::error::Slack recusou o anúncio de release (HTTP ${resposta.status})`);
  process.exit(1);
}
console.log("release anunciada em #wyd-releases");
