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
//   node tools/launcher/dist-manifest.mjs <pasta> <build> <protocolo>
//
// A pasta fica fora do git (`dist/`), como qualquer binário do projeto.
import { createHash } from "node:crypto";
import { readFileSync, readdirSync, statSync, writeFileSync } from "node:fs";
import path from "node:path";

const [dirArg, buildArg, protocolArg] = process.argv.slice(2);
if (!dirArg || !buildArg || !protocolArg) {
  console.error("uso: node tools/launcher/dist-manifest.mjs <pasta> <build> <protocolo>");
  process.exit(2);
}
const build = Number(buildArg);
const protocol = Number(protocolArg);
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
    else if (entry.isFile() && entry.name !== "manifest.txt") out.push(rel);
  }
  return out.sort();
}

const linhas = ["# wyd-launcher-manifest v1", `build ${build}`, `protocol ${protocol}`];
for (const rel of arquivos(dirArg)) {
  const abs = path.join(dirArg, rel);
  if (/\s/.test(rel)) throw new Error(`caminho com espaço não é suportado: ${rel}`);
  const bytes = readFileSync(abs);
  const sha = createHash("sha256").update(bytes).digest("hex");
  linhas.push(`${sha} ${statSync(abs).size} ${rel}`);
}

const destino = path.join(dirArg, "manifest.txt");
writeFileSync(destino, `${linhas.join("\n")}\n`, "utf8");
console.log(`manifesto: ${linhas.length - 3} arquivo(s) em ${destino}`);
