#!/usr/bin/env node
// Orquestrador do pipeline estático do cliente.
// Uso: node tools/client/analyse.mjs [caminho-do-exe]
// Gera artefatos determinísticos em docs/cliente/gerado/ e falha se o build divergir.
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { analyseEvidence, disassemble } from './evidence.mjs';
import { analyseFormats } from './formats.mjs';
import { parsePe, readExports, readImports, readRichHeader } from './pe.mjs';
import { indexStrings } from './strings.mjs';

export const BUILD = {
  id: '7662',
  exe: 'client/WYD.exe',
  sha256: '04c9d2a1235cba57f7b588347fae935a2466824ddf9d7d92e541e0cc8a27ddf8',
};

const OUT = 'docs/cliente/gerado';
const exe = process.argv[2] && !process.argv[2].startsWith('--') ? process.argv[2] : BUILD.exe;

const buf = fs.readFileSync(exe);
const sha256 = crypto.createHash('sha256').update(buf).digest('hex');
if (sha256 !== BUILD.sha256) {
  console.error(`build divergente de ${BUILD.id}: ${sha256}`);
  console.error(`esperado: ${BUILD.sha256}`);
  process.exit(1);
}

fs.mkdirSync(OUT, { recursive: true });
const writeJson = (name, value) => {
  const file = path.join(OUT, name);
  fs.writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`);
  return file;
};

const pe = parsePe(buf);
const imports = readImports(buf, pe).sort((a, b) => a.name.localeCompare(b.name));
const exportsInfo = readExports(buf, pe);
const rich = readRichHeader(buf, pe);
const strings = indexStrings(buf, pe);
const evidence = analyseEvidence({ buf, pe, exePath: exe });
const formats = analyseFormats();

const peReport = {
  build: { id: BUILD.id, exe, sha256, bytes: buf.length },
  kind: pe.kind,
  machine: pe.machineName,
  numberOfSections: pe.numberOfSections,
  timeDateStampIso: pe.timeDateStampIso,
  entryPoint: pe.entryPoint,
  entryPointVa: `0x${(pe.imageBase + pe.entryPoint).toString(16)}`,
  imageBase: pe.imageBaseHex,
  sizeOfImage: pe.sizeOfImage,
  sizeOfHeaders: pe.sizeOfHeaders,
  subsystem: pe.subsystemName,
  directories: pe.directories.filter((d) => d.rva),
  sections: pe.sections,
  imports: imports.map((dll) => ({
    dll: dll.name,
    functions: dll.functions.map((fn) => fn.name ?? `#${fn.ordinal}`),
  })),
  exports: exportsInfo,
  richHeader: rich,
};

const anchorText = [
  `# Âncoras herdadas — build ${BUILD.id} — sha256 ${sha256}`,
  '# gerado por tools/client/analyse.mjs (objdump para o desassemble)',
  '',
  ...evidence.anchors.flatMap((anchor) => [
    `## ${anchor.name} @${anchor.vaHex} — ${anchor.status}`,
    `# ${anchor.reason}`,
    anchor.disassembly || '# desassemble indisponível',
    '',
  ]),
  '## Patches de checksum',
  ...evidence.patchSites.flatMap((site) => [
    `### @${site.vaHex} — ${site.status} (${site.shape})`,
    disassemble(exe, site.va).text,
    '',
  ]),
].join('\n');

writeJson('pe.json', peReport);
writeJson('strings.json', strings);
writeJson('evidencia.json', evidence);
writeJson('formatos.json', formats);
fs.writeFileSync(path.join(OUT, 'ancoras.txt'), `${anchorText.trim()}\n`);

const line = (label, value) => console.log(`${label.padEnd(18)} ${value}`);
console.log(`build ${BUILD.id} — ${exe} — sha256 ok`);
line('formato', `${pe.kind} / ${pe.machineName} / base ${pe.imageBaseHex}`);
line('seções', pe.sections.map((s) => `${s.name}(${s.sizeOfRawData}B)`).join(' '));
line('imports', `${imports.length} DLLs, ${imports.reduce((n, d) => n + d.functions.length, 0)} funções`);
line('exports', exportsInfo.entries.length ? `${exportsInfo.entries.length} nomes` : 'nenhum');
line('strings', `${strings.totals.all} (relevantes ${strings.totals.relevant})`);
line('âncoras', `${evidence.summary.anchorsConfirmed}/${evidence.summary.anchorsTotal} CONFIRMADO`);
line('patches', `${evidence.summary.patchesConfirmed}/${evidence.summary.patchesTotal} CONFIRMADO`);
line('CPSock', `${evidence.cpsock.header.status}; tabela ${evidence.cpsock.keyTable.status}; INITCODE ${evidence.cpsock.initcode.status}`);
line('formatos', `wyt ${formats.resumo.wytReconhecidas}/${formats.resumo.wytAmostras}; msa ${formats.resumo.msaAmostras}; serverlist ${formats.resumo.serverlistInterpretado ? 'interpretado' : 'opaco'}`);
line('artefatos', `${OUT}/{pe,strings,evidencia,formatos}.json + ancoras.txt`);
