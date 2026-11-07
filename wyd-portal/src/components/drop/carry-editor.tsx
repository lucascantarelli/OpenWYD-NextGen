"use client";

import { useRouter } from "next/navigation";
import { useCallback, useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { Combobox, type ComboOption } from "@/components/Combobox";
import { Dialog } from "@/components/Dialog";
import { useToast } from "@/components/Toast";
import {
  CARRY_EDITOR_NOTE,
  CARRY_SLOTS,
  effectLabel,
  groupCarry,
  type MobCarry,
  type MobCarrySlot,
} from "@/lib/mobcarry";
import { resetMobCarry, saveMobCarry } from "@/server/actions/drops";

interface Picked {
  item_index: number;
  item_name: string;
}

function toRecord(slots: readonly MobCarrySlot[]): Record<number, Picked> {
  const out: Record<number, Picked> = {};
  for (const slot of slots) {
    if (slot.item_index > 0) out[slot.slot] = { item_index: slot.item_index, item_name: slot.item_name };
  }
  return out;
}

/** Reconstrói o formato do proto para o agrupamento (efeitos vêm do BFF na leitura). */
function fromRecord(items: Record<number, Picked>, source: readonly MobCarrySlot[]): MobCarrySlot[] {
  const effects = new Map(source.map((slot) => [slot.slot, slot]));
  return Object.entries(items).map(([slot, picked]) => {
    const prev = effects.get(Number(slot));
    return {
      slot: Number(slot),
      item_index: picked.item_index,
      item_name: picked.item_name,
      eff1: prev?.eff1 ?? 0,
      effv1: prev?.effv1 ?? 0,
      eff2: prev?.eff2 ?? 0,
      effv2: prev?.effv2 ?? 0,
      eff3: prev?.eff3 ?? 0,
      effv3: prev?.effv3 ?? 0,
      from_override: prev?.from_override ?? false,
    };
  });
}

function sameTable(a: Record<number, Picked>, b: Record<number, Picked>): boolean {
  const keys = new Set([...Object.keys(a), ...Object.keys(b)]);
  for (const key of keys) {
    if ((a[Number(key)]?.item_index ?? 0) !== (b[Number(key)]?.item_index ?? 0)) return false;
  }
  return true;
}

/**
 * `Carry[64]` de um template (issue 34, fatia 3).
 *
 * A posição é o que define a raridade: a linha se apresenta pela faixa
 * ("Sempre cai", "Ultra raro"…) e o número da posição fica como detalhe. Só
 * posições ocupadas viram linha; cada faixa mostra quantas livres restam e
 * permite ocupar a próxima — evita uma tabela de 64 campos vazios.
 *
 * Os efeitos do `STRUCT_ITEM` aparecem para leitura e **não** são editados: quem
 * decide se eles sobrevivem é o web-api (mesmo item preserva, item novo zera).
 */
export function CarryEditor({ carry }: { carry: MobCarry }) {
  const router = useRouter();
  const toast = useToast();
  const [base, setBase] = useState<Record<number, Picked>>(() => toRecord(carry.slots));
  const [items, setItems] = useState<Record<number, Picked>>(() => toRecord(carry.slots));
  const [reason, setReason] = useState("");
  const [picker, setPicker] = useState<number | null>(null);
  const [picked, setPicked] = useState<Picked | null>(null);
  const [pending, setPending] = useState(false);
  const [confirmReset, setConfirmReset] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const groups = useMemo(() => groupCarry(fromRecord(items, carry.slots)), [items, carry.slots]);
  const dirty = useMemo(() => !sameTable(base, items), [base, items]);
  const total = Object.keys(items).length;

  const loadItems = useCallback(async (query: string): Promise<ComboOption[]> => {
    const response = await fetch(`/api/search/items?q=${encodeURIComponent(query)}`);
    if (!response.ok) throw new Error("falha na busca");
    const body = (await response.json()) as { options?: ComboOption[] };
    return body.options ?? [];
  }, []);

  function openPicker(slot: number) {
    setPicker(slot);
    setPicked(null);
    setError(null);
  }

  function usePicked() {
    if (picker === null || !picked) return;
    setItems((current) => ({ ...current, [picker]: picked }));
    setPicker(null);
    setPicked(null);
  }

  function clearSlot(slot: number) {
    setItems((current) => {
      const next = { ...current };
      delete next[slot];
      return next;
    });
  }

  async function save() {
    const slots = Object.entries(items)
      .map(([slot, pickedItem]) => ({ slot: Number(slot), item_index: pickedItem.item_index }))
      .sort((a, b) => a.slot - b.slot);
    setPending(true);
    const result = await saveMobCarry({ templateName: carry.template_name, slots, reason });
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      toast(result.error, "error");
      return;
    }
    setError(null);
    setBase(items);
    toast("Carry salvo — o tmserver aplica no próximo boot", "ok");
    router.refresh();
  }

  async function reset() {
    setPending(true);
    const result = await resetMobCarry({ templateName: carry.template_name, reason });
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      toast(result.error, "error");
      return;
    }
    setConfirmReset(false);
    setError(null);
    toast("Carry restaurado do arquivo", "ok");
    router.refresh();
  }

  return (
    <section className="ui-card" style={{ marginTop: "var(--space-3)" }}>
      <div
        style={{
          display: "flex",
          gap: "var(--space-2)",
          alignItems: "baseline",
          flexWrap: "wrap",
          marginBottom: "var(--space-2)",
        }}
      >
        <strong>{carry.display_name || carry.template_name}</strong>
        <span className="ui-field__hint" style={{ margin: 0 }}>
          template <code>{carry.template_name}</code> · nível {carry.level || "—"}
        </span>
        <Badge tone={dirty ? "warn" : carry.overridden ? "accent" : "neutral"}>
          {dirty
            ? "alterações não salvas"
            : carry.overridden
              ? "tabela do banco (override)"
              : "tabela do arquivo"}
        </Badge>
        <Badge tone="neutral">
          {total}/{CARRY_SLOTS} posições ocupadas
        </Badge>
      </div>

      <label className="ui-field" style={{ maxWidth: 420 }}>
        <span>Motivo da alteração</span>
        <input
          type="text"
          value={reason}
          onChange={(event) => setReason(event.target.value)}
          placeholder="ex.: adicionar poção de HP ao drop do chefe"
          maxLength={200}
        />
        <span className="ui-field__hint">vai para a auditoria junto com a tabela antes/depois</span>
      </label>

      <div style={{ display: "grid", gap: "var(--space-2)", marginTop: "var(--space-2)" }}>
        {groups.map((group) => (
          <div key={group.label} className="ui-card" style={{ padding: "var(--space-2)" }}>
            <div
              style={{
                display: "flex",
                gap: "var(--space-2)",
                alignItems: "baseline",
                flexWrap: "wrap",
              }}
            >
              <strong>{group.label}</strong>
              <span className="ui-field__hint" style={{ margin: 0 }}>
                posições {group.from}–{group.to} · {group.occupied.length} ocupada(s) ·{" "}
                {group.free.length} livre(s)
              </span>
            </div>
            <p className="ui-field__hint" style={{ margin: "2px 0 var(--space-2)" }}>
              {group.note}
            </p>

            {group.occupied.length === 0 && (
              <p className="ui-field__hint" style={{ margin: "0 0 var(--space-2)" }}>
                nenhum item nesta faixa — nada cai com essa raridade
              </p>
            )}

            <div style={{ display: "grid", gap: 6 }}>
              {group.occupied.map((slot) => (
                <div
                  key={slot.slot}
                  className="ui-shop-slot"
                  data-filled="true"
                  style={{ alignItems: "center" }}
                >
                  <span title={`posição ${slot.slot} do Carry[]`}>{slot.item_name}</span>
                  <span className="ui-field__hint">
                    posição {slot.slot}
                    {effectLabel(slot) ? ` · efeitos ${effectLabel(slot)}` : ""}
                  </span>
                  <span style={{ display: "inline-flex", gap: 6 }}>
                    <button
                      type="button"
                      className="ui-btn ui-btn--ghost"
                      onClick={() => openPicker(slot.slot)}
                    >
                      trocar
                    </button>
                    <button
                      type="button"
                      className="ui-btn ui-btn--ghost"
                      onClick={() => clearSlot(slot.slot)}
                    >
                      limpar
                    </button>
                  </span>
                </div>
              ))}
            </div>

            {group.free.length > 0 && (
              <button
                type="button"
                className="ui-btn ui-btn--ghost"
                style={{ marginTop: "var(--space-2)" }}
                onClick={() => openPicker(group.free[0])}
              >
                adicionar em {group.label.toLowerCase()} (posição {group.free[0]})
              </button>
            )}
          </div>
        ))}
      </div>

      {error && (
        <p className="ui-field__error" role="alert" style={{ marginTop: "var(--space-2)" }}>
          {error}
        </p>
      )}

      <p className="ui-field__hint" style={{ marginTop: "var(--space-2)" }}>
        {CARRY_EDITOR_NOTE}
      </p>

      <div className="ui-form__actions">
        <button
          type="button"
          className="ui-btn ui-btn--primary"
          disabled={pending || !dirty}
          onClick={save}
        >
          {pending ? "salvando…" : "salvar tabela de drop"}
        </button>
        <button
          type="button"
          className="ui-btn ui-btn--ghost"
          disabled={pending || !dirty}
          onClick={() => setItems(base)}
        >
          descartar alterações
        </button>
        <button
          type="button"
          className="ui-btn ui-btn--ghost"
          disabled={pending || !carry.overridden}
          onClick={() => setConfirmReset(true)}
        >
          restaurar do arquivo
        </button>
      </div>

      <Dialog
        open={picker !== null}
        title={picker !== null ? `Item da posição ${picker}` : "Item do Carry"}
        onClose={() => setPicker(null)}
        description="A posição define a raridade pelo divisor daquele slot; o item é o que cai."
        footer={
          <>
            <button type="button" className="ui-btn" onClick={() => setPicker(null)}>
              cancelar
            </button>
            <button
              type="button"
              className="ui-btn ui-btn--primary"
              disabled={!picked}
              onClick={usePicked}
            >
              usar item
            </button>
          </>
        }
      >
        <Combobox
          label="Buscar item do catálogo"
          placeholder="nome ou índice…"
          load={loadItems}
          onSelect={(option) =>
            setPicked({ item_index: Number(option.value), item_name: option.label })
          }
        />
        {picked && (
          <p className="ui-field__hint">
            escolhido: {picked.item_name} · trocar o item zera os efeitos do slot
          </p>
        )}
      </Dialog>

      <Dialog
        open={confirmReset}
        title="Restaurar o Carry do arquivo"
        onClose={() => setConfirmReset(false)}
        description="Remove o override deste template; o drop volta a ser exatamente o do arquivo em tmserver/content/TMsrv/run/npc."
        footer={
          <>
            <button type="button" className="ui-btn" onClick={() => setConfirmReset(false)}>
              cancelar
            </button>
            <button
              type="button"
              className="ui-btn ui-btn--primary"
              disabled={pending}
              onClick={reset}
            >
              restaurar
            </button>
          </>
        }
      >
        <p>
          O tmserver precisa de restart para voltar a usar a tabela do arquivo. A alteração fica na
          auditoria com o motivo informado.
        </p>
      </Dialog>
    </section>
  );
}
