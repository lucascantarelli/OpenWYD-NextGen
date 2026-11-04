"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";

import { DataTable, type Column } from "@/components/DataTable";
import { ItemIcon } from "@/components/ItemIcon";
import { Badge } from "@/components/Badge";
import { fmtCoin } from "@/lib/format";
import { distinctSlots, gradeLabel, gradeTone, itemSearchText, slotsText } from "@/lib/item";
import type { CatalogItem } from "@/server/grpc";

/**
 * Navegador do catálogo: filtro por slot (multi) e grau, busca com debounce de
 * 150ms e paginação de 100/200/500. Só a página visível é montada, então 3.2k
 * ícones nunca entram no DOM de uma vez (equivalente prático à virtualização,
 * sem dependência nova).
 */
export function ItemBrowser({
  items,
  iconPackVersion,
  catalogVersion,
}: {
  items: readonly CatalogItem[];
  iconPackVersion: string;
  catalogVersion: string;
}) {
  const router = useRouter();
  const [query, setQuery] = useState("");
  const [debounced, setDebounced] = useState("");
  const [slots, setSlots] = useState<readonly string[]>([]);
  const [grade, setGrade] = useState<number | 0>(0);

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim().toLowerCase()), 150);
    return () => clearTimeout(timer);
  }, [query]);

  const slotOptions = useMemo(() => distinctSlots(items), [items]);
  const gradeOptions = useMemo(() => {
    const set = new Set<number>();
    for (const item of items) if (item.grade > 0) set.add(item.grade);
    return [...set].sort((a, b) => a - b);
  }, [items]);

  const rows = useMemo(() => {
    return items.filter((item) => {
      if (grade !== 0 && item.grade !== grade) return false;
      if (slots.length > 0 && !slots.some((slot) => item.slots?.includes(slot))) return false;
      if (!debounced) return true;
      return itemSearchText(item).includes(debounced);
    });
  }, [items, debounced, slots, grade]);

  const columns = useMemo<Column<CatalogItem>[]>(
    () => [
      {
        key: "icon",
        header: "",
        render: (item) => <ItemIcon item={item} iconPackVersion={iconPackVersion} />,
      },
      {
        key: "item_index",
        header: "#",
        align: "right",
        sortValue: (item) => item.item_index,
        render: (item) => <span className="ui-mono">{item.item_index}</span>,
      },
      {
        key: "name",
        header: "Nome (catálogo)",
        sortValue: (item) => item.name,
        render: (item) => <span className="ui-mono">{item.name}</span>,
      },
      {
        key: "display_name",
        header: "Exibição",
        sortValue: (item) => item.display_name ?? "",
      },
      {
        key: "slots",
        header: "Slots",
        render: (item) => <span className="ui-field__hint">{slotsText(item)}</span>,
      },
      {
        key: "grade",
        header: "Grau",
        sortValue: (item) => item.grade,
        render: (item) =>
          item.grade > 0 ? (
            <Badge tone={gradeTone(item.grade)}>{gradeLabel(item.grade)}</Badge>
          ) : (
            <span className="ui-field__hint">—</span>
          ),
      },
    ],
    [iconPackVersion],
  );

  function toggleSlot(slot: string) {
    setSlots((prev) =>
      prev.includes(slot) ? prev.filter((s) => s !== slot) : [...prev, slot],
    );
  }

  return (
    <>
      <div className="ui-filters">
        <input
          className="ui-input"
          style={{ maxWidth: 320 }}
          type="search"
          value={query}
          placeholder="Buscar por nome, exibição ou índice…"
          aria-label="Buscar item"
          onChange={(event) => setQuery(event.target.value)}
        />
        <label className="ui-field__hint">
          Grau{" "}
          <select
            className="ui-select"
            value={grade}
            onChange={(event) => setGrade(Number(event.target.value))}
          >
            <option value={0}>todos</option>
            {gradeOptions.map((option) => (
              <option key={option} value={option}>
                {gradeLabel(option)}
              </option>
            ))}
          </select>
        </label>
        {slots.length > 0 && (
          <button type="button" className="ui-btn ui-btn--ghost" onClick={() => setSlots([])}>
            limpar slots
          </button>
        )}
        <span className="ui-field__hint" style={{ marginLeft: "auto" }}>
          catálogo {catalogVersion.slice(0, 8) || "—"}
        </span>
      </div>

      {slotOptions.length > 0 && (
        <div className="ui-filters" role="group" aria-label="Filtrar por slot">
          {slotOptions.map((slot) => (
            <button
              key={slot}
              type="button"
              className="ui-chip"
              aria-pressed={slots.includes(slot)}
              onClick={() => toggleSlot(slot)}
            >
              {slot}
            </button>
          ))}
        </div>
      )}

      <DataTable
        rows={rows}
        columns={columns}
        getRowKey={(item) => item.item_index}
        pageSizes={[100, 200, 500]}
        initialPageSize={100}
        onRowClick={(item) => router.push(`/itens/${item.item_index}`)}
        emptyMessage="Nenhum item combina com o filtro atual."
        caption={`${fmtCoin(items.length)} itens no ItemList.csv`}
      />
    </>
  );
}
