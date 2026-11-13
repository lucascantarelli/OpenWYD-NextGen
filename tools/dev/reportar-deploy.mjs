#!/usr/bin/env node
/**
 * Relato do deploy local ao Alertmanager (DEVOPS-09) — o deploy não termina em artefato.
 *
 * Antes: o job `deploy local` publicava `deploy-provas.log` e ninguém fora do GitHub sabia do
 * resultado. Agora o mesmo resultado entra na cadeia que o time já vigia:
 *   job `deploy` → `POST /api/v2/alerts` → Alertmanager → receiver da rota padrão (`#wyd-alertas`).
 *
 * Dois alertas, com nomes **estáveis** (o fingerprint do Alertmanager é o conjunto de labels; um
 * `run_id` no label criaria um alerta novo por run e a falha antiga nunca resolveria):
 *
 *   WydDeployLocalFalhou  severity=critical, sem `endsAt` — fica ativo até um deploy passar;
 *   WydDeployLocalOk      severity=info, `endsAt` em 2 min — aparece e auto-resolve, provando o
 *                         caminho inteiro sem deixar alerta pendente.
 *
 * No sucesso mandamos também `WydDeployLocalFalhou` com `endsAt` no passado: é assim que uma
 * falha anterior é resolvida (o Slack tem `send_resolved: true`).
 *
 * Entrada (env, preenchido pelo job): RESULTADO_BUILD, RESULTADO_UP, RESULTADO_PROVAS, RUN_URL,
 * COMMIT. Opcionais: ALERTMANAGER_URL (default `http://127.0.0.1:9093`) e AM_PROVA=local, que
 * adiciona o label `prova="local"` — a rota de prova do Alertmanager manda para o `local-sink` em
 * vez do canal do time. Uso local: `AM_PROVA=local RESULTADO_PROVAS=success node ...`.
 */
const ALERTMANAGER = process.env.ALERTMANAGER_URL ?? "http://127.0.0.1:9093";
const RUN_URL = process.env.RUN_URL ?? "";
const COMMIT = (process.env.COMMIT ?? "").slice(0, 7);
const AM_PROVA = process.env.AM_PROVA ?? "";

const PASSOS = [
  ["build", process.env.RESULTADO_BUILD],
  ["up", process.env.RESULTADO_UP],
  ["provas", process.env.RESULTADO_PROVAS],
];
// Ausente = o passo nem rodou (sentinela de teste humano); conta como pulado, não como falha.
const desconhecidos = PASSOS.filter(([, r]) => !r || r === "skipped");
const falhos = PASSOS.filter(([, r]) => r && r !== "success" && r !== "skipped");

const rotulo = (alertname, severity) => ({
  alertname,
  severity,
  job: "deploy-local",
  environment: "local",
  ...(AM_PROVA ? { prova: AM_PROVA } : {}),
});

const detalhe = [
  falhos.length ? `passos com falha: ${falhos.map(([n, r]) => `${n}=${r}`).join(", ")}` : null,
  desconhecidos.length ? `sem resultado: ${desconhecidos.map(([n]) => n).join(", ")}` : null,
  COMMIT ? `commit ${COMMIT}` : null,
]
  .filter(Boolean)
  .join(" · ");

const RUNBOOK = "docs/rust/06-observabilidade-runbooks.md";
const alertas = [];

if (falhos.length === 0) {
  // Resolve falha anterior (mesmo conjunto de labels) e publica o sucesso.
  alertas.push({
    labels: rotulo("WydDeployLocalFalhou", "critical"),
    annotations: { summary: "resolvido por um deploy posterior bem-sucedido", runbook: `${RUNBOOK}#wyddeploylocalfalhou` },
    startsAt: new Date(Date.now() - 1000).toISOString(),
    endsAt: new Date(Date.now() - 500).toISOString(),
  });
  alertas.push({
    labels: rotulo("WydDeployLocalOk", "info"),
    annotations: {
      summary: `deploy local concluído${detalhe ? ` (${detalhe})` : ""}`,
      runbook: `${RUNBOOK}#wyddeploylocalok`,
      ...(RUN_URL ? { run_url: RUN_URL } : {}),
    },
    startsAt: new Date().toISOString(),
    // Auto-resolve em 2 min: sinaliza sucesso sem virar alerta pendente na tela de quem opera.
    endsAt: new Date(Date.now() + 120_000).toISOString(),
  });
} else {
  alertas.push({
    labels: rotulo("WydDeployLocalFalhou", "critical"),
    annotations: {
      summary: `deploy local FALHOU${detalhe ? ` (${detalhe})` : ""}`,
      runbook: `${RUNBOOK}#wyddeploylocalfalhou`,
      ...(RUN_URL ? { run_url: RUN_URL } : {}),
    },
    startsAt: new Date().toISOString(),
  });
}

try {
  const resposta = await fetch(`${ALERTMANAGER}/api/v2/alerts`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(alertas),
    signal: AbortSignal.timeout(10_000),
  });
  const corpo = await resposta.text();
  if (!resposta.ok) throw new Error(`HTTP ${resposta.status}: ${corpo.slice(0, 200)}`);
  console.log(
    `OK Alertmanager aceitou ${alertas.length} alerta(s) para ${alertas.map((a) => a.labels.alertname).join(", ")}`,
  );
} catch (erro) {
  // O deploy é o contrato; o relato é observabilidade. Sem Alertmanager o job não fica vermelho,
  // mas o run deixa registrado que a operação ficou sem aviso.
  console.log(`::warning::não foi possível relatar ao Alertmanager (${erro.message}) — confira a stack de monitoramento`);
  if (falhos.length > 0) {
    console.log(`::error::além disso o deploy falhou e o aviso NÃO chegou ao canal de operação`);
    process.exit(1);
  }
}
