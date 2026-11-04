"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";

import { Combobox, type ComboOption } from "@/components/Combobox";
import { DataTable, type Column } from "@/components/DataTable";
import { loadAccountOptions } from "@/lib/accountsearch";
import { fmtBRL, fmtCount } from "@/lib/money";
import {
  BUCKET_OPTIONS,
  PERIODS,
  fmtMoment,
  ledgerActionLabel,
  ordersCsv,
  paymentMethodLabel,
  todayInput,
  topupStatusLabel,
  type BucketValue,
  type DonateLedgerRowWire,
  type PeriodId,
  type TopBuyerRowWire,
  type TopupOrderRowWire,
} from "@/lib/revenue";

/**
 * Controles e tabelas do faturamento (issue 31).
 *
 * O filtro vive na URL (`periodo`, `desde`, `ate`, `serie`, `conta`, `status`,
 * `metodo`, `extrato`) e a página server reage a ele: nada de estado duplicado
 * entre cliente e servidor, e o link é compartilhável com outro admin. As
 * tabelas recebem até 100 linhas já paginadas pelo web-api; busca, ordenação e
 * paginação fina continuam no `DataTable`.
 */
export interface RevenueControlsProps {
  period: PeriodId;
  desde: string;
  ate: string;
  bucket: BucketValue;
  accountId: number;
  accountName: string;
  status: number;
  method: number;
  ledgerAction: number;
}

export function RevenueControls(props: RevenueControlsProps) {
  const router = useRouter();
  const params = useSearchParams();
  const [desde, setDesde] = useState(props.desde);
  const [ate, setAte] = useState(props.ate);
  const [account, setAccount] = useState<ComboOption | null>(
    props.accountId > 0
      ? {
          value: String(props.accountId),
          label: props.accountName || `conta #${props.accountId}`,
        }
      : null,
  );
  const [dateError, setDateError] = useState<string | null>(null);

  function apply(patch: Record<string, string | null>) {
    const next = new URLSearchParams(params.toString());
    for (const [key, value] of Object.entries(patch)) {
      if (value === null || value === "") next.delete(key);
      else next.set(key, value);
    }
    router.replace(`/faturamento?${next.toString()}`, { scroll: false });
  }

  function pickPeriod(period: PeriodId) {
    setDateError(null);
    if (period !== "custom") {
      apply({ periodo: period, desde: null, ate: null });
      return;
    }
    const start = desde || todayInput();
    const end = ate || todayInput();
    setDesde(start);
    setAte(end);
    apply({ periodo: "custom", desde: start, ate: end });
  }

  function applyCustom() {
    if (!desde || !ate) {
      setDateError("informe as duas datas");
      return;
    }
    if (desde > ate) {
      setDateError("a data inicial precisa ser anterior à final");
      return;
    }
    setDateError(null);
    apply({ periodo: "custom", desde, ate });
  }

  return (
    <div className="ui-card" style={{ marginBottom: 20 }}>
      <div className="ui-filters" style={{ marginBottom: 0 }}>
        {PERIODS.map((period) => (
          <button
            key={period.id}
            type="button"
            className="ui-chip"
            aria-pressed={props.period === period.id}
            onClick={() => pickPeriod(period.id)}
          >
            {period.label}
          </button>
        ))}

        {props.period === "custom" && (
          <div style={{ display: "flex", gap: 6, alignItems: "center", flexWrap: "wrap" }}>
            <input
              className="ui-input"
              type="date"
              aria-label="data inicial"
              value={desde}
              onChange={(event) => setDesde(event.target.value)}
            />
            <span className="ui-field__hint">até</span>
            <input
              className="ui-input"
              type="date"
              aria-label="data final"
              value={ate}
              onChange={(event) => setAte(event.target.value)}
            />
            <button type="button" className="ui-btn" onClick={applyCustom}>
              aplicar
            </button>
            {dateError && (
              <span className="ui-field__error" role="alert">
                {dateError}
              </span>
            )}
          </div>
        )}

        <label className="ui-field__hint" style={{ marginLeft: "auto", display: "flex", gap: 6 }}>
          série{" "}
          <select
            className="ui-select"
            value={String(props.bucket)}
            onChange={(event) => apply({ serie: event.target.value })}
          >
            {BUCKET_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
          gap: "var(--space-2)",
          marginTop: "var(--space-3)",
          alignItems: "end",
        }}
      >
        <div>
          <span className="ui-field__label">conta</span>
          <Combobox
            label="filtrar por conta"
            load={loadAccountOptions}
            placeholder="Login da conta (mín. 2 letras)…"
            value={account}
            onSelect={(option) => {
              setAccount(option);
              apply({ conta: option.value, nome: option.label });
            }}
          />
          {account && (
            <button
              type="button"
              className="ui-btn ui-btn--ghost"
              style={{ marginTop: 4 }}
              onClick={() => {
                setAccount(null);
                apply({ conta: null, nome: null });
              }}
            >
              limpar conta
            </button>
          )}
        </div>

        <label className="ui-field">
          <span className="ui-field__label">status do pedido</span>
          <select
            className="ui-select"
            value={String(props.status)}
            onChange={(event) => apply({ status: event.target.value })}
          >
            <option value="0">todos</option>
            <option value="1">pendente</option>
            <option value="2">pago</option>
          </select>
        </label>

        <label className="ui-field">
          <span className="ui-field__label">forma de pagamento</span>
          <select
            className="ui-select"
            value={String(props.method)}
            onChange={(event) => apply({ metodo: event.target.value })}
          >
            <option value="0">todas</option>
            <option value="1">PIX</option>
            <option value="2">cartão</option>
          </select>
        </label>

        <label className="ui-field">
          <span className="ui-field__label">extrato</span>
          <select
            className="ui-select"
            value={String(props.ledgerAction)}
            onChange={(event) => apply({ extrato: event.target.value })}
          >
            <option value="0">tudo</option>
            <option value="1">compras</option>
            <option value="2">créditos manuais</option>
          </select>
        </label>
      </div>

      <p className="ui-field__hint" style={{ margin: "6px 0 0" }}>
        O filtro de conta vale para pedidos, compradores e extrato. Pedido pago filtra por{" "}
        <code>confirmed_at</code>; qualquer outro status, por <code>created_at</code>.
      </p>
    </div>
  );
}

