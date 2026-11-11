/**
 * Distribuição pública do launcher.
 *
 * O portal **serve** a pasta de distribuição montada no container
 * (`LAUNCHER_DIST_DIR`) e o manifesto gerado por `tools/launcher/dist-manifest.mjs`.
 * Nada é inventado: sem pasta, sem manifesto ou sem o pacote, a página diz que a
 * distribuição não está publicada e mostra o comando de publicação.
 *
 * Server-only: caminho e conteúdo nunca vão para o browser como caminho de disco;
 * o download passa por rota própria (`/api/launcher/pacote`).
 */
import { createHash } from "node:crypto";
import { existsSync, readFileSync, statSync } from "node:fs";
import path from "node:path";

const DIR = process.env.LAUNCHER_DIST_DIR ?? "/dist/launcher";
const PACOTE = process.env.LAUNCHER_PACKAGE ?? "wyd-launcher.exe";
const MANIFESTO = "manifest.txt";

export interface LauncherPublicacao {
  /** Manifesto presente e legível. */
  publicada: boolean;
  /** Motivo legível quando `publicada` é falso. */
  motivo: string;
  build: number | null;
  protocol: number | null;
  arquivos: number;
  /** Soma dos tamanhos do manifesto, em bytes. */
  bytes: number;
  /** Pacote principal do launcher (nome, tamanho e SHA-256 reais). */
  pacote: { nome: string; bytes: number; sha256: string } | null;
  /** URL pública do manifesto (a mesma que o launcher consome). */
  manifestoUrl: string | null;
}

/** Manifesto cru (`text/plain`), ou `null` quando não publicado. */
export function lerManifesto(): string | null {
  const arquivo = path.join(DIR, MANIFESTO);
  if (!existsSync(arquivo)) return null;
  return readFileSync(arquivo, "utf8");
}

/** Pacote principal do launcher, ou `null` quando ausente. */
export function abrirPacote(): { nome: string; conteudo: Buffer; sha256: string } | null {
  const arquivo = path.join(DIR, PACOTE);
  if (!existsSync(arquivo) || !statSync(arquivo).isFile()) return null;
  const conteudo = readFileSync(arquivo);
  return {
    nome: PACOTE,
    conteudo,
    sha256: createHash("sha256").update(conteudo).digest("hex"),
  };
}

/** Parse do manifesto; linhas inválidas não derrubam a página (viram `motivo`). */
function parseManifesto(texto: string): {
  build: number | null;
  protocol: number | null;
  arquivos: number;
  bytes: number;
} {
  let build: number | null = null;
  let protocol: number | null = null;
  let arquivos = 0;
  let bytes = 0;
  for (const linha of texto.split(/\r?\n/)) {
    const limpa = linha.trim();
    if (!limpa || limpa.startsWith("#")) continue;
    if (limpa.startsWith("build ")) {
      build = Number(limpa.slice(6).trim()) || null;
      continue;
    }
    if (limpa.startsWith("protocol ")) {
      protocol = Number(limpa.slice(9).trim()) || null;
      continue;
    }
    const partes = limpa.split(/\s+/);
    if (partes.length !== 3) continue;
    arquivos += 1;
    bytes += Number(partes[1]) || 0;
  }
  return { build, protocol, arquivos, bytes };
}

export function getLauncherPublicacao(): LauncherPublicacao {
  const texto = lerManifesto();
  if (texto === null) {
    return {
      publicada: false,
      motivo: "manifesto não publicado",
      build: null,
      protocol: null,
      arquivos: 0,
      bytes: 0,
      pacote: null,
      manifestoUrl: null,
    };
  }
  const manifesto = parseManifesto(texto);
  const pacote = abrirPacote();

  return {
    publicada: true,
    motivo: pacote ? "" : "pacote do launcher ausente na distribuição",
    ...manifesto,
    pacote: pacote
      ? { nome: pacote.nome, bytes: pacote.conteudo.byteLength, sha256: pacote.sha256 }
      : null,
    manifestoUrl: "/api/launcher/manifesto",
  };
}
