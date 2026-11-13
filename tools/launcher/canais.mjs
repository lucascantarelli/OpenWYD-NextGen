#!/usr/bin/env node
// Deriva `canais.txt` do que o cliente REAL já tem (`serverlist.bin` + `sn.bin`).
//
// Nada é inventado: os endereços saem do `.bin` (decodificado com a mesma
// tabela do cliente) e o nome do servidor, quando existe, vem do `sn.bin`.
// O `canais.txt` gerado é o registro que o launcher lê para REGENERAR o
// `serverlist.bin` a cada execução — e o round-trip é conferido byte a byte.
//
// Uso:
//   node tools/launcher/canais.mjs <pasta-do-cliente>            # escreve
//   node tools/launcher/canais.mjs <pasta-do-cliente> --check    # confere
import { createHash } from "node:crypto";
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";

import {
  build,
  decode,
  filled,
  parseSn,
  renderSelect,
} from "../client/serverlist.mjs";

const [dirArg] = process.argv.slice(2).filter((a) => !a.startsWith("--"));
if (!dirArg) {
  console.error("uso: node tools/launcher/canais.mjs <pasta-do-cliente> [--check]");
  process.exit(2);
}
const somenteCheck = process.argv.includes("--check");
const binPath = path.join(dirArg, "serverlist.bin");
const snPath = path.join(dirArg, "sn.bin");
const destino = path.join(dirArg, "canais.txt");

if (!existsSync(binPath)) {
  console.error(`sem ${binPath}: não há canais para derivar`);
  process.exit(2);
}
const bin = readFileSync(binPath);
const grid = decode(bin);
const linhas = filled(grid);
if (linhas.length === 0) {
  console.error(`${binPath}: nenhuma célula preenchida — não vou inventar canais`);
  process.exit(1);
}
const sha = createHash("sha256").update(bin).digest("hex");
const nomes = [];
if (existsSync(snPath)) {
  for (const linha of renderSelect(parseSn(readFileSync(snPath)), grid)) {
    nomes.push(`# servidor "${linha.server}" (grupo ${linha.group}) — nome lido de sn.bin`);
  }
}
const texto = [
  `# canais derivados de serverlist.bin (${bin.length} bytes, sha256 ${sha})`,
  ...nomes,
  ...linhas,
].join("\n") + "\n";

if (somenteCheck) {
  const atual = existsSync(destino) ? readFileSync(destino, "utf8") : null;
  if (atual !== texto) {
    console.error(`canais divergentes dos bytes do cliente: ${destino}`);
    process.exit(1);
  }
  const regerado = build(filled(grid).map((l) => {
    const [group, channel, addr] = l.split(" ");
    return { group: Number(group), channel: Number(channel), addr };
  }));
  if (!regerado.equals(bin)) {
    console.error("canais.txt não reproduz o serverlist.bin do cliente byte a byte");
    process.exit(1);
  }
  console.log(`canais conferem: ${linhas.length} célula(s), serverlist.bin reproduzido`);
  process.exit(0);
}

writeFileSync(destino, texto, "utf8");
console.log(`canais: ${linhas.length} célula(s) em ${destino}`);
