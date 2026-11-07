import { redirect } from "next/navigation";

import { RetryButton } from "@/components/RetryButton";
import { Stat } from "@/components/Stat";
import { averageCents, fmtBRL, fmtCount, fmtPercent } from "@/lib/money";
import {
  bucketForDays,
  bucketNoun,
  fmtBucketLabel,
  fmtMoment,
  isBucketValue,
  isPeriodId,
  paymentMethodLabel,
  resolveWindow,
  type BucketValue,
  type PeriodId,
} from "@/lib/revenue";
import { AdminError } from "@/server/errors";
import { getRevenueSummary, listLedger, listOrders, listTopBuyers } from "@/server/revenue";
import { getSession } from "@/server/session";

import { BuyersTable, LedgerTable, OrdersTable, RevenueControls } from "./revenue-view";

export const dynamic = "force-dynamic";

export const metadata = { title: "Faturamento · WYD2" };

type SearchParams = Promise<Record<string, string | string[] | undefined>>;

const ORDERS_LIMIT = 100;
const BUYERS_LIMIT = 50;
const LEDGER_LIMIT = 100;

function one(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}

function intParam(value: string | undefined, allowed: readonly number[]): number {
  const parsed = Math.trunc(Number(value));
  return allowed.includes(parsed) ? parsed : 0;
}

/**
 * Painel de faturamento (issue 31).
 *
 * Lê o `DonateRevenueAdminService` e não escreve nada — nem uma linha de
 * auditoria no banco. Restrito a `admin`: `moderator` opera a loja, mas receita e
 * dados de pagador são informação financeira (o web-api também recusa).
 */
