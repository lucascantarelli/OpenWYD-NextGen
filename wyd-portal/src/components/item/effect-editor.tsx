"use client";

/**
 * Editor dos efeitos base de um item (issue 44, metade de escrita).
 *
 * A chave é o código do efeito: a quantidade de pares `EF_*` varia por item (a
 * 1188 tem seis) e o servidor os indexa por código. Por isso não existe "trocar o
 * código de uma linha" — remover e adicionar é o caminho, e cada operação
 * aparece separada na auditoria.
 *
 * O seletor lista o vocabulário `ItemEffect.h` que o web-api manda junto com os
 * efeitos, nunca uma cópia local: um código fora do header é recusado lá, e
 * oferecê-lo aqui só produziria erro depois do clique.
 */
import { useRouter } from "next/navigation";
import { useEffect, useState, useTransition } from "react";

import { deleteEffect, upsertEffect } from "@/server/actions/itemeffects";
import type { ItemEffectEntry, ItemEffectType } from "@/server/admin-client";

type Props = {
  itemIndex: number;
  effects: ItemEffectEntry[];
  effectTypes: ItemEffectType[];
  /** Explicação do web-api quando a lista veio vazia (item fora do catálogo). */
  reason: string;
};

export function ItemEffectEditor({ itemIndex, effects, effectTypes, reason }: Props) {
  const router = useRouter();
  const [newEffect, setNewEffect] = useState<number | "">("");
  const [newValue, setNewValue] = useState("0");
  const [feedback, setFeedback] = useState<{ ok: boolean; text: string } | null>(null);
  const [pending, startTransition] = useTransition();

  // Só oferece códigos que o item ainda não tem: reenviar um código existente é
  // a operação de editar a linha dele, não a de adicionar.
  const available = effectTypes.filter((t) => !effects.some((e) => e.effect === t.code));

  function add() {
    const parsed = Number.parseInt(newValue, 10);
    if (newEffect === "" || !Number.isInteger(parsed)) {
      setFeedback({ ok: false, text: "escolha o efeito e um valor inteiro" });
      return;
    }
    startTransition(async () => {
      const res = await upsertEffect({ itemIndex, effect: newEffect, value: parsed, reason: "" });
      setFeedback(res.ok ? { ok: true, text: res.message ?? "salvo" } : { ok: false, text: res.error });
      if (res.ok) {
        setNewEffect("");
        setNewValue("0");
        router.refresh();
      }
    });
  }

  return (
    <>
      {reason ? (
        <p className="ui-field__hint" style={{ marginTop: 0 }}>
          {reason}
        </p>
      ) : null}

      <div style={{ display: "grid", gap: 8 }}>
        {effects.map((effect) => (
          <EffectRow
            key={effect.effect}
            itemIndex={itemIndex}
            effect={effect}
            onDone={() => router.refresh()}
          />
        ))}
      </div>

      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          alignItems: "center",
          gap: 8,
          marginTop: 12,
          paddingTop: 12,
          borderTop: "1px solid var(--border)",
        }}
      >
        <select
          value={newEffect}
          onChange={(event) => setNewEffect(event.target.value === "" ? "" : Number(event.target.value))}
          style={{ minWidth: 190 }}
          aria-label="novo efeito"
          disabled={pending || available.length === 0}
        >
          <option value="">— adicionar efeito —</option>
          {available.map((type) => (
            <option key={type.code} value={type.code}>
              {type.label || type.name} ({type.code})
            </option>
          ))}
        </select>
        <input
          type="number"
          value={newValue}
          onChange={(event) => setNewValue(event.target.value)}
          style={{ width: 96 }}
          aria-label="valor do novo efeito"
          disabled={pending}
        />
        <button
          type="button"
          className="ui-btn ui-btn--primary"
          onClick={add}
          disabled={pending || newEffect === ""}
        >
          adicionar
        </button>
        <span className="ui-field__hint" style={{ margin: 0 }}>
          {available.length === 0
            ? "todos os efeitos do vocabulário já estão neste item"
            : "um efeito que o conteúdo não declara também pode ser concedido aqui"}
        </span>
        {feedback ? (
          <span
            className="ui-field__hint"
            style={{ margin: 0, color: feedback.ok ? "var(--ok, inherit)" : "var(--danger, inherit)" }}
            role="status"
          >
            {feedback.text}
          </span>
        ) : null}
      </div>
    </>
  );
}

