#!/usr/bin/env node
// Goldens de pacotes C2G do caminho login → campo (sintéticos, nunca capturados).
// Uso: node tools/client/captura.mjs --gen     → escreve fixtures/cliente/*.json
//      node tools/client/captura.mjs --check   → valida e grava gerado/goldens.json
//
// O algoritmo é o mesmo do crate `wyd-protocol` (obfuscation.rs): header em claro,
// transformação de 4 até Size, semente KEY_TABLE[key*2], checksum = soma(cifrado)-soma(claro).
// Dados de conta são sintéticos e marcados como tal; fixture de captura real entra aqui
// só depois de higienizada (ver docs/cliente/depuracao.md §5).
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { loadKeywordTable, transform } from './keytable.mjs';

const ROOT = 'fixtures/cliente';
const OUT = 'docs/cliente/gerado/goldens.json';

const SYNTHETIC = 'GOLDEN';

function loadOpcodeMap(file = 'crates/wyd-protocol/src/wire/generated/opcode_table.rs') {
  const source = fs.readFileSync(file, 'utf8');
  const map = new Map();
  for (const match of source.matchAll(/Opcode::(\w+)\s*=>\s*(0x[0-9a-fA-F]+)/g)) {
    map.set(parseInt(match[2], 16), match[1]);
  }
  return map;
}

/** Escreve o campo no payload; origens só do payload.rs (offsets documentados). */
function writeField(buf, field) {
  if (buf.length <= field.offset) throw new Error(`offset ${field.offset} fora do pacote ${buf.length}`);
  if (field.kind === 'ascii') {
    const bytes = Buffer.from(String(field.value), 'latin1');
    if (bytes.length > field.len) throw new Error(`campo ${field.name} excede ${field.len} B`);
    bytes.copy(buf, field.offset);
  } else if (field.kind === 'i16') {
    buf.writeInt16LE(field.value, field.offset);
  } else if (field.kind === 'i32') {
    buf.writeInt32LE(field.value, field.offset);
  } else if (field.kind === 'bytes') {
    const bytes = Buffer.from(String(field.value), 'hex');
    if (bytes.length !== field.len) throw new Error(`campo ${field.name} tem ${bytes.length} B, esperado ${field.len}`);
    bytes.copy(buf, field.offset);
  } else {
    throw new Error(`kind desconhecido: ${field.kind}`);
  }
}

const FIXTURES = [
  {
    id: 'account_login',
    name: 'AccountLogin',
    type: 0x020d,
    size: 116,
    keyword: 9,
    connId: 1,
    tick: 1001,
    fields: [
      { offset: 12, name: 'account_password', kind: 'ascii', len: 12, value: 'SYNTHETIC' },
      { offset: 24, name: 'account_name', kind: 'ascii', len: 16, value: SYNTHETIC },
      { offset: 40, name: 'zero', kind: 'bytes', len: 52, value: '00'.repeat(52) },
      { offset: 92, name: 'client_version', kind: 'i32', len: 4, value: 12000 },
      { offset: 96, name: 'db_need_save', kind: 'i32', len: 4, value: 0 },
      { offset: 100, name: 'adapter_name', kind: 'bytes', len: 16, value: '00'.repeat(16) },
    ],
  },
  {
    id: 'create_character',
    name: 'CreateCharacter',
    type: 0x020f,
    size: 36,
    keyword: 77,
    connId: 1,
    tick: 1002,
    fields: [
      { offset: 12, name: 'slot', kind: 'i32', len: 4, value: 0 },
      { offset: 16, name: 'mob_name', kind: 'ascii', len: 16, value: SYNTHETIC },
      { offset: 32, name: 'mob_class', kind: 'i32', len: 4, value: 0 },
    ],
  },
  {
    id: 'character_login',
    name: 'CharacterLogin',
    type: 0x0213,
    size: 20,
    keyword: 128,
    connId: 1,
    tick: 1003,
    fields: [
      { offset: 12, name: 'slot', kind: 'i32', len: 4, value: 0 },
      { offset: 16, name: 'force', kind: 'i32', len: 4, value: 0 },
    ],
  },
  {
    id: 'action',
    name: 'Action',
    type: 0x036c,
    size: 52,
    keyword: 5,
    connId: 1,
    tick: 1004,
    fields: [
      { offset: 12, name: 'pos_x', kind: 'i16', len: 2, value: 1702 },
      { offset: 14, name: 'pos_y', kind: 'i16', len: 2, value: 1728 },
      { offset: 16, name: 'effect', kind: 'i32', len: 4, value: 0 },
      { offset: 20, name: 'speed', kind: 'i32', len: 4, value: 0 },
      { offset: 24, name: 'route', kind: 'bytes', len: 24, value: '00'.repeat(24) },
      { offset: 48, name: 'target_x', kind: 'i16', len: 2, value: 1703 },
      { offset: 50, name: 'target_y', kind: 'i16', len: 2, value: 1730 },
    ],
  },
  {
    id: 'header_only',
    name: 'HeaderOnly',
    type: 0x0333,
    size: 12,
    keyword: 5,
    connId: 1,
    tick: 1005,
    fields: [],
  },
];

function buildWire(fixture, table) {
  const plain = Buffer.alloc(fixture.size, 0);
  plain.writeUInt16LE(fixture.size, 0);
  plain[2] = fixture.keyword;
  plain.writeUInt16LE(fixture.type, 4);
  plain.writeUInt16LE(fixture.connId, 6);
  plain.writeUInt32LE(fixture.tick, 8);
  for (const field of fixture.fields) writeField(plain, field);

  const wire = transform(plain, { keyWord: fixture.keyword, table, from: 4, direction: 'encode' });
  let sumPlain = 0;
  let sumCipher = 0;
  for (let i = 4; i < fixture.size; i += 1) {
    sumPlain = (sumPlain + plain[i]) & 0xff;
    sumCipher = (sumCipher + wire[i]) & 0xff;
  }
  const checksum = (sumCipher - sumPlain) & 0xff;
  wire[3] = checksum;
  return { plain, wire, checksum };
}

