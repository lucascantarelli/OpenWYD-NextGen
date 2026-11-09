#!/usr/bin/env node
// Diff de duas capturas CPSock — a correlação de bytes da FASE-08-6.
//
// Uso:
//   node tools/client/diff-captura.mjs --a tmp/captura-a.json --b tmp/captura-b.json
//   node tools/client/diff-captura.mjs --a ... --b ... --json tmp/diff.json
//   node tools/client/diff-captura.mjs --selftest
//
// Lê os artefatos do `captura-live.mjs` (frames com `direcao` e `hex`), desofusca cada frame
// com a tabela de keyword do crate e pareia por direção + opcode + tamanho. Em cada par mostra
// as faixas de bytes que mudaram, com o valor nos dois estados. Faixa que acompanha a variação
// feita no banco é candidato a campo; o resto é ruído (tick, timestamp).
import fs from 'node:fs';
import path from 'node:path';
import { loadKeywordTable, transform } from './keytable.mjs';

export const HEADER = 12;

/** Desofusca o frame e devolve `{ direcao, opcode, size, clear }` (buffer em claro). */
export function abrirFrame(frame, tabela) {
  const wire = Buffer.from(frame.hex, 'hex');
  if (wire.length < HEADER) return null;
  // `transform` devolve um buffer novo (não muta a entrada) — usar o retorno.
  const clear = transform(wire, { keyWord: wire[2], table: tabela, direction: 'decode' });
  return {
    direcao: frame.direcao === 'c2s' ? 'C2S' : 'S2C',
    opcode: clear.readUInt16LE(4),
    size: clear.readUInt16LE(0),
    clear,
  };
}

/** Faixas contíguas de bytes diferentes, com o conteúdo dos dois lados. */
export function faixas(a, b) {
  const limite = Math.min(a.length, b.length);
  const runs = [];
  let ini = -1;
  for (let i = HEADER; i <= limite; i += 1) {
    const difere = i < limite && a[i] !== b[i];
    if (difere && ini < 0) ini = i;
    if (!difere && ini >= 0) {
      runs.push({
        offset: ini,
        tamanho: i - ini,
        a: a.subarray(ini, i).toString('hex'),
        b: b.subarray(ini, i).toString('hex'),
      });
      ini = -1;
    }
  }
  return runs;
}

/** Pareia os frames das duas capturas na ordem em que aparecem, por direção+opcode+tamanho. */
export function parear(framesA, framesB, tabela) {
  const chave = (f) => `${f.direcao}:${f.opcode}:${f.size}`;
  const filaB = new Map();
  for (const bruto of framesB) {
    const f = abrirFrame(bruto, tabela);
    if (!f) continue;
    const k = chave(f);
    if (!filaB.has(k)) filaB.set(k, []);
    filaB.get(k).push(f);
  }
  const pares = [];
  const sobrando = [];
  for (const bruto of framesA) {
    const f = abrirFrame(bruto, tabela);
    if (!f) continue;
    const fila = filaB.get(chave(f)) ?? [];
    const par = fila.shift();
    if (par) pares.push({ a: f, b: par });
    else sobrando.push(f);
  }
  for (const fila of filaB.values()) sobrando.push(...fila);
  return { pares, sobrando };
}

export function relatorio(capturaA, capturaB, tabela = loadKeywordTable()) {
  const { pares, sobrando } = parear(capturaA.frames ?? [], capturaB.frames ?? [], tabela);
  const frames = pares.map(({ a, b }) => ({
    direcao: a.direcao,
    opcode: `0x${a.opcode.toString(16).padStart(4, '0')}`,
    size: a.size,
    mudou: faixas(a.clear, b.clear),
  }));
  return {
    a: capturaA.origem ?? null,
    b: capturaB.origem ?? null,
    pareados: frames.length,
    com_diferenca: frames.filter((f) => f.mudou.length > 0),
    sem_par: sobrando.map((f) => `${f.direcao} 0x${f.opcode.toString(16).padStart(4, '0')} ${f.size}B`),
  };
}

