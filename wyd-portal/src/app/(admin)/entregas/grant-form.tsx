"use client";

import { Coins } from "lucide-react";
import { useEffect, useActionState, useState } from "react";

import { grantGoldAction, type GrantGoldState } from "./actions";

/**
 * Concessão de ouro para um personagem, em uma linha só.
 *
 * A chave de idempotência não aparece para o staff: ela é gerada por personagem
 * e mantida entre reenvios da mesma página, o que já impede entrega dupla. O
 * resultado da ação fica embaixo do formulário.
 */
export function GrantForm({
  characterId,
  initialKey,
}: {
  characterId: number;
  /** Chave vinda da URL (`?key=`), para retry seguro sobreviver ao refresh. */
  initialKey?: string;
}) {
  const [fallbackKey] = useState(
    () => `portal-${characterId}-${Math.random().toString(36).slice(2, 10)}`,
  );
  const defaultKey = initialKey ?? fallbackKey;
  const [state, formAction, pending] = useActionState<GrantGoldState, FormData>(
    grantGoldAction,
    { status: "idle" },
  );
  // Antes da hidratação o `action` do formulário ainda é o do HTML e o clique não
  // dispara a server action — o botão espera o React assumir o formulário.
  const [pronto, setPronto] = useState(false);
  useEffect(() => setPronto(true), []);

  return (
    <form action={formAction} className="grant-form">
      <input type="hidden" name="character_id" value={characterId} />
      <input type="hidden" name="key" value={defaultKey} />
      <label className="sr-only" htmlFor={`ouro-${characterId}`}>
        Ouro a conceder para o personagem #{characterId}
      </label>
      <input
        id={`ouro-${characterId}`}
        className="ui-input ui-input--compact"
        name="amount"
        type="number"
        min={1}
        max={1_000_000}
        defaultValue={100}
        required
      />
      <button
        className="ui-btn ui-btn--sm"
        type="submit"
        disabled={pending || !pronto}
        aria-busy={pending}
      >
        <Coins size={13} strokeWidth={1.9} aria-hidden="true" />
        {!pronto ? "Carregando…" : pending ? "Enviando…" : "Conceder"}
      </button>
      {state.status === "ok" && (
        <span role="status" className="grant-form__result text-ok">
          {state.message}
        </span>
      )}
      {state.status === "error" && (
        <span role="alert" className="grant-form__result login-error">
          {state.message}
        </span>
      )}
    </form>
  );
}
