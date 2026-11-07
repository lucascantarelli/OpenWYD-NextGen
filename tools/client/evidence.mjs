#!/usr/bin/env node
// Evidência dos endereços herdados do ClientPatch/CPSock.
// A classificação usa duas provas independentes: bytes no endereço e fronteira de
// instrução no desassemble (objdump). Nunca promove hipótese a fato.
import { execFileSync } from 'node:child_process';
import crypto from 'node:crypto';
import { loadKeywordTable } from './keytable.mjs';
import { offsetToVa, sectionOfOffset, vaToOffset } from './pe.mjs';

export const ANCHORS = [
  { name: 'ReadMessage', va: 0x4252d6, role: 'hook de recebimento: todo pacote antes do parse' },
  { name: 'SendChat', va: 0x4676c5, role: 'hook de chat digitado' },
  { name: 'SendPack', va: 0x40126a, role: 'envio pelo socket do cliente' },
  { name: 'SendPacket', va: 0x54da23, role: 'função de envio original (ponteiro fixo)' },
  { name: 'AddVolatileMessageItem', va: 0x422007, role: 'hook de item com confirmação' },
  { name: 'SetVolatileMessageBoxText', va: 0x41fb30, role: 'hook de texto de diálogo' },
];

export const PATCH_SITES = [
  { va: 0x53ac6a, role: 'desliga a 1ª checagem de checksum' },
  { va: 0x53ad52, role: 'desliga a 2ª checagem de checksum' },
  { va: 0x53ae7e, role: 'desliga a 3ª checagem de checksum' },
];

const PROLOGUES = [
  { bytes: [0x55], label: 'push ebp' },
  { bytes: [0x8b, 0xff], label: 'mov edi, edi' },
  { bytes: [0x53], label: 'push ebx' },
  { bytes: [0x56], label: 'push esi' },
  { bytes: [0x57], label: 'push edi' },
  { bytes: [0x83, 0xec], label: 'sub esp, imm8' },
  { bytes: [0x81, 0xec], label: 'sub esp, imm32' },
  { bytes: [0x6a], label: 'push imm8' },
  { bytes: [0x68], label: 'push imm32' },
  { bytes: [0xe9], label: 'jmp rel32 (thunk)' },
];

const hex32 = (value) => `0x${value.toString(16)}`;

function hex(buf, offset, length) {
  return buf
    .subarray(offset, offset + length)
    .toString('hex')
    .replace(/(..)/g, '$1 ')
    .trim();
}

function matchPrologue(buf, offset) {
  return PROLOGUES.find((p) => p.bytes.every((b, i) => buf[offset + i] === b)) || null;
}

/** Extrai { va → { bytes, mnemonic } } das linhas do objdump. */
function parseInstructions(text) {
  const instructions = new Map();
  for (const line of text.split(/\r?\n/)) {
    const match = /^\s*([0-9a-f]+):\t([0-9a-f ]+)\t(.+)$/.exec(line);
    if (!match) continue;
    instructions.set(parseInt(match[1], 16), {
      bytes: match[2].trim(),
      mnemonic: match[3].trim(),
    });
  }
  return instructions;
}

function occurrences(buf, pe, pattern, cap = 25) {
  const found = [];
  let from = 0;
  for (;;) {
    const at = buf.indexOf(pattern, from);
    if (at < 0 || found.length >= cap) break;
    const section = sectionOfOffset(pe, at);
    found.push({ offset: at, va: offsetToVa(pe, at), section: section ? section.name : null });
    from = at + 1;
  }
  return found;
}

