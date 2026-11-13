"use client";

import { useState } from "react";

/** Recuperação: envia só o login; a resposta é a mesma exista ou não a conta. */
export default function RecoveryForm() {
  const [login, setLogin] = useState("");
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    try {
      const res = await fetch("/api/panel/recovery", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ login }),
        cache: "no-store",
      });
      const body = (await res.json().catch(() => ({}))) as { message?: string; error?: string };
      setMessage(res.ok ? (body.message ?? "Solicitação registrada.") : (body.error ?? "Falha ao solicitar."));
    } catch {
      setMessage("Serviço indisponível: tente de novo em instantes.");
    } finally {
      setPending(false);
    }
  }

  return (
    <form onSubmit={submit} className="ui-form">
      <label className="ui-field">
        <span className="ui-field__label">Login da conta</span>
        <input
          className="ui-input"
          value={login}
          maxLength={16}
          minLength={1}
          required
          autoComplete="username"
          onChange={(event) => setLogin(event.target.value)}
        />
      </label>
      <div className="ui-form__actions">
        <button type="submit" className="ui-btn" disabled={pending}>
          {pending ? "Enviando…" : "Solicitar recuperação"}
        </button>
      </div>
      {message ? (
        <p role="status" className="ui-field__hint">
          {message}
        </p>
      ) : null}
    </form>
  );
}
