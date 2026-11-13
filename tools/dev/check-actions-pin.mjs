#!/usr/bin/env node
/**
 * Pin de actions por SHA (DEVOPS-12) — confere YAML **e** a verdade no GitHub.
 *
 * Por que existe: `uses: actions/checkout@v7` é tag móvel — o conteúdo muda sem mudar o YAML, e
 * ninguém revisa o diff de um commit que não aparece no repositório. O pin por SHA congela o que
 * roda; o comentário ao lado (`# v7`) é o que um humano lê para saber o que foi congelado. Sozinho,
 * o comentário mente: por isso o script resolve tag/branch → commit na API do GitHub e compara.
 *
 * Regras por uso (ignora `uses: ./caminho` local):
 *   P1  o ref é um SHA de 40 hex — nada de `v7`, `main` ou `stable` solto;
 *   P2  existe comentário com a referência humana (`# v7`, `# v0.36.0`, `# stable`);
 *   P3  o SHA do pin é exatamente o commit para onde a referência aponta agora.
 *
 * Quando uma action lança versão nova, o Dependabot troca o par SHA+comentário no mesmo PR; o que
 * P3 impede é alguém trocar o comentário (ou colar um SHA de outro lugar) sem trocar o que roda.
 *
 * Uso:
 *   node tools/dev/check-actions-pin.mjs            # raiz = repositório atual
 *   node tools/dev/check-actions-pin.mjs <raiz>     # raiz alternativa (usado na prova negativa)
 *
 * Token: `GITHUB_TOKEN`/`GH_TOKEN` (CI). Sem token, cai para `gh api` (máquina de dev logada).
 * Exit 0 = tudo confere; 1 = divergência; 2 = não foi possível verificar (rede/credencial).
 */
import { execFileSync } from "node:child_process";
import { existsSync, readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";

const raiz = process.argv[2] ?? ".";
const TOKEN = process.env.GITHUB_TOKEN ?? process.env.GH_TOKEN ?? "";
const cache = new Map();
const erros = [];
let verificados = 0;

const USO = /uses:\s*(?<repo>[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+)(?:\/[^@\s]+)?@(?<ref>[^\s#]+)\s*(?:#\s*(?<rotulo>\S+))?/;

/** Chama a API do GitHub: fetch com token no CI, `gh api` na máquina de dev. */
async function api(rota) {
  if (cache.has(rota)) return cache.get(rota);
  let dados;
  if (TOKEN) {
    const resposta = await fetch(`https://api.github.com/${rota}`, {
      headers: {
        authorization: `Bearer ${TOKEN}`,
        accept: "application/vnd.github+json",
        "user-agent": "wyd-check-actions-pin",
      },
    });
    if (!resposta.ok) throw new Error(`GitHub ${resposta.status} em ${rota}`);
    dados = await resposta.json();
  } else {
    dados = JSON.parse(execFileSync("gh", ["api", rota], { encoding: "utf8" }));
  }
  cache.set(rota, dados);
  return dados;
}

/** Commit para onde `rotulo` aponta hoje: tag anotada precisa de um segundo GET. */
async function commitDoRotulo(repo, rotulo) {
  const tipo = /^v?\d/.test(rotulo) ? "tags" : "heads";
  const ref = await api(`repos/${repo}/git/ref/${tipo}/${rotulo}`);
  const objeto = ref.object;
  if (objeto.type === "tag") {
    const tag = await api(`repos/${repo}/git/tags/${objeto.sha}`);
    return tag.object.sha;
  }
  return objeto.sha;
}

function arquivos(prefixo) {
  const base = join(raiz, prefixo);
  if (!existsSync(base)) return [];
  if (prefixo.endsWith("workflows")) {
    return readdirSync(base)
      .filter((nome) => nome.endsWith(".yml") || nome.endsWith(".yaml"))
      .map((nome) => join(base, nome));
  }
  return readdirSync(base)
    .map((dir) => join(base, dir, "action.yml"))
    .filter((caminho) => existsSync(caminho));
}

const alvos = [...arquivos(".github/workflows"), ...arquivos(".github/actions")];
if (alvos.length === 0) {
  console.error(`check-actions-pin: nenhum workflow/action encontrado em ${raiz}`);
  process.exit(2);
}

for (const arquivo of alvos) {
  const linhas = readFileSync(arquivo, "utf8").split(/\r?\n/);
  for (const [indice, linha] of linhas.entries()) {
    if (!linha.includes("uses:") || /uses:\s*\.\//.test(linha)) continue;
    const achado = linha.match(USO);
    if (!achado) continue;
    const { repo, ref, rotulo } = achado.groups;
    const onde = `${arquivo.replace(/\\/g, "/")}:${indice + 1}`;

    if (!/^[0-9a-f]{40}$/.test(ref)) {
      erros.push(`${onde} — ${repo}@${ref}: não é SHA de 40 caracteres (P1)`);
      continue;
    }
    if (!rotulo) {
      erros.push(`${onde} — ${repo}@${ref.slice(0, 7)}: sem comentário com a referência (P2)`);
      continue;
    }
    let esperado;
    try {
      esperado = await commitDoRotulo(repo, rotulo);
    } catch (erro) {
      console.error(`check-actions-pin: não consegui resolver ${repo}#${rotulo}: ${erro.message}`);
      process.exit(2);
    }
    if (esperado !== ref) {
      erros.push(
        `${onde} — ${repo}: pin ${ref.slice(0, 7)} não é o commit de "${rotulo}" ` +
          `(${esperado.slice(0, 7)}). Aponte o SHA para ${esperado} ou corrija o comentário. (P3)`,
      );
      continue;
    }
    verificados += 1;
  }
}

if (erros.length > 0) {
  for (const erro of erros) console.error(`check-actions-pin: ${erro}`);
  console.error(
    `check-actions-pin: ${erros.length} problema(s) de pin — ${verificados} uso(s) conferido(s) contra a API.`,
  );
  process.exit(1);
}

console.log(
  `ok: ${verificados} uso(s) de action pinados por SHA, cada um conferido contra a referência do comentário`,
);
