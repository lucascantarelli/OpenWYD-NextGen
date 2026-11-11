"use client";

import Link from "next/link";
import { useMemo } from "react";

import { DataTable, type Column } from "@/components/DataTable";
import type { CatalogItem } from "@/server/admin-client";

/**
 * Navegador do catálogo.
 *
 * O gateway expõe hoje índice, nome e nome de exibição. A tabela não mostra
 * coluna que o contrato não fornece — grade, ícone e preço só entram quando
 * existirem no payload.
 */
export function ItemBrowser({
  items,
  catalogVersion,
}: {
  items: readonly CatalogItem[];
  catalogVersion: string;
}) {
  const columns = useMemo<Column<CatalogItem>[]>(
    () => [
      {
        key: "item_index",
        header: "Índice",
        align: "right",
        sortValue: (item) => item.item_index,
        render: (item) => (
        <Link href={`/itens?item=${item.item_index}`} className="mono">
          {item.item_index}
        </Link>
      ),
      },
      {
        key: "name",
        header: "Nome no catálogo",
        sortValue: (item) => item.name,
        render: (item) => <span className="mono">{item.name}</span>,
      },
      {
        key: "display_name",
        header: "Nome de exibição",
        sortValue: (item) => item.display_name ?? "",
        render: (item) => item.display_name || <span className="muted">—</span>,
      },
    ],
    [],
  );

  return (
    <>
      <div className="ui-filters">
        <span className="ui-field__hint">
          {items.length.toLocaleString("pt-BR")} itens
          {catalogVersion ? ` · versão ${catalogVersion.slice(0, 8)}` : ""}
        </span>
      </div>

      <DataTable
        rows={items}
        columns={columns}
        getRowKey={(item) => item.item_index}
        searchText={(item) => `${item.item_index} ${item.name} ${item.display_name}`}
        searchPlaceholder="Buscar por índice, nome ou exibição…"
        pageSizes={[100, 200, 500]}
        initialPageSize={100}
        emptyMessage="Nenhum item combina com a busca."
        caption={`${items.length} itens carregados no servidor`}
      />
    </>
  );
}
