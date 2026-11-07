#!/usr/bin/env node
// Gera `crates/wyd-protocol/src/opcodes.rs` e `crates/wyd-protocol/src/keyword.rs`
// a partir das fontes C++ arquivadas (fonte canônica):
//   - .deprecated-go/Source/Code/Basedef.h  (198 constantes `_MSG_*` + FLAG_*)
//   - .deprecated-go/Source/Code/CPSock.cpp (tabela `pKeyWord[512]`)
//
// Uso: node tools/protocol/gen-protocol.mjs
// Regenerar só quando a fonte canônica mudar; o resultado é versionado.
import fs from 'node:fs';
import path from 'node:path';

const root = process.cwd();
const BASEDEF = path.join(root, '.deprecated-go/Source/Code/Basedef.h');
const CPSOCK = path.join(root, '.deprecated-go/Source/Code/CPSock.cpp');
const OUT_DIR = path.join(root, 'crates/wyd-protocol/src');

const FLAGS = [
  ['Game2Client', 'FLAG_GAME2CLIENT', 0x0100],
  ['Client2Game', 'FLAG_CLIENT2GAME', 0x0200],
  ['Db2Game', 'FLAG_DB2GAME', 0x0400],
  ['Game2Db', 'FLAG_GAME2DB', 0x0800],
  ['Db2Np', 'FLAG_DB2NP', 0x1000],
  ['Np2Db', 'FLAG_NP2DB', 0x2000],
  ['New', 'FLAG_NEW', 0x4000],
];
const flagValue = new Map(FLAGS.map(([, c, v]) => [c, v]));
const flagName = new Map(FLAGS.map(([n, c]) => [c, n]));

function parseOpcodes() {
  const source = fs.readFileSync(BASEDEF, 'utf8');
  const re = /const\s+(?:short|int)\s+(_MSG_\w+)\s*=\s*\(([^)]*)\)\s*;/g;
  const entries = [];
  let match;
  while ((match = re.exec(source)) !== null) {
    const [, constName, expr] = match;
    let value = 0;
    const flows = new Set();
    for (const raw of expr.split('|')) {
      const token = raw.trim();
      if (/^0x[0-9a-f]+$/i.test(token)) value |= parseInt(token, 16);
      else if (/^\d+$/.test(token)) value |= parseInt(token, 10);
      else if (flagValue.has(token)) {
        value |= flagValue.get(token);
        flows.add(flagName.get(token));
      } else throw new Error(`token desconhecido "${token}" em ${constName}`);
    }
    entries.push({
      constName,
      variant: constName
        .replace(/^_MSG_/, '')
        .split('_')
        .filter(Boolean)
        .map((w) => w[0].toUpperCase() + w.slice(1))
        .join(''),
      value,
      flows: FLAGS.map(([n]) => n).filter((n) => flows.has(n)),
    });
  }
  entries.sort((a, b) => a.value - b.value || a.constName.localeCompare(b.constName));

  const seenVariant = new Map();
  for (const e of entries) {
    if (seenVariant.has(e.variant)) {
      throw new Error(`variante duplicada ${e.variant}: ${seenVariant.get(e.variant)} / ${e.constName}`);
    }
    seenVariant.set(e.variant, e.constName);
  }

  const byValue = new Map();
  const duplicates = [];
  for (const e of entries) {
    if (byValue.has(e.value)) duplicates.push([e.value, e.constName, byValue.get(e.value).constName]);
    else byValue.set(e.value, e);
  }
  return { entries, unique: [...byValue.values()], duplicates };
}

function rustArray(values, perLine = 16) {
  const lines = [];
  for (let i = 0; i < values.length; i += perLine) {
    lines.push('    ' + values.slice(i, i + perLine).map((v) => `0x${v.toString(16).padStart(2, '0')}`).join(', ') + ',');
  }
  return lines.join('\n');
}