export default async function RevenuePage({ searchParams }: { searchParams: SearchParams }) {
  const session = await getSession();
  if (!session) redirect("/login");
  if (session.role !== "admin") redirect("/sem-permissao");

  const sp = await searchParams;
  const rawPeriod = one(sp.periodo);
  const period: PeriodId = isPeriodId(rawPeriod) ? rawPeriod : "30d";
  const desde = one(sp.desde) ?? "";
  const ate = one(sp.ate) ?? "";
  const accountId = Math.max(0, Math.trunc(Number(one(sp.conta)) || 0));
  const accountName = one(sp.nome) ?? "";
  const status = intParam(one(sp.status), [1, 2]);
  const method = intParam(one(sp.metodo), [1, 2]);
  const ledgerAction = intParam(one(sp.extrato), [1, 2]);

  const resolved = resolveWindow({ period, desde, ate });
  const defaultBucket: BucketValue = resolved.ok ? bucketForDays(resolved.days) : 1;
  const rawBucket = Math.trunc(Number(one(sp.serie)));
  const bucket: BucketValue = isBucketValue(rawBucket) ? rawBucket : defaultBucket;

  const controls = (
    <RevenueControls
      period={period}
      desde={desde}
      ate={ate}
      bucket={bucket}
      accountId={accountId}
      accountName={accountName}
      status={status}
      method={method}
      ledgerAction={ledgerAction}
    />
  );

  if (!resolved.ok) {
    return (
      <>
        <h1 className="ui-page-title">Faturamento</h1>
        {controls}
        <div className="ui-card ui-empty">
          <strong>Período inválido</strong>
          <p>{resolved.error}</p>
        </div>
      </>
    );
  }

  const window = { from: resolved.from, to: resolved.to };

  let error: string | null = null;
  let summary: Awaited<ReturnType<typeof getRevenueSummary>> | null = null;
  let orders: Awaited<ReturnType<typeof listOrders>> | null = null;
  let buyers: Awaited<ReturnType<typeof listTopBuyers>> | null = null;
  let ledger: Awaited<ReturnType<typeof listLedger>> | null = null;

  try {
    [summary, orders, buyers, ledger] = await Promise.all([
      getRevenueSummary({ ...window, bucket, accountId }),
      listOrders({ ...window, status, method, accountId, limit: ORDERS_LIMIT, offset: 0 }),
      listTopBuyers({ ...window, limit: BUYERS_LIMIT, offset: 0 }),
      listLedger({ ...window, action: ledgerAction, accountId, limit: LEDGER_LIMIT, offset: 0 }),
    ]);
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível ler o faturamento do web-api";
  }

  const appliedFrom = summary?.from ?? resolved.from;
  const appliedTo = summary?.to ?? resolved.to;
  const totals = summary?.totals;
  const series = summary?.series ?? [];
  const maxGross = series.reduce((max, point) => Math.max(max, point.gross_cents), 0);

  return (
    <>
      <h1 className="ui-page-title">Faturamento</h1>
      <p className="ui-page-sub">
        Receita <strong>reconhecida em <code>confirmed_at</code></strong>, em reais. Donate é outra
        unidade: créditos vendidos, gastos e concedidos aparecem separados e nunca entram no total
        em R$.
      </p>

      {controls}

      {error ? (
        <div className="ui-card ui-empty">
          <strong>Faturamento indisponível</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      ) : (
        <>
          <p className="ui-field__hint" style={{ marginTop: 0 }}>
            Janela aplicada pelo servidor: <span className="ui-mono">{fmtMoment(appliedFrom)}</span>{" "}
            até <span className="ui-mono">{fmtMoment(appliedTo)}</span> · buckets fecham em
            America/Sao_Paulo
            {accountId > 0 && (
              <>
                {" "}
                · filtrado por{" "}
                <strong>{accountName || `conta #${accountId}`}</strong>
              </>
            )}
          </p>

          <div className="ui-stats">
            <Stat
              label="Receita paga"
              value={fmtBRL(totals?.gross_cents ?? 0)}
              hint={`${fmtCount(totals?.paid_orders ?? 0)} pedidos pagos no período`}
            />
            <Stat
              label="Ticket médio"
              value={fmtBRL(averageCents(totals?.gross_cents ?? 0, totals?.paid_orders ?? 0))}
              hint="receita ÷ pedidos pagos"
            />
            <Stat
              label="Compradores"
              value={fmtCount(totals?.distinct_buyers ?? 0)}
              hint="contas distintas que pagaram"
            />
            <Stat
              label="Créditos vendidos"
              value={fmtCount(totals?.credits_sold ?? 0)}
              hint="unidade de donate — não é reais"
            />
          </div>

          <div className="ui-stats">
            <Stat
              label="Pedidos criados"
              value={fmtCount(totals?.created_orders ?? 0)}
              hint="qualquer status, base da conversão"
            />
            <Stat
              label="Aguardando pagamento"
              value={fmtCount(totals?.pending_orders ?? 0)}
              hint={`${fmtBRL(totals?.pending_cents ?? 0)} pendentes — não é receita`}
            />
            <Stat
              label="Compras na loja"
              value={fmtCount(totals?.shop_purchases ?? 0)}
              hint={`${fmtCount(totals?.credits_spent ?? 0)} donate gastos`}
            />
            <Stat
              label="Créditos manuais"
              value={fmtCount(totals?.credits_granted ?? 0)}
              hint={`${fmtCount(totals?.manual_credits ?? 0)} concessões auditadas`}
            />
          </div>

          <div className="ui-card" style={{ marginBottom: 20 }}>
            <h2 style={{ fontSize: 14, margin: "0 0 6px" }}>
              Receita por forma de pagamento
            </h2>
            {(summary?.byMethod.length ?? 0) === 0 ? (
              <p style={{ margin: 0, color: "var(--muted)", fontSize: 13 }}>
                Nenhum pedido pago nesta janela.
              </p>
            ) : (
              <ul style={{ listStyle: "none", margin: 0, padding: 0, display: "grid", gap: 6 }}>
                {summary?.byMethod.map((row) => (
                  <li
                    key={row.payment_method}
                    style={{
                      display: "grid",
                      gridTemplateColumns: "110px 1fr auto",
                      gap: 10,
                      alignItems: "center",
                      fontSize: 13,
                    }}
                  >
                    <span>{paymentMethodLabel(row.payment_method)}</span>
                    <span className="ui-field__hint">
                      {fmtCount(row.paid_orders)} pedidos ·{" "}
                      {fmtPercent(row.gross_cents, totals?.gross_cents ?? 0)} da receita
                    </span>
                    <span className="ui-mono">{fmtBRL(row.gross_cents)}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="ui-card" style={{ marginBottom: 20 }}>
            <h2 style={{ fontSize: 14, margin: "0 0 6px" }}>
              {bucket === 0 ? "Série desligada" : `Receita por ${bucketNoun(bucket)}`}
            </h2>
            {bucket === 0 ? (
              <p style={{ margin: 0, color: "var(--muted)", fontSize: 13 }}>
                Escolha dia, semana ou mês no seletor de série para ver a evolução.
              </p>
            ) : series.length === 0 ? (
              <p style={{ margin: 0, color: "var(--muted)", fontSize: 13 }}>
                Sem pontos na janela.
              </p>
            ) : (
              <div
                role="img"
                aria-label={`Receita por ${bucketNoun(bucket)}: ${fmtBRL(
                  totals?.gross_cents ?? 0,
                )} em ${series.length} buckets`}
              >
                {series.map((point) => {
                  const width = maxGross > 0 ? Math.round((point.gross_cents / maxGross) * 100) : 0;
                  const label = fmtBucketLabel(point.bucket_start_unix, bucket);
                  return (
                    <div
                      key={point.bucket_start_unix}
                      title={`${label}: ${fmtBRL(point.gross_cents)} · ${fmtCount(
                        point.paid_orders,
                      )} pedidos · ${fmtCount(point.credits_sold)} donate`}
                      style={{
                        display: "grid",
                        gridTemplateColumns: "96px 1fr 118px",
                        gap: 10,
                        alignItems: "center",
                        padding: "2px 0",
                        fontSize: 12,
                      }}
                    >
                      <span className="ui-field__hint">{label}</span>
                      <span
                        style={{
                          height: 14,
                          background: "var(--surface-2)",
                          borderRadius: 4,
                          overflow: "hidden",
                        }}
                      >
                        <span
                          style={{
                            display: "block",
                            height: "100%",
                            width: `${width}%`,
                            background: "var(--accent)",
                          }}
                        />
                      </span>
                      <span className="ui-mono" style={{ textAlign: "right" }}>
                        {fmtBRL(point.gross_cents)}
                      </span>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          <h2 style={{ fontSize: 14, margin: "0 0 8px" }}>Pedidos</h2>
          <OrdersTable
            rows={orders?.rows ?? []}
            total={orders?.total ?? 0}
            accountName={accountName}
          />

          <h2 style={{ fontSize: 14, margin: "20px 0 8px" }}>
            Quem pagou
            {buyers && buyers.total > buyers.rows.length
              ? ` (top ${buyers.rows.length} de ${fmtCount(buyers.total)})`
              : ""}
          </h2>
          <BuyersTable rows={buyers?.rows ?? []} />

          <h2 style={{ fontSize: 14, margin: "20px 0 8px" }}>Movimento da carteira de donate</h2>
          <LedgerTable rows={ledger?.rows ?? []} total={ledger?.total ?? 0} />
        </>
      )}
    </>
  );
}
