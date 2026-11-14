// Testes da própria ferramenta: sem eles, o harness seria mais uma caixa-preta.
import assert from "node:assert/strict";
import { mkdtempSync, readFileSync } from "node:fs";
import net from "node:net";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";

import { FrameSplitter, INITCODE, readSession, replaySession, startProxy } from "./lib.mjs";

function frame(size, marca = 0) {
  const buffer = Buffer.alloc(size, marca);
  buffer.writeUInt16LE(size, 0);
  return buffer;
}

async function servidorFalso(tamanhoResposta) {
  const servidor = net.createServer((socket) => {
    const splitter = new FrameSplitter({ initcode: true });
    socket.on("data", (chunk) => {
      for (const item of splitter.push(chunk)) {
        if (item.frame) socket.write(frame(tamanhoResposta, 7));
      }
    });
    socket.on("error", () => {});
  });
  await new Promise((resolve) => servidor.listen(0, "127.0.0.1", resolve));
  return { servidor, porta: servidor.address().port };
}

test("splitter delimita INITCODE e frames partidos", () => {
  const splitter = new FrameSplitter({ initcode: true });
  const um = frame(12, 1);
  const dois = frame(13, 2);
  const bytes = Buffer.concat([INITCODE, um, dois]);
  const itens = [...splitter.push(bytes.subarray(0, 7)), ...splitter.push(bytes.subarray(7))];
  assert.equal(itens[0].initcode, true);
  const frames = itens.filter((item) => item.frame).map((item) => item.frame.length);
  assert.deepEqual(frames, [12, 13]);
});

test("splitter recusa Size fora da faixa", () => {
  const splitter = new FrameSplitter();
  const ruim = Buffer.alloc(12);
  ruim.writeUInt16LE(9000, 0);
  assert.throws(() => splitter.push(ruim), /Size fora da faixa/);
});

test("proxy grava as duas direções e o replay compara tamanhos", async () => {
  const dir = mkdtempSync(join(tmpdir(), "harness-"));
  const saida = join(dir, "sessao.jsonl");
  const { servidor, porta } = await servidorFalso(20);
  const proxy = await startProxy({ targetHost: "127.0.0.1", targetPort: porta, outPath: saida });

  const cliente = net.connect({ host: "127.0.0.1", port: proxy.port });
  await new Promise((resolve) => cliente.once("connect", resolve));
  cliente.write(Buffer.concat([INITCODE, frame(12), frame(13)]));
  await new Promise((r) => setTimeout(r, 150));
  cliente.destroy();
  await new Promise((r) => setTimeout(r, 100));
  await proxy.close();

  const linhas = readFileSync(saida, "utf8").trim().split("\n").map((l) => JSON.parse(l));
  assert.equal(linhas[0].hex, INITCODE.toString("hex"));
  assert.equal(linhas[0].init, true);
  assert.deepEqual(
    linhas
      .filter((l) => l.dir === "C" && !l.init)
      .map((l) => l.size),
    [12, 13],
  );
  assert.ok(linhas.some((l) => l.dir === "S" && l.size === 20), "resposta S→C gravada");

  const frames = readSession(saida);
  const bom = await replaySession({ frames, host: "127.0.0.1", port: porta, idleMs: 150 });
  assert.deepEqual(bom.divergencias, []);

  const outro = await servidorFalso(30);
  const ruim = await replaySession({ frames, host: "127.0.0.1", port: outro.porta, idleMs: 150 });
  assert.match(ruim.divergencias.join("\n"), /size esperado 20, recebido 30/);

  await new Promise((r) => servidor.close(r));
  await new Promise((r) => outro.servidor.close(r));
});
