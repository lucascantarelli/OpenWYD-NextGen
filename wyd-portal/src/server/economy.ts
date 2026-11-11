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

/** Alerta derivado só de número já resumido (sem linha por jogador). */
export interface EconomyAlert {
  tone: "warn" | "accent";
  title: string;
  detail: string;
}

/**
 * Alertas econômicos a partir do resumo (regras fixas, sem limiar inventado no dado).
 *
 * - cidade sem movimento com outras ativas: possível dreno (vale conferir o recolhimento);
 * - concessões de ouro acima de 1M somado: volume alto de staff (vale conferir a fila);
 * - sem movimento algum: silêncio, não problema — a tela mostra "sem movimento".
 * Moderador vê agregados; nunca linha por jogador (regra da issue).
 */
export function computeEconomyAlerts(summary: EconomyView): EconomyAlert[] {
  const alerts: EconomyAlert[] = [];
  const gold = summary.grants.find((row) => row.kind === "gold");
  if (gold && gold.gold >= 1_000_000) {
    alerts.push({
      tone: "warn",
      title: "Volume alto de ouro concedido",
      detail: `${gold.count} concessão(ões) somando ${gold.gold.toLocaleString("pt-BR")} de ouro na fila. Confira a fila antes de novas concessões.`,
    });
  }
  const zeradas = summary.taxByCity.filter((row) => row.total <= 0);
  if (summary.taxByCity.length > 1 && zeradas.length > 0) {
    alerts.push({
      tone: "accent",
      title: "Cidade sem recolhimento",
      detail: `${zeradas.map((row) => row.city).join(", ")} sem movimento no livro-caixa enquanto outras arrecadam.`,
    });
  }
  return alerts;
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
