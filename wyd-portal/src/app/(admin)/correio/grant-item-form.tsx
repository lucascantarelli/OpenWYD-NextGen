"use client";

import { PackagePlus } from "lucide-react";
import { useActionState, useState } from "react";

import { grantItemAction, type GrantItemState } from "./actions";

/**
 * Formulário de concessão de item para um personagem.
 *
 * O índice do item é conferido contra o catálogo na tela (datalist), mas a
 * fonte da verdade continua o gateway: índice inexistente volta como recusa de
 * negócio, nunca como item inventado.
 */
export function GrantItemForm({
  characterId,
  itemNames,
}: {
  characterId: number;
  /** Sugestões de índice vindas do catálogo lido do gateway. */
  itemNames: readonly string[];
}) {
  const [fallbackKey] = useState(
    () => `correio-${characterId}-${Math.random().toString(36).slice(2, 10)}`,
  );
  const [state, formAction, pending] = useActionState<GrantItemState, FormData>(
    grantItemAction,
    { status: "idle" },
  );

  return (
    <form action={formAction} className="grant-form">
      <input type="hidden" name="character_id" value={characterId} />
      <label className="login-field grant-form__amount">
        <span>Índice do item</span>
        <input
          className="ui-input"
          name="item_id"
          type="number"
          min={1}
          required
          list={`catalogo-${characterId}`}
        />
        <datalist id={`catalogo-${characterId}`}>
          {itemNames.map((name) => (
            <option key={name} value={name} />
          ))}
        </datalist>
      </label>
      <label className="login-field grant-form__amount">
        <span>Qtd.</span>
        <input className="ui-input" name="qty" type="number" min={1} max={999} defaultValue={1} required />
      </label>
      <label className="login-field grant-form__key-field">
        <span>Chave de idempotência</span>
        <input
          className="ui-input"
          name="key"
          type="text"
          defaultValue={fallbackKey}
          minLength={8}
          maxLength={120}
          required
        />
      </label>
      <button className="ui-btn ui-btn--primary" type="submit" disabled={pending}>
        <PackagePlus size={15} strokeWidth={1.8} aria-hidden="true" />
        {pending ? "Enviando…" : "Conceder item"}
      </button>

      {state.status === "ok" && (
        <p role="status" className="grant-form__result">
          {state.message}
        </p>
      )}
      {state.status === "error" && (
        <p role="alert" className="login-error">
          {state.message}
        </p>
      )}
    </form>
  );
}