function imprimir(r) {
  console.log(`a: ${r.a ?? '(sem origem)'}`);
  console.log(`b: ${r.b ?? '(sem origem)'}`);
  console.log(`frames pareados: ${r.pareados}; com diferença: ${r.com_diferenca.length}`);
  for (const f of r.com_diferenca) {
    console.log(`\n${f.direcao} ${f.opcode} ${f.size}B`);
    for (const faixa of f.mudou) {
      const pos = `+${faixa.offset} (0x${faixa.offset.toString(16)})`;
      console.log(`  ${pos} ${faixa.tamanho}B  a=${faixa.a}  b=${faixa.b}`);
    }
  }
  if (r.sem_par.length > 0) {
    console.log(`\nsem par (só em uma das capturas): ${r.sem_par.join(', ')}`);
  }
}

/** Frame sintético cifrado para o autoteste: cabeçalho de 12 B + corpo. */
function frameSintetico({ size, opcode, corpo, tabela }) {
  const buf = Buffer.alloc(size);
  buf.writeUInt16LE(size, 0);
  buf[2] = 11;
  buf.writeUInt16LE(opcode, 4);
  buf.writeUInt16LE(1, 6);
  corpo.copy(buf, HEADER);
  // `transform` devolve um buffer novo — sem atribuir, o frame fica em claro.
  const wire = transform(buf, { keyWord: 11, table: tabela, direction: 'encode' });
  return { direcao: 's2c', hex: wire.toString('hex') };
}

function selftest() {
  const tabela = loadKeywordTable();
  const corpoA = Buffer.alloc(1996);
  const corpoB = Buffer.from(corpoA);
  corpoB.writeInt32LE(42, 400); // nível 42 no offset 400+12
  const capturaA = { frames: [frameSintetico({ size: 2008, opcode: 0x010a, corpo: corpoA, tabela })] };
  const capturaB = { frames: [frameSintetico({ size: 2008, opcode: 0x010a, corpo: corpoB, tabela })] };
  const r = relatorio(capturaA, capturaB, tabela);
  const faixa = r.com_diferenca[0]?.mudou[0];
  // 42 em little-endian é `2a 00 00 00`: só o primeiro byte difere de zero — o diff correto
  // devolve uma faixa de 1 byte no offset 412 (12 de cabeçalho + 400 do corpo).
  const okOffsets = r.com_diferenca[0]?.mudou.length === 1 && faixa.offset === 412 && faixa.tamanho === 1;
  const okValores = faixa && faixa.a === '00' && faixa.b === '2a';
  if (!okOffsets || !okValores) {
    console.error('diff-captura: autoteste falhou', JSON.stringify(r, null, 2));
    return 1;
  }
  console.log('diff-captura: autoteste ok (0x010A, faixa +412 1B, 0 → 42)');
  return 0;
}

function arg(nome) {
  const i = process.argv.indexOf(`--${nome}`);
  return i >= 0 ? process.argv[i + 1] : null;
}

function main() {
  if (process.argv.includes('--selftest')) return selftest();
  const a = arg('a');
  const b = arg('b');
  if (!a || !b) {
    console.error('uso: node tools/client/diff-captura.mjs --a <a.json> --b <b.json> [--json saída.json] | --selftest');
    return 2;
  }
  const r = relatorio(
    JSON.parse(fs.readFileSync(path.resolve(a), 'utf8')),
    JSON.parse(fs.readFileSync(path.resolve(b), 'utf8')),
  );
  imprimir(r);
  const saida = arg('json');
  if (saida) {
    const destino = path.resolve(saida);
    fs.mkdirSync(path.dirname(destino), { recursive: true });
    fs.writeFileSync(destino, `${JSON.stringify(r, null, 2)}\n`, 'utf8');
    console.log(`\njson: ${destino}`);
  }
  return r.com_diferenca.length === 0 ? 1 : 0;
}

if (process.argv[1]?.endsWith('diff-captura.mjs')) process.exitCode = main();
