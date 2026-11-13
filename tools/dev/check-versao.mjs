#!/usr/bin/env node
/**
 * Fonte única de versão (DEVOPS-10): `Cargo.toml` raiz × crates × portal.
 *
 * O que é cobrado:
 *   1. `[workspace.package] version` existe no `Cargo.toml` raiz — é a fonte;
 *   2. todo membro declara `version.workspace = true` (nada de versão solta por crate);
 *   3. `wyd-portal/package.json` publica a mesma versão.
 *
 * Por que gate e não aviso: com três versões independentes, a tag `vX.Y.Z` deixa de identificar o
 * que foi construído — e a imagem passa a mentir sobre a proveniência (DEVOPS-06/DEVOPS-10).
 *
 * Uso: `node tools/dev/check-versao.mjs` (ou `make versao-check`).
 */
import { readFileSync, readdirSync, existsSync } from "node:fs";
import { join } from "node:path";

const RAIZ = new URL("../..", import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1");
const erros = [];

/** Versão de `[workspace.package]` no Cargo.toml raiz. */
function versaoDoWorkspace() {
  const texto = readFileSync(join(RAIZ, "Cargo.toml"), "utf8");
  const secao = /^\[workspace\.package\]\s*$([\s\S]*?)(?=^\[|(?![\s\S]))/m.exec(texto);
  if (!secao) {
    erros.push("Cargo.toml raiz sem [workspace.package] — a versão não tem fonte única");
    return null;
  }
  const versao = /^\s*version\s*=\s*"([^"]+)"/m.exec(secao[1])?.[1];
  if (!versao) {
    erros.push("Cargo.toml raiz: [workspace.package] sem `version`");
    return null;
  }
  return versao;
}

/** Membros do workspace declarados no Cargo.toml raiz. */
function membros() {
  const texto = readFileSync(join(RAIZ, "Cargo.toml"), "utf8");
  return [.../members\s*=\s*\[([\s\S]*?)\]/m.exec(texto)?.[1]?.matchAll(/"([^"]+)"/g) ?? []].map(
    (m) => m[1],
  );
}

function main() {
  const versao = versaoDoWorkspace();
  if (!versao) {
    relatar();
    return;
  }

  const soltos = [];
  for (const membro of membros()) {
    const caminho = join(RAIZ, membro, "Cargo.toml");
    if (!existsSync(caminho)) {
      erros.push(`${membro}: membro do workspace sem Cargo.toml`);
      continue;
    }
    const texto = readFileSync(caminho, "utf8");
    const herdada = /^\s*version\.workspace\s*=\s*true\s*$/m.test(texto);
    const propria = /^\[package\][\s\S]*?^\s*version\s*=\s*"([^"]+)"/m.exec(texto)?.[1];
    if (!herdada) {
      soltos.push(`${membro}${propria ? ` (version = "${propria}")` : ""}`);
    } else if (propria) {
      erros.push(`${membro}: tem version literal e version.workspace ao mesmo tempo`);
    }
  }
  if (soltos.length > 0) {
    erros.push(
      `crates com versão própria (use \`version.workspace = true\`): ${soltos.join(", ")}`,
    );
  }

  const portal = JSON.parse(readFileSync(join(RAIZ, "wyd-portal/package.json"), "utf8"));
  if (portal.version !== versao) {
    erros.push(`wyd-portal/package.json version="${portal.version}" ≠ workspace "${versao}"`);
  }

  if (erros.length === 0) {
    console.log(`versão única ok: ${versao} (workspace, ${membros().length} membros e portal)`);
    return;
  }
  relatar();
}

function relatar() {
  for (const erro of erros) {
    console.error(`check-versao: ${erro}`);
  }
  console.error("check-versao: corrija para a versão ser rastreável até a tag (DEVOPS-10)");
  process.exit(1);
}

main();
