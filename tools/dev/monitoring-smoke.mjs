// Smoke da stack de observabilidade (QA-08, ampliado pela DEVOPS-02) — ferramenta de
// dev, sem runtime de jogo.
//
// Verifica o que um humano verificaria a mão, com exit code de verdade:
//   1. Prometheus pronto (`/-/ready`);
//   2. alvos `prometheus`, `node`, `wyd-gateway`, `alertmanager` e `wyd-portal` em `up`
//      (este último é a sonda do BFF, que não publica `/metrics`);
//   3. as regras de alerta carregadas;
//   4. Grafana saudável, datasource Prometheus provisionada e dashboard `wyd-saude`;
//   5. Alertmanager pronto e com receiver configurado;
//   6. o `WydWatchdog` visível no Alertmanager — a prova de que a ligação
//      Prometheus → Alertmanager existe de fato (regra em `firing` vira alerta lá).
//
// A senha do Grafana vem do ambiente ou do `.env` (mesma precedência do compose), e a
// origem usada é impressa: o smoke não depende de um default escondido no código.
//
// Uso: node tools/dev/monitoring-smoke.mjs [--prometheus 9090] [--grafana 3001]
//                                        [--alertmanager 9093]
import { readFileSync } from "node:fs";
import { request } from "node:http";

const args = process.argv.slice(2);
const opt = (nome, padrao) => {
  const i = args.indexOf(`--${nome}`);
  return i >= 0 && args[i + 1] ? args[i + 1] : padrao;
};
const prom = Number(opt("prometheus", process.env.PROMETHEUS_PORT || 9090));
const graf = Number(opt("grafana", process.env.GRAFANA_PORT || 3001));
const alertmanager = Number(opt("alertmanager", process.env.ALERTMANAGER_PORT || 9093));

