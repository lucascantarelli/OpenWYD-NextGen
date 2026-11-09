/**
 * Estatísticas de economia do painel (FASE-07-3).
 *
 * Cada número mostra de onde veio: imposto somado de `city_tax_ledger` e
 * concessões contadas em `delivery_queue`. Nada é estimado no portal —
 * `/economia` exibe a fonte ao lado da tabela, e lista vazia significa
 * "sem movimento registrado", não "indisponível".
 */
import { adminCall } from "./admin";
import { economyAdmin, type EconomyResponse } from "./admin-client";

export interface CityTaxView {
  city: string;
  total: number;
}

export interface GrantTotalView {
  kind: string;
  count: number;
  gold: number;
}

export interface EconomyView {
  taxByCity: CityTaxView[];
  grants: GrantTotalView[];
}

/** Foto da economia auditada (`economy_admin/summary`, staff). */
export async function getEconomySummary(): Promise<EconomyView> {
  const res = await adminCall<Record<string, never>, EconomyResponse>(
    economyAdmin.summary.bind(economyAdmin),
    {},
    { action: "economy.summary", target: "economy" },
  );
  const wire = res.economy ?? { tax_by_city: [], grants: [] };
  return {
    taxByCity: wire.tax_by_city.map((row) => ({ city: row.city_code, total: row.total })),
    grants: wire.grants.map((row) => ({ kind: row.kind, count: row.count, gold: row.gold })),
  };
}
