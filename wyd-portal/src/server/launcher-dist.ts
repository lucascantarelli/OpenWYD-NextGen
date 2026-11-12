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
import {
  appendFileSync,
  copyFileSync,
  existsSync,
  mkdirSync,
  readFileSync,
  renameSync,
  statSync,
  writeFileSync,
} from "node:fs";
import path from "node:path";

const DIR = process.env.LAUNCHER_DIST_DIR ?? "/dist/launcher";
const PACOTE = process.env.LAUNCHER_PACKAGE ?? "wyd-launcher.exe";
const MANIFESTO = "manifest.txt";
const CONFIGURACAO = "launcher.conf";

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

/**
 * Configuração assinada publicada (`launcher.conf`), ou `null` quando ausente.
 *
 * O portal não interpreta nem re-assina: entrega os bytes exatos que
 * `tools/launcher/sign-config.mjs` assinou. Quem autentica é o launcher, com a
 * chave (`wyd-launcher config ... --key-file`).
 */
export function lerConfiguracao(): string | null {
  const arquivo = path.join(DIR, CONFIGURACAO);
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

/** Arquivo de histórico (uma revisão publicada por linha, JSONL). */
export const HISTORICO_CONFIG = "launcher.conf.historico.jsonl";

/** Teto aceito na publicação (a config real tem poucas centenas de bytes). */
export const LIMITE_CONFIG = 32 * 1024;

/** Revisão publicada, como fica no histórico do portal. */
export interface RevisaoPublicada {
  revisao: number;
  sha256: string;
  bytes: number;
  /** Quem publicou (`portal#<conta>`). */
  autor: string;
  /** ISO-8601 do momento da publicação. */
  em: string;
}

/** Configuração em vigor, com o texto assinado (quem autentica é o launcher). */
export interface ConfiguracaoAtual {
  texto: string;
  revisao: number;
  sha256: string;
  bytes: number;
  autor: string | null;
  em: string | null;
}

const sha256 = (texto: string): string =>
  createHash("sha256").update(texto, "utf8").digest("hex");

/** Revisão declarada no texto (`revisao = N`), ou `null`. */
export function revisaoDe(texto: string): number | null {
  const achado = /^revisao\s*=\s*(\d+)\s*$/m.exec(texto);
  if (!achado) return null;
  const numero = Number(achado[1]);
  return Number.isSafeInteger(numero) && numero > 0 ? numero : null;
}

/** Histórico de revisões publicadas (mais antiga primeiro), ou lista vazia. */
export function historicoConfiguracao(dir = DIR): RevisaoPublicada[] {
  const arquivo = path.join(dir, HISTORICO_CONFIG);
  if (!existsSync(arquivo)) return [];
  const revisoes: RevisaoPublicada[] = [];
  for (const linha of readFileSync(arquivo, "utf8").split(/\r?\n/)) {
    if (!linha.trim()) continue;
    try {
      const lido = JSON.parse(linha) as Partial<RevisaoPublicada>;
      if (
        typeof lido.revisao === "number" &&
        typeof lido.sha256 === "string" &&
        typeof lido.bytes === "number" &&
        typeof lido.autor === "string" &&
        typeof lido.em === "string"
      ) {
        revisoes.push(lido as RevisaoPublicada);
      }
    } catch {
      // Linha corrompida não esconde as demais: o histórico é diagnóstico.
    }
  }
  return revisoes;
}

/** Configuração em vigor com metadados (ou `null` quando não publicada). */
export function configuracaoAtual(dir = DIR): ConfiguracaoAtual | null {
  const texto = lerConfiguracaoEm(dir);
  if (texto === null) return null;
  const revisao = revisaoDe(texto);
  if (revisao === null) return null;
  const sha = sha256(texto);
  const entrada = historicoConfiguracao(dir).find((item) => item.sha256 === sha);
  return {
    texto,
    revisao,
    sha256: sha,
    bytes: Buffer.byteLength(texto, "utf8"),
    autor: entrada?.autor ?? null,
    em: entrada?.em ?? null,
  };
}

/**
 * Publica a configuração **já assinada** por `tools/launcher/sign-config.mjs`.
 *
 * O portal não assina nem reescreve bytes: só recusa o que não tem cara de config
 * assinada (revisão ausente, assinatura fora do formato, revisão não monotônica) e
 * grava o texto exato, com histórico append-only. A autenticação HMAC acontece no
 * launcher, que tem a chave — o portal nunca vê a chave.
 */
export function publicarConfiguracao(
  texto: string,
  autor: string,
  dir = DIR,
): RevisaoPublicada {
  const bytes = Buffer.byteLength(texto, "utf8");
  if (bytes === 0) throw new Error("configuração vazia");
  if (bytes > LIMITE_CONFIG) {
    throw new Error(`configuração acima de ${LIMITE_CONFIG / 1024} KiB`);
  }
  if (!texto.endsWith("\n")) {
    throw new Error("a configuração precisa terminar com \\n (o prefixo assinado é byte a byte)");
  }
  const linhas = texto
    .split(/\r?\n/)
    .map((linha) => linha.trim())
    .filter((linha) => linha !== "" && !linha.startsWith("#"));
  const ultima = linhas.at(-1) ?? "";
  if (!/^assinatura\s*=\s*[0-9a-f]{64}$/.test(ultima)) {
    throw new Error(
      "última linha precisa ser `assinatura = <64 hex>`; assine com tools/launcher/sign-config.mjs",
    );
  }
  const revisao = revisaoDe(texto);
  if (revisao === null) throw new Error("configuração sem `revisao = N`");

  const historico = historicoConfiguracao(dir);
  if (historico.some((item) => item.revisao === revisao)) {
    throw new Error(`revisão ${revisao} já foi publicada`);
  }
  const atual = configuracaoAtual(dir);
  if (atual && revisao <= atual.revisao) {
    throw new Error(`revisão ${revisao} não é maior que a publicada (${atual.revisao})`);
  }

  mkdirSync(dir, { recursive: true });
  const destino = path.join(dir, CONFIGURACAO);
  const temporario = `${destino}.tmp`;
  writeFileSync(temporario, texto, "utf8");
  if (existsSync(destino)) copyFileSync(destino, `${destino}.bak`);
  renameSync(temporario, destino);

  const entrada: RevisaoPublicada = {
    revisao,
    sha256: sha256(texto),
    bytes,
    autor,
    em: new Date().toISOString(),
  };
  appendFileSync(path.join(dir, HISTORICO_CONFIG), `${JSON.stringify(entrada)}\n`, "utf8");
  return entrada;
}

/** Texto do `launcher.conf` em um diretório específico (testes e staging). */
export function lerConfiguracaoEm(dir: string): string | null {
  const arquivo = path.join(dir, CONFIGURACAO);
  if (!existsSync(arquivo)) return null;
  return readFileSync(arquivo, "utf8");
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
