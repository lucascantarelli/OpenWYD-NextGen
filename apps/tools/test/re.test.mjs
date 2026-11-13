import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { test } from "node:test";

import { candidatosFuncao, candidatosVariaveis } from "../lib/pe-analise.mjs";
import { analisarPacotes, lerCatalogo } from "../lib/pacotes.mjs";
import { renderBinarios, renderPacotes } from "../lib/re-docs.mjs";

function projetoFalso() {
  const raiz = mkdtempSync(path.join(tmpdir(), "wyd-tools-re-"));
  mkdirSync(path.join(raiz, "crates/wyd-protocol/src/wire/generated"), { recursive: true });
  mkdirSync(path.join(raiz, "crates/wyd-protocol/src/payload"), { recursive: true });
  mkdirSync(path.join(raiz, "crates/wyd-server/src"), { recursive: true });
  writeFileSync(
    path.join(raiz, "crates/wyd-protocol/src/wire/generated/opcode_table.rs"),
    `fn wire(op: Opcode) -> u16 {\n    match op {\n        Opcode::Ping => 0x03a0,\n        Opcode::Pong => 0x03a1,\n        Opcode::ShopList => 0x017c,\n    }\n}\n`,
  );
  writeFileSync(
    path.join(raiz, "crates/wyd-server/src/action.rs"),
    `fn handle(op: Opcode) { match op { Opcode::Ping => {}, _ => {} } }\n`,
  );
  writeFileSync(
    path.join(raiz, "crates/wyd-protocol/src/payload/shop.rs"),
    `pub struct ShopList {\n    pub shop_type: i32,\n    pub tax: i32,\n}\n\nimpl WirePayload for ShopList {\n    const OPCODE: Opcode = Opcode::ShopList;\n    const WIRE_LEN: usize = 236;\n}\n`,
  );
  return raiz;
}

test("candidatosFuncao conta destinos de call rel32 e prólogos", () => {
  const buf = Buffer.alloc(0x100);
  // .text em offset 0 / VA 0x1000; call em 0x1000 -> 0x1010.
  buf[0] = 0xe8;
  buf.writeInt32LE(0x0b, 1); // 0x1000 + 5 + 0x0b = 0x1010
  // prólogo em 0x1020.
  buf.writeUInt8(0x55, 0x20);
  buf.writeUInt8(0x8b, 0x21);
  buf.writeUInt8(0xec, 0x22);
  const pe = {
    sections: [
      {
        name: ".text",
        virtualAddress: 0x1000,
        virtualSize: 0x100,
        sizeOfRawData: 0x100,
        pointerToRawData: 0,
        flags: { code: true },
      },
    ],
  };
  const r = candidatosFuncao(buf, pe);
  assert.equal(r.totalAlvos, 1);
  assert.deepEqual(r.alvos[0], { va: 0x1010, chamadas: 1 });
  assert.equal(r.prologos, 1);
  assert.equal(r.secao, ".text");
});

test("candidatosVariaveis acha referência absoluta a seção de dados", () => {
  const buf = Buffer.alloc(0x300);
  buf[0] = 0xa1; // mov eax, [0x2000]
  buf.writeUInt32LE(0x2000, 1);
  buf[5] = 0x89; // mov [0x2004], eax
  buf[6] = 0x05;
  buf.writeUInt32LE(0x2004, 7);
  const pe = {
    sections: [
      { name: ".text", virtualAddress: 0x1000, virtualSize: 0x100, sizeOfRawData: 0x100, pointerToRawData: 0, flags: { code: true } },
      { name: ".data", virtualAddress: 0x2000, virtualSize: 0x100, sizeOfRawData: 0x100, pointerToRawData: 0x200, flags: { code: false } },
    ],
  };
  const r = candidatosVariaveis(buf, pe);
  assert.equal(r.total, 2);
  assert.deepEqual(r.top[0], { va: 0x2000, secao: ".data", refs: 1 });
});

test("catálogo lê wire e direção pelos bits de FLAG", () => {
  const catalogo = lerCatalogo(projetoFalso());
  assert.equal(catalogo.length, 3);
  const ping = catalogo.find((o) => o.nome === "Ping");
  assert.equal(ping.wire, "0x03a0");
  assert.equal(ping.direcao, "S→C");
  assert.equal(catalogo.find((o) => o.nome === "ShopList").direcao, "S→C");
});

test("pacotes cruzam handler e estrutura sem inventar", () => {
  const analise = analisarPacotes(projetoFalso());
  const ping = analise.linhas.find((l) => l.nome === "Ping");
  assert.deepEqual(ping.handlers, ["crates/wyd-server/src/action.rs"]);
  const shop = analise.linhas.find((l) => l.nome === "ShopList");
  assert.equal(shop.estrutura.struct, "ShopList");
  assert.equal(shop.estrutura.wireLen, 236);
  assert.deepEqual(shop.estrutura.campos, ["shop_type: i32", "tax: i32"]);
  const pong = analise.linhas.find((l) => l.nome === "Pong");
  assert.equal(pong.handlers.length, 0);
  assert.equal(pong.estrutura, null);
  assert.equal(analise.totalSemHandler, 2);
});

test("render dos documentos é determinístico", () => {
  const analise = analisarPacotes(projetoFalso());
  assert.equal(renderPacotes(analise), renderPacotes(analise));
  assert.match(renderPacotes(analise), /198|3\*\* opcodes/);
  const bin = {
    caminho: "Fake.exe",
    arquitetura: "x86",
    subsystem: "gui",
    entryPoint: { rva: 1, rvaHex: "0x1", vaHex: "0x2" },
    imageBaseHex: "0x1",
    timestampIso: null,
    pdb: null,
    secoes: [{ nome: ".text", vaHex: "0x1000", virtualSize: 16, rawSize: 16 }],
    imports: [{ dll: "WS2_32.dll", total: 1, funcoes: ["send"] }],
    exports: { dllName: null, entradas: [] },
    strings: { totais: { all: 1, ascii: 1, utf16le: 0, relevant: 1 }, relevant: [{ offset: 1, va: 2, section: ".text", encoding: "ascii", tags: ["rede"], text: "a|b" }] },
    variaveis: { total: 1, top: [{ va: 0x2000, secao: ".data", refs: 1 }] },
    funcoes: { secao: ".text", prologos: 1, totalAlvos: 1, alvos: [{ va: 0x1010, chamadas: 2 }] },
  };
  assert.equal(renderBinarios([bin]), renderBinarios([bin]));
  assert.match(renderBinarios([bin]), /a\\\|b/);
});
