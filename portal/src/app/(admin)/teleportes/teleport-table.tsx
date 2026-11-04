"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { DataTable, type Column } from "@/components/DataTable";
import { Dialog } from "@/components/Dialog";
import { Field } from "@/components/Field";
import { useToast } from "@/components/Toast";
import {
  REQUIREMENTS,
  isRequirement,
  requirementLabel,
  routeFormSchema,
  routeOrigin,
  roundTile,
  toFormValues,
  type RouteFormValues,
  type TeleportRoute,
} from "@/lib/teleport";
import {
  deleteRoute,
  duplicateReverse,
  setRouteEnabled,
  simulateRoute,
  upsertRoute,
} from "@/server/actions/teleports";

/**
 * Tabela + editor de rotas (issue 42).
 *
 * Duas regras guiam os botões:
 *
 * - rota do conteúdo (sem linha no banco) não tem “apagar”: desativar cria a linha
 *   com `enabled=false`, que é o caminho suportado;
 * - “voltar ao padrão” só aparece quando existe linha no overlay.
 *
 * A simulação é server-side (`SimulateRoute`) e não consome o RNG do mundo.
 */
export function TeleportTable({ routes }: { routes: readonly TeleportRoute[] }) {
  const router = useRouter();
  const toast = useToast();
  const [editing, setEditing] = useState<TeleportRoute | null>(null);
  const [creating, setCreating] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [warnings, setWarnings] = useState<string[]>([]);
  const [simulation, setSimulation] = useState<string | null>(null);
  const [form, setForm] = useState<RouteFormValues>(blankRoute());

  function openCreate() {
    setForm(blankRoute());
    setEditing(null);
    setCreating(true);
    setError(null);
    setWarnings([]);
    setSimulation(null);
  }

  function openEdit(route: TeleportRoute) {
    setForm(toFormValues(route));
    setEditing(route);
    setCreating(false);
    setError(null);
    setWarnings([]);
    setSimulation(null);
  }

  function close() {
    setEditing(null);
    setCreating(false);
    setError(null);
    setWarnings([]);
    setSimulation(null);
  }

  async function save() {
    const parsed = routeFormSchema.safeParse(form);
    if (!parsed.success) {
      setError(parsed.error.issues[0]?.message ?? "dados inválidos");
      return;
    }
    setPending(true);
    const result = await upsertRoute(parsed.data);
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    toast("Rota salva — o tmserver aplica no próximo poll (15 s)", "ok");
    if (result.warnings.length > 0) setWarnings(result.warnings);
    router.refresh();
    close();
  }

  async function toggle(route: TeleportRoute, enabled: boolean) {
    setPending(true);
    const result = await setRouteEnabled({ srcX: route.src_x, srcY: route.src_y, enabled });
    setPending(false);
    if (!result.ok) {
      toast(result.error, "error");
      return;
    }
    toast(enabled ? "Rota ativada" : "Rota desativada — o tile deixa de teleportar", "ok");
    router.refresh();
  }

  async function restore(route: TeleportRoute) {
    setPending(true);
    const result = await deleteRoute({ srcX: route.src_x, srcY: route.src_y });
    setPending(false);
    if (!result.ok) {
      toast(result.error, "error");
      return;
    }
    toast("Linha removida — a rota voltou ao padrão do servidor", "ok");
    router.refresh();
  }

  async function simulate() {
    setPending(true);
    const result = await simulateRoute({ srcX: form.src_x, srcY: form.src_y });
    setPending(false);
    if (!result.ok) {
      setSimulation(result.error);
      return;
    }
    const sim = result.simulation;
    const parts = [
      sim.ok
        ? `resolve para (${sim.dst_x}..${sim.dst_x + 2}, ${sim.dst_y}..${sim.dst_y + 2}) · custo ${sim.cost} · ${requirementLabel[isRequirement(sim.requirement) ? sim.requirement : "none"]}`
        : "este tile não teleporta",
      `origem usada: ${sim.src_x},${sim.src_y}`,
      ...sim.warnings,
    ];
    setSimulation(parts.join(" · "));
  }

  async function reverse() {
    setPending(true);
    const result = await duplicateReverse(form);
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    toast("Rota de volta criada", "ok");
    router.refresh();
    close();
  }

  const columns = useMemo<Column<TeleportRoute>[]>(
    () => [
      {
        key: "origin",
        header: "Origem",
        sortValue: (route) => route.src_x * 10000 + route.src_y,
        render: (route) => <Place zone={route.src_zone} x={route.src_x} y={route.src_y} />,
      },
      {
        key: "dest",
        header: "Destino",
        sortValue: (route) => route.dst_x * 10000 + route.dst_y,
        render: (route) => <Place zone={route.dst_zone} x={route.dst_x} y={route.dst_y} />,
      },
      {
        key: "cost",
        header: "Custo",
        sortValue: (route) => route.cost,
        render: (route) => (route.cost > 0 ? `${route.cost} gold` : "grátis"),
      },
      {
        key: "requirement",
        header: "Requisito",
        sortValue: (route) => route.requirement,
        render: (route) =>
          isRequirement(route.requirement) ? requirementLabel[route.requirement] : route.requirement,
      },
      {
        key: "state",
        header: "Estado",
        sortValue: (route) => (route.enabled ? 1 : 0),
        render: (route) => (
          <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
            <Badge tone={route.enabled ? "ok" : "warn"}>
              {route.enabled ? "ativa" : "desativada"}
            </Badge>
            <Badge tone={route.has_override ? "accent" : "neutral"}>{routeOrigin(route)}</Badge>
          </div>
        ),
      },
      {
        key: "actions",
        header: "",
        render: (route) => (
          <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
            <button className="ui-btn ui-btn--ghost" onClick={() => openEdit(route)}>
              editar
            </button>
            <button
              className="ui-btn ui-btn--ghost"
              disabled={pending}
              onClick={() => toggle(route, !route.enabled)}
            >
              {route.enabled ? "desativar" : "ativar"}
            </button>
            {route.has_override && (
              <button className="ui-btn ui-btn--ghost" disabled={pending} onClick={() => restore(route)}>
                voltar ao padrão
              </button>
            )}
          </div>
        ),
      },
    ],
    [pending],
  );

  return (
    <>
      <div className="ui-table-toolbar">
        <button className="ui-btn ui-btn--primary" onClick={openCreate}>
          nova rota
        </button>
      </div>

      <DataTable<TeleportRoute>
        rows={[...routes]}
        columns={columns}
        getRowKey={(route) => `${route.src_x},${route.src_y}`}
        searchText={(route) =>
          `${route.src_zone ?? ""} ${route.dst_zone ?? ""} ${route.src_x},${route.src_y} ${route.dst_x},${route.dst_y} ${route.requirement}`
        }
        emptyMessage="Nenhuma rota carregada."
      />

      <Dialog
        open={creating || editing !== null}
        title={creating ? "Nova rota" : `Rota ${form.src_x}, ${form.src_y}`}
        onClose={close}
        footer={
          <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
            <button className="ui-btn ui-btn--ghost" onClick={close} disabled={pending}>
              cancelar
            </button>
            <button className="ui-btn ui-btn--ghost" onClick={simulate} disabled={pending}>
              simular
            </button>
            {!creating && (
              <button className="ui-btn ui-btn--ghost" onClick={reverse} disabled={pending}>
                duplicar inverso
              </button>
            )}
            <button className="ui-btn ui-btn--primary" onClick={save} disabled={pending}>
              salvar
            </button>
          </div>
        }
      >
        <div style={{ display: "grid", gap: "var(--space-2)" }}>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "var(--space-2)" }}>
            <Field label="origem X" hint={`tile ${roundTile(form.src_x)}`}>
              <input
                className="ui-input"
                type="number"
                value={form.src_x}
                onChange={(e) => setForm({ ...form, src_x: Number(e.target.value) })}
              />
            </Field>
            <Field label="origem Y" hint={`tile ${roundTile(form.src_y)}`}>
              <input
                className="ui-input"
                type="number"
                value={form.src_y}
                onChange={(e) => setForm({ ...form, src_y: Number(e.target.value) })}
              />
            </Field>
            <Field label="destino X">
              <input
                className="ui-input"
                type="number"
                value={form.dst_x}
                onChange={(e) => setForm({ ...form, dst_x: Number(e.target.value) })}
              />
            </Field>
            <Field label="destino Y">
              <input
                className="ui-input"
                type="number"
                value={form.dst_y}
                onChange={(e) => setForm({ ...form, dst_y: Number(e.target.value) })}
              />
            </Field>
            <Field label="custo (gold)">
              <input
                className="ui-input"
                type="number"
                value={form.cost}
                onChange={(e) => setForm({ ...form, cost: Number(e.target.value) })}
              />
            </Field>
            <Field label="requisito">
              <select
                className="ui-input"
                value={form.requirement}
                onChange={(e) =>
                  setForm({ ...form, requirement: e.target.value as RouteFormValues["requirement"] })
                }
              >
                {REQUIREMENTS.map((req) => (
                  <option key={req} value={req}>
                    {requirementLabel[req]}
                  </option>
                ))}
              </select>
            </Field>
          </div>

          <Field label="nota" hint="aparece só no portal">
            <input
              className="ui-input"
              value={form.note}
              maxLength={120}
              onChange={(e) => setForm({ ...form, note: e.target.value })}
            />
          </Field>

          <label style={{ display: "flex", gap: 8, alignItems: "center" }}>
            <input
              type="checkbox"
              checked={form.enabled}
              onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
            />
            rota ativa
          </label>

          {simulation && <p className="ui-field__hint">{simulation}</p>}
          {warnings.map((warning) => (
            <p key={warning} className="ui-field__hint" style={{ color: "var(--warn)" }}>
              {warning}
            </p>
          ))}
          {error && <p style={{ color: "var(--danger)", margin: 0 }}>{error}</p>}
        </div>
      </Dialog>
    </>
  );
}

/**
 * Origem/destino na tabela: nome do lugar primeiro, coordenada como detalhe
 * (issue 47 — nenhuma tela mostra número cru quando existe nome). Sem atlas, o
 * nome é omitido e a coordenada continua legível.
 */
function Place({ zone, x, y }: { zone?: string; x: number; y: number }) {
  return (
    <span style={{ display: "grid", gap: "var(--space-1)" }}>
      {zone && <span>{zone}</span>}
      <span className="ui-mono ui-field__hint" style={{ margin: 0 }} title={`coordenada ${x}, ${y}`}>
        {x}, {y}
      </span>
    </span>
  );
}

function blankRoute(): RouteFormValues {
  return {
    src_x: 0,
    src_y: 0,
    dst_x: 0,
    dst_y: 0,
    cost: 0,
    requirement: "none",
    enabled: true,
    note: "",
  };
}
