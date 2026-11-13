#!/usr/bin/env node
// Limpeza de `tmp/` (política em docs/arquitetura/visao-pastas.md).
//
// Remove só artefatos que o próprio projeto gera e pode recriar: logs de teste,
// drivers baixados, pastas de E2E/instalação e cópias de cliente. Capturas
// nomeadas pelo operador (`tmp/captura*`, `tmp/*.png`, `tmp/basemob`) ficam.
//
// Uso:
//   node tools/dev/clean-tmp.mjs            # mostra o que removeria
//   node tools/dev/clean-tmp.mjs --yes      # remove
import { existsSync, readdirSync, rmSync, statSync } from "node:fs";
import path from "node:path";

const RAIZ = path.resolve(path.dirname(new URL(import.meta.url).pathname.replace(/^\/([A-Za-z]:)/, "$1")), "../..");
const TMP = path.join(RAIZ, "tmp");
const remover = process.argv.includes("--yes");

/** Pastas/arquivos gerados pelo projeto (recriáveis). */
const CONHECIDOS = [
  "e2e-janela",
  "instala-teste",
  "drivers",
  "cliente-pristino",
  "portal-cliente.log",
  "check-client.log",
  "check-final.log",
  "check-limpo.log",
  "check-pos-reparo.log",
  "check-instala-teste.log",
  "baixado.txt",
  "orig-WYDLauncher.exe.bak",
];
const PADROES = [/\.zip-bak-/, /\.reparo-bak-/, /\.log$/];

/** Caches de build dentro de `tmp/` (Cargo target) — fonte e Cargo.toml ficam. */
function target_aninhados() {
  const alvos = [];
  for (const nome of readdirSync(TMP)) {
    const completo = path.join(TMP, nome);
    if (!statSync(completo).isDirectory()) continue;
    const target = path.join(completo, "target");
    if (existsSync(target) && statSync(target).isDirectory()) alvos.push(target);
  }
  return alvos;
}

if (!existsSync(TMP)) {
  console.log("tmp/: nada a limpar (pasta ausente)");
  process.exit(0);
}

const alvos = [];
for (const nome of readdirSync(TMP)) {
  const completo = path.join(TMP, nome);
  const conhecido = CONHECIDOS.includes(nome) || PADROES.some((p) => p.test(nome));
  // Só remove arquivo/pasta conhecido; `captura`, `basemob`, `*.png` ficam.
  if (!conhecido) continue;
  const tamanho = statSync(completo).isDirectory() ? "pasta" : `${statSync(completo).size} B`;
  alvos.push({ completo, tamanho });
}

for (const completo of target_aninhados()) {
  alvos.push({ completo, tamanho: "pasta target" });
}

if (alvos.length === 0) {
  console.log("tmp/: nenhum artefato conhecido para limpar");
  process.exit(0);
}
for (const alvo of alvos) {
  console.log(`${remover ? "removendo" : "removeria"} ${path.relative(RAIZ, alvo.completo)} (${alvo.tamanho})`);
  if (remover) rmSync(alvo.completo, { recursive: true, force: true });
}
console.log(remover ? `tmp/: ${alvos.length} artefato(s) removido(s)` : "dry-run: use --yes para remover");
