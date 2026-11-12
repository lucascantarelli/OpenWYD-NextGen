// Smoke da stack de observabilidade (QA-08) — ferramenta de dev, sem runtime de jogo.
//
// Verifica o que um humano verificaria a mão, com exit code de verdade:
//   1. Prometheus pronto (`/-/ready`);
//   2. alvos `prometheus`, `node` e `wyd-gateway` em `up` (o job `wyd-server` fica sem alvo por desenho);
//   3. as oito regras de alerta carregadas;
//   4. Grafana saudável, datasource Prometheus provisionada e dashboard `wyd-saude` presente.
//
// Uso: node tools/dev/monitoring-smoke.mjs [--prometheus 9090] [--grafana 3001]
import { request } from 'node:http';

const args = process.argv.slice(2);
const opt = (nome, padrao) => {
  const i = args.indexOf(`--${nome}`);
  return i >= 0 && args[i + 1] ? args[i + 1] : padrao;
};
const prom = Number(opt('prometheus', process.env.PROMETHEUS_PORT || 9090));
const graf = Number(opt('grafana', process.env.GRAFANA_PORT || 3001));
const usuario = process.env.GRAFANA_USER || 'admin';
const senha = process.env.GRAFANA_PASSWORD || 'troque-aqui-dev';

let falhas = 0;
const ok = (texto) => console.log(`OK    ${texto}`);
const falha = (texto) => {
  falhas += 1;
  console.log(`FALHA ${texto}`);
};

function pega(url, { auth = false } = {}) {
  return new Promise((resolve) => {
    const headers = auth
      ? { authorization: `Basic ${Buffer.from(`${usuario}:${senha}`).toString('base64')}` }
      : {};
    const req = request(url, { headers, timeout: 5000 }, (res) => {
      let corpo = '';
      res.on('data', (parte) => (corpo += parte));
      res.on('end', () => resolve({ status: res.statusCode, corpo }));
    });
    req.on('timeout', () => req.destroy(new Error('timeout')));
    req.on('error', (e) => resolve({ status: 0, corpo: e.message }));
    req.end();
  });
}

async function main() {
  const ready = await pega(`http://127.0.0.1:${prom}/-/ready`);
  ready.status === 200 ? ok('prometheus /-/ready 200') : falha(`prometheus /-/ready ${ready.status}`);

  const alvos = await pega(`http://127.0.0.1:${prom}/api/v1/targets`);
  try {
    const lista = JSON.parse(alvos.corpo).data.activeTargets;
    for (const job of ['prometheus', 'node', 'wyd-gateway']) {
      const alvo = lista.find((t) => t.labels.job === job);
      if (alvo && alvo.health === 'up') ok(`alvo ${job} up (${alvo.scrapeUrl})`);
      else falha(`alvo ${job} ausente ou fora (${alvo ? alvo.health : 'sem alvo'})`);
    }
    const motor = lista.filter((t) => t.labels.job === 'wyd-server');
    if (motor.length === 0) {
      falha('alvos do motor (wyd-server): nenhum (o binario existe desde a ADR-0022)');
    } else {
      for (const alvo of motor) {
        alvo.health === 'up'
          ? ok(`alvo wyd-server canal ${alvo.labels.canal ?? '?'} up (${alvo.scrapeUrl})`)
          : falha(`alvo wyd-server canal ${alvo.labels.canal ?? '?'} fora (${alvo.health})`);
      }
    }
  } catch (e) {
    falha(`/api/v1/targets ilegível: ${e.message}`);
  }

  const regras = await pega(`http://127.0.0.1:${prom}/api/v1/rules?type=alert`);
  try {
    const grupos = JSON.parse(regras.corpo).data.groups;
    const nomes = grupos.flatMap((g) => g.rules.map((r) => r.name));
    nomes.length >= 8 ? ok(`regras carregadas: ${nomes.join(', ')}`) : falha(`regras: ${nomes.length}`);
  } catch (e) {
    falha(`/api/v1/rules ilegível: ${e.message}`);
  }

  const saude = await pega(`http://127.0.0.1:${graf}/api/health`);
  saude.status === 200 && /"database"\s*:\s*"ok"/.test(saude.corpo)
    ? ok('grafana /api/health ok')
    : falha(`grafana /api/health ${saude.status} ${saude.corpo.slice(0, 80)}`);

  const fontes = await pega(`http://127.0.0.1:${graf}/api/datasources`, { auth: true });
  try {
    const lista = JSON.parse(fontes.corpo);
    Array.isArray(lista) && lista.some((d) => d.type === 'prometheus' && d.uid === 'prometheus')
      ? ok('datasource prometheus provisionada (uid=prometheus)')
      : falha(`datasource prometheus ausente (status ${fontes.status})`);
  } catch (e) {
    falha(`/api/datasources ilegível: ${e.message}`);
  }

  const dash = await pega(`http://127.0.0.1:${graf}/api/dashboards/uid/wyd-saude`, { auth: true });
  try {
    const dados = JSON.parse(dash.corpo);
    dados.dashboard && dados.dashboard.title
      ? ok(`dashboard provisionado: ${dados.dashboard.title} (${dados.dashboard.panels.length} painéis)`)
      : falha(`dashboard wyd-saude ausente (status ${dash.status})`);
  } catch (e) {
    falha(`/api/dashboards/uid/wyd-saude ilegível (status ${dash.status}): ${e.message}`);
  }

  console.log(falhas === 0 ? 'smoke de monitoramento: tudo verde' : `smoke de monitoramento: ${falhas} falha(s)`);
  process.exit(falhas === 0 ? 0 : 1);
}

main();
