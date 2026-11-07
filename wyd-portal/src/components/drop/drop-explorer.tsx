"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { DataTable, type Column } from "@/components/DataTable";
import { Tabs } from "@/components/Tabs";
import {
  DROP_NOTE,
  dropCounters,
  dropsToCsv,
  flattenItemDrops,
  flattenMobDrops,
  rarityOf,
  type ItemDropEntry,
  type ItemDropRow,
  type MobDropEntry,
  type MobDropRow,
} from "@/lib/drop";
import { searchItemDrops, searchMobDrops } from "@/server/actions/drops";

const TABS = [
  { id: "item", label: "Por item" },
  { id: "mob", label: "Por mob" },
] as const;

const MIN_QUERY = 2;

/**
 * Relatório de drops (issue 26): duas visões do mesmo `Carry[]`.
 *
 * A busca é server-side e com debounce porque consultar sem filtro devolve o
 * catálogo inteiro de pares. O botão "tentar de novo" existe porque todo erro
 * precisa de uma ação de recuperação (issue 47), não só da mensagem.
 */
export function DropExplorer() {
  const [tab, setTab] = useState<string>("item");
  const [itemQuery, setItemQuery] = useState("");
  const [mobQuery, setMobQuery] = useState("");
  const [includeZero, setIncludeZero] = useState(false);
  const [itemEntries, setItemEntries] = useState<ItemDropEntry[]>([]);
  const [mobEntries, setMobEntries] = useState<MobDropEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (tab !== "item") return;
    const query = itemQuery.trim();
    if (query.length < MIN_QUERY) {
      setItemEntries([]);
      setLoading(false);
      setError(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    const timer = setTimeout(() => {
      void searchItemDrops({ itemQuery: query, includeZeroDropItems: includeZero }).then(
        (result) => {
          if (cancelled) return;
          setLoading(false);
          if (!result.ok) {
            setError(result.error);
            return;
          }
          setError(null);
          setItemEntries(result.entries);
        },
      );
    }, 350);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [tab, itemQuery, includeZero, attempt]);

  useEffect(() => {
    if (tab !== "mob") return;
    const query = mobQuery.trim();
    if (query.length < MIN_QUERY) {
      setMobEntries([]);
      setLoading(false);
      setError(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    const timer = setTimeout(() => {
      void searchMobDrops({ mobQuery: query }).then((result) => {
        if (cancelled) return;
        setLoading(false);
        if (!result.ok) {
          setError(result.error);
          return;
        }
        setError(null);
        setMobEntries(result.entries);
      });
    }, 350);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [tab, mobQuery, attempt]);

  const itemRows = useMemo(() => flattenItemDrops(itemEntries), [itemEntries]);
  const mobRows = useMemo(() => flattenMobDrops(mobEntries), [mobEntries]);
  const counters = useMemo(() => dropCounters(tab === "item" ? itemRows : mobRows), [tab, itemRows, mobRows]);

  const rarityColumn = <T extends { rate_divisor: number }>(): Column<T> => ({
    key: "rarity",
    header: "Raridade",
    sortValue: (row) => rarityOf(row.rate_divisor).order,
    render: (row) => {
      const band = rarityOf(row.rate_divisor);
      return (
        <span style={{ display: "inline-flex", gap: "var(--space-1)", alignItems: "center" }}>
          <Badge tone={band.tone}>{band.label}</Badge>
          <span className="ui-mono ui-field__hint" style={{ margin: 0 }}>
            ÷{row.rate_divisor}
          </span>
        </span>
      );
    },
  });

  const itemColumns = useMemo<Column<ItemDropRow>[]>(
    () => [
      {
        key: "item",
        header: "Item",
        sortValue: (row) => row.item_name.toLowerCase(),
        render: (row) => (
          <Link href={`/itens/${row.item_index}`} title={`código ${row.item_index}`}>
            {row.item_name || `#${row.item_index}`}
          </Link>
        ),
      },
      {
        key: "mob",
        header: "Mob",
        sortValue: (row) => (row.mob_name || row.template_name).toLowerCase(),
        render: (row) => row.mob_name || row.template_name,
      },
      {
        key: "template",
        header: "Template",
        sortValue: (row) => row.template_name,
        render: (row) => <span className="ui-mono">{row.template_name}</span>,
      },
      { key: "level", header: "Nível", sortValue: (row) => row.mob_level, render: (row) => row.mob_level },
      { key: "slot", header: "Slot", sortValue: (row) => row.slot, render: (row) => row.slot },
      rarityColumn<ItemDropRow>(),
    ],
    [],
  );

  const mobColumns = useMemo<Column<MobDropRow>[]>(
    () => [
      {
        key: "mob",
        header: "Mob",
        sortValue: (row) => (row.mob_name || row.template_name).toLowerCase(),
        render: (row) => row.mob_name || row.template_name,
      },
      {
        key: "template",
        header: "Template",
        sortValue: (row) => row.template_name,
        render: (row) => <span className="ui-mono">{row.template_name}</span>,
      },
      { key: "level", header: "Nível", sortValue: (row) => row.mob_level, render: (row) => row.mob_level },
      { key: "slot", header: "Slot", sortValue: (row) => row.slot, render: (row) => row.slot },
      {
        key: "item",
        header: "Item",
        sortValue: (row) => (row.item_name || String(row.item_index)).toLowerCase(),
        render: (row) => (
          <Link href={`/itens/${row.item_index}`} title={`código ${row.item_index}`}>
            {row.item_name || `#${row.item_index}`}
          </Link>
        ),
      },
      rarityColumn<MobDropRow>(),
    ],
    [],
  );

  function exportCsv() {
    const rows: Record<string, string | number>[] =
      tab === "item"
        ? itemRows.map((row) => ({ ...row, raridade: rarityOf(row.rate_divisor).label }))
        : mobRows.map((row) => ({ ...row, raridade: rarityOf(row.rate_divisor).label }));
    const csv = dropsToCsv(rows);
    if (!csv) return;
    const distinct = new Set(rows.map((row) => (tab === "item" ? row.item_index : row.template_name)));
    const suffix = distinct.size === 1 ? `-${[...distinct][0]}` : "";
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `drops-${tab === "item" ? "item" : "mob"}${suffix}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  }

  const query = tab === "item" ? itemQuery.trim() : mobQuery.trim();
  const ready = query.length >= MIN_QUERY;
  const rows = tab === "item" ? itemRows.length : mobRows.length;

  return (
    <>
      <div className="ui-card" style={{ marginBottom: "var(--space-3)" }}>
        <strong>Como ler</strong>
        <p style={{ margin: "var(--space-1) 0 0" }}>{DROP_NOTE}</p>
      </div>

      <div className="ui-filters" style={{ alignItems: "flex-end" }}>
        <Tabs tabs={TABS} value={tab} onChange={setTab} label="Visão do relatório" />
        {tab === "item" ? (
          <input
            className="ui-input"
            style={{ maxWidth: 320 }}
            type="search"
            value={itemQuery}
            placeholder="item: nome ou código (mín. 2 letras)"
            aria-label="Buscar item"
            onChange={(event) => setItemQuery(event.target.value)}
          />
        ) : (
          <input
            className="ui-input"
            style={{ maxWidth: 320 }}
            type="search"
            value={mobQuery}
            placeholder="mob: template ou nome (mín. 2 letras)"
            aria-label="Buscar mob"
            onChange={(event) => setMobQuery(event.target.value)}
          />
        )}
        {tab === "item" && (
          <label style={{ display: "flex", gap: 8, alignItems: "center" }}>
            <input
              type="checkbox"
              checked={includeZero}
              onChange={(event) => setIncludeZero(event.target.checked)}
            />
            incluir itens sem drop
          </label>
        )}
        <button
          className="ui-btn ui-btn--ghost"
          onClick={exportCsv}
          disabled={rows === 0}
          style={{ marginLeft: "auto" }}
        >
          exportar CSV
        </button>
      </div>

      <p className="ui-field__hint" style={{ marginBottom: "var(--space-3)" }}>
        {!ready
          ? `digite ao menos ${MIN_QUERY} caracteres para consultar`
          : loading
            ? "consultando o catálogo…"
            : `${counters.pairs} pares · ${counters.items} itens · ${counters.mobs} mobs`}
      </p>

      {error && (
        <div className="ui-card ui-empty" style={{ marginBottom: "var(--space-3)" }}>
          <strong>Não deu para consultar os drops</strong>
          <p>{error}</p>
          <button className="ui-btn" onClick={() => setAttempt((value) => value + 1)}>
            tentar de novo
          </button>
        </div>
      )}

      {!error && ready && rows === 0 && !loading && (
        <div className="ui-card ui-empty">
          <strong>Nenhum drop encontrado</strong>
          <p>
            Nada casa com “{query}”. Tente parte do nome do item ou o nome do template do mob
            (ex.: <span className="ui-mono">alp</span> para os lobos).
          </p>
        </div>
      )}

      {!error && rows > 0 && tab === "item" && (
        <DataTable<ItemDropRow>
          rows={itemRows}
          columns={itemColumns}
          getRowKey={(row) => `${row.item_index}:${row.template_name}:${row.slot}`}
          searchText={(row) =>
            `${row.item_index} ${row.item_name} ${row.template_name} ${row.mob_name}`
          }
          searchPlaceholder="filtrar resultado…"
          caption="Item → mobs que podem dropá-lo, com o slot do Carry[]"
        />
      )}

      {!error && rows > 0 && tab === "mob" && (
        <DataTable<MobDropRow>
          rows={mobRows}
          columns={mobColumns}
          getRowKey={(row) => `${row.template_name}:${row.slot}`}
          searchText={(row) =>
            `${row.template_name} ${row.mob_name} ${row.item_index} ${row.item_name}`
          }
          searchPlaceholder="filtrar resultado…"
          caption="Mob → slots Carry[] do template"
        />
      )}
    </>
  );
}
