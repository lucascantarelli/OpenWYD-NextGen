"use client";

import { CircleMinus } from "lucide-react";
import { useEffect, useActionState, useState } from "react";

import { debitDonateAction, type DebitDonateState } from "./actions";

/**
 * Débito de saldo de donate, em um formulário só.
 *
 * O botão só habilita depois que o React assume o formulário: antes disso o clique
 * cairia no HTML sem ação e não faria nada. O resultado abaixo é o recibo real.
 */
export function DebitForm() {
  const [state, formAction, pending] = useActionState<DebitDonateState, FormData>(
    debitDonateAction,
    { status: "idle" },
  );
  const [pronto, setPronto] = useState(false);
  useEffect(() => setPronto(true), []);

  return (
    <form action={formAction} className="grant-form">
      <label className="sr-only" htmlFor="debit-conta">
        Login ou id da conta a debitar
      </label>
      <input
        id="debit-conta"
        className="ui-input ui-input--compact"
        name="conta"
        type="text"
        placeholder="login ou id da conta"
        autoComplete="off"
        required
      />
      <label className="sr-only" htmlFor="debit-credits">
        Créditos a debitar
      </label>
      <input
        id="debit-credits"
        className="ui-input ui-input--compact"
        name="credits"
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
        <CircleMinus size={13} strokeWidth={1.9} aria-hidden="true" />
        {!pronto ? "Carregando…" : pending ? "Debitando…" : "Debitar"}
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
