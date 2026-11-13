#!/usr/bin/env node
// Assina pacotes de patch do launcher com Ed25519 (ADR-0003, revisão 2026-11-11).
//
// O `wyd-launcher` verifica com a chave **pública embutida** no binário
// (`crates/wyd-launcher/src/distribuicao.rs`, `CHAVE_PUBLICA_HEX`) sobre os bytes
// exatos de `manifesto`. A chave privada nunca é versionada: no CI ela vem de um
// secret (`WYD_PACKAGE_KEY` virando arquivo), nunca do repositório.
//
// Uso:
//   node tools/launcher/sign-package.mjs keygen <chave.pem>
//   node tools/launcher/sign-package.mjs pubkey <chave.pem>
//   node tools/launcher/sign-package.mjs sign <pacote-dir> <chave.pem>
//
// `sign` grava `<pacote-dir>/assinatura` com 128 hex minúsculos (64 bytes), que é
// o formato lido por `distribuicao::verify_package`.
import { createPrivateKey, createPublicKey, generateKeyPairSync, sign } from "node:crypto";
import { readFileSync, writeFileSync } from "node:fs";
import path from "node:path";

const [acao, ...args] = process.argv.slice(2);

function uso() {
  console.error("uso: node tools/launcher/sign-package.mjs keygen|pubkey|sign <args...>");
  process.exit(2);
}

/** 32 bytes crus da chave pública (o SPKI DER termina com a chave bruta). */
function publicoBruto(chave) {
  const publica = chave.type === "public" ? chave : createPublicKey(chave);
  const der = publica.export({ type: "spki", format: "der" });
  if (der.length < 32) throw new Error("SPKI menor que 32 bytes");
  return der.subarray(der.length - 32).toString("hex");
}

if (acao === "keygen") {
  const [destino] = args;
  if (!destino) uso();
  const { publicKey, privateKey } = generateKeyPairSync("ed25519");
  writeFileSync(destino, privateKey.export({ type: "pkcs8", format: "pem" }), { mode: 0o600 });
  console.log(`chave privada: ${destino} (nunca versionar)`);
  console.log(`chave pública (hex para CHAVE_PUBLICA_HEX): ${publicoBruto(publicKey)}`);
} else if (acao === "pubkey") {
  const [arquivo] = args;
  if (!arquivo) uso();
  console.log(publicoBruto(createPrivateKey(readFileSync(arquivo, "utf8"))));
} else if (acao === "sign") {
  const [dir, arquivo] = args;
  if (!dir || !arquivo) uso();
  const manifesto = readFileSync(path.join(dir, "manifesto"));
  if (manifesto.length === 0) throw new Error("manifesto vazio");
  const chave = createPrivateKey(readFileSync(arquivo, "utf8"));
  const assinatura = sign(null, manifesto, chave).toString("hex");
  writeFileSync(path.join(dir, "assinatura"), `${assinatura}\n`);
  console.log(`assinado: ${path.join(dir, "assinatura")} (${assinatura.length / 2} bytes)`);
} else {
  uso();
}
