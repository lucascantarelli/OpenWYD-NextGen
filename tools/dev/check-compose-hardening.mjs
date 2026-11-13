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
 *   R6  o socket do Docker é montado só `:ro` (o `alloy` lê; leitura de `inspect` expõe `env`);
 *   R7  todo serviço herda log com teto (`logging:`) e a âncora define `max-size` — um laço de
 *       erro não pode encher o disco do host;
 *   R8  o portal de produção só escuta em loopback (`127.0.0.1:3000`) — não há TLS no stack por
 *       decisão registrada (host único de dev), então expor `0.0.0.0:3000` seria HTTP puro na rede.
 *
 * O QUE AINDA NÃO ESTÁ COBERTO: `cap_drop` por serviço (exige provar boot de cada imagem) e
 * `healthcheck` dos exporters sem shell na imagem. Documentado, não fingido.
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

/** Serviços do arquivo base, dentro do bloco `services:` (volumes e âncoras ficam de fora). */
function servicosDoBase(texto) {
  const linhas = texto.split(/\r?\n/);
  const inicio = linhas.findIndex((linha) => linha === "services:");
  if (inicio === -1) return [];
  const corte = linhas.findIndex((linha, i) => i > inicio && /^(volumes|networks):/.test(linha));
  return servicos(linhas.slice(inicio, corte === -1 ? linhas.length : corte).join("\n"));
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

// R7 — log com teto em todo serviço (via âncora) e âncora com `max-size` definido.
if (!/x-log-teto:\s*&log-teto/.test(base) || !/max-size:/.test(base)) {
  erros.push(`${BASE}: âncora x-log-teto sem max-size (R7)`);
}
for (const servico of servicosDoBase(base)) {
  const trecho = bloco(base, servico) ?? "";
  if (!/^\s+logging:/m.test(trecho)) {
    erros.push(`${BASE}/${servico}: sem logging com teto (R7)`);
  }
}

// R8 — portal de produção só em loopback (sem TLS no stack, por decisão).
const portalProd = bloco(prod, "portal") ?? "";
const portaPortal = portalProd.match(/-\s*["']?([^"'\s]+:3000)["']?/);
if (!portaPortal) {
  erros.push(`${PROD}/portal: sem mapeamento de porta 3000 — se não publica, documente (R8)`);
} else if (!portaPortal[1].startsWith("127.0.0.1:")) {
  erros.push(`${PROD}/portal: publica ${portaPortal[1]} fora do loopback sem TLS (R8)`);
}

if (erros.length === 0) {
  console.log("superfície do compose ok (R1–R8) — TLS não adotado por decisão, documentado no §6");
} else {
  for (const erro of erros) console.error(`compose-hardening: ${erro}`);
  console.error("compose-hardening: o overlay de produção não está seguro por construção (DEVOPS-03)");
  process.exit(1);
}
