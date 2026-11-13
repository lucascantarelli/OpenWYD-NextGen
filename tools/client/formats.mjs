#!/usr/bin/env node
// Estrutura inicial dos formatos de dados do cliente, a partir de amostras reais.
// O que não se sustenta com bytes é declarado opaco — nunca inferido como fato.
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { loadKeywordTable, transform } from './keytable.mjs';
import { parseWyt as decodificarWyt } from './wyt.mjs';

const head = (buf, n = 64) =>
  buf
    .subarray(0, Math.min(n, buf.length))
    .toString('hex')
    .replace(/(..)/g, '$1 ')
    .trim();

function cstringAt(buf, offset, max = 260) {
  let end = offset;
  while (end < buf.length && buf[end] !== 0 && end - offset < max) end += 1;
  return buf.toString('latin1', offset, end);
}

function entropy(buf, window = 4096) {
  const slice = buf.subarray(0, Math.min(window, buf.length));
  const counts = new Array(256).fill(0);
  for (const byte of slice) counts[byte] += 1;
  let value = 0;
  for (const count of counts) {
    if (!count) continue;
    const p = count / slice.length;
    value -= p * Math.log2(p);
  }
  return Number(value.toFixed(4));
}

function printableRatio(buf) {
  let printable = 0;
  for (const byte of buf) if ((byte >= 0x20 && byte < 0x7f) || byte === 0x09) printable += 1;
  return printable / buf.length;
}

function scoreText(buf) {
  const text = buf.toString('latin1');
  const printable = printableRatio(buf);
  let alnum = 0;
  for (const byte of buf) {
    const isAlnum =
      (byte >= 0x30 && byte <= 0x39) ||
      (byte >= 0x41 && byte <= 0x5a) ||
      (byte >= 0x61 && byte <= 0x7a) ||
      byte === 0x20;
    if (isAlnum) alnum += 1;
  }
  const ratioAlnum = Number((alnum / buf.length).toFixed(4));
  const hasToken = /192\.168\.|10\.\d+\.\d+\.|127\.0\.0\.1|:8281|serv00|http/i.test(text);
  // Texto "imprimível" cheio de símbolos não é estrutura: exigimos alfanuméricos.
  const meaningful = hasToken || (printable > 0.95 && ratioAlnum > 0.6);
  return {
    printable: Number(printable.toFixed(4)),
    alnum: ratioAlnum,
    hasToken,
    score: Number((printable + (hasToken ? 0.5 : 0) + ratioAlnum * 0.2).toFixed(4)),
    meaningful,
  };
}

function describe(file, buf, extra = {}) {
  return {
    file: file.split(path.sep).join('/'),
    size: buf.length,
    sha256: crypto.createHash('sha256').update(buf).digest('hex').slice(0, 16),
    entropia: entropy(buf),
    headHex: head(buf, 64),
    headAscii: buf
      .subarray(0, 48)
      .toString('latin1')
      .replace(/[^\x20-\x7e]/g, '.'),
    ...extra,
  };
}

function findSamples(root, dir, extension, limit = 2) {
  const full = path.join(root, dir);
  if (!fs.existsSync(full)) return [];
  return fs
    .readdirSync(full)
    .filter((name) => name.toLowerCase().endsWith(extension))
    .sort()
    .slice(0, limit)
    .map((name) => path.join(full, name));
}

function parseWyt(buf) {
  try {
    const wyt = decodificarWyt(buf);
    return {
      magic: 'WT10',
      recognised: true,
      version: wyt.versao,
      width: wyt.largura,
      height: wyt.altura,
      bits: wyt.bits,
      pixelsOffset: wyt.pixelsOffset,
      notes: [
        'raster puro: cabeçalho de 22 B + largura*altura*(bits/8) + 8 B finais',
        'fórmula confirmada em 256/259 amostras de client/NUI + client/UI (NUI 8/8)',
        'texto, fontes e controles são desenhados pelo .exe; o arquivo só tem pixels',
        'decoder/export: tools/client/wyt.mjs (--png, --bmp, --scan)',
      ],
    };
  } catch (e) {
    return { magic: buf.toString('latin1', 0, 4), recognised: false, motivo: e.message };
  }
}

function parseMsa(buf) {
  if (buf.length < 64) return { recognised: false };
  return {
    recognised: false,
    version: buf.readUInt32LE(0),
    field4: buf.readUInt32LE(4),
    field8: buf.readUInt32LE(8),
    field20: buf.readUInt32LE(20),
    field28: buf.readUInt32LE(28),
    texturePathOffset: 34,
    texturePath: cstringAt(buf, 34, 200),
    notes: [
      'u32 @0 = 0x112 (274) idêntico nas duas amostras — candidato a versão do formato',
      'u32 @4 = 0x20 (32) e u32 @8 = 1 idênticos nas duas amostras',
      'nome de textura ASCII terminado em NUL no offset 34 nas duas amostras ("copy.tga", "\\ori.psd")',
      'geometria, esqueleto e animação permanecem opacos',
    ],
  };
}

