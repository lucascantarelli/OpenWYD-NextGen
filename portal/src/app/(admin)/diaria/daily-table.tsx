"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { Combobox } from "@/components/Combobox";
import { DataTable, type Column } from "@/components/DataTable";
import { Dialog } from "@/components/Dialog";
import { Field } from "@/components/Field";
import { ItemIcon } from "@/components/ItemIcon";
import { useToast } from "@/components/Toast";
import {
  DESCRIPTION_MAX,
  EFFECT_MAX,
  EXPIRES_MAX,
  TITLE_MAX,
  TITLE_MIN,
  effectsSummary,
  emptyDailyReward,
  expiryLabel,
  validateDailyReward,
  type DailyRewardItem,
} from "@/lib/daily";
import { loadItemOptions } from "@/lib/itemsearch";
import { deleteReward, duplicateReward, saveReward, setRewardEnabled } from "@/server/actions/daily";
import type { CatalogItem, ItemEffectType } from "@/server/grpc";

export interface DailyRewardRow extends DailyRewardItem {
  /** `null` quando o item não está no catálogo (oferta antiga ou catálogo fora). */
  item: CatalogItem | null;
}

const EFFECT_PAIRS = [
  ["eff1", "effv1"],
  ["eff2", "effv2"],
  ["eff3", "effv3"],
] as const;

/**
 * Tabela + editor da recompensa diária (issue 30).
 *
 * A coluna de item mostra nome e ícone porque a oferta é lida por jogadores —
 * um `item_index` cru na vitrine é o bug de UX que a issue 47 padronizou.
 */
