#!/usr/bin/env node
// Grava uma sessão real do cliente 7662: cliente → proxy → canal.
//
// Uso:
//   node tools/cliente-harness/proxy.mjs --out tmp/cliente/sessao.jsonl \
//     --listen 127.0.0.1:8299 --target 127.0.0.1:8281
//
// Aponte o `serverlist.bin` do cliente para 127.0.0.1:8299 e jogue uma vez. O JSONL tem os
// frames das duas direções, com `Size` e hex — é o oráculo externo que falta nos testes.
import { mkdirSync } from "node:fs";
import { dirname } from "node:path";

import { startProxy } from "./lib.mjs";

function argumento(nome, padrao) {
  const indice = process.argv.indexOf(`--${nome}`);
  return indice === -1 ? padrao : process.argv[indice + 1];
}

function endereco(valor) {
  const [host, porta] = valor.split(":");
  if (!host || !porta) throw new Error(`endereço inválido: ${valor} (use ip:porta)`);
  return { host, port: Number(porta) };
}

const escuta = endereco(argumento("listen", "127.0.0.1:8299"));
const alvo = endereco(argumento("target", "127.0.0.1:8281"));
const saida = argumento("out", "tmp/cliente/sessao.jsonl");
mkdirSync(dirname(saida), { recursive: true });

const proxy = await startProxy({
  listenHost: escuta.host,
  listenPort: escuta.port,
  targetHost: alvo.host,
  targetPort: alvo.port,
  outPath: saida,
});
console.log(`proxy em ${escuta.host}:${proxy.port} → ${alvo.host}:${alvo.port}`);
console.log(`gravando em ${saida} (Ctrl+C encerra)`);

const encerrar = async () => {
  await proxy.close();
  console.log(`sessão salva em ${saida}`);
  process.exit(0);
};
process.on("SIGINT", encerrar);
process.on("SIGTERM", encerrar);