function fixtureDocument(fixture, built) {
  return {
    id: fixture.id,
    origin: 'generated',
    warning: 'bytes sintéticos para teste de codec; NÃO são captura de rede',
    builtBy: 'node tools/client/captura.mjs --gen',
    algorithm: 'mesmas regras de crates/wyd-protocol/src/obfuscation.rs',
    header: {
      size: fixture.size,
      type: fixture.type,
      typeHex: `0x${fixture.type.toString(16).padStart(4, '0')}`,
      connId: fixture.connId,
      tick: fixture.tick,
      keyword: fixture.keyword,
      checksum: built.checksum,
      checksumHex: `0x${built.checksum.toString(16).padStart(2, '0')}`,
    },
    fields: fixture.fields,
    payloadHex: built.plain.toString('hex'),
    wireHex: built.wire.toString('hex'),
  };
}

function checkFixture(document, table, opcodeMap) {
  const errors = [];
  const wire = Buffer.from(document.wireHex, 'hex');
  const expected = Buffer.from(document.payloadHex, 'hex');

  // O wire viaja ofuscado a partir de 4: Type e corpo só são legíveis depois do decode.
  // Bytes 2 (KeyWord) e 3 (CheckSum) ficam em claro e NÃO voltam ao original no
  // round-trip (mesma regra do crate: só Size @0..2 e corpo @4.. são identidade).
  if (wire.length !== document.header.size) errors.push(`tamanho ${wire.length} != Size ${document.header.size}`);
  if (wire.readUInt16LE(0) !== document.header.size) errors.push('Size do header divergente');
  if (wire[2] !== document.header.keyword) errors.push('KeyWord do header divergente');

  const decoded = transform(wire, {
    keyWord: document.header.keyword,
    table,
    from: 4,
    direction: 'decode',
  });
  if (decoded.readUInt16LE(4) !== document.header.type) errors.push('Type decodificado divergente');
  if (!opcodeMap.has(document.header.type)) errors.push(`opcode 0x${document.header.type.toString(16)} fora da tabela gerada`);

  let sumPlain = 0;
  let sumCipher = 0;
  for (let i = 4; i < wire.length; i += 1) {
    sumPlain = (sumPlain + decoded[i]) & 0xff;
    sumCipher = (sumCipher + wire[i]) & 0xff;
  }
  const computed = (sumCipher - sumPlain) & 0xff;
  if (computed !== wire[3]) errors.push('checksum não confere');
  if (wire[3] !== document.header.checksum) errors.push('checksum do header divergente');
  if (!decoded.subarray(0, 2).equals(expected.subarray(0, 2))) errors.push('Size decodificado divergente');
  if (!decoded.subarray(4).equals(expected.subarray(4))) errors.push('corpo decodificado diverge do payload declarado');

  const source = Buffer.from(expected);
  source[3] = 0;
  const reEncoded = transform(source, {
    keyWord: document.header.keyword,
    table,
    from: 4,
    direction: 'encode',
  });
  reEncoded[3] = computed;
  if (!reEncoded.equals(wire)) errors.push('re-encode diverge do wire declarado');
  return errors;
}

const mode = process.argv[2];
if (mode === '--gen') {
  const table = loadKeywordTable();
  const opcodeMap = loadOpcodeMap();
  fs.mkdirSync(ROOT, { recursive: true });
  for (const fixture of [...FIXTURES].sort((a, b) => a.id.localeCompare(b.id))) {
    if (!opcodeMap.has(fixture.type)) {
      throw new Error(`opcode 0x${fixture.type.toString(16)} fora da tabela gerada`);
    }
    const built = buildWire(fixture, table);
    const document = fixtureDocument(fixture, built);
    fs.writeFileSync(path.join(ROOT, `${fixture.id}.json`), `${JSON.stringify(document, null, 2)}\n`);
    console.log(`ok ${fixture.id}: type=${document.header.typeHex} size=${fixture.size} checksum=${document.header.checksumHex}`);
  }
} else if (mode === '--check') {
  const table = loadKeywordTable();
  const opcodeMap = loadOpcodeMap();
  const files = fs.readdirSync(ROOT).filter((name) => name.endsWith('.json')).sort();
  if (!files.length) throw new Error(`nenhuma fixture em ${ROOT}`);
  const results = [];
  for (const file of files) {
    const document = JSON.parse(fs.readFileSync(path.join(ROOT, file), 'utf8'));
    const errors = checkFixture(document, table, opcodeMap);
    results.push({
      id: document.id,
      origin: document.origin,
      typeHex: document.header.typeHex,
      size: document.header.size,
      checksumHex: document.header.checksumHex,
      ok: errors.length === 0,
      errors,
    });
    console.log(`${errors.length === 0 ? 'ok' : 'FALHA'} ${document.id} (${document.origin})`);
    for (const error of errors) console.log(`  - ${error}`);
  }
  fs.mkdirSync(path.dirname(OUT), { recursive: true });
  fs.writeFileSync(OUT, `${JSON.stringify({ source: 'fixtures/cliente/*.json', results }, null, 2)}\n`);
  const failed = results.filter((r) => !r.ok).length;
  console.log(`goldens: ${results.length - failed}/${results.length} válidos → ${OUT}`);
  process.exit(failed ? 1 : 0);
} else {
  console.error('uso: node tools/client/captura.mjs --gen | --check');
  process.exit(2);
}