// Precedência idêntica à do compose (`${GRAFANA_PASSWORD:-troque-aqui-dev}`):
// ambiente → `.env` do host → default documentado. A origem é publicada para que
// ninguém precise adivinhar de onde veio a credencial.
function senhaDoGrafana() {
  if (process.env.GRAFANA_PASSWORD) return { senha: process.env.GRAFANA_PASSWORD, origem: "ambiente" };
  try {
    const linha = readFileSync(".env", "utf8")
      .split(/\r?\n/)
      .find((l) => l.startsWith("GRAFANA_PASSWORD="));
    if (linha) {
      const valor = linha.slice("GRAFANA_PASSWORD=".length).trim().replace(/^["']|["']$/g, "");
      if (valor) return { senha: valor, origem: ".env" };
    }
  } catch {
    // Sem `.env`: segue para o default do compose.
  }
  return { senha: "troque-aqui-dev", origem: "default do compose" };
}

const { senha, origem } = senhaDoGrafana();
const usuario = process.env.GRAFANA_USER || "admin";

let falhas = 0;
const ok = (texto) => console.log(`OK    ${texto}`);
const falha = (texto) => {
  falhas += 1;
  console.log(`FALHA ${texto}`);
};

function pega(url, { auth = false } = {}) {
  return new Promise((resolve) => {
    const headers = auth
      ? { authorization: `Basic ${Buffer.from(`${usuario}:${senha}`).toString("base64")}` }
      : {};
    const req = request(url, { headers, timeout: 5000 }, (res) => {
      let corpo = "";
      res.on("data", (parte) => (corpo += parte));
      res.on("end", () => resolve({ status: res.statusCode, corpo }));
    });
    req.on("timeout", () => req.destroy(new Error("timeout")));
    req.on("error", (e) => resolve({ status: 0, corpo: e.message }));
    req.end();
  });
}

const leJson = (corpo) => {
  try {
    return JSON.parse(corpo);
  } catch {
    return null;
  }
};

async function main() {
  const ready = await pega(`http://127.0.0.1:${prom}/-/ready`);
  ready.status === 200 ? ok("prometheus /-/ready 200") : falha(`prometheus /-/ready ${ready.status}`);

  const alvos = await pega(`http://127.0.0.1:${prom}/api/v1/targets`);
  const dadosAlvos = leJson(alvos.corpo)?.data?.activeTargets;
  if (!Array.isArray(dadosAlvos)) {
    falha(`/api/v1/targets ilegível: ${alvos.corpo.slice(0, 80)}`);
  } else {
    for (const job of ["prometheus", "node", "wyd-gateway", "alertmanager", "wyd-portal"]) {
      const alvo = dadosAlvos.find((t) => t.labels.job === job);
      if (alvo && alvo.health === "up") ok(`alvo ${job} up (${alvo.scrapeUrl})`);
      else falha(`alvo ${job} ausente ou fora (${alvo ? alvo.health : "sem alvo"})`);
    }
    const motor = dadosAlvos.filter((t) => t.labels.job === "wyd-server");
    if (motor.length === 0) {
      falha("alvos do motor (wyd-server): nenhum (o binario existe desde a ADR-0022)");
    } else {
      for (const alvo of motor) {
        alvo.health === "up"
          ? ok(`alvo wyd-server canal ${alvo.labels.canal ?? "?"} up (${alvo.scrapeUrl})`)
          : falha(`alvo wyd-server canal ${alvo.labels.canal ?? "?"} fora (${alvo.health})`);
      }
    }
  }

  const regras = await pega(`http://127.0.0.1:${prom}/api/v1/rules?type=alert`);
  const grupos = leJson(regras.corpo)?.data?.groups;
  if (!Array.isArray(grupos)) {
    falha(`/api/v1/rules ilegível: ${regras.corpo.slice(0, 80)}`);
  } else {
    const nomes = grupos.flatMap((g) => g.rules.map((r) => r.name));
    // Host, portal e vigia entraram na DEVOPS-02: menos que isso é regra sumindo.
    nomes.length >= 17 ? ok(`regras carregadas: ${nomes.length}`) : falha(`regras: ${nomes.length} (esperado >= 17)`);
    for (const exigida of ["WydPortalForaDoAr", "WydDiscoQuaseCheio", "WydWatchdog"]) {
      nomes.includes(exigida) ? ok(`regra ${exigida} presente`) : falha(`regra ${exigida} ausente`);
    }
  }

  const saude = await pega(`http://127.0.0.1:${graf}/api/health`);
  saude.status === 200 && /"database"\s*:\s*"ok"/.test(saude.corpo)
    ? ok("grafana /api/health ok")
    : falha(`grafana /api/health ${saude.status} ${saude.corpo.slice(0, 80)}`);

  console.log(`      senha do Grafana: ${origem}`);
  const fontes = await pega(`http://127.0.0.1:${graf}/api/datasources`, { auth: true });
  const listaFontes = leJson(fontes.corpo);
  if (!Array.isArray(listaFontes)) {
    falha(
      `datasource prometheus: /api/datasources status ${fontes.status} — ` +
        "defina GRAFANA_PASSWORD no ambiente ou no .env",
    );
  } else if (listaFontes.some((d) => d.type === "prometheus" && d.uid === "prometheus")) {
    ok("datasource prometheus provisionada (uid=prometheus)");
  } else {
    falha("datasource prometheus ausente");
  }

  const dash = await pega(`http://127.0.0.1:${graf}/api/dashboards/uid/wyd-saude`, { auth: true });
  const dadosDash = leJson(dash.corpo);
  if (dadosDash?.dashboard?.title) {
    ok(`dashboard provisionado: ${dadosDash.dashboard.title} (${dadosDash.dashboard.panels.length} painéis)`);
  } else {
    falha(`dashboard wyd-saude ausente (status ${dash.status})`);
  }

  const amReady = await pega(`http://127.0.0.1:${alertmanager}/-/ready`);
  amReady.status === 200 ? ok("alertmanager /-/ready 200") : falha(`alertmanager /-/ready ${amReady.status}`);

  const amStatus = await pega(`http://127.0.0.1:${alertmanager}/api/v2/status`);
  const status = leJson(amStatus.corpo);
  if (!status) {
    falha(`alertmanager /api/v2/status ${amStatus.status}`);
  } else {
    const config = String(status.config?.original ?? "");
    const versao = status.versionInfo?.version ?? status.version ?? "?";
    // `config.original` é a config **renderizada** que o Alertmanager carregou.
    const destino = /^\s*receiver:\s*(\S+)\s*$/m.exec(config)?.[1] ?? "";
    if (!config.includes("receivers")) {
      falha("alertmanager sem bloco `receivers` na configuração");
    } else if (!destino) {
      falha("alertmanager sem `receiver` na rota padrão");
    } else if (destino === "local-sink") {
      falha("rota padrão aponta para o receptor local — o destino real (slack/email) não está ligado");
    } else {
      ok(`alertmanager ${versao}, rota padrão → ${destino} (destino real)`);
    }
  }

  const amAlertas = await pega(`http://127.0.0.1:${alertmanager}/api/v2/alerts`);
  const ativos = leJson(amAlertas.corpo);
  if (!Array.isArray(ativos)) {
    falha(`alertmanager /api/v2/alerts ${amAlertas.status}`);
  } else {
    const watchdog = ativos.find((a) => a.labels?.alertname === "WydWatchdog");
    watchdog
      ? ok(`WydWatchdog no Alertmanager (state=${watchdog.status?.state}) — Prometheus → Alertmanager ligado`)
      : falha("WydWatchdog ausente no Alertmanager: a ligação Prometheus → Alertmanager não chegou");
  }

  console.log(falhas === 0 ? "smoke de monitoramento: tudo verde" : `smoke de monitoramento: ${falhas} falha(s)`);
  process.exit(falhas === 0 ? 0 : 1);
}

main();