function anchorReport(buf, pe, anchor, instructions, disassembly) {
  const offset = vaToOffset(pe, anchor.va);
  const base = { name: anchor.name, va: anchor.va, vaHex: hex32(anchor.va), role: anchor.role };
  if (offset === null) {
    return { ...base, status: 'NÃO LOCALIZADO', reason: 'VA fora das seções com bytes no arquivo' };
  }
  const section = sectionOfOffset(pe, offset);
  const prologue = matchPrologue(buf, offset);
  const instruction = instructions.get(anchor.va);
  const boundary = Boolean(instruction);
  const status = boundary
    ? 'CONFIRMADO'
    : disassembly.available
      ? 'REFUTADO'
      : 'NÃO VERIFICADO (sem desassemble)';
  return {
    ...base,
    offset,
    section: section ? section.name : null,
    bytes: hex(buf, offset, 16),
    instruction: instruction ? instruction.mnemonic : null,
    instructionBoundary: boundary,
    functionEntry: Boolean(prologue),
    prologue: prologue ? prologue.label : null,
    status,
    reason: boundary
      ? instruction.mnemonic.startsWith('j')
        ? `fronteira de instrução confirmada; o endereço é um desvio interno que o hook reescreve, não a entrada da função`
        : `fronteira de instrução confirmada; ${prologue ? 'prólogo de função' : 'instrução interna (ponto de patch/hook)'}`
      : 'o endereço não é fronteira de instrução no desassemble',
    disassembly: disassembly.text,
    disassemblyAvailable: disassembly.available,
  };
}

function patchReport(buf, pe, site, instructions, disassembly) {
  const offset = vaToOffset(pe, site.va);
  const base = { va: site.va, vaHex: hex32(site.va), role: site.role };
  if (offset === null) return { ...base, status: 'NÃO LOCALIZADO', reason: 'VA fora do arquivo' };
  const b0 = buf[offset];
  const b1 = buf[offset + 1];
  const section = sectionOfOffset(pe, offset);
  const instruction = instructions.get(site.va);

  let shape;
  let alreadyPatched = false;
  let status;
  let note;
  if (b0 === 0xeb) {
    shape = 'jmp rel8 (EB)';
    alreadyPatched = true;
    status = instruction ? 'CONFIRMADO' : 'NÃO VERIFICADO (sem desassemble)';
    note = 'o binário em disco já contém o salto incondicional: a checagem foi desativada antes da distribuição';
  } else if (b0 === 0x74) {
    shape = 'jz rel8 (74)';
    status = instruction ? 'CONFIRMADO' : 'NÃO VERIFICADO (sem desassemble)';
    note = 'salto condicional intacto; o patch troca 0x74 por 0xEB preservando o deslocamento';
  } else if (b0 === 0x0f && b1 === 0x84) {
    shape = 'jz rel32 (0F 84)';
    status = instruction ? 'CONFIRMADO' : 'NÃO VERIFICADO (sem desassemble)';
    note = 'salto longo; patch de 1 byte exigiria reescrita maior';
  } else {
    shape = `0x${b0.toString(16)}`;
    status = 'REFUTADO';
    note = 'não é salto condicional nem incondicional curto';
  }

  return {
    ...base,
    offset,
    section: section ? section.name : null,
    bytes: hex(buf, offset, 8),
    instruction: instruction ? instruction.mnemonic : null,
    instructionBoundary: Boolean(instruction),
    shape,
    alreadyPatched,
    status,
    note,
    disassembly: disassembly.text,
  };
}

