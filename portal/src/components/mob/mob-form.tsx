"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { Dialog } from "@/components/Dialog";
import { Field } from "@/components/Field";
import { useToast } from "@/components/Toast";
import {
  FIELD_GROUPS,
  changedFields,
  toStatValues,
  type MobStat,
  type MobStatValues,
  type NumericField,
} from "@/lib/mob";
import { EquipEditor } from "@/components/mob/equip-editor";
import { revertMobStat, saveMobStat } from "@/server/actions/mobs";

/**
 * Editor de stats de um template (issue 25).
 *
 * Três decisões que a tela precisa deixar explícitas:
 *
 * - o formulário abre com **os valores do arquivo** quando ainda não há override
 *   (read-through do servidor), e o badge diz qual dos dois casos é;
 * - o diff mostrado é o **da sessão** (o que mudou desde o carregamento) — o
 *   valor original do arquivo só é conhecido justamente quando não há override;
 * - o `Equip[]` tem ação própria (`saveMobEquip`) e o BFF relê o equip do
 *   servidor ao salvar stats, então editar equip e depois salvar stats (ou o
 *   contrário) não apaga uma alteração com cópia velha da outra.
 */
export function MobForm({
  stat,
  itemNames,
}: {
  stat: MobStat;
  itemNames: Record<string, string>;
}) {
  const router = useRouter();
  const toast = useToast();
  const initial = useMemo<MobStatValues>(() => toStatValues(stat), [stat]);
  const [values, setValues] = useState<MobStatValues>(initial);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [confirmRevert, setConfirmRevert] = useState(false);

  const changed = changedFields(initial, values);

  function setField(key: NumericField, raw: string) {
    const parsed = raw === "" ? 0 : Number(raw);
    setValues((prev) => ({ ...prev, [key]: Number.isFinite(parsed) ? parsed : 0 }));
  }

  async function save() {
    setPending(true);
    const result = await saveMobStat({ stat: values });
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    setError(null);
    toast("Stats salvos — o tmserver aplica no próximo restart", "ok");
    router.refresh();
  }

  async function revert() {
    setPending(true);
    const result = await revertMobStat(stat.template_name);
    setPending(false);
    setConfirmRevert(false);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    toast("Override removido — o template voltou ao arquivo", "ok");
    router.refresh();
  }

  return (
    <>
      <div className="ui-card" style={{ marginBottom: "var(--space-3)" }}>
        <div style={{ display: "flex", gap: "var(--space-2)", alignItems: "center", flexWrap: "wrap" }}>
          <Badge tone={stat.overridden ? "warn" : "neutral"}>
            {stat.overridden ? "override no banco" : "valores do arquivo"}
          </Badge>
          <span className="ui-field__hint" style={{ margin: 0 }}>
            {stat.overridden
              ? "este template já tem override; o arquivo em Release/ não foi tocado"
              : "read-through: nada salvo ainda para este template"}
          </span>
        </div>
      </div>

      <Field label="nome no jogo" hint="vazio = mantém o nome do arquivo">
        <input
          className="ui-input"
          value={values.display_name}
          maxLength={63}
          onChange={(event) => setValues({ ...values, display_name: event.target.value })}
        />
      </Field>

      {FIELD_GROUPS.map((group) => (
        <section key={group.id} className="ui-card" style={{ marginTop: "var(--space-3)" }}>
          <strong>{group.title}</strong>
          {group.note && (
            <p className="ui-field__hint" style={{ margin: "var(--space-1) 0 0" }}>
              {group.note}
            </p>
          )}
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
              gap: "var(--space-2)",
              marginTop: "var(--space-2)",
            }}
          >
            {group.fields.map((field) => {
              const dirty = changed.includes(field.key);
              return (
                <Field
                  key={field.key}
                  label={dirty ? `${field.label} · alterado` : field.label}
                  hint={field.hint}
                >
                  <input
                    className="ui-input"
                    type="number"
                    step={1}
                    value={values[field.key] as number}
                    style={dirty ? { borderColor: "var(--warn)" } : undefined}
                    onChange={(event) => setField(field.key, event.target.value)}
                  />
                </Field>
              );
            })}
          </div>
        </section>
      ))}

      <EquipEditor
        templateName={stat.template_name}
        initial={stat.equip ?? []}
        itemNames={itemNames}
        overridden={stat.overridden}
      />

      {error && (
        <p className="ui-field__error" role="alert" style={{ marginTop: "var(--space-2)" }}>
          {error}
        </p>
      )}

      <div className="ui-form__actions" style={{ marginTop: "var(--space-3)" }}>
        <button className="ui-btn ui-btn--primary" onClick={save} disabled={pending}>
          salvar
        </button>
        {stat.overridden && (
          <button className="ui-btn ui-btn--ghost" onClick={() => setConfirmRevert(true)} disabled={pending}>
            reverter ao arquivo
          </button>
        )}
        <span className="ui-field__hint" style={{ margin: 0 }}>
          {changed.length === 0
            ? "nenhum campo alterado nesta sessão"
            : `${changed.length} campo${changed.length === 1 ? "" : "s"} alterado${changed.length === 1 ? "" : "s"} nesta sessão`}
        </span>
      </div>

      <Dialog
        open={confirmRevert}
        title="Reverter ao arquivo?"
        onClose={() => setConfirmRevert(false)}
        footer={
          <div style={{ display: "flex", gap: "var(--space-2)" }}>
            <button className="ui-btn ui-btn--ghost" onClick={() => setConfirmRevert(false)}>
              cancelar
            </button>
            <button className="ui-btn ui-btn--danger" onClick={revert} disabled={pending}>
              remover override
            </button>
          </div>
        }
      >
        <p style={{ margin: 0 }}>
          O override de <span className="ui-mono">{stat.template_name}</span> sai do banco e o
          template volta a valer pelo arquivo em <code>Release/TMsrv/run/npc/</code>. O arquivo nunca
          é alterado.
        </p>
      </Dialog>
    </>
  );
}
