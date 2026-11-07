#!/usr/bin/env node
// Tabela `pKeyWord[512]` e transformação CPSock em JS — mesmas regras do crate
// `wyd-protocol` (`obfuscation.rs`): `Size/KeyWord/CheckSum` em claro, transformação
// de 4 até o fim, semente `KEY_TABLE[key_word*2]`, trans `KEY_TABLE[pos*2+1]`.
import fs from 'node:fs';

export const KEYWORD_OFFSET = 4;

/** Lê a tabela do arquivo gerado (fonte única, já validada pelo crate). */
export function loadKeywordTable(file = 'crates/wyd-protocol/src/keyword.rs') {
  const source = fs.readFileSync(file, 'utf8');
  const start = source.indexOf('KEY_TABLE');
  const end = source.indexOf('];', start);
  if (start < 0 || end < 0) throw new Error(`KEY_TABLE não encontrada em ${file}`);
  const bytes = [];
  for (const match of source.slice(start, end).matchAll(/0x([0-9a-fA-F]{2})/g)) {
    bytes.push(parseInt(match[1], 16));
  }
  if (bytes.length !== 512) throw new Error(`KEY_TABLE com ${bytes.length} bytes (esperado 512)`);
  return Buffer.from(bytes);
}

/**
 * Aplica a transformação CPSock a um buffer inteiro.
 * `from` é o offset do primeiro byte transformado (4 em pacote; 0 para arquivos de dados).
 */
export function transform(bytes, { keyWord, table, from = KEYWORD_OFFSET, direction = 'decode' }) {
  const out = Buffer.from(bytes);
  const seed = table[(keyWord & 0xff) * 2];
  for (let i = from; i < out.length; i += 1) {
    const trans = table[((seed + (i - from)) % 256) * 2 + 1];
    const cipher = out[i];
    const phase = i & 3;
    let plain;
    if (direction === 'decode') {
      plain =
        phase === 0
          ? cipher - ((trans << 1) & 0xff)
          : phase === 1
            ? cipher + (trans >> 3)
            : phase === 2
              ? cipher - ((trans << 2) & 0xff)
              : cipher + (trans >> 5);
    } else {
      plain =
        phase === 0
          ? cipher + ((trans << 1) & 0xff)
          : phase === 1
            ? cipher - (trans >> 3)
            : phase === 2
              ? cipher + ((trans << 2) & 0xff)
              : cipher - (trans >> 5);
    }
    out[i] = plain & 0xff;
  }
  return out;
}
