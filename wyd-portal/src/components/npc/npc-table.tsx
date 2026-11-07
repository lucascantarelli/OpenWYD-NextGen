"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { DataTable, type Column } from "@/components/DataTable";
import { Dialog } from "@/components/Dialog";
import { Field } from "@/components/Field";
import { useToast } from "@/components/Toast";
import { hasShop, merchantLabel, originLabel, type MapZone, type NpcDefinition } from "@/lib/npc";
import { deleteNpc, duplicateNpc, setNpcVisibility } from "@/server/actions/npcs";

/**
 * Lista de NPCs com as ações de linha.
 *
 * `DeleteNpc` em NPC de conteúdo devolve `CONTENT_OWNED` → o diálogo troca o
 * botão de apagar por **ocultar**, em vez de mostrar "erro interno" (bug #257).
 */
export function NpcTable({
  npcs,
  zones,
}: {
  npcs: readonly NpcDefinition[];
  zones: readonly MapZone[];
}) {
  const router = useRouter();
  const toast = useToast();
  const [onlyHidden, setOnlyHidden] = useState(false);
  const [target, setTarget] = useState<NpcDefinition | null>(null);
  const [confirmSlug, setConfirmSlug] = useState("");
  const [owned, setOwned] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  const zoneName = useMemo(() => {
    const map = new Map(zones.map((zone) => [zone.id, zone.name]));
    return (mapId: number) => map.get(mapId) ?? `#${mapId}`;
  }, [zones]);

  const rows = useMemo(
    () => (onlyHidden ? npcs.filter((npc) => !npc.enabled) : npcs),
    [npcs, onlyHidden],
  );

  async function toggle(npc: NpcDefinition, enabled: boolean) {
    setPending(true);
    const result = await setNpcVisibility({ npcId: npc.id, enabled });
    setPending(false);
    if (!result.ok) {
      toast(result.error, "error");
      return;
    }
    toast(enabled ? "NPC visível" : "NPC oculto — o jogo reflete no próximo restart", "ok");
    router.refresh();
  }

  async function duplicate(npc: NpcDefinition) {
    setPending(true);
    const result = await duplicateNpc({ npcId: npc.id });
    setPending(false);
    if (!result.ok) {
      toast(result.error, "error");
      return;
    }
    toast(`Cópia criada (${npc.slug}-copy)`, "ok");
    router.push(`/npcs/${result.npcId}`);
  }

  async function remove() {
    if (!target || confirmSlug !== target.slug) return;
    setPending(true);
    const result = await deleteNpc({ npcId: target.id, slug: target.slug });
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      setOwned(Boolean(result.contentOwned));
      return;
    }
    toast("NPC apagado", "ok");
    setTarget(null);
    router.refresh();
  }

  async function hideInstead() {
    if (!target) return;
    setPending(true);
    const result = await setNpcVisibility({ npcId: target.id, enabled: false });
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    toast("NPC ocultado — o jogo reflete no próximo restart", "ok");
    setTarget(null);
    router.refresh();
  }

  const columns = useMemo<Column<NpcDefinition>[]>(
    () => [
      {
        key: "slug",
        header: "Slug",
        sortValue: (npc) => npc.slug,
        render: (npc) => (
          <Link href={`/npcs/${npc.id}`} className="ui-mono">
            {npc.slug}
          </Link>
        ),
      },
      { key: "display_name", header: "Nome", sortValue: (npc) => npc.display_name },
      {
        key: "template_name",
        header: "Template",
        sortValue: (npc) => npc.template_name,
        render: (npc) => <span className="ui-mono">{npc.template_name}</span>,
      },
      {
        key: "map_id",
        header: "Mapa",
        sortValue: (npc) => npc.map_id,
        render: (npc) => zoneName(npc.map_id),
      },
      {
        key: "pos",
        header: "Posição",
        render: (npc) => (
          <span className="ui-mono">
            {npc.pos_x},{npc.pos_y}
          </span>
        ),
      },
      {
        key: "merchant",
        header: "Mercador",
        sortValue: (npc) => npc.merchant,
        render: (npc) => (
          <span title={merchantLabel(npc.merchant)}>
            {npc.merchant > 0 ? (
              <Badge tone="accent">{npc.merchant}</Badge>
            ) : (
              <span className="ui-field__hint">—</span>
            )}
          </span>
        ),
      },
      {
        key: "shop",
        header: "Loja",
        align: "right",
        render: (npc) => (
          <span className="ui-mono">
            {hasShop(npc.merchant) ? `${npc.shop?.length ?? 0}/${27}` : "—"}
          </span>
        ),
      },
      {
        key: "origin",
        header: "Origem",
        sortValue: (npc) => npc.origin,
        render: (npc) => (
          <Badge tone={npc.origin === "content" ? "warn" : "ok"}>{originLabel(npc.origin)}</Badge>
        ),
      },
      {
        key: "enabled",
        header: "Status",
        sortValue: (npc) => (npc.enabled ? 1 : 0),
        render: (npc) =>
          npc.enabled ? <Badge tone="ok">visível</Badge> : <Badge tone="danger">oculto</Badge>,
      },
      {
        key: "actions",
        header: "Ações",
        render: (npc) => (
          <span style={{ display: "inline-flex", gap: 6 }}>
            <button
              type="button"
              className="ui-btn ui-btn--ghost"
              disabled={pending}
              onClick={() => toggle(npc, !npc.enabled)}
            >
              {npc.enabled ? "ocultar" : "mostrar"}
            </button>
            <button
              type="button"
              className="ui-btn ui-btn--ghost"
              disabled={pending}
              onClick={() => duplicate(npc)}
            >
              duplicar
            </button>
            <button
              type="button"
              className="ui-btn ui-btn--danger"
              disabled={pending}
              onClick={() => {
                setTarget(npc);
                setConfirmSlug("");
                setOwned(false);
                setError(null);
              }}
            >
              apagar
            </button>
          </span>
        ),
      },
    ],
    // `pending` entra porque os botões desabilitam durante uma ação.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [pending, zoneName],
  );

  return (
    <>
      <div className="ui-filters">
        <label className="ui-field__hint">
          <input
            type="checkbox"
            checked={onlyHidden}
            onChange={(event) => setOnlyHidden(event.target.checked)}
          />{" "}
          somente ocultos
        </label>
        <span className="ui-field__hint" style={{ marginLeft: "auto" }}>
          {npcs.length} definições · {npcs.filter((npc) => npc.enabled).length} visíveis
        </span>
      </div>

      <DataTable
        rows={rows}
        columns={columns}
        getRowKey={(npc) => npc.id}
        searchText={(npc) => `${npc.slug} ${npc.display_name} ${npc.template_name}`}
        searchPlaceholder="Buscar por slug, nome ou template…"
        pageSizes={[25, 50, 100]}
        initialPageSize={25}
        emptyMessage="Nenhum NPC com esse filtro."
        caption="Definições de NPC/mercador (Postgres, aplicadas pelo tmserver)"
      />

      <Dialog
        open={target !== null}
        onClose={() => {
          setTarget(null);
          setOwned(false);
          setError(null);
        }}
        title={`Apagar ${target?.slug ?? ""}?`}
        description={
          owned
            ? "Este NPC é conteúdo versionado (NPCGener.txt): o servidor recusa apagar. Oculte em vez de apagar — o próximo import-npcs o traria de volta."
            : "A definição sai do Postgres. Para NPC de conteúdo o servidor recusa e sugere ocultar."
        }
        footer={
          <>
            <button
              type="button"
              className="ui-btn"
              onClick={() => {
                setTarget(null);
                setOwned(false);
                setError(null);
              }}
            >
              cancelar
            </button>
            {owned ? (
              <button
                type="button"
                className="ui-btn ui-btn--primary"
                disabled={pending}
                onClick={hideInstead}
              >
                ocultar em vez de apagar
              </button>
            ) : (
              <button
                type="button"
                className="ui-btn ui-btn--danger"
                disabled={pending || confirmSlug !== target?.slug}
                onClick={remove}
              >
                apagar definitivamente
              </button>
            )}
          </>
        }
      >
        {target && (
          <>
            <Field
              label="Digite o slug para confirmar"
              hint={`template ${target.template_name} · origem ${originLabel(target.origin)}`}
              htmlFor="confirm-slug"
            >
              <input
                id="confirm-slug"
                className="ui-input"
                value={confirmSlug}
                onChange={(event) => setConfirmSlug(event.target.value)}
                autoComplete="off"
              />
            </Field>
            {error && (
              <p className="ui-field__error" role="alert">
                {error}
              </p>
            )}
          </>
        )}
      </Dialog>
    </>
  );
}
