"use client";

import { useState } from "react";

/** Formulário de recuperação: só envia o login; a resposta é genérica por desenho. */
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
    <form onSubmit={submit} style={{ display: "grid", gap: 8, maxWidth: 360 }}>
      <label htmlFor="recovery-login" style={{ fontSize: 13, color: "var(--muted)" }}>
        Login da conta
      </label>
      <input
        id="recovery-login"
        value={login}
        maxLength={16}
        minLength={1}
        required
        onChange={(event) => setLogin(event.target.value)}
        style={{ padding: "8px 10px", borderRadius: 6, border: "1px solid var(--border)" }}
      />
      <button type="submit" disabled={pending} style={{ padding: "8px 12px", borderRadius: 6 }}>
        {pending ? "Enviando…" : "Solicitar recuperação"}
      </button>
      {message ? (
        <p role="status" style={{ fontSize: 12, color: "var(--muted)", margin: 0 }}>
          {message}
        </p>
      ) : null}
    </form>
  );
}
