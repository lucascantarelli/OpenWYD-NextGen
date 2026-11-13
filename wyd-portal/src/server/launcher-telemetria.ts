/**
 * Telemetria pública do launcher (CLIENT-11, item 5 do plano).
 *
 * O launcher não tem sessão: o POST chega anônimo. Por isso o contrato é
 * **fechado** (nada de chave extra), os números são limitados e o único texto
 * aceito é nome de arquivo (`patch` falhou em `jogo.dat`), nunca caminho
 * absoluto, login, IP ou mensagem livre. O que sai daqui é agregado por
 * `versao + operacao` — não uma trilha por jogador.
 *
 * A agregação vive no processo do BFF (Map limitado) e é escrita no log
 * estruturado a cada evento aceito: Prometheus/Loki contam o resto. Não há
 * tabela nova nem escrita no banco do jogo — telemetria não pode virar caminho
 * de escrita do portal para o mundo vivo (AGENTS §3.7).
 */
import { z } from "zod";

/** Teto do corpo aceito (um evento cabe em ~200 bytes). */
export const LIMITE_CORPO = 4 * 1024;

/** Chaves distintas mantidas no processo; o resto é contado como descartado. */
const MAX_CHAVES = 64;

const arquivo = z
  .string()
  .min(1)
  .max(120)
  // Só nome de arquivo: `/`, `\`, `:` e espaço não passam (caminho de máquina
  // do jogador não é dado nosso).
  .regex(/^[A-Za-z0-9._-]+$/, "nome de arquivo inválido");

/** Contrato aceito pelo `POST /api/launcher/telemetria`. */
export const esquemaEvento = z.strictObject({
  /** Build do launcher que mandou o evento. */
  versao: z.number().int().min(0).max(1_000_000),
  /** Build do cliente em `client/` no momento da operação. */
  build_cliente: z.number().int().min(0).max(1_000_000),
  operacao: z.enum(["check", "config", "patch", "launch"]),
  ok: z.boolean(),
  /** Duração da operação observada pelo launcher. */
  duracao_ms: z.number().int().min(0).max(600_000),
  /** Arquivos tocados (0 quando a operação não baixa nada). */
  arquivos: z.number().int().min(0).max(10_000).default(0),
  /** Falhas por arquivo, no máximo 10, com motivo enumerado. */
  falhas: z
    .array(
      z.strictObject({
        arquivo,
        motivo: z.enum(["hash", "io", "assinatura", "rede", "desconhecido"]),
      }),
    )
    .max(10)
    .default([]),
});

export type EventoTelemetria = z.infer<typeof esquemaEvento>;
export type Operacao = EventoTelemetria["operacao"];

/** Agregado por `versao + operacao` desde o boot do processo. */
export interface ResumoTelemetria {
  versao: number;
  operacao: Operacao;
  eventos: number;
  falhas: number;
  duracaoTotalMs: number;
  duracaoMaxMs: number;
  /** ISO-8601 do último evento aceito para esta chave. */
  ultimaEm: string;
}

interface Acumulado {
  versao: number;
  operacao: Operacao;
  eventos: number;
  falhas: number;
  duracaoTotalMs: number;
  duracaoMaxMs: number;
  ultimaEm: string;
}

const agregados = new Map<string, Acumulado>();
let descartados = 0;
let totalEventos = 0;

function chave(versao: number, operacao: Operacao): string {
  return `${versao}:${operacao}`;
}

/**
 * Registra um evento já validado e devolve o agregado da chave.
 *
 * `null` = chave nova além do teto (o evento é contado em `descartados`, não
 * derruba o launcher e não cresce memória sem limite).
 */
export function registrar(evento: EventoTelemetria, agora = new Date()): ResumoTelemetria | null {
  const id = chave(evento.versao, evento.operacao);
  let atual = agregados.get(id);
  if (!atual) {
    if (agregados.size >= MAX_CHAVES) {
      descartados += 1;
      return null;
    }
    atual = {
      versao: evento.versao,
      operacao: evento.operacao,
      eventos: 0,
      falhas: 0,
      duracaoTotalMs: 0,
      duracaoMaxMs: 0,
      ultimaEm: agora.toISOString(),
    };
    agregados.set(id, atual);
  }
  atual.eventos += 1;
  atual.falhas += evento.falhas.length;
  atual.duracaoTotalMs += evento.duracao_ms;
  atual.duracaoMaxMs = Math.max(atual.duracaoMaxMs, evento.duracao_ms);
  atual.ultimaEm = agora.toISOString();
  totalEventos += 1;

  // Linha única e sem texto livre: o Loki agrega; nada de PII no log.
  console.info(
    "[launcher-telemetria]",
    JSON.stringify({
      versao: evento.versao,
      build_cliente: evento.build_cliente,
      operacao: evento.operacao,
      ok: evento.ok,
      duracao_ms: evento.duracao_ms,
      arquivos: evento.arquivos,
      falhas: evento.falhas.length,
    }),
  );

  return { ...atual };
}

/** Linha pronta do painel: agregado + média e taxa de falha por evento. */
export interface LinhaTelemetria extends ResumoTelemetria {
  mediaMs: number;
  /** Falhas por evento (0..1). */
  taxaFalha: number;
}

/**
 * Agregados na ordem do painel: versão mais nova primeiro, operação alfabética.
 *
 * Cálculo puro sobre a foto de `resumo()` — a página não faz conta própria.
 */
export function resumoTabela(): LinhaTelemetria[] {
  return resumo()
    .chaves.map((chave) => ({
      ...chave,
      mediaMs: chave.eventos > 0 ? Math.round(chave.duracaoTotalMs / chave.eventos) : 0,
      taxaFalha: chave.eventos > 0 ? chave.falhas / chave.eventos : 0,
    }))
    .sort(
      (a, b) =>
        b.versao - a.versao || a.operacao.localeCompare(b.operacao, "pt-BR"),
    );
}

/** Foto dos agregados (cópia: quem lê não altera o estado do processo). */
export function resumo(): {
  chaves: ResumoTelemetria[];
  totalEventos: number;
  descartados: number;
} {
  return {
    chaves: [...agregados.values()].map((item) => ({ ...item })),
    totalEventos,
    descartados,
  };
}