function cpsockReport(buf, pe) {
  const table = loadKeywordTable();
  const initcode = Buffer.from([0x11, 0xf3, 0x11, 0x1f]);
  const initcodeData = occurrences(buf, pe, initcode);
  const initcodePush = occurrences(buf, pe, Buffer.from([0x68, 0x11, 0xf3, 0x11, 0x1f]));
  const initcodeMov = [];
  for (let reg = 0xb8; reg <= 0xbf; reg += 1) {
    initcodeMov.push(...occurrences(buf, pe, Buffer.from([reg, 0x11, 0xf3, 0x11, 0x1f]), 4));
  }
  const tableHits = occurrences(buf, pe, table, 4);

  const imm = [0x00, 0x20, 0x00, 0x00];
  const sizePatterns = [
    { name: 'cmp eax, 0x2000', bytes: [0x3d, ...imm] },
    { name: 'push 0x2000', bytes: [0x68, ...imm] },
    ...['c0', 'c8', 'd0', 'd8', 'e0', 'e8', 'f0', 'f8'].map((modrm) => ({
      name: `cmp reg, 0x2000 (81 /7 ${modrm})`,
      bytes: [0x81, parseInt(modrm, 16), ...imm],
    })),
  ];
  const sizeLimit = sizePatterns.flatMap((pattern) =>
    occurrences(buf, pe, Buffer.from(pattern.bytes), 5).map((hit) => ({ ...hit, pattern: pattern.name })),
  );

  const tableHash = crypto.createHash('sha256').update(table).digest('hex');
  const codeRefs = initcodePush.length + initcodeMov.length;
  const confirmed = tableHits.length > 0 && initcodeData.length > 0;

  return {
    header: {
      sizeBytes: 12,
      layout: 'Size(2) KeyWord(1) CheckSum(1) Type(2) ID(2) ClientTick(4)',
      status: confirmed ? 'CONFIRMADO INDIRETO' : 'REFUTADO',
      evidence: confirmed
        ? 'tabela de 512 bytes embutida byte-idêntica + constante INITCODE presente no binário'
        : 'tabela e/ou INITCODE ausentes',
      pending: 'offsets de campo e a leitura de Size por instrução ficam para 08-6 (trace dinâmico)',
    },
    keyTable: {
      sha256: tableHash,
      occurrences: tableHits,
      status: tableHits.length > 0 ? 'CONFIRMADO' : 'REFUTADO',
    },
    initcode: {
      value: '0x1F11F311',
      dataOccurrences: initcodeData,
      codeReferences: [...initcodePush, ...initcodeMov],
      status:
        codeRefs > 0
          ? 'CONFIRMADO (referência em código)'
          : initcodeData.length > 0
            ? 'CONFIRMADO (dado)'
            : 'REFUTADO',
    },
    sizeLimit: {
      value: 8192,
      note: 'MAX_MESSAGE_SIZE do CPSock; os padrões abaixo são candidatos a checagem de enquadramento',
      candidates: sizeLimit,
    },
  };
}

export function disassemble(exePath, va, after = 48) {
  // Objdump começa a decodificar no endereço pedido; se ele cair no meio de uma
  // instrução, o fluxo desalinha e o endereço-alvo nunca aparece. Por isso tentamos
  // janelas com deslocamentos diferentes até achar o VA como início de instrução.
  let fallback = null;
  for (const before of [16, 8, 4, 12, 20, 24, 32, 40, 48, 64, 96]) {
    const start = Math.max(0, va - before);
    const range = [
      `--start-address=0x${start.toString(16)}`,
      `--stop-address=0x${(va + after).toString(16)}`,
    ];
    const attempts = [
      ['-d', '-M', 'intel', ...range, exePath],
      ['-d', ...range, exePath],
    ];
    for (const args of attempts) {
      try {
        const text = execFileSync('objdump', args, {
          encoding: 'utf8',
          windowsHide: true,
          maxBuffer: 8 << 20,
        })
          .trim()
          .replace(/\r\n/g, '\n');
        const aligned = parseInstructions(text).has(va);
        if (aligned) return { available: true, aligned, text: text.slice(0, 4000) };
        fallback = fallback || { available: true, aligned: false, text: text.slice(0, 4000) };
      } catch {
        // tenta a próxima forma
      }
    }
  }
  return fallback || { available: false, aligned: false, text: 'objdump ausente ou incapaz de ler o PE' };
}

export function analyseEvidence({ buf, pe, exePath, withDisassembly = true }) {
  const anchors = ANCHORS.map((anchor) => {
    const disassembly = withDisassembly
      ? disassemble(exePath, anchor.va)
      : { available: false, text: 'desassemble desativado' };
    return anchorReport(buf, pe, anchor, parseInstructions(disassembly.text), disassembly);
  });

  const patchSites = PATCH_SITES.map((site) => {
    const disassembly = withDisassembly
      ? disassemble(exePath, site.va)
      : { available: false, text: 'desassemble desativado' };
    return patchReport(buf, pe, site, parseInstructions(disassembly.text), disassembly);
  });

  return {
    build: { format: 'PE32', machine: 'i386', imageBase: hex32(pe.imageBase) },
    anchors,
    patchSites,
    cpsock: cpsockReport(buf, pe),
    summary: {
      anchorsConfirmed: anchors.filter((a) => a.status === 'CONFIRMADO').length,
      anchorsFunctionEntry: anchors.filter((a) => a.functionEntry).length,
      anchorsTotal: anchors.length,
      patchesConfirmed: patchSites.filter((p) => p.status === 'CONFIRMADO').length,
      patchesAlreadyPatched: patchSites.filter((p) => p.alreadyPatched).length,
      patchesTotal: patchSites.length,
    },
  };
}
