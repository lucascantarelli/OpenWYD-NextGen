#!/usr/bin/env node
/**
 * Prova de rota e de entrega da notificação, do DISPARO à RESOLUÇÃO (DEVOPS-02).
 *
 * O que é provado aqui:
 *   1. ROTA — `amtool config routes test` mostra que um alerta crítico vai para o
 *      **destino real** (`slack` ou `email`, conforme `ALERTMANAGER_DESTINO`) e que só o
 *      caminho de prova (`prova=local`) vai para o receptor local;
 *   2. DISPARO — um alerta injetado percorre a rota e é entregue;
 *   3. RESOLUÇÃO — o mesmo alerta é resolvido pela API e o Alertmanager envia a
 *      notificação de `resolved` (é isso que fecha o ciclo: sem essa mensagem o canal
 *      fica com um alarme que "nunca acabou").
 *
 * Como a entrega é medida:
 *   - externo (Slack/e-mail real): contador `alertmanager_notifications_total` sobe sem
 *     `failed_total` correspondente, uma vez no disparo e outra na resolução;
 *   - bootstrap (sem `SLACK_API_URL`): o payload que chegaria ao Slack é registrado pelo
 *     `alert-sink`, então o script confere `status=firing` e depois `status=resolved`.
 *
 * O que NÃO é provado aqui: a regra disparar/resolver sozinha (`promtool test rules`) e o
 * Prometheus → Alertmanager (`WydWatchdog` em `make monitoring-check`).
 *
 * Uso: `node tools/dev/alerts-prova.mjs` (ou `make monitoring-alertas`), com a stack no ar.
 */
import { spawnSync } from "node:child_process";

const CONFIG = "/tmp/wyd-secrets/alertmanager.yml";
const LIMITE_MS = 90_000;
// A resolução só é despachada no `group_interval` do grupo (1m neste projeto).
const LIMITE_RESOLUCAO_MS = 150_000;

const compose = (args, opcoes = {}) =>
  spawnSync("docker", ["compose", ...args], { encoding: "utf8", timeout: 60000, ...opcoes });

function falhar(mensagem) {
  console.error(`FALHOU: ${mensagem}`);
  process.exit(1);
}

function exigirStack() {
  const ps = compose(["ps", "--status", "running", "--services"]);
  if (ps.status !== 0) falhar(`docker compose ps saiu com status ${ps.status}`);
  const ativos = new Set((ps.stdout ?? "").split(/\r?\n/).filter(Boolean));
  for (const servico of ["alertmanager", "alert-sink"]) {
    if (!ativos.has(servico)) falhar(`${servico} não está rodando (make stack-up)`);
  }
}

/** Receiver que o Alertmanager escolheria para estes labels. */
function rota(...labels) {
  const r = compose([
    "exec", "-T", "alertmanager", "amtool", "config", "routes", "test",
    `--config.file=${CONFIG}`, ...labels,
  ]);
  if (r.status !== 0) falhar(`amtool config routes test saiu com status ${r.status}: ${r.stderr?.trim()}`);
  return (r.stdout ?? "").trim().split(/\r?\n/).filter(Boolean).at(-1) ?? "";
}

function injetar(nome, ...labels) {
  const r = compose([
    "exec", "-T", "alertmanager", "amtool", "alert", "add",
    `alertname=${nome}`, ...labels, "--alertmanager.url=http://127.0.0.1:9093",
  ]);
  if (r.status !== 0) falhar(`amtool alert add saiu com status ${r.status}: ${r.stderr?.trim()}`);
}

/** Resolve o alerta pela API (mesmos labels do disparo, `endsAt` no passado). */
function resolver(nome, labels = {}) {
  const corpo = JSON.stringify([{ labels: { alertname: nome, ...labels }, endsAt: new Date(Date.now() - 1000).toISOString() }]);
  const r = compose(
    [
      "exec", "-T", "alertmanager", "wget", "-q", "-O", "/dev/null",
      "--header=Content-Type: application/json",
      `--post-data=${corpo}`,
      "http://127.0.0.1:9093/api/v2/alerts",
    ],
  );
  if (r.status !== 0) falhar(`resolver ${nome} falhou (status ${r.status}): ${r.stderr?.trim()}`);
}

function registros() {
  const r = compose(["exec", "-T", "alert-sink", "cat", "/var/log/wyd-alerts/alertas.jsonl"]);
  if (r.status !== 0) return [];
  return (r.stdout ?? "")
    .split(/\r?\n/)
    .filter(Boolean)
    .map((linha) => {
      try {
        return JSON.parse(linha);
      } catch {
        return null;
      }
    })
    .filter(Boolean);
}

/**
 * O destino é externo (Slack real / e-mail) ou está em bootstrap para o receptor local?
 * Lido do container, não de `process.env`: o `.env` é lido pelo compose, não por este
 * processo, então `process.env.SLACK_API_URL` ficaria sempre vazio.
 */
function destinoExterno() {
  if (destino === "email") return true;
  const r = compose([
    "exec", "-T", "alertmanager", "sh", "-c",
    // Bootstrap é o único destino apontando para o `alert-sink`; qualquer outra URL é
    // destino externo (Slack real, proxy ou webhook compatível).
    'grep -q "alert-sink" /tmp/wyd-secrets/slack_api_url 2>/dev/null && echo bootstrap || echo externo',
  ]);
  return (r.stdout ?? "").trim() === "externo";
}