function emitOpcodes({ entries, unique, duplicates }) {
  const header = `//! Catálogo de \`Type\` do protocolo CPSock — GERADO, não editar à mão.
//!
//! Fonte canônica: \`.deprecated-go/Source/Code/Basedef.h\` (${entries.length} constantes \`_MSG_*\`).
//! Gerador: \`tools/protocol/gen-protocol.mjs\`.
//! Especificação: \`docs/arquitetura/protocolo.md\` + \`.deprecated-go/docs/migration/protocol-spec.md\`.
//!
//! O valor de fio inclui os bits de direção (\`FLAG_*\`); \`Opcode::wire()\` devolve o
//! valor exato que trafega no \`HEADER.Type\`.

`;

  const flags = FLAGS.map(([n, c, v]) => `pub const ${c}: u16 = 0x${v.toString(16).padStart(4, '0')};`).join('\n');

  const variants = entries.map((e) => `    ${e.variant},`).join('\n');
  const all = entries.map((e) => `    Opcode::${e.variant},`).join('\n');
  const wireArms = entries.map((e) => `            Opcode::${e.variant} => 0x${e.value.toString(16).padStart(4, '0')},`).join('\n');
  const nameArms = entries.map((e) => `            Opcode::${e.variant} => "${e.constName}",`).join('\n');
  const flowArms = entries
    .map((e) => {
      const arms = e.flows.map((f) => `Flow::${f}`).join(', ');
      return `            Opcode::${e.variant} => &[${arms}],`;
    })
    .join('\n');
  const fromArms = unique.map((e) => `            0x${e.value.toString(16).padStart(4, '0')} => Some(Opcode::${e.variant}),`).join('\n');
  const dupComment = duplicates.length
    ? `// Valores de fio repetidos na fonte (o primeiro nome vence em \`from_wire\`):\n` +
      duplicates.map(([v, a, b]) => `//   0x${v.toString(16).padStart(4, '0')}: ${a} (já mapeado por ${b})`).join('\n') +
      '\n\n'
    : '';

  return `${header}${dupComment}${flags}

/// Direção de tráfego declarada nos bits \`FLAG_*\` do \`Type\`.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum Flow {
    Game2Client,
    Client2Game,
    Db2Game,
    Game2Db,
    Db2Np,
    Np2Db,
    New,
}

/// Identificador de mensagem do protocolo (valor de fio já com os bits de direção).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, PartialOrd, Ord)]
pub enum Opcode {
${variants}
}

impl Opcode {
    /// Todas as constantes catalogadas, em ordem de valor de fio.
    pub const ALL: &'static [Opcode] = &[
${all}
    ];

    /// Valor exato transmitido em \`HEADER.Type\`.
    #[must_use]
    pub const fn wire(self) -> u16 {
        match self {
${wireArms}
        }
    }

    /// Nome da constante original em \`Basedef.h\` (rastreabilidade com o legado).
    #[must_use]
    pub const fn const_name(self) -> &'static str {
        match self {
${nameArms}
        }
    }

    /// Direções declaradas na constante original.
    #[must_use]
    pub const fn flows(self) -> &'static [Flow] {
        match self {
${flowArms}
        }
    }

    /// Resolve um valor de fio para a constante correspondente.
    #[must_use]
    pub fn from_wire(value: u16) -> Option<Opcode> {
        match value {
${fromArms}
            _ => None,
        }
    }
}
`;
}

function emitKeyword() {
  const source = fs.readFileSync(CPSOCK, 'utf8');
  const table = source.match(/pKeyWord\s*\[\s*512\s*\]\s*=\s*\{([\s\S]*?)\};/);
  if (!table) throw new Error('pKeyWord[512] não encontrada em CPSock.cpp');
  const values = [...table[1].matchAll(/0x([0-9a-fA-F]{1,2})/g)].map((m) => parseInt(m[1], 16));
  if (values.length !== 512) throw new Error(`pKeyWord tem ${values.length} bytes, esperado 512`);
  return `//! Tabela de ofuscação \`pKeyWord[512]\` — GERADO, não editar à mão.
//!
//! Fonte canônica: \`CPSock.cpp\` (\`pKeyWord[512] = { // 7.xx keys }\`).
//! Gerador: \`tools/protocol/gen-protocol.mjs\`. Bytes pares = semente (\`KeyWord\`),
//! ímpares = byte de transformação (\`Trans\`), como em \`docs/arquitetura/protocolo.md\`.

pub const KEY_TABLE: [u8; 512] = [
${rustArray(values)}
];
`;
}

fs.mkdirSync(OUT_DIR, { recursive: true });
const opcodes = parseOpcodes();
fs.writeFileSync(path.join(OUT_DIR, 'opcodes.rs'), emitOpcodes(opcodes));
fs.writeFileSync(path.join(OUT_DIR, 'keyword.rs'), emitKeyword());
console.log(`opcodes.rs: ${opcodes.entries.length} opcodes (${opcodes.unique.length} valores únicos, ${opcodes.duplicates.length} repetidos)`);
console.log('keyword.rs: 512 bytes');