function reference(value: string): string {
  return value.length > 10 ? `${value.slice(0, 8)}…` : value || "—";
}

function downloadCsv(rows: readonly TopupOrderRowWire[]) {
  const blob = new Blob([ordersCsv(rows)], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = "pedidos-donate.csv";
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

const ORDER_COLUMNS: Column<TopupOrderRowWire>[] = [
  {
    key: "id",
    header: "pedido",
    sortValue: (row) => row.id,
    render: (row) => (
      <div>
        <span className="ui-mono">#{row.id}</span>
        <span className="ui-field__hint" style={{ display: "block" }} title={row.external_reference}>
          {reference(row.external_reference)}
        </span>
      </div>
    ),
  },
  {
    key: "created",
    header: "criado",
    sortValue: (row) => row.created_at_unix,
    render: (row) => <span className="ui-mono">{fmtMoment(row.created_at_unix)}</span>,
  },
  {
    key: "account",
    header: "conta",
    sortValue: (row) => row.account_name,
    render: (row) => (
      <div>
        <span>{row.account_name || `#${row.account_id}`}</span>
        {row.account_email && (
          <span className="ui-field__hint" style={{ display: "block" }}>
            {row.account_email}
          </span>
        )}
      </div>
    ),
  },
  {
    key: "payer",
    header: "pagador",
    sortValue: (row) => row.payer_name,
    render: (row) => (
      <div>
        <span>{row.payer_name || "não informado"}</span>
        {row.payer_cpf_masked && (
          <span className="ui-field__hint ui-mono" style={{ display: "block" }}>
            {row.payer_cpf_masked}
          </span>
        )}
      </div>
    ),
  },
  {
    key: "amount",
    header: "valor",
    align: "right",
    sortValue: (row) => row.amount_cents,
    render: (row) => <span className="ui-mono">{fmtBRL(row.amount_cents)}</span>,
  },
  {
    key: "credits",
    header: "donate",
    align: "right",
    sortValue: (row) => row.credits,
    render: (row) => fmtCount(row.credits),
  },
  {
    key: "method",
    header: "método",
    sortValue: (row) => row.payment_method,
    render: (row) => paymentMethodLabel(row.payment_method),
  },
  {
    key: "status",
    header: "status",
    sortValue: (row) => row.status,
    render: (row) => (
      <span className="ui-field__hint">
        {topupStatusLabel(row.status)}
        {row.status !== 2 && " — não é receita"}
      </span>
    ),
  },
  {
    key: "confirmed",
    header: "pago em",
    sortValue: (row) => row.confirmed_at_unix,
    render: (row) => <span className="ui-mono">{fmtMoment(row.confirmed_at_unix)}</span>,
  },
];

export function OrdersTable({
  rows,
  total,
  accountName,
}: {
  rows: readonly TopupOrderRowWire[];
  total: number;
  accountName: string;
}) {
  return (
    <>
      <div className="ui-filters" style={{ marginBottom: 8 }}>
        <span className="ui-field__hint">
          {fmtCount(total)} pedido{total === 1 ? "" : "s"} na janela
          {rows.length < total ? ` · mostrando os ${rows.length} primeiros — estreite o período` : ""}
          {accountName ? ` · ${accountName}` : ""}
        </span>
        <button
          type="button"
          className="ui-btn"
          style={{ marginLeft: "auto" }}
          disabled={rows.length === 0}
          onClick={() => downloadCsv(rows)}
        >
          exportar CSV
        </button>
      </div>
      <DataTable
        rows={rows}
        columns={ORDER_COLUMNS}
        getRowKey={(row) => row.id}
        searchText={(row) =>
          `${row.id} ${row.external_reference} ${row.account_name} ${row.account_email} ${row.payer_name} ${row.payer_cpf_masked} ${row.amount_cents}`
        }
        searchPlaceholder="Buscar por pedido, referência, conta ou pagador…"
        emptyMessage="Nenhum pedido nesta janela."
        caption="Valor em reais; donate é a unidade da carteira e não entra no total."
      />
    </>
  );
}

const BUYER_COLUMNS: Column<TopBuyerRowWire>[] = [
  {
    key: "account",
    header: "conta",
    sortValue: (row) => row.account_name,
    render: (row) => (
      <div>
        <span>{row.account_name || `#${row.account_id}`}</span>
        {row.account_email && (
          <span className="ui-field__hint" style={{ display: "block" }}>
            {row.account_email}
          </span>
        )}
      </div>
    ),
  },
  {
    key: "window",
    header: "receita na janela",
    align: "right",
    sortValue: (row) => row.window_gross_cents,
    render: (row) => <span className="ui-mono">{fmtBRL(row.window_gross_cents)}</span>,
  },
  {
    key: "orders",
    header: "pedidos",
    align: "right",
    sortValue: (row) => row.window_paid_orders,
    render: (row) => fmtCount(row.window_paid_orders),
  },
  {
    key: "ticket",
    header: "ticket",
    align: "right",
    sortValue: (row) =>
      row.window_paid_orders ? row.window_gross_cents / row.window_paid_orders : 0,
    render: (row) =>
      row.window_paid_orders
        ? fmtBRL(Math.round(row.window_gross_cents / row.window_paid_orders))
        : "—",
  },
  {
    key: "ltv",
    header: "total histórico",
    align: "right",
    sortValue: (row) => row.lifetime_gross_cents,
    render: (row) => (
      <div>
        <span className="ui-mono">{fmtBRL(row.lifetime_gross_cents)}</span>
        <span className="ui-field__hint" style={{ display: "block" }}>
          {fmtCount(row.lifetime_paid_orders)} pedidos · {fmtCount(row.lifetime_credits)} donate
        </span>
      </div>
    ),
  },
  {
    key: "balance",
    header: "saldo donate",
    align: "right",
    sortValue: (row) => row.donate_balance,
    render: (row) => fmtCount(row.donate_balance),
  },
  {
    key: "last",
    header: "último pagamento",
    sortValue: (row) => row.last_paid_at_unix,
    render: (row) => <span className="ui-mono">{fmtMoment(row.last_paid_at_unix)}</span>,
  },
];

export function BuyersTable({ rows }: { rows: readonly TopBuyerRowWire[] }) {
  return (
    <DataTable
      rows={rows}
      columns={BUYER_COLUMNS}
      getRowKey={(row) => row.account_id}
      searchText={(row) => `${row.account_name} ${row.account_email} ${row.account_id}`}
      searchPlaceholder="Buscar comprador…"
      emptyMessage="Ninguém pagou nesta janela."
      caption="Ranking pela receita da janela; o total histórico é o LTV da conta."
    />
  );
}

const LEDGER_COLUMNS: Column<DonateLedgerRowWire>[] = [
  {
    key: "when",
    header: "quando",
    sortValue: (row) => row.created_at_unix,
    render: (row) => <span className="ui-mono">{fmtMoment(row.created_at_unix)}</span>,
  },
  {
    key: "action",
    header: "movimento",
    sortValue: (row) => row.action,
    render: (row) => (
      <span className="ui-field__hint">{ledgerActionLabel(row.action)}</span>
    ),
  },
  {
    key: "subject",
    header: "carteira",
    sortValue: (row) => row.subject_account_name,
    render: (row) => (
      <span>
        {row.subject_account_name || `#${row.subject_account_id}`}
      </span>
    ),
  },
  {
    key: "actor",
    header: "autor",
    sortValue: (row) => row.actor_account_name,
    render: (row) => (
      <span>
        {row.actor_account_name || `#${row.actor_account_id}`}
        {row.actor_account_id === row.subject_account_id && (
          <span className="ui-field__hint"> (próprio)</span>
        )}
      </span>
    ),
  },
  {
    key: "delta",
    header: "donate",
    align: "right",
    sortValue: (row) => row.credits_delta,
    render: (row) => (
      <span
        className="ui-mono"
        style={{ color: row.credits_delta < 0 ? "var(--danger)" : undefined }}
      >
        {row.credits_delta > 0 ? "+" : ""}
        {fmtCount(row.credits_delta)}
      </span>
    ),
  },
  {
    key: "balance",
    header: "saldo após",
    align: "right",
    sortValue: (row) => row.balance_after,
    render: (row) => fmtCount(row.balance_after),
  },
  {
    key: "detail",
    header: "detalhe",
    render: (row) =>
      row.action === 1
        ? row.shop_item_title || (row.shop_item_id ? `oferta #${row.shop_item_id}` : "—")
        : row.reason || "—",
  },
];

export function LedgerTable({
  rows,
  total,
}: {
  rows: readonly DonateLedgerRowWire[];
  total: number;
}) {
  return (
    <DataTable
      rows={rows}
      columns={LEDGER_COLUMNS}
      getRowKey={(row) => `${row.action}-${row.id}`}
      searchText={(row) =>
        `${row.subject_account_name} ${row.actor_account_name} ${row.credits_delta} ${
          row.shop_item_title
        } ${row.reason}`
      }
      searchPlaceholder="Buscar por conta, oferta ou motivo…"
      emptyMessage="Nenhum movimento de carteira nesta janela."
      caption={`${fmtCount(total)} movimento${total === 1 ? "" : "s"} · créditos de donate, não reais`}
    />
  );
}
