#!/usr/bin/env node
/**
 * Notificação de CI no Slack (DEVOPS-11) — mensagem de operação, não log de job.
 *
 * Por que existe: o resumo antigo saía de `needs.*.result`, que só diz "success/skipped". Não
 * mostrava tempo, e tempo é a informação que muda a decisão (qual job otimizar, se o run demorou
 * mais que o normal, se o deploy segurou o relógio). Aqui os tempos vêm da API do próprio run:
 * `GET /repos/{repo}/actions/runs/{id}/jobs`, com `started_at`/`completed_at` reais por job.
 *
 * Política de ruído (uma mensagem por evento, nunca duas):
 *   - `push`/`schedule`: sempre fala, verde ou vermelho;
 *   - `pull_request`: só fala quando algo falhou ou foi cancelado (verde de PR é ruído no canal).
 * Sempre escreve a aba *Summary* do run, então nada se perde quando o Slack não recebe.
 *
 * Entrada: SLACK_CI_WEBHOOK (opcional), GITHUB_TOKEN, REPOSITORIO, RUN_ID, RUN_URL, EVENTO, REF,
 * SHA, ATOR, WORKFLOW. Nada de segredo em argumento: o webhook só existe no env do job.
 */
const WEBHOOK = process.env.SLACK_CI_WEBHOOK ?? "";
const TOKEN = process.env.GITHUB_TOKEN ?? "";
const REPO = process.env.REPOSITORIO ?? "";
const RUN_ID = process.env.RUN_ID ?? "";
const RUN_URL = process.env.RUN_URL ?? "";
const EVENTO = process.env.EVENTO ?? "";
const REF = process.env.REF ?? "";
const SHA = (process.env.SHA ?? "").slice(0, 7);
const ATOR = process.env.ATOR ?? "";
const WORKFLOW = process.env.WORKFLOW ?? "CI";

const EMOJI = { success: "✅", failure: "❌", cancelled: "🚫", skipped: "⏭️", timed_out: "⏱️", neutral: "➖" };

/** 42 s / 2 min 07 s / 1 h 04 min — unidade que importa, sem casas decimais inventadas. */
function dur(segundos) {
  const s = Math.max(0, Math.round(segundos));
  if (s < 60) return `${s} s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m} min ${String(s % 60).padStart(2, "0")} s`;
  return `${Math.floor(m / 60)} h ${String(m % 60).padStart(2, "0")} min`;
}

/** Duração de cada job; job pulado (nunca iniciou) não tem tempo e não entra nas somas. */
function duracao(job) {
  if (!job.started_at || !job.completed_at) return null;
  return (Date.parse(job.completed_at) - Date.parse(job.started_at)) / 1000;
}

async function buscarJobs() {
  const jobs = [];
  for (let pagina = 1; pagina <= 5; pagina += 1) {
    const resposta = await fetch(
      `https://api.github.com/repos/${REPO}/actions/runs/${RUN_ID}/jobs?per_page=100&page=${pagina}`,
      {
        headers: {
          Authorization: `Bearer ${TOKEN}`,
          Accept: "application/vnd.github+json",
          "X-GitHub-Api-Version": "2022-11-28",
        },
        signal: AbortSignal.timeout(15_000),
      },
    );
    if (!resposta.ok) throw new Error(`API de jobs respondeu HTTP ${resposta.status}`);
    const corpo = await resposta.json();
    jobs.push(...(corpo.jobs ?? []));
    if ((corpo.jobs ?? []).length < 100) break;
  }
  return jobs;
}

const jobs = (await buscarJobs()).filter((j) => !j.name.startsWith("notify"));
const conta = (estado) => jobs.filter((j) => j.conclusion === estado).length;
const falhas = jobs.filter((j) => ["failure", "timed_out"].includes(j.conclusion));
const pulados = jobs.filter((j) => j.conclusion === "skipped" || j.conclusion === null);
const comTempo = jobs.map((j) => ({ job: j, s: duracao(j) })).filter((x) => x.s !== null);
const somaSegundos = comTempo.reduce((acc, x) => acc + x.s, 0);
const lentos = [...comTempo].sort((a, b) => b.s - a.s).slice(0, 5);

