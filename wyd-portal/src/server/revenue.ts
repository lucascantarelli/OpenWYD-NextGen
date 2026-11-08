/**
 * Leitura do `DonateRevenueAdminService` (issue 31).
 *
 * Serviço **read-only**: não escreve nada, nem auditoria no banco — todo RPC
 * admin passa por `adminCall`, que revalida o papel da conta no web-api e injeta
 * o `moderator_id` da sessão. Nenhuma page chama `donateRevenueAdmin` direto.
 *
 * As janelas chegam já resolvidas (segundos Unix, `to` exclusivo) de
 * `lib/revenue`. O servidor devolve `from_unix`/`to_unix` aplicados; é isso que
 * a tela exibe, não o que o operador digitou.
 */
import {
  EMPTY_TOTALS,
  type AccountSummaryWire,
  type DonateLedgerRowWire,
  type RevenueByMethodWire,
  type RevenuePointWire,
  type RevenueTotalsWire,
  type TopBuyerRowWire,
  type TopupOrderRowWire,
} from "@/lib/revenue";

import { adminCall } from "./admin";
import {
  donateRevenueAdmin,
  type ListDonateSpendResponse,
  type ListTopBuyersResponse,
  type ListTopupOrdersResponse,
  type RevenueSummaryRequest,
  type RevenueSummaryResponse,
  type SearchAccountsResponse,
} from "./admin-client";

export interface RevenueSummary {
  totals: RevenueTotalsWire;
  byMethod: RevenueByMethodWire[];
  series: RevenuePointWire[];
  /** Janela efetivamente aplicada pelo web-api. */
  from: number;
  to: number;
}

function numbers(totals: Partial<RevenueTotalsWire> | undefined): RevenueTotalsWire {
  const base = { ...EMPTY_TOTALS };
  if (!totals) return base;
  for (const key of Object.keys(base) as (keyof RevenueTotalsWire)[]) {
    base[key] = Number(totals[key] ?? 0);
  }
  return base;
}

export async function getRevenueSummary(input: {
  from: number;
  to: number;
  bucket: number;
  accountId: number;
}): Promise<RevenueSummary> {
  const res = await adminCall<RevenueSummaryRequest, RevenueSummaryResponse>(
    donateRevenueAdmin.getRevenueSummary.bind(donateRevenueAdmin),
    {
      window: { from_unix: input.from, to_unix: input.to },
      bucket: input.bucket,
      account_id: input.accountId,
    },
    { action: "revenue.summary", target: input.accountId ? `account:${input.accountId}` : undefined },
  );

  return {
    totals: numbers(res.totals),
    byMethod: (res.by_method ?? []).map((row) => ({
      payment_method: Number(row.payment_method ?? 0),
      paid_orders: Number(row.paid_orders ?? 0),
      gross_cents: Number(row.gross_cents ?? 0),
    })),
    series: (res.series ?? []).map((point) => ({
      bucket_start_unix: Number(point.bucket_start_unix ?? 0),
      paid_orders: Number(point.paid_orders ?? 0),
      gross_cents: Number(point.gross_cents ?? 0),
      credits_sold: Number(point.credits_sold ?? 0),
      distinct_buyers: Number(point.distinct_buyers ?? 0),
    })),
    from: Number(res.from_unix ?? input.from),
    to: Number(res.to_unix ?? input.to),
  };
}

export interface OrderPage {
  rows: TopupOrderRowWire[];
  total: number;
  from: number;
  to: number;
}

export async function listOrders(input: {
  from: number;
  to: number;
  status: number;
  method: number;
  accountId: number;
  limit: number;
  offset: number;
}): Promise<OrderPage> {
  const res = await adminCall<
    {
      window: { from_unix: number; to_unix: number };
      status: number;
      payment_method: number;
      account_id: number;
      limit: number;
      offset: number;
    },
    ListTopupOrdersResponse
  >(
    donateRevenueAdmin.listTopupOrders.bind(donateRevenueAdmin),
    {
      window: { from_unix: input.from, to_unix: input.to },
      status: input.status,
      payment_method: input.method,
      account_id: input.accountId,
      limit: input.limit,
      offset: input.offset,
    },
    { action: "revenue.orders", target: input.accountId ? `account:${input.accountId}` : undefined },
  );

  return {
    rows: (res.orders ?? []).map((row) => ({
      id: Number(row.id ?? 0),
      external_reference: row.external_reference ?? "",
      account_id: Number(row.account_id ?? 0),
      account_name: row.account_name ?? "",
      account_email: row.account_email ?? "",
      payer_name: row.payer_name ?? "",
      payer_cpf_masked: row.payer_cpf_masked ?? "",
      credits: Number(row.credits ?? 0),
      amount_cents: Number(row.amount_cents ?? 0),
      payment_method: Number(row.payment_method ?? 0),
      provider: row.provider ?? "",
      status: Number(row.status ?? 0),
      created_at_unix: Number(row.created_at_unix ?? 0),
      confirmed_at_unix: Number(row.confirmed_at_unix ?? 0),
    })),
    total: Number(res.total_count ?? 0),
    from: Number(res.from_unix ?? input.from),
    to: Number(res.to_unix ?? input.to),
  };
}

