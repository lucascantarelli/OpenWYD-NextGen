"use client";

import { useCallback, useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { Dialog } from "@/components/Dialog";
import { Combobox, type ComboOption } from "@/components/Combobox";
import { Field } from "@/components/Field";
import { useToast } from "@/components/Toast";
import { SHOP_TABS, emptyShopItem, shopDiff, sortShop, type NpcShopItem } from "@/lib/npc";
import { saveNpcShop } from "@/server/actions/npcs";

/**
 * Loja do mercador: 27 slots em 3 abas de 9 (`§5.2`).
 *
 * `SetNpcShop` **substitui** a loja inteira — o editor sempre envia o array
 * completo e mostra o diff antes de salvar, para o moderador ver o que sai.
 */
export function ShopEditor({
  npcId,
  initial,
  itemNames,
  onSaved,
}: {
  npcId: number;
  initial: readonly NpcShopItem[];
  itemNames: Record<string, string>;
  onSaved?: () => void;
}) {
  const toast = useToast();
  const [items, setItems] = useState<NpcShopItem[]>(() => sortShop(initial));
  const [names, setNames] = useState<Record<string, string>>(itemNames);
  const [tab, setTab] = useState(0);
  const [pickerSlot, setPickerSlot] = useState<number | null>(null);
  const [picked, setPicked] = useState<{ index: number; label: string } | null>(null);
  const [quantity, setQuantity] = useState(1);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const diff = useMemo(() => shopDiff(initial, items), [initial, items]);
  const dirty = diff.added + diff.removed + diff.changed > 0;

  const bySlot = useMemo(() => new Map(items.map((item) => [item.slot, item])), [items]);

  /** Busca no catálogo já carregado pelo BFF (`/api/search/items`). */
  const loadItems = useCallback(async (query: string): Promise<ComboOption[]> => {
    const response = await fetch(`/api/search/items?q=${encodeURIComponent(query)}`);
    if (!response.ok) throw new Error("falha na busca");
    const body = (await response.json()) as { options?: ComboOption[] };
    return body.options ?? [];
  }, []);

  function openPicker(slot: number) {
    setPickerSlot(slot);
    setPicked(null);
    setQuantity(1);
    setError(null);
  }

  function addItem() {
    if (pickerSlot === null || !picked) return;
    setItems((current) =>
      sortShop([
        ...current.filter((item) => item.slot !== pickerSlot),
        { ...emptyShopItem(pickerSlot), item_index: picked.index, quantity },
      ]),
    );
    setNames((current) => ({ ...current, [String(picked.index)]: picked.label }));
    setPickerSlot(null);
    setPicked(null);
  }

  function removeSlot(slot: number) {
    setItems((current) => current.filter((item) => item.slot !== slot));
  }

  async function save() {
    setPending(true);
    setError(null);
    const result = await saveNpcShop({ npcId, items });
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      toast(result.error, "error");
      return;
    }
    toast("Loja salva — aplica no próximo restart do tmserver", "ok");
    onSaved?.();
  }

  return (
    <>
      <div className="ui-filters" role="tablist" aria-label="Abas da loja">
        {SHOP_TABS.map((shopTab, index) => (
          <button
            key={shopTab.label}
            type="button"
            className="ui-chip"
            role="tab"
            aria-pressed={tab === index}
            onClick={() => setTab(index)}
          >
            {shopTab.label} ({shopTab.from}–{shopTab.to})
          </button>
        ))}
        <span className="ui-field__hint" style={{ marginLeft: "auto" }}>
          {items.length}/27 slots ·{" "}
          {dirty ? (
            <Badge tone="warn">
              {diff.added} novo(s), {diff.removed} removido(s), {diff.changed} alterado(s)
            </Badge>
          ) : (
            <Badge tone="ok">sem alterações</Badge>
          )}
        </span>
      </div>

      <div className="ui-shop-grid">
        {Array.from({ length: SHOP_TABS[tab].to - SHOP_TABS[tab].from + 1 }, (_, offset) => {
          const slot = SHOP_TABS[tab].from + offset;
          const item = bySlot.get(slot);
          return (
            <div key={slot} className="ui-shop-slot" data-filled={item ? "true" : "false"}>
              <span className="ui-field__hint">slot {slot}</span>
              {item ? (
                <>
                  <span title={`índice ${item.item_index}`}>
                    {names[String(item.item_index)] ?? `item #${item.item_index}`}
                  </span>
                  <span className="ui-field__hint">quantidade {item.quantity}</span>
                  <span style={{ display: "inline-flex", gap: 6 }}>
                    <button type="button" className="ui-btn ui-btn--ghost" onClick={() => openPicker(slot)}>
                      trocar
                    </button>
                    <button type="button" className="ui-btn ui-btn--ghost" onClick={() => removeSlot(slot)}>
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

      <div className="ui-form__actions" style={{ marginTop: 12 }}>
        <button type="button" className="ui-btn ui-btn--primary" disabled={pending} onClick={save}>
          {pending ? "Salvando…" : "Salvar loja"}
        </button>
        <button
          type="button"
          className="ui-btn"
          disabled={pending || items.length === 0}
          onClick={() => {
            setItems([]);
            toast("Loja esvaziada na tela — salve para aplicar", "info");
          }}
        >
          esvaziar loja
        </button>
        <button
          type="button"
          className="ui-btn ui-btn--ghost"
          disabled={pending || !dirty}
          onClick={() => setItems(sortShop(initial))}
        >
          descartar alterações
        </button>
      </div>

      {error && (
        <p className="ui-field__error" role="alert">
          {error}
        </p>
      )}

      <Dialog
        open={pickerSlot !== null}
        onClose={() => setPickerSlot(null)}
        title={`Item do slot ${pickerSlot ?? ""}`}
        description="O item vem do catálogo do ItemList.csv; o preço é global (issue 24), não por NPC."
        footer={
          <>
            <button type="button" className="ui-btn" onClick={() => setPickerSlot(null)}>
              cancelar
            </button>
            <button type="button" className="ui-btn ui-btn--primary" disabled={!picked} onClick={addItem}>
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
        <Field label="Quantidade" hint="1–255 (vira EF_AMOUNT no item entregue)" htmlFor="quantity">
          <input
            id="quantity"
            className="ui-input"
            type="number"
            min={1}
            max={255}
            value={quantity}
            onChange={(event) => setQuantity(Math.min(255, Math.max(1, Number(event.target.value))))}
          />
        </Field>
        {picked && (
          <p className="ui-field__hint" title={`índice ${picked.index}`}>
            escolhido: {picked.label}
          </p>
        )}
      </Dialog>
    </>
  );
}
