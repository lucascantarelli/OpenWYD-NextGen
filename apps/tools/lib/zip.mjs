// Verificação de `client.zip` contra o manifesto do cliente.
//
// O launcher instala o ZIP conferindo cada entrada do manifesto (tamanho +
// SHA-256). Este módulo faz a mesma conferência **antes** de publicar o pacote,
// sem depender de `unzip`/`zip` do sistema: lê o diretório central, infla quando
// preciso e compara. Entrada ausente/divergente é erro com o caminho, nunca
// "quase tudo certo".
import { createHash } from "node:crypto";
import { inflateRawSync } from "node:zlib";

const EOCD = 0x0605_4b50;
const CENTRAL = 0x0201_4b50;
const LOCAL = 0x0403_4b50;

/** Manifesto: `caminho → { sha256, size }` (ignora `build`/`protocol`/comentários). */
export function parseManifesto(texto) {
  const entradas = new Map();
  for (const linha of texto.split(/\r?\n/)) {
    const limpa = linha.trim();
    if (!limpa || limpa.startsWith("#") || limpa.startsWith("build ") || limpa.startsWith("protocol ")) {
      continue;
    }
    const m = /^([0-9a-f]{64})\s+(\d+)\s+(.+)$/i.exec(limpa);
    if (!m) throw new Error(`linha de manifesto inválida: ${limpa}`);
    entradas.set(m[3].trim(), { sha256: m[1].toLowerCase(), size: Number(m[2]) });
  }
  return entradas;
}

/** Índice do diretório central: `nome → { metodo, tamanho, comprimido, dados }`. */
export function indiceZip(zip) {
  const eocd = zip.lastIndexOf(Buffer.from([0x50, 0x4b, 0x05, 0x06]));
  if (eocd < 0) throw new Error("ZIP sem fim de diretório central");
  const total = zip.readUInt16LE(eocd + 10);
  let off = zip.readUInt32LE(eocd + 16);
  const entradas = new Map();
  for (let i = 0; i < total; i += 1) {
    if (zip.readUInt32LE(off) !== CENTRAL) throw new Error("cabeçalho central inválido");
    const flags = zip.readUInt16LE(off + 8);
    if (flags & 0x1) throw new Error("entrada criptografada");
    const metodo = zip.readUInt16LE(off + 10);
    const comprimido = zip.readUInt32LE(off + 20);
    const tamanho = zip.readUInt32LE(off + 24);
    const nomeLen = zip.readUInt16LE(off + 28);
    const extraLen = zip.readUInt16LE(off + 30);
    const comentarioLen = zip.readUInt16LE(off + 32);
    const local = zip.readUInt32LE(off + 42);
    const nome = zip.subarray(off + 46, off + 46 + nomeLen).toString("utf8");
    if (zip.readUInt32LE(local) !== LOCAL) throw new Error(`cabeçalho local inválido: ${nome}`);
    const nomeLocal = zip.readUInt16LE(local + 26);
    const extraLocal = zip.readUInt16LE(local + 28);
    const inicio = local + 30 + nomeLocal + extraLocal;
    if (!nome.endsWith("/")) {
      entradas.set(nome, { metodo, comprimido, tamanho, inicio });
    }
    off += 46 + nomeLen + extraLen + comentarioLen;
  }
  return entradas;
}

/** Bytes descomprimidos de uma entrada (stored ou deflate). */
export function conteudoZip(zip, nome, entrada) {
  const bruto = zip.subarray(entrada.inicio, entrada.inicio + entrada.comprimido);
  const bytes = entrada.metodo === 0 ? bruto : entrada.metodo === 8 ? inflateRawSync(bruto) : null;
  if (!bytes) throw new Error(`método ${entrada.metodo} não suportado: ${nome}`);
  if (bytes.length !== entrada.tamanho) throw new Error(`tamanho divergente: ${nome}`);
  return bytes;
}

/**
 * Confere o ZIP contra o manifesto. Devolve `{ ok, ausentes, divergentes, extras }`.
 * `extras` são arquivos do ZIP fora do manifesto (permitidos, só reportados).
 */
export function verificarZip(zip, manifesto) {
  const indice = indiceZip(zip);
  const ausentes = [];
  const divergentes = [];
  for (const [caminho, esperado] of manifesto) {
    const entrada = indice.get(caminho);
    if (!entrada) {
      ausentes.push(caminho);
      continue;
    }
    const bytes = conteudoZip(zip, caminho, entrada);
    const sha = createHash("sha256").update(bytes).digest("hex");
    if (bytes.length !== esperado.size || sha !== esperado.sha256) divergentes.push(caminho);
  }
  const extras = [...indice.keys()].filter((nome) => !manifesto.has(nome));
  return { ok: ausentes.length === 0 && divergentes.length === 0, ausentes, divergentes, extras };
}
