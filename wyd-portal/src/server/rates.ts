/**
 * Leitura do `ContentRatesService` (issue 36).
 *
 * O web-api devolve cada célula com valor do arquivo, valor efetivo e a faixa
 * aceita — o portal não recalcula nem inventa limites. Sem `-content` no web-api,
 * a resposta vem vazia com um aviso, e a tela mostra o aviso em vez de erro.
 */
import type { Rate, ServerFlag } from "@/lib/rates";

import { adminCall } from "./admin";
import { contentRates, type GetRatesResponse, type GetServerFlagsResponse } from "./grpc";

export interface RatesSnapshot {
  rates: Rate[];
  warning: string;
  contentOk: boolean;
}

export interface FlagsSnapshot {
  flags: ServerFlag[];
  available: boolean;
  capturedAt: string;
  message: string;
}

/**
 * Snapshot de flags que o tmserver gravou no último boot. `available=false`
 * significa que ele ainda não gravou (ou subiu sem `-dbserver`) — a tela mostra
 * isso em vez de uma tabela vazia.
 */
export async function getServerFlags(): Promise<FlagsSnapshot> {
  const res = await adminCall<Record<string, never>, GetServerFlagsResponse>(
    contentRates.getServerFlags.bind(contentRates),
    {},
    { action: "rates.flags" },
  );
  return {
    flags: res.flags ?? [],
    available: Boolean(res.available),
    capturedAt: res.captured_at ?? "",
    message: res.message ?? "",
  };
}

export async function getRates(): Promise<RatesSnapshot> {
  const res = await adminCall<Record<string, never>, GetRatesResponse>(
    contentRates.getRates.bind(contentRates),
    {},
    { action: "rates.get" },
  );
  return {
    rates: res.rates ?? [],
    warning: res.warning ?? "",
    contentOk: Boolean(res.content_ok),
  };
}