// Parede do run: do primeiro job iniciado ao último concluído (o run ainda está aberto por causa
// deste próprio job, então `updated_at` do run não serve de fim).
const inicios = comTempo.map((x) => Date.parse(x.job.started_at));
const fins = comTempo.map((x) => Date.parse(x.job.completed_at));
const parede = inicios.length ? (Math.max(...fins) - Math.min(...inicios)) / 1000 : 0;
// Fator de paralelismo: soma do tempo de todos os jobs ÷ parede. >1 é o quanto a matriz comprou.
const fator = parede > 0 ? somaSegundos / parede : 0;

const titulo = falhas.length ? "❌ CI falhou" : jobs.some((j) => j.conclusion === "cancelled") ? "🚫 CI cancelado" : "✅ CI passou";
const deploy = jobs.find((j) => j.name.startsWith("deploy"));
const deployLinha = deploy
  ? `🚀 deploy local: ${EMOJI[deploy.conclusion] ?? deploy.conclusion}${duracao(deploy) !== null ? ` (${dur(duracao(deploy))})` : ""}`
  : null;

const linhas = [
  `*${titulo} — ${REF}*  \`${SHA}\``,
  `${EVENTO} · ${ATOR}`,
  "",
  `*⏱ ${dur(parede)} de parede* · ${jobs.length} jobs · ${conta("success")} ok · ${falhas.length} falha · ${pulados.length} pulado`,
  `soma ${dur(somaSegundos)} em ${jobs.length} jobs → ${fator.toFixed(1)}× de paralelismo`,
];

if (lentos.length) linhas.push(`*🐢 mais lentos:* ${lentos.map((x) => `${x.job.name.split(" / ").pop()} ${dur(x.s)}`).join(" · ")}`);

if (falhas.length) {
  linhas.push("", "*❌ falhas:*");
  for (const job of falhas) {
    const tempo = duracao(job);
    linhas.push(`• ${job.name}${tempo !== null ? ` — ${dur(tempo)}` : ""} <${job.html_url}|logs>`);
  }
}
if (deployLinha) linhas.push("", deployLinha);

const puladosNomes = pulados.filter((j) => j.conclusion === "skipped").map((j) => j.name.split(" / ").pop());
if (puladosNomes.length) linhas.push(`⏭️ pulados: ${[...new Set(puladosNomes)].join(", ")}`);

linhas.push("", `<${RUN_URL}|ver run> · <https://github.com/${REPO}/commit/${process.env.SHA ?? ""}|código>`);
const mensagem = linhas.join("\n");

// Aba Summary: existe com ou sem webhook, então o run sempre conta a própria história.
const resumo = [
  `### ${titulo} — ${REF}`,
  "",
  `| métrica | valor |`,
  `|---|---|`,
  `| parede | ${dur(parede)} |`,
  `| soma dos jobs | ${dur(somaSegundos)} (${fator.toFixed(1)}× paralelismo) |`,
  `| ok / falha / pulado | ${conta("success")} / ${falhas.length} / ${pulados.length} |`,
  `| commit | \`${SHA}\` |`,
  "",
  ...(lentos.length ? ["**Mais lentos:** " + lentos.map((x) => `${x.job.name} (${dur(x.s)})`).join(", ")] : []),
  ...(falhas.length ? ["", "**Falhas:** " + falhas.map((j) => `[${j.name}](${j.html_url})`).join(", ")] : []),
  "",
  `[ver run](${RUN_URL})`,
].join("\n");
if (process.env.GITHUB_STEP_SUMMARY) {
  const { appendFileSync } = await import("node:fs");
  appendFileSync(process.env.GITHUB_STEP_SUMMARY, `${resumo}\n`);
}

console.log(mensagem.replace(/\*/g, "").replace(/<[^|>]+\|([^>]+)>/g, "$1"));

const avisarNoSlack = EVENTO !== "pull_request" || falhas.length > 0 || jobs.some((j) => j.conclusion === "cancelled");
if (!avisarNoSlack) {
  console.log("::notice::PR sem falha — mensagem não vai ao Slack (política de ruído); o resumo está na aba Summary");
} else if (!WEBHOOK) {
  console.log("::warning::SLACK_CI_WEBHOOK ausente — nada enviado ao canal de CI (docs/rust/10-devops-ci.md §11)");
} else {
  const resposta = await fetch(WEBHOOK, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ text: mensagem, unfurl_links: false, unfurl_media: false }),
    signal: AbortSignal.timeout(15_000),
  });
  if (!resposta.ok) {
    console.log(`::error::Slack recusou a mensagem (HTTP ${resposta.status}) — confira o webhook do canal de CI`);
    process.exit(1);
  }
  console.log("mensagem enviada ao canal de CI");
}
