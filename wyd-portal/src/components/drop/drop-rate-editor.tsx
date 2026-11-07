"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { Field } from "@/components/Field";
import { useToast } from "@/components/Toast";
import {
  DROP_EDITOR_NOTE,
  DROP_RATE_MAX,
  DROP_RATE_MIN,
  effectiveDropRate,
  hardcodedRate,
  oddsLabel,
  slotGroup,
  type DropRateSlot,
} from "@/lib/droprate";
import { resetDropRate, saveDropRate } from "@/server/actions/drops";

/**
 * Editor da taxa de drop por slot (issue 34).
 *
 * A linha é identificada pelo **papel** do slot ("Equipamento comum", "Ultra
 * raro", "Sempre cai"); o número aparece como posição secundária, porque é ele
 * que ordena o `Carry[]`. O simulador é a própria coluna de chance: nivel do mob
 * e bônus do matador mudam a conta de todas as linhas ao vivo, antes de salvar.
 *
 * Slots 8–11 são marcados como fixos: `EffectiveDropRate` reescreve o resultado
 * deles depois de qualquer ajuste, então salvar ali seria uma mentira silenciosa.
 */
export function DropRateEditor({ slots }: { slots: readonly DropRateSlot[] }) {
  const router = useRouter();
  const toast = useToast();
  const [drafts, setDrafts] = useState<Record<number, number>>({});
  const [level, setLevel] = useState(50);
  const [bonus, setBonus] = useState(0);
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  const ordered = useMemo(() => [...slots].sort((a, b) => a.slot - b.slot), [slots]);
  const overrides = ordered.filter((slot) => slot.overridden).length;

  /** Tabela do simulador: parte do valor de código e aplica o que está na tela. */
  const table = useMemo(() => {
    const base = new Array<number>(64).fill(0);
    for (const slot of ordered) {
      base[slot.slot] = drafts[slot.slot] ?? slot.current_value;
    }
    return base;
  }, [ordered, drafts]);

  function valueOf(slot: DropRateSlot): number {
    return drafts[slot.slot] ?? slot.current_value;
  }

  function setValue(slot: number, raw: string) {
    const parsed = Number(raw);
    setDrafts((current) => ({ ...current, [slot]: Number.isFinite(parsed) ? parsed : 0 }));
  }

  async function save(slot: DropRateSlot) {
    setPending(slot.slot);
    const result = await saveDropRate({ slot: slot.slot, value: valueOf(slot), reason });
    setPending(null);
    if (!result.ok) {
      setError(result.error);
      toast(result.error, "error");
      return;
    }
    setError(null);
    toast("divisor salvo — o tmserver aplica no próximo restart", "ok");
    router.refresh();
  }

  async function restore(slot: DropRateSlot) {
    setPending(slot.slot);
    const result = await resetDropRate({ slot: slot.slot, reason });
    setPending(null);
    if (!result.ok) {
      setError(result.error);
      toast(result.error, "error");
      return;
    }
    setError(null);
    setDrafts((current) => {
      const next = { ...current };
      delete next[slot.slot];
      return next;
    });
    toast("posição restaurada para o divisor do código", "ok");
    router.refresh();
  }

  return (
    <>
      <section className="ui-card" style={{ marginBottom: "var(--space-3)" }}>
        <strong>Simulador de chance</strong>
        <p className="ui-field__hint" style={{ margin: "var(--space-1) 0 var(--space-2)" }}>
          {DROP_EDITOR_NOTE}
        </p>
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
            gap: "var(--space-2)",
          }}
        >
          <Field label="nível do mob" hint="0–255; o corte muda a cada faixa de 10/170/200/230/255/320">
            <input
              className="ui-input"
              type="number"
              min={0}
              max={255}
              value={level}
              onChange={(event) => setLevel(Math.max(0, Math.min(255, Number(event.target.value) || 0)))}
            />
          </Field>
          <Field label="bônus de drop do matador" hint="0 = sem bônus; soma no divisor base">
            <input
              className="ui-input"
              type="number"
              min={0}
              max={10000}
              value={bonus}
              onChange={(event) => setBonus(Math.max(0, Math.min(10000, Number(event.target.value) || 0)))}
            />
          </Field>
          <Field label="motivo da alteração" hint="vai para a auditoria; opcional">
            <input
              className="ui-input"
              value={reason}
              maxLength={200}
              onChange={(event) => setReason(event.target.value)}
              placeholder="ex.: rebalanceamento do nível 40"
            />
          </Field>
        </div>
      </section>

      <p className="ui-field__hint" style={{ marginBottom: "var(--space-2)" }}>
        {ordered.length} posições · {overrides} fora do valor do código · chance calculada para nível{" "}
        {level}
        {bonus > 0 ? ` e bônus ${bonus}` : ""}
      </p>

      {error && (
        <p className="ui-field__error" role="alert" style={{ marginBottom: "var(--space-2)" }}>
          {error}
        </p>
      )}

      <div className="ui-table-scroll">
        <table className="ui-table">
          <caption className="ui-field__hint" style={{ padding: 6 }}>
            Divisor por posição do <code>Carry[]</code> — o papel do slot é o nome da linha
          </caption>
          <thead>
            <tr>
              <th>Faixa de drop</th>
              <th>Posição</th>
              <th>Divisor</th>
              <th>Chance</th>
              <th>Estado</th>
              <th>Ações</th>
            </tr>
          </thead>
          <tbody>
            {ordered.map((slot) => {
              const group = slotGroup(slot.slot);
              const hard = hardcodedRate(slot.slot);
              const value = valueOf(slot);
              const chance = oddsLabel(effectiveDropRate(slot.slot, bonus, level, table));
              const dirty = value !== slot.current_value;
              const invalid =
                !Number.isInteger(value) || value < DROP_RATE_MIN || value > DROP_RATE_MAX;
              return (
                <tr key={slot.slot}>
                  <td>
                    {group.label}
                    <span className="ui-field__hint" style={{ display: "block" }}>
                      {group.note}
                    </span>
                  </td>
                  <td className="ui-mono" title={`posição ${slot.slot} do Carry[]`}>
                    {group.from === group.to ? "única" : `${group.from}–${group.to}`}
                  </td>
                  <td>
                    <input
                      className="ui-input"
                      type="number"
                      min={DROP_RATE_MIN}
                      max={DROP_RATE_MAX}
                      value={value}
                      disabled={hard !== null || pending === slot.slot}
                      aria-label={`divisor da faixa ${group.label}`}
                      onChange={(event) => setValue(slot.slot, event.target.value)}
                    />
                  </td>
                  <td>{chance}</td>
                  <td>
                    {hard !== null ? (
                      <Badge tone="warn">fixo no código</Badge>
                    ) : slot.overridden ? (
                      <Badge tone="accent">ajustado</Badge>
                    ) : (
                      <Badge tone="neutral">arquivo</Badge>
                    )}
                  </td>
                  <td>
                    <span style={{ display: "inline-flex", gap: "var(--space-1)", flexWrap: "wrap" }}>
                      <button
                        type="button"
                        className="ui-btn ui-btn--primary"
                        disabled={hard !== null || !dirty || invalid || pending === slot.slot}
                        onClick={() => save(slot)}
                      >
                        salvar
                      </button>
                      <button
                        type="button"
                        className="ui-btn ui-btn--ghost"
                        disabled={!slot.overridden || pending === slot.slot}
                        onClick={() => restore(slot)}
                      >
                        restaurar
                      </button>
                    </span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </>
  );
}
