// Inventário determinístico do cliente (nada inventado: só o que os bytes dizem).
//
// Lê a pasta do cliente, agrega por extensão/diretório, calcula SHA-256 dos
// arquivos e, quando encontra um PE, resume cabeçalho/seções com o parser já
// existente em `tools/client/pe.mjs` (fonte única; este app não duplica formato).
import { createHash } from "node:crypto";
import { readFileSync, readdirSync, statSync } from "node:fs";
import path from "node:path";

import { parsePe } from "../../../tools/client/pe.mjs";

/** Caminhos que não são conteúdo do jogo (artefatos do launcher/ambiente). */
export const IGNORADOS = new Set([
  "manifest.txt",
  "canais.txt",
  "launcher.conf",
  "launcher.conf.bak",
  "WYDLauncher.exe",
  "WYDLauncher.exe.bak",
  "serverlist.bin",
  "serverlist.bin.bak",
  "serverlist.txt",
  "sn.bin",
]);

const ignorar = (nome) => IGNORADOS.has(nome) || /\.(bak|tmp|part|pdb)$/i.test(nome);

/** Todos os arquivos do cliente, relativo com `/`, em ordem determinística. */
export function listarArquivos(raiz, prefixo = "") {
  const saida = [];
  for (const entrada of readdirSync(raiz, { withFileTypes: true })) {
    const rel = prefixo ? `${prefixo}/${entrada.name}` : entrada.name;
    if (entrada.isDirectory()) saida.push(...listarArquivos(path.join(raiz, entrada.name), rel));
    else if (entrada.isFile() && !ignorar(entrada.name)) saida.push(rel);
  }
  return saida.sort((a, b) => (a < b ? -1 : a > b ? 1 : 0));
}

function sha256(arquivo) {
  return createHash("sha256").update(readFileSync(arquivo)).digest("hex");
}

/** Resumo PE enxuto (o dossiê completo fica em `tools/client/analyse.mjs`). */
export function resumoPe(bytes) {
  try {
    const pe = parsePe(bytes);
    return {
      machine: `${pe.machineName} (0x${pe.machine.toString(16)})`,
      subsystem: `${pe.subsystemName} (${pe.subsystem})`,
      sections: pe.sections.map((s) => ({
        name: s.name,
        virtualSize: s.virtualSize,
        rawSize: s.rawSize,
      })),
      entryPoint: pe.entryPointHex ?? pe.entryPoint,
      imageBase: pe.imageBase,
      timestamp: pe.timestamp,
    };
  } catch (erro) {
    return { erro: String(erro.message ?? erro) };
  }
}

/**
 * Inventário: arquivos, bytes, buckets por extensão/diretório e PEs executáveis.
 */
export function inventariar(raiz) {
  const arquivos = listarArquivos(raiz);
  const porExtensao = new Map();
  const porDiretorio = new Map();
  const executaveis = [];
  let bytes = 0;
  for (const rel of arquivos) {
    const completo = path.join(raiz, rel);
    const tamanho = statSync(completo).size;
    bytes += tamanho;
    const ext = (path.extname(rel) || "(sem extensão)").toLowerCase();
    const bucket = porExtensao.get(ext) ?? { arquivos: 0, bytes: 0 };
    bucket.arquivos += 1;
    bucket.bytes += tamanho;
    porExtensao.set(ext, bucket);
    const topo = rel.includes("/") ? rel.slice(0, rel.indexOf("/")) : ".";
    const dir = porDiretorio.get(topo) ?? { arquivos: 0, bytes: 0 };
    dir.arquivos += 1;
    dir.bytes += tamanho;
    porDiretorio.set(topo, dir);
    if (ext === ".exe" || ext === ".dll") {
      const resumo = resumoPe(readFileSync(completo));
      executaveis.push({ caminho: rel, bytes: tamanho, sha256: sha256(completo), pe: resumo });
    }
  }
  return {
    raiz: path.resolve(raiz),
    arquivos: arquivos.length,
    bytes,
    porExtensao: [...porExtensao.entries()].sort((a, b) => b[1].bytes - a[1].bytes),
    porDiretorio: [...porDiretorio.entries()].sort((a, b) => b[1].bytes - a[1].bytes),
    executaveis: executaveis.sort((a, b) => (a.caminho < b.caminho ? -1 : 1)),
  };
}
