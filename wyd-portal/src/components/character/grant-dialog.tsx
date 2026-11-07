"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { Dialog } from "@/components/Dialog";
import { Field } from "@/components/Field";
import { useToast } from "@/components/Toast";
import { grantCharacterItem, type GrantInput } from "@/server/actions/characters";

/**
 * Concessão de item para um personagem (issue 43).
 *
 * Deixa explícito que o item **entra na fila**: o tmserver drena
 * `delivery_queue` no login, então nada é escrito no save agora. Os pares de
 * efeito são opcionais — `adicionar efeito` preenche `eff`/`effv` em sequência.
 */
export interface EffectOption {
  code: number;
  label: string;
}

export function GrantDialog({
  characterId,
  characterName,
  effectTypes,
}: {
  characterId: number;
  characterName: string;
  /** Vocabulário de ItemEffect.h vindo do catálogo (issue 44). */
  effectTypes: readonly EffectOption[];
}) {
  const router = useRouter();
  const toast = useToast();
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState<GrantInput>({
    character_id: characterId,
    item_index: 0,
    quantity: 1,
    eff1: 0,
    effv1: 0,
    eff2: 0,
    effv2: 0,
    eff3: 0,
    effv3: 0,
    reason: "",
  });

  function patch(values: Partial<GrantInput>) {
    setForm((current) => ({ ...current, ...values }));
  }

  const slots = [1, 2, 3] as const;
  const effectAt = (slot: 1 | 2 | 3) => (slot === 1 ? form.eff1 : slot === 2 ? form.eff2 : form.eff3);
  const valueAt = (slot: 1 | 2 | 3) => (slot === 1 ? form.effv1 : slot === 2 ? form.effv2 : form.effv3);

  function setEffect(slot: 1 | 2 | 3, eff: number) {
    patch(slot === 1 ? { eff1: eff } : slot === 2 ? { eff2: eff } : { eff3: eff });
  }

  function setEffectValue(slot: 1 | 2 | 3, value: number) {
    patch(slot === 1 ? { effv1: value } : slot === 2 ? { effv2: value } : { effv3: value });
  }

  async function submit() {
    setPending(true);
    const result = await grantCharacterItem({ ...form, character_id: characterId });
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    toast(
      `${result.deliveryIds.length} unidade(s) na fila — ${characterName} recebe no próximo dreno`,
      "ok",
    );
    setOpen(false);
    setError(null);
    router.refresh();
  }

  return (
    <>
      <button className="ui-btn ui-btn--primary" onClick={() => setOpen(true)}>
        conceder item
      </button>

      <Dialog
        open={open}
        title={`Conceder item — ${characterName}`}
        onClose={() => setOpen(false)}
        footer={
          <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
            <button className="ui-btn ui-btn--ghost" onClick={() => setOpen(false)} disabled={pending}>
              cancelar
            </button>
            <button className="ui-btn ui-btn--primary" onClick={submit} disabled={pending}>
              enfileirar
            </button>
          </div>
        }
      >
        <div style={{ display: "grid", gap: 10 }}>
          <p className="ui-field__hint" style={{ margin: 0 }}>
            Vai para <code>delivery_queue</code> e o personagem recebe no próximo dreno do tmserver
            (login). Nada é escrito no save agora.
          </p>

          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
            <Field label="item (índice)" hint="veja o nome em /itens">
              <input
                className="ui-input"
                type="number"
                min={1}
                value={form.item_index}
                onChange={(e) => patch({ item_index: Number(e.target.value) })}
              />
            </Field>
            <Field label="quantidade" hint="1–255 (uma linha por unidade)">
              <input
                className="ui-input"
                type="number"
                min={1}
                max={255}
                value={form.quantity}
                onChange={(e) => patch({ quantity: Number(e.target.value) })}
              />
            </Field>
          </div>

          <div style={{ display: "grid", gap: 8 }}>
            <p className="ui-field__hint" style={{ margin: 0 }}>
              Adds do item: um par por linha. O efeito vem do vocabulário de <code>ItemEffect.h</code> e é
              validado de novo no web-api — código fora da tabela é recusado.
            </p>
            {slots.map((slot) => (
              <div key={slot} style={{ display: "grid", gridTemplateColumns: "1fr 120px", gap: 8 }}>
                <Field label={`efeito ${slot}`}>
                  <select
                    className="ui-input"
                    value={effectAt(slot)}
                    onChange={(e) => setEffect(slot, Number(e.target.value))}
                  >
                    <option value={0}>nenhum</option>
                    {effectTypes.map((type) => (
                      <option key={type.code} value={type.code}>
                        {type.label} ({type.code})
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label={`valor ${slot}`}>
                  <input
                    className="ui-input"
                    type="number"
                    min={0}
                    max={255}
                    value={valueAt(slot)}
                    onChange={(e) => setEffectValue(slot, Number(e.target.value))}
                  />
                </Field>
              </div>
            ))}
          </div>

          <Field label="motivo" hint="obrigatório — vai para a auditoria">
            <input
              className="ui-input"
              maxLength={200}
              value={form.reason}
              onChange={(e) => patch({ reason: e.target.value })}
            />
          </Field>

          {error && <p style={{ color: "var(--danger)", margin: 0 }}>{error}</p>}
        </div>
      </Dialog>
    </>
  );
}
