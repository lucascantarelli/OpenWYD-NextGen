#!/usr/bin/env node
// Verifica um frame MSG_Trade (0x0383) capturado e decide entre o layout
// natural do Basedef.h (156 B) e o packed usado pelo porte Go (154 B).
//
// Uso:
//   node tools/client/trade-layout.mjs <hex-do-frame-decodificado>
//   node tools/client/trade-layout.mjs --self-test
//
// O frame deve estar com o Type claro (o mesmo ponto em que o servidor já
// desofuscou o pacote). Size/KeyWord/CheckSum continuam legíveis no header.
// Nada é gravado: a saída serve para anexar à captura da CLIENT-10.

const TYPE_TRADE = 0x0383;
const MAX_TRADE = 15;
const ITEM_SIZE = 8;
const MAX_USER = 1000;
const MAX_MONEY = 2_000_000_000;

// O C++ de Basedef.h:2435-2445 não está dentro de #pragma pack(1):
// sizeof(MSG_Trade) = 156, com padding em 147 e 153. O Go usa pack(1):
// 154 B, sem padding. A captura decide qual o cliente 7662 envia.
const LAYOUTS = new Map([
  [
    156,
    {
      name: 'natural',
      source: 'Basedef.h:2435-2445 (sem pack)',
      money: 148,
      check: 152,
      opponent: 154,
      padding: [147, 153],
    },
  ],
  [
    154,
    {
      name: 'packed',
      source: 'porte Go MsgTradeBodySize (pack(1))',
      money: 147,
      check: 151,
      opponent: 152,
      padding: [],
    },
  ],
]);

function fail(message) {
  throw new Error(message);
}

function decodeItem(frame, offset) {
  const index = frame.readInt16LE(offset);
  const effects = [];
  for (let i = 0; i < 3; i += 1) {
    effects.push({ effect: frame[offset + 2 + i * 2], value: frame[offset + 3 + i * 2] });
  }
  return { index, effects };
}

function parse(frame) {
  if (frame.length < 12) fail(`frame curto: ${frame.length} B (mínimo 12)`);
  const size = frame.readUInt16LE(0);
  const type = frame.readUInt16LE(4);
  const keyword = frame[2];
  const checksum = frame[3];
  if (size !== frame.length) {
    fail(`Size=${size} não bate com os ${frame.length} B recebidos`);
  }
  if (type !== TYPE_TRADE) {
    fail(`Type=0x${type.toString(16)}; esperado 0x0383 (MSG_Trade)`);
  }
  const layout = LAYOUTS.get(size);
  if (!layout) {
    fail(
      `Size=${size}: nenhum layout conhecido (natural=156, packed=154). ` +
        'Não invente offsets; registre a captura bruta.',
    );
  }

  const items = [];
  const slots = [];
  for (let i = 0; i < MAX_TRADE; i += 1) {
    const offset = 12 + i * ITEM_SIZE;
    const item = decodeItem(frame, offset);
    const slot = frame.readUInt8(132 + i);
    const position = slot === 0xff ? -1 : slot;
    if (item.index !== 0) {
      items.push({ index: i, position, item });
      slots.push(position);
    }
  }

  const money = frame.readInt32LE(layout.money);
  const myCheck = frame.readUInt8(layout.check);
  const opponent = frame.readUInt16LE(layout.opponent);
  const padding = layout.padding.map((offset) => ({ offset, value: frame[offset] }));
  const duplicated = slots.filter((slot, index) => slots.indexOf(slot) !== index);
  const errors = [];
  if (money < 0 || money > MAX_MONEY) errors.push(`TradeMoney fora da faixa: ${money}`);
  if (myCheck !== 0 && myCheck !== 1) errors.push(`MyCheck deve ser 0/1: ${myCheck}`);
  if (opponent <= 0 || opponent >= MAX_USER) {
    errors.push(`OpponentID fora de 1..${MAX_USER - 1}: ${opponent}`);
  }
  for (const { position } of items) {
    if (position < 0 || position >= 64) errors.push(`InvenPos inválido: ${position}`);
  }
  if (duplicated.length) errors.push(`InvenPos repetido: ${[...new Set(duplicated)].join(', ')}`);

  return {
    size,
    layout: layout.name,
    source: layout.source,
    keyword,
    checksum,
    money,
    myCheck,
    opponent,
    padding,
    items,
    duplicatedPositions: [...new Set(duplicated)],
    errors,
  };
}

function buildFrame({ size, keyword = 7, checksum = 0, money = 1234, myCheck = 1, opponent = 2 }) {
  const layout = LAYOUTS.get(size);
  if (!layout) fail(`sem layout para ${size}`);
  const frame = Buffer.alloc(size);
  frame.writeUInt16LE(size, 0);
  frame[2] = keyword;
  frame[3] = checksum;
  frame.writeUInt16LE(TYPE_TRADE, 4);
  frame.writeUInt16LE(1, 6);
  // Item 0: index 861 (Adaga), efeito 0/0.
  frame.writeInt16LE(861, 12);
  frame.writeUInt8(16, 132); // InvenPos[0]
  frame.writeInt32LE(money, layout.money);
  frame.writeUInt8(myCheck, layout.check);
  frame.writeUInt16LE(opponent, layout.opponent);
  return frame;
}

function selfTest() {
  for (const size of [156, 154]) {
    const parsed = parse(buildFrame({ size }));
    if (parsed.size !== size) fail(`self-test ${size}: Size não preservado`);
    if (parsed.layout !== LAYOUTS.get(size).name) fail(`self-test ${size}: layout errado`);
    if (parsed.money !== 1234 || parsed.myCheck !== 1 || parsed.opponent !== 2) {
      fail(`self-test ${size}: campos não decodificados`);
    }
    if (parsed.items.length !== 1 || parsed.items[0].item.index !== 861) {
      fail(`self-test ${size}: item não decodificado`);
    }
    if (parsed.items[0].position !== 16) fail(`self-test ${size}: InvenPos não decodificado`);
    if (parsed.errors.length) fail(`self-test ${size}: ${parsed.errors.join('; ')}`);
  }
  console.log('ok: layouts 156 (natural) e 154 (packed) decodificados sem ambiguidade');
}

const args = process.argv.slice(2);
try {
  if (args.includes('--self-test')) {
    selfTest();
  } else if (args.length !== 1 || args[0].startsWith('--')) {
    console.error('uso: node tools/client/trade-layout.mjs <hex-do-frame-decodificado>');
    console.error('     node tools/client/trade-layout.mjs --self-test');
    process.exitCode = 2;
  } else {
    const hex = args[0].replace(/^0x/i, '').replace(/\s+/g, '');
    if (!/^[0-9a-f]+$/i.test(hex) || hex.length % 2 !== 0) fail('hex inválido');
    const result = parse(Buffer.from(hex, 'hex'));
    console.log(JSON.stringify(result, null, 2));
  }
} catch (error) {
  console.error(`erro: ${error.message}`);
  process.exitCode = 1;
}