export function DailyTable({
  rows,
  effectTypes,
  iconPackVersion,
}: {
  rows: readonly DailyRewardRow[];
  effectTypes: readonly ItemEffectType[];
  iconPackVersion: string;
}) {
  const router = useRouter();
  const toast = useToast();
  const [draft, setDraft] = useState<DailyRewardItem | null>(null);
  const [picked, setPicked] = useState<{ index: number; name: string } | null>(null);
  const [pending, setPending] = useState(false);
  const [toDelete, setToDelete] = useState<DailyRewardRow | null>(null);

  const errors = draft ? validateDailyReward(draft) : {};
  const invalid = Object.keys(errors).length > 0;

  function openNew() {
    setDraft(emptyDailyReward());
    setPicked(null);
  }

  function openEdit(row: DailyRewardRow) {
    setDraft({ ...row });
    setPicked({
      index: row.item_index,
      name: row.item ? row.item.display_name || row.item.name : `item #${row.item_index}`,
    });
  }

  function set<K extends keyof DailyRewardItem>(key: K, value: DailyRewardItem[K]) {
    setDraft((current) => (current ? { ...current, [key]: value } : current));
  }

  async function run(
    action: () => Promise<{ ok: true; message: string } | { ok: false; error: string }>,
  ): Promise<boolean> {
    setPending(true);
    const result = await action();
    setPending(false);
    if (!result.ok) {
      toast(result.error, "error");
      return false;
    }
    toast(result.message, "ok");
    router.refresh();
    return true;
  }

  async function submit() {
    if (!draft) return;
    if (invalid) {
      toast(Object.values(errors)[0] ?? "oferta inválida", "error");
      return;
    }
    if (await run(() => saveReward(draft))) {
      setDraft(null);
      setPicked(null);
    }
  }

  const columns = useMemo<Column<DailyRewardRow>[]>(
    () => [
      {
        key: "title",
        header: "oferta",
        sortValue: (row) => row.title,
        render: (row) => (
          <div>
            <span>{row.title}</span>
            {row.description && (
              <span className="ui-field__hint" style={{ display: "block" }}>
                {row.description}
              </span>
            )}
          </div>
        ),
      },
      {
        key: "item",
        header: "item entregue",
        sortValue: (row) => row.item?.display_name || row.item?.name || row.item_index,
        render: (row) => (
          <div style={{ display: "flex", alignItems: "center", gap: "var(--space-2)" }}>
            {row.item && <ItemIcon item={row.item} iconPackVersion={iconPackVersion} />}
            <span>
              {row.item ? row.item.display_name || row.item.name : `item #${row.item_index}`}
            </span>
          </div>
        ),
      },
      {
        key: "expires_days",
        header: "validade",
        align: "right",
        sortValue: (row) => row.expires_days,
        render: (row) => expiryLabel(row.expires_days),
      },
      {
        key: "effects",
        header: "adicionais",
        render: (row) => effectsSummary(row),
      },
      {
        key: "enabled",
        header: "estado",
        render: (row) => (
          <Badge tone={row.enabled ? "ok" : "neutral"}>
            {row.enabled ? "habilitada" : "desabilitada"}
          </Badge>
        ),
      },
      {
        key: "actions",
        header: "ações",
        render: (row) => (
          <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
            <button className="ui-btn ui-btn--ghost" onClick={() => openEdit(row)}>
              editar
            </button>
            <button
              className="ui-btn ui-btn--ghost"
              disabled={pending}
              onClick={() =>
                void run(() => setRewardEnabled({ id: row.id, enabled: !row.enabled }))
              }
            >
              {row.enabled ? "desabilitar" : "habilitar"}
            </button>
            <button
              className="ui-btn ui-btn--ghost"
              disabled={pending}
              onClick={() => void run(() => duplicateReward({ id: row.id }))}
            >
              duplicar
            </button>
            <button
              className="ui-btn ui-btn--danger"
              disabled={pending}
              onClick={() => setToDelete(row)}
            >
              apagar
            </button>
          </div>
        ),
      },
    ],
    // `run`/`openEdit` fecham sobre estado local e são recriados a cada render.
    [iconPackVersion, pending],
  );

  return (
    <>
      <div className="ui-filters">
        <div>
          <p className="ui-page-sub" style={{ margin: 0 }}>
            O catálogo é livre de preço: o limite é <strong>uma oferta por conta por dia (UTC)</strong>{" "}
            e a entrega acontece no <strong>próximo login</strong> do personagem, pelo{" "}
            <code>delivery_queue</code>. Habilitar duas ofertas não dá dois resgates.
          </p>
        </div>
        <button className="ui-btn ui-btn--primary" style={{ marginLeft: "auto" }} onClick={openNew}>
          nova oferta
        </button>
      </div>

      <DataTable
        rows={rows}
        columns={columns}
        getRowKey={(row) => row.id}
        searchText={(row) =>
          `${row.title} ${row.description} ${row.item?.display_name ?? ""} ${
            row.item?.name ?? ""
          } ${row.item_index}`
        }
        searchPlaceholder="Buscar por título ou item…"
        emptyMessage="Nenhuma oferta cadastrada ainda."
        caption={`${rows.length} oferta${rows.length === 1 ? "" : "s"} · resgates de hoje ainda não têm RPC (issue 31)`}
      />

      <Dialog
        open={draft !== null}
        onClose={() => setDraft(null)}
        title={draft && draft.id > 0 ? "Editar oferta" : "Nova recompensa diária"}
        description="Grátis para o jogador; o resgate é limitado a um por conta por dia UTC."
        footer={
          <>
            <button className="ui-btn ui-btn--ghost" onClick={() => setDraft(null)}>
              cancelar
            </button>
            <button
              className="ui-btn ui-btn--primary"
              disabled={pending || invalid}
              onClick={() => void submit()}
            >
              {pending ? "salvando…" : draft && draft.id > 0 ? "salvar oferta" : "criar oferta"}
            </button>
          </>
        }
      >
        {draft && (
          <div className="ui-form">
            <Field
              label="título"
              error={errors.title}
              hint={`${draft.title.trim().length}/${TITLE_MAX} · mínimo ${TITLE_MIN}`}
            >
              <input
                className="ui-input"
                value={draft.title}
                maxLength={TITLE_MAX}
                onChange={(event) => set("title", event.target.value)}
                placeholder="ex.: Poção de cura diária"
              />
            </Field>

            <Field
              label="descrição"
              error={errors.description}
              hint={`${draft.description.length}/${DESCRIPTION_MAX} · aparece na vitrine do jogador`}
            >
              <textarea
                className="ui-input"
                rows={2}
                value={draft.description}
                maxLength={DESCRIPTION_MAX}
                onChange={(event) => set("description", event.target.value)}
              />
            </Field>

            <Field label="item" error={errors.item_index} hint="o item vai para o carry no próximo login">
              <Combobox
                label="item da recompensa"
                load={loadItemOptions}
                placeholder="Buscar item por nome ou índice…"
                value={picked ? { value: String(picked.index), label: picked.name } : null}
                onSelect={(option) => {
                  const index = Number(option.value);
                  set("item_index", index);
                  setPicked({ index, name: option.label });
                }}
              />
            </Field>

            <Field
              label="validade do item"
              error={errors.expires_days}
              hint="0 = permanente; o cliente mostra os dias no nome do item"
            >
              <input
                className="ui-input"
                type="number"
                min={0}
                max={EXPIRES_MAX}
                value={draft.expires_days}
                onChange={(event) => set("expires_days", Number(event.target.value))}
              />
            </Field>

            <div>
              <span className="ui-field__label">adicionais do item</span>
              <div style={{ display: "grid", gap: "var(--space-2)", marginTop: "var(--space-1)" }}>
                {EFFECT_PAIRS.map(([effectKey, valueKey], index) => (
                  <div
                    key={effectKey}
                    style={{
                      display: "grid",
                      gridTemplateColumns: "1fr 120px",
                      gap: "var(--space-2)",
                      alignItems: "center",
                    }}
                  >
                    <select
                      className="ui-select"
                      aria-label={`efeito ${index + 1}`}
                      value={String(draft[effectKey])}
                      onChange={(event) => set(effectKey, Number(event.target.value))}
                    >
                      <option value="0">nenhum</option>
                      {effectTypes.map((type) => (
                        <option key={type.code} value={type.code}>
                          {type.label || type.name}
                        </option>
                      ))}
                      {draft[effectKey] !== 0 &&
                        !effectTypes.some((type) => type.code === draft[effectKey]) && (
                          <option value={draft[effectKey]}>efeito {draft[effectKey]}</option>
                        )}
                    </select>
                    <input
                      className="ui-input"
                      type="number"
                      min={0}
                      max={EFFECT_MAX}
                      aria-label={`valor do efeito ${index + 1}`}
                      value={draft[valueKey]}
                      onChange={(event) => set(valueKey, Number(event.target.value))}
                    />
                    {(errors[effectKey] || errors[valueKey]) && (
                      <span className="ui-field__error" style={{ gridColumn: "1 / -1" }}>
                        {errors[effectKey] ?? errors[valueKey]}
                      </span>
                    )}
                  </div>
                ))}
              </div>
            </div>

            <label
              style={{
                display: "flex",
                alignItems: "flex-start",
                gap: "var(--space-2)",
                minHeight: 32,
                cursor: "pointer",
              }}
            >
              <input
                type="checkbox"
                checked={draft.enabled}
                onChange={(event) => set("enabled", event.target.checked)}
                style={{ accentColor: "var(--accent)", width: 16, height: 16, marginTop: 8 }}
              />
              <span>
                <span style={{ fontSize: 13 }}>habilitada</span>
                <span className="ui-field__hint" style={{ display: "block" }}>
                  desabilitada, a oferta sai da vitrine e um resgate em curso é recusado
                </span>
              </span>
            </label>
          </div>
        )}
      </Dialog>

      <Dialog
        open={toDelete !== null}
        onClose={() => setToDelete(null)}
        title="Apagar oferta?"
        description="Os resgates já feitos permanecem no histórico."
        footer={
          <>
            <button className="ui-btn ui-btn--ghost" onClick={() => setToDelete(null)}>
              cancelar
            </button>
            <button
              className="ui-btn ui-btn--danger"
              disabled={pending}
              onClick={async () => {
                const target = toDelete;
                setToDelete(null);
                if (target) await run(() => deleteReward({ id: target.id }));
              }}
            >
              apagar
            </button>
          </>
        }
      >
        <p style={{ margin: 0 }}>
          A oferta <strong>{toDelete?.title}</strong> sai da vitrine imediatamente. Se a intenção é só
          tirar do ar, use <strong>desabilitar</strong>.
        </p>
      </Dialog>
    </>
  );
}