export interface BuyerPage {
  rows: TopBuyerRowWire[];
  total: number;
}

export async function listTopBuyers(input: {
  from: number;
  to: number;
  limit: number;
  offset: number;
}): Promise<BuyerPage> {
  const res = await adminCall<
    { window: { from_unix: number; to_unix: number }; limit: number; offset: number },
    ListTopBuyersResponse
  >(
    donateRevenueAdmin.listTopBuyers.bind(donateRevenueAdmin),
    {
      window: { from_unix: input.from, to_unix: input.to },
      limit: input.limit,
      offset: input.offset,
    },
    { action: "revenue.buyers" },
  );

  return {
    rows: (res.buyers ?? []).map((row) => ({
      account_id: Number(row.account_id ?? 0),
      account_name: row.account_name ?? "",
      account_email: row.account_email ?? "",
      window_paid_orders: Number(row.window_paid_orders ?? 0),
      window_gross_cents: Number(row.window_gross_cents ?? 0),
      lifetime_paid_orders: Number(row.lifetime_paid_orders ?? 0),
      lifetime_gross_cents: Number(row.lifetime_gross_cents ?? 0),
      lifetime_credits: Number(row.lifetime_credits ?? 0),
      first_paid_at_unix: Number(row.first_paid_at_unix ?? 0),
      last_paid_at_unix: Number(row.last_paid_at_unix ?? 0),
      donate_balance: Number(row.donate_balance ?? 0),
    })),
    total: Number(res.total_count ?? 0),
  };
}

export interface LedgerPage {
  rows: DonateLedgerRowWire[];
  total: number;
}

export async function listLedger(input: {
  from: number;
  to: number;
  action: number;
  accountId: number;
  limit: number;
  offset: number;
}): Promise<LedgerPage> {
  const res = await adminCall<
    {
      window: { from_unix: number; to_unix: number };
      action: number;
      account_id: number;
      limit: number;
      offset: number;
    },
    ListDonateSpendResponse
  >(
    donateRevenueAdmin.listDonateSpend.bind(donateRevenueAdmin),
    {
      window: { from_unix: input.from, to_unix: input.to },
      action: input.action,
      account_id: input.accountId,
      limit: input.limit,
      offset: input.offset,
    },
    { action: "revenue.ledger", target: input.accountId ? `account:${input.accountId}` : undefined },
  );

  return {
    rows: (res.entries ?? []).map((row) => ({
      id: Number(row.id ?? 0),
      action: Number(row.action ?? 0),
      created_at_unix: Number(row.created_at_unix ?? 0),
      subject_account_id: Number(row.subject_account_id ?? 0),
      subject_account_name: row.subject_account_name ?? "",
      actor_account_id: Number(row.actor_account_id ?? 0),
      actor_account_name: row.actor_account_name ?? "",
      credits_delta: Number(row.credits_delta ?? 0),
      balance_after: Number(row.balance_after ?? 0),
      shop_item_id: Number(row.shop_item_id ?? 0),
      shop_item_title: row.shop_item_title ?? "",
      reason: row.reason ?? "",
    })),
    total: Number(res.total_count ?? 0),
  };
}

/** Prefixo de login → contas. Usado pelo Combobox e pelo filtro de conta. */
export async function searchAccounts(prefix: string, limit = 20): Promise<AccountSummaryWire[]> {
  const res = await adminCall<{ name_prefix: string; limit: number }, SearchAccountsResponse>(
    donateRevenueAdmin.searchAccounts.bind(donateRevenueAdmin),
    { name_prefix: prefix, limit },
    { action: "revenue.search_accounts", target: prefix.slice(0, 40) },
  );
  return (res.accounts ?? []).map((account) => ({
    id: Number(account.id ?? 0),
    name: account.name ?? "",
    email: account.email ?? "",
    donate_balance: Number(account.donate_balance ?? 0),
    role: account.role ?? "",
    is_blocked: Boolean(account.is_blocked),
  }));
}