function parseSnBin(buf) {
  const names = [];
  let start = 0;
  for (let i = 0; i <= buf.length; i += 1) {
    if (i === buf.length || buf[i] === 0) {
      if (i > start) names.push(buf.toString('latin1', start, i));
      start = i + 1;
    }
  }
  return {
    firstByte: `0x${buf[0].toString(16).padStart(2, '0')}`,
    namesCount: names.length,
    names: names.slice(0, 12).map((name) => name.replace(/[^\x20-\x7e]/g, '.')),
    notes: [
      'nomes em texto claro separados por NUL a partir do offset 1',
      'a partir do offset 1 lê-se "eserved", coerente com "Reserved" com o primeiro caractere sobrescrito',
      'o primeiro byte (0xd6) não foi explicado por nenhum padrão testado — permanece UNVERIFIED',
    ],
  };
}

function serverlistAttempts(buf, table) {
  const attempts = [];
  for (const from of [4, 0]) {
    for (let keyWord = 0; keyWord < 256; keyWord += 1) {
      const decoded = transform(buf, { keyWord, table, from, direction: 'decode' });
      attempts.push({
        method: 'cpsock-decode',
        keyWord,
        from,
        score: scoreText(decoded),
        preview: decoded
          .subarray(0, 96)
          .toString('latin1')
          .replace(/[^\x20-\x7e]/g, '.'),
      });
    }
  }
  for (let key = 0; key < 256; key += 1) {
    const decoded = Buffer.from(buf.map((byte) => byte ^ key));
    attempts.push({
      method: 'xor1',
      keyWord: key,
      from: 0,
      score: scoreText(decoded),
      preview: decoded
        .subarray(0, 96)
        .toString('latin1')
        .replace(/[^\x20-\x7e]/g, '.'),
    });
  }
  const tableXor = Buffer.from(buf.map((byte, i) => byte ^ table[i % table.length]));
  attempts.push({
    method: 'xor-tabela',
    keyWord: null,
    from: 0,
    score: scoreText(tableXor),
    preview: tableXor
      .subarray(0, 96)
      .toString('latin1')
      .replace(/[^\x20-\x7e]/g, '.'),
  });

  attempts.sort((a, b) => b.score - a.score || a.method.localeCompare(b.method) || (a.keyWord ?? 0) - (b.keyWord ?? 0));
  const best = attempts[0];
  return {
    best,
    top5: attempts.slice(0, 5),
    interpreted: Boolean(best.meaningful),
    notes: [
      'tentativas: CPSock (256 keywords, a partir de 4 e de 0), XOR de 1 byte e XOR com a tabela',
      best.meaningful
        ? 'o método vencedor produz texto com estrutura reconhecível (token ou maioria alfanumérica)'
        : 'nenhuma tentativa produziu texto estruturalmente coerente — arquivo permanece opaco',
      'layout (grupo/índice/IP/porta) não foi comprovado nesta passada',
    ],
  };
}

export function analyseFormats({ root = 'client', table = loadKeywordTable() } = {}) {
  const wytSamples = [...findSamples(root, 'NUI', '.wyt', 2), ...findSamples(root, 'UI', '.wyt', 2)];
  const msaSamples = findSamples(root, 'Mesh', '.msa', 2);

  const wyt = wytSamples.map((file) => {
    const buf = fs.readFileSync(file);
    return describe(file, buf, parseWyt(buf));
  });

  const msa = msaSamples.map((file) => {
    const buf = fs.readFileSync(file);
    return describe(file, buf, parseMsa(buf));
  });

  const snPath = path.join(root, 'sn.bin');
  const snBuf = fs.existsSync(snPath) ? fs.readFileSync(snPath) : null;
  const snBin = snBuf ? describe(snPath, snBuf, parseSnBin(snBuf)) : null;

  const serverlistPath = path.join(root, 'serverlist.bin');
  const serverlistBuf = fs.existsSync(serverlistPath) ? fs.readFileSync(serverlistPath) : null;
  const serverlist = serverlistBuf
    ? describe(serverlistPath, serverlistBuf, serverlistAttempts(serverlistBuf, table))
    : null;

  const txtPath = path.join(root, 'serverlist.txt');
  const txtBuf = fs.existsSync(txtPath) ? fs.readFileSync(txtPath) : null;
  const serverlistTxt = txtBuf
    ? describe(txtPath, txtBuf, {
        conteudo: txtBuf.toString('utf8').split(/\r?\n/).filter(Boolean),
      })
    : null;

  return {
    wyt,
    msa,
    snBin,
    serverlist,
    serverlistTxt,
    resumo: {
      wytAmostras: wyt.length,
      wytReconhecidas: wyt.filter((w) => w.recognised).length,
      msaAmostras: msa.length,
      msaComVersao: msa.filter((m) => m.version === 0x112).length,
      serverlistInterpretado: Boolean(serverlist && serverlist.interpreted),
    },
  };
}
