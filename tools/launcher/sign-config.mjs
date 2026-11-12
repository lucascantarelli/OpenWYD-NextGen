#!/usr/bin/env node
// Assina a configuração remota do launcher (`launcher.conf`).
//
// O `wyd-launcher` confere HMAC-SHA256 sobre os bytes **exatos** anteriores à
// linha `assinatura = <64 hex>` (crates/wyd-launcher/src/configuracao.rs).
// Este script reproduz esse recorte à risca: o corpo recebido é assinado como
// está e a linha de assinatura é anexada no fim.
//
// Uso:
//   node tools/launcher/sign-config.mjs <corpo.conf> <chave.bin> <saida.conf>
//
// Exemplo:
//   node tools/launcher/sign-config.mjs tools/launcher/launcher.conf.exemplo \
//     tmp/chave.bin dist/launcher/launcher.conf
//
// A chave é o mesmo arquivo entregue ao launcher (`--key-file`), com bytes crus
// e sem `\n`/`\r` final — igual a `configuracao::carregar_chave`. Ela nunca é versionada.
import { createHmac } from "node:crypto";
import { readFileSync, writeFileSync } from "node:fs";

const [corpoArg, chaveArg, saidaArg] = process.argv.slice(2);
if (!corpoArg || !chaveArg || !saidaArg) {
  console.error(
    "uso: node tools/launcher/sign-config.mjs <corpo.conf> <chave.bin> <saida.conf>",
  );
  process.exit(2);
}

/** Bytes da chave sem terminador de linha (paridade com `configuracao::carregar_chave`). */
function chave(caminho) {
  let bytes = readFileSync(caminho);
  while (
    bytes.length > 0 &&
    (bytes[bytes.length - 1] === 0x0a || bytes[bytes.length - 1] === 0x0d)
  ) {
    bytes = bytes.subarray(0, bytes.length - 1);
  }
  if (bytes.length === 0) throw new Error(`chave vazia: ${caminho}`);
  return bytes;
}

const corpo = readFileSync(corpoArg);
if (/^assinatura\s*=/m.test(corpo.toString("utf8"))) {
  throw new Error("o corpo já tem linha de assinatura; assine o arquivo sem ela");
}
if (corpo.length === 0 || corpo[corpo.length - 1] !== 0x0a) {
  throw new Error("o corpo precisa terminar com \\n (o prefixo assinado é byte a byte)");
}

const assinatura = createHmac("sha256", chave(chaveArg)).update(corpo).digest("hex");
writeFileSync(saidaArg, Buffer.concat([corpo, Buffer.from(`assinatura = ${assinatura}\n`)]));
console.log(
  `config assinada: ${saidaArg} (corpo ${corpo.length} B, assinatura ${assinatura.slice(0, 12)}…)`,
);
