/**
 * Dados públicos do servidor para a home e para `/downloads`.
 *
 * Só entra o que tem fonte real e sem segredo: saúde do banco pelo `/health` do
 * gateway, revisão de conteúdo pelo `content_rates/get_rates` e as constantes
 * documentadas do cliente (build 7662, protocolo 12000). Nenhum número é
 * inventado e nada aqui usa a sessão do jogador.
 *
 * Toda fonte falha isolada: portal no ar com gateway fora é um estado que a
 * home precisa mostrar, não esconder.
 */
import { contentRates, promisify, type GetRatesResponse } from "./admin-client";
import { getDbHealth } from "./db-health";

/** Constantes do cliente/protocolo (docs/arquitetura/protocolo.md). */
export const CLIENT_BUILD = 7662;
export const PROTOCOL_VERSION = 12000;

export interface StatusPublico {
  /** O gateway respondeu a pelo menos uma chamada. */
  gateway: boolean;
  database: boolean;
  /** Revisão vigente do conteúdo; `null` sem gateway. */
  configRevision: number | null;
  /** Latência do `/health` do gateway, em ms. */
  dbLatencyMs: number | null;
}

export async function getStatusPublico(): Promise<StatusPublico> {
  const [db, rates] = await Promise.all([
    getDbHealth(),
    promisify<Record<string, never>, GetRatesResponse>(
      contentRates.getRates.bind(contentRates),
    )({}).catch(() => null),
  ]);

  return {
    gateway: db.reachable || rates !== null,
    database: db.reachable,
    configRevision: rates?.revision ?? null,
    dbLatencyMs: db.latencyMs,
  };
}
