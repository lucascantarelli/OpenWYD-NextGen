#!/usr/bin/env node
// Reenvia ao servidor os frames que o cliente real enviou e compara a resposta com a gravação.
//
// Uso:
//   node tools/cliente-harness/replay.mjs --session tmp/cliente/sessao.jsonl \
//     --target 127.0.0.1:8281 [--exact]
//
// Exit 0 = a sequência de respostas (tamanho; bytes com --exact) bate com a do cliente real.
// Exit 1 = divergência, com o primeiro frame quebrado no relatório.
import { readSession, replaySession } from "./lib.mjs";

function argumento(nome, padrao) {
  const indice = process.argv.indexOf(`--${nome}`);
  return indice === -1 ? padrao : process.argv[indice + 1];
}

const sessao = argumento("session", "tmp/cliente/sessao.jsonl");
const [host, porta] = argumento("target", "127.0.0.1:8281").split(":");
const exato = process.argv.includes("--exact");

const frames = readSession(sessao);
if (frames.filter((f) => f.dir === "C").length === 0) {
  console.error(`sessão sem frames C→S: ${sessao}`);
  process.exit(1);
}

const resultado = await replaySession({
  frames,
  host,
  port: Number(porta),
  exact: exato,
});

console.log(
  `replay: ${resultado.enviados} enviados | ${resultado.recebidos} recebidos | ${resultado.esperados} esperados`,
);
if (resultado.divergencias.length) {
  for (const linha of resultado.divergencias.slice(0, 20)) console.error(`  ✗ ${linha}`);
  console.error(`${resultado.divergencias.length} divergência(s)`);
  process.exit(1);
}
console.log("ok: resposta do servidor bate com a sessão gravada");