/** Contadores de notificação do Alertmanager por integração (prova de entrega). */
function notificacoesDeSlack() {
  const r = compose(["exec", "-T", "alertmanager", "wget", "-qO-", "http://127.0.0.1:9093/metrics"]);
  const texto = r.stdout ?? "";
  const total = /^alertmanager_notifications_total\{integration="slack"\} (\d+)/m.exec(texto)?.[1];
  const falhas = [
    ...texto.matchAll(/^alertmanager_notifications_failed_total\{integration="slack"[^}]*\} (\d+)/gm),
  ].reduce((soma, m) => soma + Number(m[1]), 0);
  return { total: total === undefined ? NaN : Number(total), falhas };
}

const espera = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

/** Espera uma nova notificação no destino externo (contador) ou no sink (payload). */
async function esperarNotificacao({ base, limiteMs, alvo, rotulo }) {
  for (const inicio = Date.now(); Date.now() - inicio < limiteMs; ) {
    if (externo) {
      const agora = notificacoesDeSlack();
      if (agora.falhas > base.falhas) {
        falhar(
          `o destino externo recusou a notificação de ${rotulo} (failed_total ${base.falhas} → ` +
            `${agora.falhas}). Confira SLACK_API_URL/credencial e \`docker compose logs alertmanager\`.`,
        );
      }
      if (agora.total > base.total) return { externo: true, contador: agora, entregues: agora.total - base.total };
    } else {
      const registro = registros().find(
        (r) => JSON.stringify(r.payload ?? {}).includes(rotulo) && (r.payload?.status ?? "") === alvo,
      );
      if (registro) return { externo: false, registro };
    }
    await espera(2000);
  }
  falhar(
    `nenhuma notificação de ${rotulo} (${alvo}) saiu em ${Math.round(limiteMs / 1000)}s. ` +
      `Veja \`docker compose logs --tail=50 alertmanager\` (config: ${CONFIG}).`,
  );
}

// ---------------------------------------------------------------------------
exigirStack();

const destino = rota("alertname=WydTickP99Alto", "severity=critical");
if (!["slack", "email"].includes(destino)) {
  falhar(`alerta crítico foi para "${destino}" — o destino real (slack/email) não está ligado`);
}
console.log(`OK rota: alerta crítico → ${destino} (destino real)`);

const destinoProva = rota("alertname=WydProva", "severity=info", "prova=local");
if (destinoProva !== "local-sink") {
  falhar(`caminho de prova foi para "${destinoProva}" em vez de local-sink`);
}
console.log("OK rota: caminho de prova (prova=local) → local-sink");

const externo = destinoExterno();
const prova = `WydProvaRota-${Date.now()}`;
const real = `WydEntregaReal-${Date.now()}`;
const labelsReais = { severity: "warning" };

// 1) caminho de prova: payload de webhook (`alerts`), independente do destino real.
injetar(prova, "severity=info", "prova=local");
let webhook = null;
for (const inicio = Date.now(); Date.now() - inicio < LIMITE_MS; ) {
  webhook = registros().find((r) => JSON.stringify(r.payload ?? {}).includes(prova));
  if (webhook) break;
  await espera(2000);
}
if (!webhook) falhar(`alerta de prova ${prova} não chegou ao local-sink em ${LIMITE_MS / 1000}s`);
if (!Array.isArray(webhook.payload?.alerts)) falhar("payload do caminho de prova sem `alerts`");
console.log(`OK entrega (prova): local-sink sequência ${webhook.sequencia}`);

// 2) DISPARO no destino real.
const antes = externo ? notificacoesDeSlack() : { total: NaN, falhas: 0 };
if (externo && Number.isNaN(antes.total)) falhar("não consegui ler `alertmanager_notifications_total` em /metrics");

injetar(real, ...Object.entries(labelsReais).map(([k, v]) => `${k}=${v}`));
const disparo = await esperarNotificacao({ base: antes, limiteMs: LIMITE_MS, alvo: "firing", rotulo: real });
if (disparo.externo) {
  console.log(
    `OK entrega (disparo): ${destino} externo aceitou ${disparo.entregues} notificação(ões) — ` +
      `failed_total=${disparo.contador.falhas}`,
  );
} else {
  const titulo = (disparo.registro.payload?.attachments ?? []).map((a) => a.title ?? "").join(" ");
  console.log(`OK entrega (disparo): bootstrap registrou o payload de Slack — ${titulo}`);
}

// 3) RESOLUÇÃO no destino real.
resolver(real, labelsReais);
const baseResolucao = externo ? notificacoesDeSlack() : { total: NaN, falhas: 0 };
const resolucao = await esperarNotificacao({
  base: baseResolucao,
  limiteMs: LIMITE_RESOLUCAO_MS,
  alvo: "resolved",
  rotulo: real,
});
if (resolucao.externo) {
  console.log(
    `OK entrega (resolução): ${destino} externo aceitou a notificação de RESOLVED — ` +
      `failed_total=${resolucao.contador.falhas}`,
  );
} else {
  const anexo = (resolucao.registro.payload?.attachments ?? [])[0] ?? {};
  console.log(`OK entrega (resolução): bootstrap registrou RESOLVED — ${anexo.title ?? "sem título"}`);
  if (!String(anexo.title ?? "").includes("RESOLVED")) {
    falhar("a mensagem de resolução não traz RESOLVED no título");
  }
}

if (externo) {
  console.log(
    "   Confirme as duas mensagens (disparo e resolução) no canal/caixa do destino.",
  );
}
