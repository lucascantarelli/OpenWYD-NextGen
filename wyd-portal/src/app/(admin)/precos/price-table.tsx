"use client";

import { useCallback, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import { Badge } from "@/components/Badge";
import { Combobox, type ComboOption } from "@/components/Combobox";
import { DataTable, type Column } from "@/components/DataTable";
import { Dialog } from "@/components/Dialog";
import { Field } from "@/components/Field";
import { ItemIcon } from "@/components/ItemIcon";
import { useToast } from "@/components/Toast";
import { fmtCoin } from "@/lib/format";
import { CLEAR_PRICE } from "@/lib/price";
import { setItemPrice } from "@/server/actions/prices";
import type { CatalogItem } from "@/server/admin-client";

export interface PriceRow {
  index: number;
  price: number;
  base: number | null;
  item: CatalogItem | null;
}

/**
 * Tabela de overrides de preço + formulário de novo preço.
 *
 * O preço é **global**: vale em toda loja que não fixe preço. A coluna "base" é
 * o `ItemList.csv`; o Δ mostra o desvio para o operador perceber um zero a mais.
 */
export function PriceTable({
  rows,
  catalogVersion,
}: {
  rows: readonly PriceRow[];
  catalogVersion: string;
}) {
  const router = useRouter();
  const toast = useToast();
  const [open, setOpen] = useState(false);
  const [picked, setPicked] = useState<{ index: number; label: string } | null>(null);
  const [fixed, setFixed] = useState<{ index: number; label: string; base: number | null } | null>(null);
  const [price, setPrice] = useState("");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirming, setConfirming] = useState<number | null>(null);

  const loadItems = useCallback(async (query: string): Promise<ComboOption[]> => {
    const response = await fetch(`/api/search/items?q=${encodeURIComponent(query)}`);
    if (!response.ok) throw new Error("falha na busca");
    const body = (await response.json()) as { options?: ComboOption[] };
    return body.options ?? [];
  }, []);

  async function submit(index: number, value: number) {
    setPending(true);
    const result = await setItemPrice({ itemIndex: index, price: value });
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      toast(result.error, "error");
      return;
    }
    toast(
      value === CLEAR_PRICE
        ? "Override removido — volta ao preço do ItemList.csv"
        : "Preço global salvo — aplica no próximo restart do tmserver",
      "ok",
    );
    setOpen(false);
    setPicked(null);
    setFixed(null);
    setPrice("");
    setError(null);
    router.refresh();
  }

  function startNew() {
    setPicked(null);
    setFixed(null);
    setPrice("");
    setError(null);
    setOpen(true);
  }

  function startEdit(row: PriceRow) {
    setFixed({ index: row.index, label: row.item?.display_name || row.item?.name || `#${row.index}`, base: row.base });
    setPicked(null);
    setPrice(String(row.price));
    setError(null);
    setOpen(true);
  }

  const target = fixed ?? (picked ? { index: picked.index, label: picked.label, base: null } : null);
  const parsedPrice = Number(price);
  const priceValid = price.trim() !== "" && Number.isInteger(parsedPrice) && parsedPrice >= 0;

  const columns = useMemo<Column<PriceRow>[]>(
    () => [
      {
        key: "icon",
        header: "",
        render: (row) =>
          row.item ? (
            <ItemIcon item={row.item} iconPackVersion={catalogVersion} />
          ) : (
            <span className="ui-field__hint">—</span>
          ),
      },
      {
        key: "index",
        header: "#",
        align: "right",
        sortValue: (row) => row.index,
        render: (row) => <span className="ui-mono">{row.index}</span>,
      },
      {
        key: "name",
        header: "Item",
        sortValue: (row) => row.item?.display_name || row.item?.name || "",
        render: (row) =>
          row.item ? (
            row.item.display_name || row.item.name
          ) : (
            <span className="ui-field__hint">não está no catálogo</span>
          ),
      },
      {
        key: "base",
        header: "Base (ItemList.csv)",
        align: "right",
        sortValue: (row) => row.base ?? -1,
        render: (row) => <span className="ui-mono">{fmtCoin(row.base)}</span>,
      },
      {
        key: "price",
        header: "Preço global",
        align: "right",
        sortValue: (row) => row.price,
        render: (row) => <span className="ui-mono">{fmtCoin(row.price)}</span>,
      },
      {
        key: "delta",
        header: "Δ sobre a base",
        align: "right",
        render: (row) => {
          if (row.base === null) return <span className="ui-field__hint">—</span>;
          if (row.price === row.base) return <Badge>igual</Badge>;
          const diff = row.price - row.base;
          return (
            <Badge tone={diff > 0 ? "warn" : "ok"}>
              {diff > 0 ? `+${fmtCoin(diff)}` : `-${fmtCoin(Math.abs(diff))}`}
            </Badge>
          );
        },
      },
      {
        key: "actions",
        header: "Ações",
        render: (row) => (
          <span style={{ display: "inline-flex", gap: 6 }}>
            <button
              type="button"
              className="ui-btn ui-btn--ghost"
              disabled={pending}
              onClick={() => startEdit(row)}
            >
              alterar
            </button>
            {confirming === row.index ? (
              <>
                <button
                  type="button"
                  className="ui-btn ui-btn--danger"
                  disabled={pending}
                  onClick={() => {
                    setConfirming(null);
                    void submit(row.index, CLEAR_PRICE);
                  }}
                >
                  confirmar
                </button>
                <button
                  type="button"
                  className="ui-btn ui-btn--ghost"
                  onClick={() => setConfirming(null)}
                >
                  cancelar
                </button>
              </>
            ) : (
              <button
                type="button"
                className="ui-btn ui-btn--ghost"
                disabled={pending}
                onClick={() => setConfirming(row.index)}
              >
                remover override
              </button>
            )}
          </span>
        ),
      },
    ],
    // Os botões dependem de `pending`/`confirming`; `submit`/`startEdit` são estáveis por closure.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [catalogVersion, pending, confirming],
  );

  return (
    <>
      <div className="ui-filters">
        <button type="button" className="ui-btn ui-btn--primary" onClick={startNew}>
          definir preço
        </button>
        <span className="ui-field__hint" style={{ marginLeft: "auto" }}>
          {rows.length} {rows.length === 1 ? "item com override" : "itens com override"}
        </span>
      </div>

      <DataTable
        rows={rows}
        columns={columns}
        getRowKey={(row) => row.index}
        searchText={(row) => `${row.index} ${row.item?.name ?? ""} ${row.item?.display_name ?? ""}`}
        searchPlaceholder="Buscar override por nome ou índice…"
        pageSizes={[25, 50, 100]}
        initialPageSize={25}
        emptyMessage="Nenhum item com preço global definido. Use “definir preço”."
        caption="Preço global por item (tabela item_price). Item sem override usa o ItemList.csv."
      />

      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        title={fixed ? "Alterar preço global" : "Definir preço global"}
        description="O preço vale em todas as lojas que não fixam preço — não é por NPC."
        footer={
          <>
            <button type="button" className="ui-btn" onClick={() => setOpen(false)}>
              cancelar
            </button>
            <button
              type="button"
              className="ui-btn ui-btn--primary"
              disabled={pending || !target || !priceValid}
              onClick={() => target && void submit(target.index, parsedPrice)}
            >
              salvar
            </button>
          </>
        }
      >
        {!fixed && (
          <Combobox
            label="Item"
            placeholder="nome ou índice…"
            load={loadItems}
            onSelect={(option) => setPicked({ index: Number(option.value), label: option.label })}
          />
        )}

        <Field
          label="Preço global (inteiro)"
          hint={
            target
              ? `item #${target.index} — ${target.label}${target.base !== null ? ` · base ${fmtCoin(target.base)}` : ""}`
              : "escolha um item acima"
          }
          htmlFor="price"
        >
          <input
            id="price"
            className="ui-input"
            type="number"
            min={0}
            step={1}
            value={price}
            onChange={(event) => setPrice(event.target.value)}
            autoComplete="off"
          />
        </Field>

        {error && (
          <p className="ui-field__error" role="alert">
            {error}
          </p>
        )}
      </Dialog>
    </>
  );
}
