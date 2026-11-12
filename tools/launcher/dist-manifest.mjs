#!/usr/bin/env node
// Gera o manifesto de distribuição do launcher (`manifest.txt`) para uma pasta.
//
// Formato lido por `wyd-launcher` (`crates/wyd-launcher/src/manifest.rs`):
//   # wyd-launcher-manifest v1
//   build 7662
//   protocol 12000
//   <sha256> <tamanho> <caminho relativo>
//
// Uso:
//   node tools/launcher/dist-manifest.mjs <pasta> <build> <protocolo> [--check]
//
// `--check` não escreve: recalcula os hashes e falha se o manifesto no disco
// divergir dos bytes reais (manifesto que mente não passa).
//
// O cliente real tem arquivos mutáveis (serverlist/bin, sn.bin gerados pelo
// launcher) e o próprio launcher instalado; esses ficam FORA do manifesto — se
// entrassem, o `--check` ficaria vermelho depois do primeiro Jogar. As exclusões
// são as mesmas do README do launcher.
//
// A pasta fica fora do git (`dist/`, `client/`), como qualquer binário do projeto.
import { createHash } from "node:crypto";
import { existsSync, readFileSync, readdirSync, statSync, writeFileSync } from "node:fs";
import path from "node:path";

const [dirArg, buildArg, protocolArg] = process.argv.slice(2);
if (!dirArg || !buildArg || !protocolArg) {
  console.error("uso: node tools/launcher/dist-manifest.mjs <pasta> <build> <protocolo>");
  process.exit(2);
}
const build = Number(buildArg);
const protocol = Number(protocolArg);
const somenteCheck = process.argv.includes("--check");

/** Artefatos mutáveis/de ambiente que nunca entram no manifesto do cliente. */
const EXCLUIDOS = new Set([
  "manifest.txt",
  "canais.txt",
  "launcher.conf",
  "launcher.conf.bak",
  "WYDLauncher.exe",
  "serverlist.bin",
  "serverlist.bin.bak",
  "serverlist.txt",
  "sn.bin",
]);
const excluido = (rel) =>
  EXCLUIDOS.has(rel) || /\.(bak|tmp|part|pdb)$/i.test(rel) || rel.split("/").some((p) => p === "__pycache__");
if (!Number.isInteger(build) || !Number.isInteger(protocol)) {
  console.error("build e protocolo precisam ser inteiros");
  process.exit(2);
}

/** Todos os arquivos da pasta, relativo com `/` (sem diretórios vazios). */
function arquivos(dir, prefixo = "") {
  const out = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const rel = prefixo ? `${prefixo}/${entry.name}` : entry.name;
    if (entry.isDirectory()) out.push(...arquivos(path.join(dir, entry.name), rel));
    else if (entry.isFile() && !excluido(rel)) out.push(rel);
  }
  return out.sort();
}

const linhas = ["# wyd-launcher-manifest v1", `build ${build}`, `protocol ${protocol}`];
for (const rel of arquivos(dirArg)) {
  const abs = path.join(dirArg, rel);
  // Caminho é o resto da linha: espaços e não-ASCII são suportados pelo parser Rust.
  const bytes = readFileSync(abs);
  const sha = createHash("sha256").update(bytes).digest("hex");
  linhas.push(`${sha} ${statSync(abs).size} ${rel}`);
}

const destino = path.join(dirArg, "manifest.txt");
const texto = `${linhas.join("\n")}\n`;
if (somenteCheck) {
  const atual = existsSync(destino) ? readFileSync(destino, "utf8") : null;
  if (atual !== texto) {
    console.error(`manifesto divergente dos bytes reais: ${destino}`);
    process.exit(1);
  }
  console.log(`manifesto confere: ${linhas.length - 3} arquivo(s) em ${destino}`);
  process.exit(0);
}
writeFileSync(destino, texto, "utf8");
console.log(`manifesto: ${linhas.length - 3} arquivo(s) em ${destino}`);
