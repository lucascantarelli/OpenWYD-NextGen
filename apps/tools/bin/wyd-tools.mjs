#!/usr/bin/env node
// `wyd-tools` — app de análise do projeto (nunca runtime).
//
// Comandos:
//   inventario <dir> [--json|--out md]         resumo medido (arquivos, bytes, PE)
//   binarios <dir> [--out <md>]                imports/exports/seções/strings/funções
//   strings <dir> [--bin WYD.exe] [--out <md>] tabela de strings do binário
//   pacotes [--client <dir>] [--out <md>]      opcodes × handlers × estruturas × capturas
//   docs <dir>                                 regenera TODOS os docs em docs/cliente/
//   zip-check <zip> [--manifest <arq>]         confere client.zip contra o manifesto
import { readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { inventariar } from "../lib/inventario.mjs";
import { renderInventario } from "../lib/docs.mjs";
import { analisarBinario } from "../lib/pe-analise.mjs";
import { analisarPacotes } from "../lib/pacotes.mjs";
import { renderBinarios, renderPacotes, renderStrings } from "../lib/re-docs.mjs";
import { parseManifesto, verificarZip } from "../lib/zip.mjs";

const RAIZ = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");
const [comando, dir, ...flags] = process.argv.slice(2);

function arg(nome, padrao = null) {
  const i = flags.indexOf(nome);
  return i >= 0 && flags[i + 1] ? flags[i + 1] : padrao;
}

function uso() {
  console.error(
    "uso: wyd-tools inventario <dir> [--json] | binarios <dir> [--out md] | " +
      "strings <dir> [--bin WYD.exe] [--out md] | pacotes [--client dir] [--out md] | " +
      "docs <dir> | zip-check <client.zip> [--manifest <manifest.txt>]",
  );
  process.exit(2);
}

function escrever(destino, texto) {
  const alvo = path.resolve(destino);
  writeFileSync(alvo, texto, "utf8");
  console.log(`gerado: ${path.relative(RAIZ, alvo)}`);
}

/** Analisa todos os PEs listados pelo inventário. */
function analisarBinarios(raizCliente) {
  const inv = inventariar(raizCliente);
  return inv.executaveis.map((exe) =>
    analisarBinario(readFileSync(path.join(raizCliente, exe.caminho)), exe.caminho),
  );
}

if (!comando) uso();

try {
  if (comando === "inventario") {
    if (!dir) uso();
    const inventario = inventariar(dir);
    if (arg("--out")) {
      escrever(arg("--out"), renderInventario(inventario));
    } else if (flags.includes("--json")) {
      console.log(JSON.stringify(inventario, null, 2));
    } else {
      console.log(
        `${inventario.arquivos} arquivo(s), ${inventario.bytes} byte(s), ${inventario.executaveis.length} PE`,
      );
      for (const exe of inventario.executaveis) {
        const pe = exe.pe ?? {};
        console.log(`  ${exe.caminho}: ${pe.machine ?? "?"}/${pe.subsystem ?? "?"} ${exe.sha256}`);
      }
    }
  } else if (comando === "binarios") {
    if (!dir) uso();
    const binarios = analisarBinarios(dir);
    escrever(arg("--out", "docs/cliente/re-binarios.md"), renderBinarios(binarios));
  } else if (comando === "strings") {
    if (!dir) uso();
    const nome = arg("--bin", "WYD.exe");
    const bin = analisarBinario(readFileSync(path.join(dir, nome)), nome);
    escrever(arg("--out", `docs/cliente/re-strings.md`), renderStrings(bin));
  } else if (comando === "pacotes") {
    const analise = analisarPacotes(RAIZ);
    const cliente = arg("--client", "client");
    let rede = null;
    try {
      rede = analisarBinario(readFileSync(path.join(cliente, "WYD.exe")), "WYD.exe");
    } catch {
      rede = null;
    }
    escrever(arg("--out", "docs/cliente/re-pacotes.md"), renderPacotes(analise, rede));
  } else if (comando === "docs") {
    if (!dir) uso();
    const inventario = inventariar(dir);
    escrever("docs/cliente/inventario-gerado.md", renderInventario(inventario));
    escrever("docs/cliente/re-binarios.md", renderBinarios(analisarBinarios(dir)));
    const wyd = analisarBinario(readFileSync(path.join(dir, "WYD.exe")), "WYD.exe");
    escrever("docs/cliente/re-strings.md", renderStrings(wyd));
    escrever("docs/cliente/re-pacotes.md", renderPacotes(analisarPacotes(RAIZ), wyd));
  } else if (comando === "zip-check") {
    if (!dir) uso();
    const manifesto = arg("--manifest", path.join(dir, "manifest.txt"));
    const esperado = parseManifesto(readFileSync(manifesto, "utf8"));
    const resultado = verificarZip(readFileSync(dir), esperado);
    if (!resultado.ok) {
      for (const caminho of resultado.ausentes) console.error(`AUSENTE   ${caminho}`);
      for (const caminho of resultado.divergentes) console.error(`DIVERGENTE ${caminho}`);
      console.error(
        `client.zip não confere: ${resultado.ausentes.length} ausente(s), ${resultado.divergentes.length} divergente(s)`,
      );
      process.exit(1);
    }
    console.log(
      `client.zip confere: ${esperado.size} arquivo(s) do manifesto, ${resultado.extras.length} extra(s) no ZIP`,
    );
  } else {
    uso();
  }
} catch (erro) {
  console.error(`wyd-tools: ${erro.message}`);
  process.exit(1);
}