function EffectRow({
  itemIndex,
  effect,
  onDone,
}: {
  itemIndex: number;
  effect: ItemEffectEntry;
  onDone: () => void;
}) {
  const [value, setValue] = useState(String(effect.value));
  const [reason, setReason] = useState("");
  const [feedback, setFeedback] = useState<{ ok: boolean; text: string } | null>(null);
  const [pending, startTransition] = useTransition();

  // Depois de salvar ou reverter o servidor devolve outro valor para este efeito;
  // sem isto o input continuaria mostrando o rascunho antigo (o "48" quando o
  // slot já voltou a 47), dando a impressão de que a escrita não pegou.
  useEffect(() => {
    setValue(String(effect.value));
  }, [effect.value, effect.has_override]);

  const parsed = Number.parseInt(value, 10);
  const valid = Number.isInteger(parsed);
  const changed = parsed !== effect.value;

  function save() {
    startTransition(async () => {
      if (!valid) {
        setFeedback({ ok: false, text: "valor precisa ser um número inteiro" });
        return;
      }
      const res = await upsertEffect({
        itemIndex,
        effect: effect.effect,
        value: parsed,
        reason: reason.trim(),
      });
      setFeedback(res.ok ? { ok: true, text: res.message ?? "salvo" } : { ok: false, text: res.error });
      if (res.ok) onDone();
    });
  }

  function revert() {
    startTransition(async () => {
      const res = await deleteEffect({ itemIndex, effect: effect.effect, reason: reason.trim() });
      setFeedback(res.ok ? { ok: true, text: res.message ?? "restaurado" } : { ok: false, text: res.error });
      if (res.ok) onDone();
    });
  }

  return (
    <div
      style={{
        display: "flex",
        flexWrap: "wrap",
        alignItems: "center",
        gap: 8,
        paddingTop: 8,
        borderTop: "1px solid var(--border)",
      }}
    >
      <span style={{ minWidth: 130 }}>{effect.label || effect.name || "—"}</span>
      <span className="ui-mono" style={{ color: "var(--muted)" }} title="código em ItemEffect.h">
        EF_{effect.name || "?"} ({effect.effect})
      </span>
      <input
        type="number"
        value={value}
        onChange={(event) => setValue(event.target.value)}
        style={{ width: 96 }}
        aria-label={`valor de ${effect.label || effect.name}`}
        disabled={pending}
      />
      <button
        type="button"
        className="ui-btn ui-btn--primary"
        onClick={save}
        disabled={pending || !changed || !valid}
        title={changed ? "gravar este efeito no overlay" : "nada mudou"}
      >
        salvar
      </button>
      {effect.has_override ? (
        <button
          type="button"
          className="ui-btn"
          onClick={revert}
          disabled={pending}
          title="remover a edição: o efeito volta a valer o ItemList.csv"
        >
          voltar ao conteúdo
        </button>
      ) : null}
      <input
        type="text"
        value={reason}
        onChange={(event) => setReason(event.target.value)}
        placeholder="motivo (vai para a auditoria)"
        style={{ flex: "1 1 180px", minWidth: 140 }}
        aria-label={`motivo de ${effect.label || effect.name}`}
        disabled={pending}
      />
      <span className="ui-field__hint" style={{ margin: 0 }}>
        {effect.has_override
          ? `editado na web · ${effect.updated_at || "sem data"}`
          : effect.from_content
            ? "padrão do conteúdo"
            : "só na web"}
      </span>
      {feedback ? (
        <span
          className="ui-field__hint"
          style={{ margin: 0, color: feedback.ok ? "var(--ok, inherit)" : "var(--danger, inherit)" }}
          role="status"
        >
          {feedback.text}
        </span>
      ) : null}
    </div>
  );
}
