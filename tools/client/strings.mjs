#!/usr/bin/env node
// Índice de strings do binário: ASCII e UTF-16LE, com offset, VA, seção e categoria.
// Somente strings classificadas como relevantes entram no artefato versionado.
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import { parsePe, offsetToVa, sectionOfOffset } from './pe.mjs';

const CATEGORIES = [
  ['dados', /\.(bin|dat|wyt|msa|ani|bon|txt|log)\b/i],
  ['rede', /serv00\.htm|serverlist|sn\.bin|:8281|cpsock|initcode|socket|checksum/i],
  ['protecao', /xtrap|jprotect|jlock|nprotect/i],
  ['protocolo', /_msg_|opcode|packet|header|keyword/i],
  ['versao', /version|vers[ãa]o|patch|update/i],
  ['ui', /mesh\\|sound\\|music\\|interface|scene/i],
];

const DATA_FILE = /^[A-Za-z0-9_\\/.\- ]+\.(bin|wyt|msa|dat|ani|bon|txt)$/i;

export function classify(text) {
  const tags = [];
  for (const [tag, pattern] of CATEGORIES) if (pattern.test(text)) tags.push(tag);
  if (DATA_FILE.test(text)) tags.push('arquivo-de-dados');
  return tags;
}

export function extractStrings(buf, minLength = 5) {
  const found = [];

  let start = -1;
  for (let i = 0; i <= buf.length; i += 1) {
    const byte = i < buf.length ? buf[i] : 0;
    const printable = byte >= 0x20 && byte < 0x7f;
    if (printable) {
      if (start < 0) start = i;
    } else if (start >= 0) {
      if (i - start >= minLength) {
        found.push({ offset: start, encoding: 'ascii', text: buf.toString('latin1', start, i) });
      }
      start = -1;
    }
  }

  start = -1;
  for (let i = 0; i + 1 < buf.length; i += 1) {
    const byte = buf[i];
    const printable = byte >= 0x20 && byte < 0x7f && buf[i + 1] === 0;
    if (printable) {
      if (start < 0) start = i;
      i += 1;
    } else {
      if (start >= 0) {
        const chars = (i - start) / 2;
        if (chars >= minLength) {
          found.push({
            offset: start,
            encoding: 'utf16le',
            text: buf.toString('utf16le', start, i),
          });
        }
        start = -1;
      }
    }
  }

  return found.sort((a, b) => a.offset - b.offset);
}

export function indexStrings(buf, pe) {
  const all = extractStrings(buf, 5);
  const relevant = [];
  const seen = new Set();
  for (const item of all) {
    const tags = classify(item.text);
    if (!tags.length) continue;
    const key = `${item.encoding}:${item.text}`;
    if (seen.has(key)) continue;
    seen.add(key);
    const section = sectionOfOffset(pe, item.offset);
    relevant.push({
      offset: item.offset,
      va: offsetToVa(pe, item.offset),
      section: section ? section.name : null,
      encoding: item.encoding,
      text: item.text,
      tags,
    });
  }
  relevant.sort((a, b) => a.text.localeCompare(b.text) || a.offset - b.offset);
  return {
    totals: {
      all: all.length,
      ascii: all.filter((s) => s.encoding === 'ascii').length,
      utf16le: all.filter((s) => s.encoding === 'utf16le').length,
      relevant: relevant.length,
    },
    relevant: relevant.slice(0, 600),
  };
}

if (process.argv[1] && import.meta.url === pathToFileURL(path.resolve(process.argv[1])).href) {
  const file = process.argv[2] || 'client/WYD.exe';
  const buf = fs.readFileSync(file);
  console.log(JSON.stringify(indexStrings(buf, parsePe(buf)), null, 2));
}
