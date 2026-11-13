#!/usr/bin/env node
// Extrai as tabelas de dados do cliente 7662 (ItemList/SkillData/InitItem) e grava
// um perfil de colunas em docs/cliente/gerado/tabelas.json.
//
// POR QUE PERFIL E NÃO "SCHEMA": o layout dos registros não está publicado. O que se
// pode provar com bytes é (a) a chave de ofuscação, (b) o tamanho do registro e
// (c) quais colunas são sequência, enum pequeno ou vazia. Nomear campo sem prova
// seria inventar estrutura — e estrutura inventada vira bug silencioso.
//
// Uso: node tools/client/extract-tables.mjs [--verbose]
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';

const OUT = 'docs/cliente/gerado';
const VERBOSE = process.argv.includes('--verbose');

/** Chave medida: maior taxa de zeros em 200 kB (ItemList 83,5%; SkillData 92,7%). */
const XOR_KEY = 0x5a;
const HEADER = 4;

const TARGETS = [
  {
    file: 'SkillData.bin',
    kind: 'skill',
    record: 192,
    evidencia: 'período 192 em 93,6% das posições; (23812-4)/192 = 124 registros exatos',
  },
  {
    file: 'ItemList.bin',
    kind: 'item',
    record: 140,
    evidencia: 'período 140 em 90,3% das posições; (910004-4)/140 = 6500 registros exatos',
  },
];

const sha = (buf) => crypto.createHash('sha256').update(buf).digest('hex');

function decode(buf) {
  return Buffer.from(buf.subarray(HEADER).map((b) => b ^ XOR_KEY));
}

/** u16/u32 por registro, nas duas ordens — a que fizer sentido fica evidente no perfil. */
function columns(records, width) {
  const cols = [];
  for (let off = 0; off + width <= records[0].length; off += width) {
    const values = records.map((r) => (width === 2 ? r.readUInt16LE(off) : r.readUInt32LE(off)));
    const distinct = new Set(values);
    const nonZero = values.filter((v) => v !== 0).length;
    const sorted = [...values].sort((a, b) => a - b);
    // Sequência: valores estritamente crescentes de 1 em 1 (índice da tabela).
    let sequential = 0;
    for (let i = 1; i < values.length; i += 1) if (values[i] === values[i - 1] + 1) sequential += 1;
    cols.push({
      offset: off,
      width,
      zeros: values.length - nonZero,
      distinct: distinct.size,
      min: sorted[0],
      max: sorted[sorted.length - 1],
      // "enum": poucos valores distintos e todos pequenos — candidato a classe/linhagem.
      enumLike: distinct.size > 1 && distinct.size <= 16 && sorted[sorted.length - 1] <= 31,
      seqRatio: Number((sequential / Math.max(1, values.length - 1)).toFixed(3)),
      sample: values.slice(0, 6),
    });
  }
  return cols;
}

const report = { key: XOR_KEY, headerBytes: HEADER, arquivos: [], ausentes: [] };

for (const target of TARGETS) {
  const abs = path.join('client', target.file);
  if (!fs.existsSync(abs)) {
    report.ausentes.push(target.file);
    continue;
  }
  const raw = fs.readFileSync(abs);
  const buf = decode(raw);
  const count = Math.floor(buf.length / target.record);
  const resto = buf.length % target.record;
  const records = [];
  for (let i = 0; i < count; i += 1) {
    records.push(buf.subarray(i * target.record, (i + 1) * target.record));
  }

  const cols32 = columns(records, 4);
  const cols16 = columns(records, 2);
  const sequencia = cols32
    .filter((c) => c.seqRatio > 0.9 && c.distinct >= count * 0.8)
    .map((c) => c.offset);
  const enums = cols16.filter((c) => c.enumLike).map((c) => ({ offset: c.offset, distinct: c.distinct, max: c.max }));
  const vazias = cols32.filter((c) => c.zeros === count).map((c) => c.offset);

  report.arquivos.push({
    file: target.file,
    kind: target.kind,
    bytes: raw.length,
    sha256: sha(raw).slice(0, 16),
    registroBytes: target.record,
    registros: count,
    sobraBytes: resto,
    evidencia: target.evidencia,
    colunas: {
      sequenciaIndex: sequencia,
      candidatasEnum16: enums,
      sempreZero32: vazias,
    },
    perfil32: cols32,
    amostra: records.slice(0, 4).map((r) => [...r.subarray(0, 48)].map((b) => b.toString(16).padStart(2, '0')).join(' ')),
  });

  if (resto !== 0) {
    console.error(`${target.file}: ${resto} byte(s) sobrando — tamanho de registro provavelmente errado`);
  }
}

// O que NÃO está no cliente: registrado explicitamente para não virar promessa falsa.
report.ausentesNoCliente = [
  {
    dado: 'taxa de drop por item',
    onde: 'configuração do servidor (TMSrv), não no cliente',
    evidencia: 'nenhum dos .bin do cliente tem tabela por monstro; drop é decidido no servidor',
  },
  {
    dado: 'curva de EXP por nível',
    onde: 'lógica do servidor; o cliente só recebe o valor de EXP atual',
    evidencia: 'SkillData/ItemList/InitItem não contêm coluna com 400 valores crescentes de EXP',
  },
];

fs.mkdirSync(OUT, { recursive: true });
const outFile = path.join(OUT, 'tabelas.json');
fs.writeFileSync(outFile, `${JSON.stringify(report, null, 2)}\n`);

for (const arq of report.arquivos) {
  console.log(
    `${arq.file.padEnd(14)} ${String(arq.registros).padStart(5)} × ${arq.registroBytes} B  ` +
      `índice@${arq.colunas.sequenciaIndex.join(',') || '?'}  ` +
      `enums=${arq.colunas.candidatasEnum16.length}  vazias=${arq.colunas.sempreZero32.length}`,
  );
  if (VERBOSE) {
    for (const col of arq.colunas.candidatasEnum16) {
      console.log(`   enum u16 @${col.offset}: ${col.distinct} valores (max ${col.max})`);
    }
  }
}
for (const ausente of report.ausentesNoCliente) console.log(`ausente: ${ausente.dado} — ${ausente.onde}`);
console.log(`artefato: ${outFile}`);
