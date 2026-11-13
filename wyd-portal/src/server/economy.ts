/**
 * Estatísticas de economia do painel (WEB-03).
 *
 * Cada número mostra de onde veio: imposto somado de `city_tax_ledger` e
 * concessões contadas em `delivery_queue`. Nada é estimado no portal —
 * `/economia` exibe a fonte ao lado da tabela, e lista vazia significa
 * "sem movimento registrado", não "indisponível".
 */
import { adminCall } from "./admin";
import {
  donateAdmin,
  donateRevenue,
  economyAdmin,
  type DonateDebitWire,
  type DonateLedgerResponse,
  type DonateRevenueResponse,
  type EconomyResponse,
} from "./admin-client";

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

/** Entrada do extrato de donate (recarga ou débito). */
export interface DonateEntryView {
  id: number;
  accountId: number;
  login: string;
  credits: number;
  createdAt: string;
}

export interface DonateBuyerView {
  accountId: number;
  login: string;
  credits: number;
  topups: number;
}

/** Receita de donate + extratos, sempre do razão gravado. */
export interface DonateRevenueView {
  creditsTotal: number;
  debitsTotal: number;
  balance: number;
  topups: number;
  debits: number;
  topupOrders: DonateEntryView[];
  spendOrders: DonateEntryView[];
  buyers: DonateBuyerView[];
}

function toEntry(row: {
  id: number;
  account_id: number;
  login: string;
  credits: number;
  created_at: string;
}): DonateEntryView {
  return {
    id: row.id,
    accountId: row.account_id,
    login: row.login,
    credits: row.credits,
    createdAt: row.created_at,
  };
}

/**
 * Receita de donate (`donate_revenue_admin/*`).
 *
 * Quatro leituras independentes: total, recargas, débitos e compradores. Tabela
 * vazia continua sendo resposta válida — a tela mostra "sem movimento".
 */
export async function getDonateRevenue(): Promise<DonateRevenueView> {
  const [summary, topups, spend, buyers] = await Promise.all([
    adminCall<Record<string, never>, DonateRevenueResponse>(
      donateRevenue.getRevenueSummary.bind(donateRevenue),
      {},
      { action: "donate.revenue", target: "donate" },
    ),
    adminCall<{ limit: number }, DonateLedgerResponse>(
      donateRevenue.listTopupOrders.bind(donateRevenue),
      { limit: 20 },
      { action: "donate.revenue", target: "donate" },
    ),
    adminCall<{ limit: number }, DonateLedgerResponse>(
      donateRevenue.listDonateSpend.bind(donateRevenue),
      { limit: 20 },
      { action: "donate.revenue", target: "donate" },
    ),
    adminCall<{ limit: number }, DonateLedgerResponse>(
      donateRevenue.listTopBuyers.bind(donateRevenue),
      { limit: 10 },
      { action: "donate.revenue", target: "donate" },
    ),
  ]);

  const revenue = summary.revenue ?? {
    credits_total: 0,
    debits_total: 0,
    balance: 0,
    topups: 0,
    debits: 0,
  };
  return {
    creditsTotal: revenue.credits_total,
    debitsTotal: revenue.debits_total,
    balance: revenue.balance,
    topups: revenue.topups,
    debits: revenue.debits,
    topupOrders: (topups.orders ?? []).map(toEntry),
    spendOrders: (spend.spend ?? []).map(toEntry),
    buyers: (buyers.buyers ?? []).map((row) => ({
      accountId: row.account_id,
      login: row.login,
      credits: row.credits,
      topups: row.topups,
    })),
  };
}

/** Resolve o login digitado para uma conta única (busca por prefixo no gateway). */
export async function findAccountByLogin(login: string): Promise<number | null> {
  const res = await adminCall<{ login: string; limit: number }, DonateLedgerResponse>(
    donateRevenue.searchAccounts.bind(donateRevenue),
    { login, limit: 5 },
    { action: "donate.search", target: "account" },
  );
  const contas = res.accounts ?? [];
  const alvo = login.trim().toLowerCase();
  const exata = contas.filter((conta) => conta.login.toLowerCase() === alvo);
  if (exata.length === 1) return exata[0].id;
  if (exata.length === 0 && contas.length === 1) return contas[0].id;
  return null;
}

/** Débito de saldo de donate, idempotente pela chave gerada na ação. */
export async function debitDonateBalance(
  accountId: number,
  credits: number,
  key: string,
): Promise<{ debitId: number; replay: boolean }> {
  const res = await adminCall<
    { account_id: number; credits: number; idempotency_key: string },
    DonateDebitWire
  >(
    donateAdmin.debitDonateBalance.bind(donateAdmin),
    { account_id: accountId, credits, idempotency_key: key },
    { action: "donate.debit", target: `account:${accountId}` },
  );
  return { debitId: res.debit_id ?? 0, replay: res.replay ?? false };
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
