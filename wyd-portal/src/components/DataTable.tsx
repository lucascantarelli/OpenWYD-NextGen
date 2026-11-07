"use client";

import { useMemo, useState } from "react";

export interface Column<T> {
  key: string;
  header: string;
  render?: (row: T) => React.ReactNode;
  /** Presente = coluna ordenável. */
  sortValue?: (row: T) => string | number;
  align?: "left" | "right";
}

export interface DataTableProps<T> {
  rows: readonly T[];
  columns: readonly Column<T>[];
  getRowKey: (row: T) => string | number;
  /** Texto usado pela busca (concatene os campos pesquisáveis). */
  searchText?: (row: T) => string;
  searchPlaceholder?: string;
  onRowClick?: (row: T) => void;
  emptyMessage?: string;
  pageSizes?: readonly number[];
  initialPageSize?: number;
  /** Descrição para leitores de tela. */
  caption?: string;
}

type Sort = { key: string; dir: "asc" | "desc" } | null;

/**
 * Tabela única do portal: busca client-side, ordenação opcional por coluna e
 * paginação. Nenhum fetch aqui — a tela dona dos dados decide de onde vêm.
 */
export function DataTable<T>({
  rows,
  columns,
  getRowKey,
  searchText,
  searchPlaceholder = "Buscar…",
  onRowClick,
  emptyMessage = "Nada encontrado.",
  pageSizes = [50, 100, 200],
  initialPageSize = 50,
  caption,
}: DataTableProps<T>) {
  const [query, setQuery] = useState("");
  const [pageSize, setPageSize] = useState(initialPageSize);
  const [page, setPage] = useState(0);
  const [sort, setSort] = useState<Sort>(null);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle || !searchText) return rows;
    return rows.filter((row) => searchText(row).toLowerCase().includes(needle));
  }, [rows, query, searchText]);

  const sorted = useMemo(() => {
    if (!sort) return filtered;
    const column = columns.find((c) => c.key === sort.key);
    if (!column?.sortValue) return filtered;
    const value = column.sortValue;
    const factor = sort.dir === "asc" ? 1 : -1;
    return [...filtered].sort((a, b) => {
      const x = value(a);
      const y = value(b);
      const result =
        typeof x === "number" && typeof y === "number"
          ? x - y
          : String(x).localeCompare(String(y), "pt-BR");
      return result * factor;
    });
  }, [filtered, sort, columns]);

  const pageCount = Math.max(1, Math.ceil(sorted.length / pageSize));
  const current = Math.min(page, pageCount - 1);
  const visible = sorted.slice(current * pageSize, (current + 1) * pageSize);

  function toggleSort(key: string) {
    setSort((prev) =>
      prev?.key === key
        ? { key, dir: prev.dir === "asc" ? "desc" : "asc" }
        : { key, dir: "asc" },
    );
    setPage(0);
  }

  return (
    <div className="ui-table-wrap">
      <div className="ui-table-toolbar">
        {searchText && (
          <input
            className="ui-input"
            style={{ maxWidth: 320 }}
            type="search"
            value={query}
            placeholder={searchPlaceholder}
            aria-label={searchPlaceholder}
            onChange={(event) => {
              setQuery(event.target.value);
              setPage(0);
            }}
          />
        )}
        <span className="ui-field__hint" style={{ marginLeft: "auto" }}>
          {sorted.length} registro{sorted.length === 1 ? "" : "s"}
        </span>
      </div>

      <div className="ui-table-scroll">
        <table className="ui-table">
          {caption && <caption className="ui-field__hint" style={{ padding: 6 }}>{caption}</caption>}
          <thead>
            <tr>
              {columns.map((column) => {
                const active = sort?.key === column.key;
                return (
                  <th
                    key={column.key}
                    className={column.align === "right" ? "ui-align-right" : undefined}
                    aria-sort={active ? (sort.dir === "asc" ? "ascending" : "descending") : undefined}
                  >
                    {column.sortValue ? (
                      <button type="button" onClick={() => toggleSort(column.key)}>
                        {column.header}
                      </button>
                    ) : (
                      <span style={{ display: "block", padding: "0 10px", height: 32, lineHeight: "32px" }}>
                        {column.header}
                      </span>
                    )}
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody>
            {visible.length === 0 ? (
              <tr>
                <td className="ui-table__empty" colSpan={columns.length}>
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              visible.map((row) => (
                <tr
                  key={getRowKey(row)}
                  data-clickable={onRowClick ? "true" : "false"}
                  onClick={onRowClick ? () => onRowClick(row) : undefined}
                >
                  {columns.map((column) => (
                    <td
                      key={column.key}
                      className={column.align === "right" ? "ui-align-right" : undefined}
                    >
                      {column.render
                        ? column.render(row)
                        : String((row as Record<string, unknown>)[column.key] ?? "—")}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <div className="ui-table-footer">
        <span>
          Página {current + 1} de {pageCount}
        </span>
        <div className="ui-table-footer__pager">
          <label className="ui-field__hint">
            Linhas{" "}
            <select
              className="ui-select"
              value={pageSize}
              onChange={(event) => {
                setPageSize(Number(event.target.value));
                setPage(0);
              }}
            >
              {pageSizes.map((size) => (
                <option key={size} value={size}>
                  {size}
                </option>
              ))}
            </select>
          </label>
          <button
            type="button"
            className="ui-btn"
            disabled={current === 0}
            onClick={() => setPage((p) => Math.max(p - 1, 0))}
          >
            Anterior
          </button>
          <button
            type="button"
            className="ui-btn"
            disabled={current >= pageCount - 1}
            onClick={() => setPage((p) => Math.min(p + 1, pageCount - 1))}
          >
            Próxima
          </button>
        </div>
      </div>
    </div>
  );
}
