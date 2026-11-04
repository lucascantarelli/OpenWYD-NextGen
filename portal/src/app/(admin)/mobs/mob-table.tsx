"use client";

import Link from "next/link";
import { useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { DataTable, type Column } from "@/components/DataTable";
import type { MobTemplateFile } from "@/lib/mob";

/**
 * Tabela de templates (issue 25). O nome do arquivo é a chave estável, então a
 * coluna principal é ele — o `display_name` do `STRUCT_MOB` vem ao lado como
 * o nome que aparece no jogo (e pode estar em branco).
 *
 * O filtro "somente com override" é client-side porque a lista inteira (1.991
 * templates) já vem numa chamada só; o flag `overridden` vem do web-api, que o
 * resolve com uma consulta ao banco por listagem.
 */
export function MobTable({ templates }: { templates: readonly MobTemplateFile[] }) {
  const [onlyOverridden, setOnlyOverridden] = useState(false);
  const overrides = useMemo(() => templates.filter((t) => t.overridden).length, [templates]);
  const rows = useMemo(
    () => (onlyOverridden ? templates.filter((t) => t.overridden) : templates),
    [templates, onlyOverridden],
  );

  const columns = useMemo<Column<MobTemplateFile>[]>(
    () => [
      {
        key: "template",
        header: "Template",
        sortValue: (row) => row.template_name,
        render: (row) => (
          <Link href={`/mobs/${encodeURIComponent(row.template_name)}`} className="ui-mono">
            {row.template_name}
          </Link>
        ),
      },
      {
        key: "name",
        header: "Nome no jogo",
        sortValue: (row) => (row.display_name || "").toLowerCase(),
        render: (row) => row.display_name || <span className="ui-field__hint">—</span>,
      },
      {
        key: "merchant",
        header: "Merchant",
        sortValue: (row) => row.merchant,
        render: (row) =>
          row.merchant > 0 ? <Badge tone="accent">{row.merchant}</Badge> : <span>—</span>,
      },
      {
        key: "overridden",
        header: "Override",
        sortValue: (row) => (row.overridden ? 1 : 0),
        render: (row) =>
          row.overridden ? <Badge tone="warn">no banco</Badge> : <span>—</span>,
      },
      {
        key: "open",
        header: "",
        render: (row) => (
          <Link className="ui-btn ui-btn--ghost" href={`/mobs/${encodeURIComponent(row.template_name)}`}>
            editar
          </Link>
        ),
      },
    ],
    [],
  );

  return (
    <>
      <div className="ui-table-toolbar" style={{ marginBottom: "var(--space-2)" }}>
        <label style={{ display: "flex", gap: "var(--space-1)", alignItems: "center" }}>
          <input
            type="checkbox"
            checked={onlyOverridden}
            disabled={overrides === 0}
            onChange={(event) => setOnlyOverridden(event.target.checked)}
          />
          <span className="ui-field__hint" style={{ margin: 0 }}>
            somente com override ({overrides})
          </span>
        </label>
      </div>
      <DataTable<MobTemplateFile>
        rows={[...rows]}
        columns={columns}
        getRowKey={(row) => row.template_name}
        searchText={(row) => `${row.template_name} ${row.display_name}`}
        searchPlaceholder="template ou nome do mob…"
        caption="Templates de Release/TMsrv/run/npc/ — monstros e mercadores"
      />
    </>
  );
}
