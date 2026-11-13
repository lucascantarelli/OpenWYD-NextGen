"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

const MESSAGES: Record<number, string> = {
  401: "Usuário ou senha inválidos.",
  403: "Esta conta está bloqueada.",
  422: "Preencha usuário e senha.",
  502: "O servidor de jogo não respondeu. Tente novamente em instantes.",
};

/** Destino por papel: jogador vai para a área dele; staff vai para a administração. */
export function destinoPorPapel(role: string | undefined): string {
  return role === "admin" || role === "moderator" ? "/painel" : "/conta";
}

export default function LoginForm() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [pass, setPass] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ name, pass }),
      });
      if (res.ok) {
        const body = (await res.json().catch(() => ({}))) as { role?: string };
        router.replace(destinoPorPapel(body.role));
        router.refresh();
        return;
      }
      setError(MESSAGES[res.status] ?? "Não foi possível entrar.");
    } catch {
      setError("Falha de rede. Verifique se o portal está no ar.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="login-form">
      <label className="login-field">
        <span>Usuário</span>
        <input
          name="name"
          className="ui-input"
          value={name}
          onChange={(event) => setName(event.target.value)}
          autoComplete="username"
          autoFocus
          required
          maxLength={16}
        />
      </label>

      <label className="login-field">
        <span>Senha</span>
        <input
          name="pass"
          className="ui-input"
          type="password"
          value={pass}
          onChange={(event) => setPass(event.target.value)}
          autoComplete="current-password"
          required
          maxLength={64}
        />
      </label>

      {error && (
        <p role="alert" className="login-error">
          {error}
        </p>
      )}

      <button type="submit" className="ui-btn ui-btn--primary" disabled={busy}>
        {busy ? "Entrando…" : "Entrar"}
      </button>
    </form>
  );
}
