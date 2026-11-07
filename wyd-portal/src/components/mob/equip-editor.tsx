"use client";

import { useRouter } from "next/navigation";
import { useCallback, useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { Combobox, type ComboOption } from "@/components/Combobox";
import { Dialog } from "@/components/Dialog";
import { useToast } from "@/components/Toast";
import {
  MOB_EQUIP_SLOTS,
  emptyEquipSlot,
  equipDiff,
  sortEquip,
  type MobEquipSlot,
} from "@/lib/mob";
import { saveMobEquip } from "@/server/actions/mobs";

/**
 * `Equip[16]` do template (issue 25).
 *
 * Decisões que a tela deixa explícitas:
 *
 * - `SetMobTemplateEquip` **substitui** a lista inteira e exige que o template já
 *   tenha override de stats; sem override o botão fica bloqueado e o texto diz
 *   o que fazer (salvar os stats primeiro acima).
 * - trocar o item de um slot **zera os efeitos** daquele slot: `eff1/effv1…`
 *   descrevem o item anterior, então herdá-los seria dado inventado.
 * - os efeitos são exibidos, mas não editados aqui — o editor de adds vive na
 *   issue 44.
 */
export function EquipEditor({
  templateName,
  initial,
  itemNames,
  overridden,
}: {
  templateName: string;
  initial: readonly MobEquipSlot[];
  /** índices de item já resolvidos para nome pelo BFF. */
  itemNames: Record<string, string>;
  overridden: boolean;
}) {
  const router = useRouter();
  const toast = useToast();
  const [base, setBase] = useState<MobEquipSlot[]>(() => sortEquip(initial));
  const [items, setItems] = useState<MobEquipSlot[]>(() => sortEquip(initial));
  const [names, setNames] = useState<Record<string, string>>(itemNames);
  const [pickerSlot, setPickerSlot] = useState<number | null>(null);
  const [picked, setPicked] = useState<{ index: number; label: string } | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const diff = useMemo(() => equipDiff(base, items), [base, items]);
  const dirty = diff.added + diff.removed + diff.changed > 0;
  const bySlot = useMemo(() => new Map(items.map((item) => [item.slot, item])), [items]);

  const loadItems = useCallback(async (query: string): Promise<ComboOption[]> => {
    const response = await fetch(`/api/search/items?q=${encodeURIComponent(query)}`);
    if (!response.ok) throw new Error("falha na busca");
    const body = (await response.json()) as { options?: ComboOption[] };
    return body.options ?? [];
  }, []);

  function openPicker(slot: number) {
    setPickerSlot(slot);
    setPicked(null);
    setError(null);
  }

  function usePicked() {
    if (pickerSlot === null || !picked) return;
    setItems((current) =>
      sortEquip([
        ...current.filter((item) => item.slot !== pickerSlot),
        { ...emptyEquipSlot(pickerSlot), item_index: picked.index },
      ]),
    );
    setNames((current) => ({ ...current, [String(picked.index)]: picked.label }));
    setPickerSlot(null);
    setPicked(null);
  }

  function clearSlot(slot: number) {
    setItems((current) => current.filter((item) => item.slot !== slot));
  }

  async function save() {
    setPending(true);
    const result = await saveMobEquip({ templateName, items });
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      toast(result.error, "error");
      return;
    }
    setError(null);
    setBase(items);
    toast("Equip salvo — o tmserver aplica no próximo restart", "ok");
    router.refresh();
  }

  return (
    <section className="ui-card" style={{ marginTop: "var(--space-3)" }}>
      <div
        style={{
          display: "flex",
          gap: "var(--space-2)",
          alignItems: "center",
          flexWrap: "wrap",
          marginBottom: "var(--space-2)",
        }}
      >
        <strong>Equip</strong>
        <Badge tone={dirty ? "warn" : "neutral"}>
          {dirty
            ? `${diff.added} novo(s), ${diff.removed} removido(s), ${diff.changed} trocado(s)`
            : `${items.length}/${MOB_EQUIP_SLOTS} slots`}
        </Badge>
        {!overridden && (
          <span className="ui-field__hint" style={{ margin: 0 }}>
            salve os stats antes de editar o equip: o override precisa existir
          </span>
        )}
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
          gap: "var(--space-2)",
        }}
      >
        {Array.from({ length: MOB_EQUIP_SLOTS }, (_, slot) => {
          const item = bySlot.get(slot);
          const name = item ? (names[String(item.item_index)] ?? `item #${item.item_index}`) : "";
          const effects = item
            ? [
                [item.eff1, item.effv1],
                [item.eff2, item.effv2],
                [item.eff3, item.effv3],
              ]
                .filter(([eff]) => eff !== 0)
                .map(([eff, value]) => `${eff}:${value}`)
                .join(" ")
            : "";
          return (
            <div key={slot} className="ui-shop-slot" data-filled={item ? "true" : "false"}>
              <span className="ui-field__hint">slot {slot}</span>
              {item ? (
                <>
                  <span title={`índice ${item.item_index}`}>{name}</span>
                  <span className="ui-field__hint">{effects ? `efeitos ${effects}` : "sem efeitos"}</span>
                  <span style={{ display: "inline-flex", gap: 6 }}>
                    <button
                      type="button"
                      className="ui-btn ui-btn--ghost"
                      onClick={() => openPicker(slot)}
                    >
                      trocar
                    </button>
                    <button
                      type="button"
                      className="ui-btn ui-btn--ghost"
                      onClick={() => clearSlot(slot)}
                    >
                      limpar
                    </button>
                  </span>
                </>
              ) : (
                <button type="button" className="ui-btn ui-btn--ghost" onClick={() => openPicker(slot)}>
                  adicionar item
                </button>
              )}
            </div>
          );
        })}
      </div>

      {error && (
        <p className="ui-field__error" role="alert" style={{ marginTop: "var(--space-2)" }}>
          {error}
        </p>
      )}

      <div className="ui-form__actions" style={{ marginTop: "var(--space-2)" }}>
        <button
          type="button"
          className="ui-btn ui-btn--primary"
          disabled={pending || !overridden || !dirty}
          onClick={save}
        >
          {pending ? "salvando…" : "salvar equip"}
        </button>
        <button
          type="button"
          className="ui-btn ui-btn--ghost"
          disabled={pending || !dirty}
          onClick={() => setItems(sortEquip(base))}
        >
          descartar alterações
        </button>
        <span className="ui-field__hint" style={{ margin: 0 }}>
          trocar o item zera os efeitos do slot; efeitos são editados na issue 44
        </span>
      </div>

      <Dialog
        open={pickerSlot !== null}
        title={`Item do slot ${pickerSlot ?? ""}`}
        onClose={() => setPickerSlot(null)}
        description="O item vem do ItemList.csv. O slot é a chave do Equip[16]."
        footer={
          <>
            <button type="button" className="ui-btn" onClick={() => setPickerSlot(null)}>
              cancelar
            </button>
            <button type="button" className="ui-btn ui-btn--primary" disabled={!picked} onClick={usePicked}>
              usar item
            </button>
          </>
        }
      >
        <Combobox
          label="Buscar item do catálogo"
          placeholder="nome ou índice…"
          load={loadItems}
          onSelect={(option) => setPicked({ index: Number(option.value), label: option.label })}
        />
        {picked && (
          <p className="ui-field__hint" title={`índice ${picked.index}`}>
            escolhido: {picked.label}
          </p>
        )}
      </Dialog>
    </section>
  );
}
