#!/usr/bin/env node
// CONGELADO (2026-11): tabelas versionadas em `crates/wyd-protocol/src/wire/generated/`
// e catálogo em `docs/arquitetura/protocolo.md` §4 são a fonte canônica atual.
// Fonte histórica: Basedef.h v7662 + CPSock.cpp (arquivo Go/C++ removido do repo).
// Este script é mantido como documentação do processo de geração; NÃO regenera
// sem ADR (falha explícita se a fonte histórica não existir).
//
// Uso (somente conferência): node tools/protocol/gen-protocol.mjs --check
import fs from 'node:fs';
import path from 'node:path';

const root = process.cwd();
const BASEDEF = path.join(root, 'docs/cliente/gerado/Basedef.h');
const CPSOCK = path.join(root, 'docs/cliente/gerado/CPSock.cpp');
const OUT_OPCODES = path.join(root, 'crates/wyd-protocol/src/wire/generated/opcode_table.rs');
const OUT_KEYWORD = path.join(root, 'crates/wyd-protocol/src/codec/generated/keyword_table.rs');
const OUT_DOC = path.join(root, 'docs/arquitetura/protocolo.md');

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
  const re = /const\s+(?:short|int)\s+(_MSG_\w+)\s*=\s*([^;]+);/g;
  const entries = [];
  let match;
  while ((match = re.exec(source)) !== null) {
    const [, constName, rawExpr] = match;
    const expr = rawExpr.trim().replace(/^\(/, '').replace(/\)$/, '');
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
//! Fonte canônica: \`docs/arquitetura/protocolo.md\` §4 (${entries.length} constantes \`_MSG_*\`, tabelas congeladas; fonte histórica Basedef.h v7662).
//! Gerador arquivado: \`tools/protocol/gen-protocol.mjs\` (congelado; ver ADR).
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
//! Fonte canônica: \`docs/arquitetura/protocolo.md\` §3 (tabela \`pKeyWord[512]\` congelada; fonte histórica CPSock.cpp v7662).
//! Gerador: \`tools/protocol/gen-protocol.mjs\`. Bytes pares = semente (\`KeyWord\`),
//! ímpares = byte de transformação (\`Trans\`), como em \`docs/arquitetura/protocolo.md\`.

pub const KEY_TABLE: [u8; 512] = [
${rustArray(values)}
];
`;
}

const FLOW_ARROW = {
  Game2Client: 'J→C',
  Client2Game: 'C→J',
  Db2Game: 'D→J',
  Game2Db: 'J→D',
  Db2Np: 'D→N',
  Np2Db: 'N→D',
  New: 'novo',
};

function emitCatalog(opcodes) {
  const rows = opcodes.entries.map((e) => {
    const dirs = e.flows.map((f) => FLOW_ARROW[f]).join(', ') || '—';
    return `| \`${e.constName}\` | 0x${e.value.toString(16).padStart(4, '0').toUpperCase()} | ${dirs} |`;
  });
  return `<!-- GERADO:catalogo-opcodes (tools/protocol/gen-protocol.mjs de Basedef.h) -->
| Constante | Fio | Direção |
|---|---|---|
${rows.join('\n')}
<!-- /GERADO:catalogo-opcodes -->`;
}

function spliceCatalog(doc, catalog) {
  const start = '<!-- GERADO:catalogo-opcodes';
  const end = '<!-- /GERADO:catalogo-opcodes -->';
  const i = doc.indexOf(start);
  const j = doc.indexOf(end);
  if (i < 0 || j < 0) throw new Error('protocolo.md sem marcadores GERADO do catálogo');
  return doc.slice(0, i) + catalog + doc.slice(j + end.length);
}

if (!fs.existsSync(BASEDEF) || !fs.existsSync(CPSOCK)) {
  console.log('gen-protocol: fonte histórica ausente (arquivo Go/C++ removido em 2026-11).');
  console.log('Tabelas congeladas em crates/wyd-protocol/src/*/generated/ + docs/arquitetura/protocolo.md §4 são canônicas. Nada a regenerar.');
  process.exit(0);
}
fs.mkdirSync(path.dirname(OUT_OPCODES), { recursive: true });
fs.mkdirSync(path.dirname(OUT_KEYWORD), { recursive: true });
const opcodes = parseOpcodes();
fs.writeFileSync(OUT_OPCODES, emitOpcodes(opcodes));
fs.writeFileSync(OUT_KEYWORD, emitKeyword());
console.log(`opcode_table.rs: ${opcodes.entries.length} opcodes (${opcodes.unique.length} valores únicos, ${opcodes.duplicates.length} repetidos)`);
console.log('keyword_table.rs: 512 bytes');
if (fs.existsSync(OUT_DOC)) {
  const doc = fs.readFileSync(OUT_DOC, 'utf8');
  fs.writeFileSync(OUT_DOC, spliceCatalog(doc, emitCatalog(opcodes)));
  console.log('protocolo.md: catálogo atualizado');
} else {
  console.log('protocolo.md ainda não existe: catálogo não aplicado');
}
