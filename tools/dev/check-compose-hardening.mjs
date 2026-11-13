#!/usr/bin/env node
/**
 * Contrato de superfície do compose (DEVOPS-03) — sem Docker e sem dependência nova.
 *
 * Verificação por texto porque o compose é YAML simples e o projeto não aceita dependência só
 * para ler dois arquivos; `docker compose config` continua sendo a prova de sintaxe, chamada
 * logo depois no mesmo job.
 *
 * O QUE ESTE JOB COBRA (e por quê):
 *   R1  prod exige TAG (`${TAG:?}`) — subir imagem `dev` em produção não pode ser silencioso;
 *   R2  prod desliga o anônimo do Grafana — observabilidade não é superfície pública;
 *   R3  todo serviço em prod tem `deploy.resources.limits` — sem teto, um vazamento de memória
 *       do motor derruba o host inteiro (inclusive o banco);
 *   R4  `--web.enable-lifecycle` fica FORA do arquivo base — reload por HTTP sem auth em prod;
 *   R5  prod não publica portas de banco/gateway/monitoramento no host;
 *   R6  o socket do Docker é montado só `:ro` (o `alloy` lê; leitura de `inspect` expõe `env`).
 *
 * O QUE AINDA NÃO ESTÁ COBERTO (checklist da issue, não deste script): TLS na frente do portal,
 * `healthcheck` em prometheus/grafana/loki/alloy/node-exporter e `security_opt`/`cap_drop`
 * por serviço. Entram aqui quando implementados — o job não finge cobri-los.
 *
 * Uso: `node tools/dev/check-compose-hardening.mjs` (ou `make compose-hardening`).
 */
import { readFileSync } from "node:fs";

const BASE = "docker-compose.yml";
const PROD = "docker-compose.prod.yml";

const base = readFileSync(BASE, "utf8");
const prod = readFileSync(PROD, "utf8");
const erros = [];

/** Linhas sem comentário (o compose tem muito comentário explicando o próprio contrato). */
const codigo = (texto) =>
  texto
    .split(/\r?\n/)
    .filter((linha) => !linha.trim().startsWith("#"))
    .join("\n");

/** Bloco de um serviço em um arquivo de compose (indentação de 2 espaços). */
function bloco(texto, servico) {
  const linhas = texto.split(/\r?\n/);
  const inicio = linhas.findIndex((linha) => new RegExp(`^  ${servico}:\\s*$`).test(linha));
  if (inicio === -1) return null;
  let fim = linhas.length;
  for (let i = inicio + 1; i < linhas.length; i += 1) {
    if (/^  [a-z0-9_-]+:\s*$/.test(linhas[i])) {
      fim = i;
      break;
    }
  }
  return linhas.slice(inicio + 1, fim).join("\n");
}

function servicos(texto) {
  return [...texto.matchAll(/^  ([a-z0-9][a-z0-9_-]*):\s*$/gm)].map((m) => m[1]);
}

// R1 — TAG obrigatória em produção.
for (const servico of ["portal", "gateway", "server-canal-1", "server-canal-2"]) {
  const trecho = bloco(prod, servico);
  if (trecho === null) continue; // serviço que o overlay não redefine não é cobrado aqui
  if (!/image:\s*\S*\$\{TAG:\?/.test(trecho)) {
    erros.push(`${PROD}/${servico}: image sem \${TAG:?defina TAG} (R1)`);
  }
}

// R2 — anônimo do Grafana desligado em produção.
const grafanaProd = bloco(prod, "grafana");
if (grafanaProd === null || !/GF_AUTH_ANONYMOUS_ENABLED:\s*["']?false/.test(grafanaProd)) {
  erros.push(`${PROD}/grafana: GF_AUTH_ANONYMOUS_ENABLED precisa ser "false" em produção (R2)`);
}

// R3 — teto de recurso em todo serviço de produção.
for (const servico of servicos(prod)) {
  const trecho = bloco(prod, servico) ?? "";
  if (!/limits:/.test(trecho)) {
    erros.push(`${PROD}/${servico}: sem deploy.resources.limits (R3)`);
  }
}

// R4 — lifecycle fora do base (só no override de dev).
if (/--web\.enable-lifecycle/.test(codigo(base))) {
  erros.push(`${BASE}: --web.enable-lifecycle no arquivo base (R4) — mova para o override de dev`);
}

// R5 — nenhuma porta de infraestrutura publicada em produção.
for (const proibida of ["5432:", "7600:", "9090:", "9093:", "9898:", "3001:", "8000:"]) {
  if (codigo(prod).includes(`"${proibida}`) || codigo(prod).includes(`${proibida}"`)) {
    erros.push(`${PROD}: publica porta de infraestrutura (${proibida}) no host (R5)`);
  }
}

// R6 — socket do Docker só leitura.
for (const linha of codigo(base).split(/\r?\n/)) {
  if (linha.includes("/var/run/docker.sock") && !linha.includes(":ro")) {
    erros.push(`${BASE}: socket do Docker montado sem :ro (R6) — ${linha.trim()}`);
  }
}

if (erros.length === 0) {
  console.log("superfície do compose ok (R1–R6) — pendente na issue: TLS, healthchecks, caps");
} else {
  for (const erro of erros) console.error(`compose-hardening: ${erro}`);
  console.error("compose-hardening: o overlay de produção não está seguro por construção (DEVOPS-03)");
  process.exit(1);
}
