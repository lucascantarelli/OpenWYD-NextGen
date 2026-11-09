"use client";

import { Coins } from "lucide-react";
import { useActionState, useState } from "react";

import { grantGoldAction, type GrantGoldState } from "./actions";

/**
 * Formulário de concessão de ouro para um personagem.
 *
 * A chave de idempotência é visível e editável de propósito: reenviar a mesma
 * chave é um retry seguro (o gateway devolve o mesmo `grant_id` com replay), e
 * o staff vê a chave que está sendo usada.
 */
export function GrantForm({ characterId }: { characterId: number }) {
  const [initialKey] = useState(
    () => `portal-${characterId}-${Math.random().toString(36).slice(2, 10)}`,
  );
  const [state, formAction, pending] = useActionState<GrantGoldState, FormData>(
    grantGoldAction,
    { status: "idle" },
  );

  return (
    <form action={formAction} className="grant-form">
      <input type="hidden" name="character_id" value={characterId} />
      <label className="login-field grant-form__amount">
        <span>Ouro</span>
        <input
          className="ui-input"
          name="amount"
          type="number"
          min={1}
          max={1_000_000}
          defaultValue={100}
          required
        />
      </label>
      <label className="login-field grant-form__key-field">
        <span>Chave de idempotência</span>
        <input
          className="ui-input"
          name="key"
          type="text"
          defaultValue={initialKey}
          minLength={8}
          maxLength={64}
          required
        />
      </label>
      <button className="ui-btn ui-btn--primary" type="submit" disabled={pending}>
        <Coins size={15} strokeWidth={1.8} aria-hidden="true" />
        {pending ? "Enviando…" : "Conceder ouro"}
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
