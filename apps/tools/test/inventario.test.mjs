import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { test } from "node:test";

import { inventariar, listarArquivos } from "../lib/inventario.mjs";
import { renderInventario } from "../lib/docs.mjs";

function clienteFalso() {
  const raiz = mkdtempSync(path.join(tmpdir(), "wyd-tools-"));
  mkdirSync(path.join(raiz, "Mesh"));
  writeFileSync(path.join(raiz, "AniSound4.txt"), "abc");
  writeFileSync(path.join(raiz, "Mesh", "a - Cópia.bin"), Buffer.alloc(2048, 7));
  writeFileSync(path.join(raiz, "WYD.exe"), "nao-e-pe");
  writeFileSync(path.join(raiz, "manifest.txt"), "build 1\n");
  writeFileSync(path.join(raiz, "serverlist.bin"), Buffer.alloc(7040));
  writeFileSync(path.join(raiz, "WYDLauncher.exe.bak"), "velho");
  return raiz;
}

test("ignora artefatos do launcher e conta só conteúdo do jogo", () => {
  const raiz = clienteFalso();
  assert.deepEqual(listarArquivos(raiz), ["AniSound4.txt", "Mesh/a - Cópia.bin", "WYD.exe"]);
  const inv = inventariar(raiz);
  assert.equal(inv.arquivos, 3);
  assert.equal(inv.bytes, 3 + 2048 + "nao-e-pe".length);
  assert.equal(inv.executaveis.length, 1);
  assert.equal(inv.executaveis[0].caminho, "WYD.exe");
  assert.ok(inv.executaveis[0].pe.erro, "PE inválido vira erro registrado, não exceção");
});

test("documento repete exatamente os números medidos", () => {
  const inv = inventariar(clienteFalso());
  const md = renderInventario(inv, { geradoEm: "2026-11-12T00:00:00.000Z" });
  assert.match(md, /3\*\* arquivos de jogo/);
  assert.ok(md.includes("`sn.bin`") && md.includes("serverlist.bin"));
  assert.ok(md.includes(inv.executaveis[0].sha256));
  assert.ok(md.includes("Mesh/a - Cópia.bin") || md.includes("Mesh"));
});
