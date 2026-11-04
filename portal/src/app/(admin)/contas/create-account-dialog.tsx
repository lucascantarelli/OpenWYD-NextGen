"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { Dialog } from "@/components/Dialog";
import { Field } from "@/components/Field";
import { useToast } from "@/components/Toast";
import {
  ACCOUNT_ROLES,
  emailError,
  type AccountRole,
  nameError,
  passwordError,
  reasonError,
  roleLabel,
} from "@/lib/account";
import { createAccount } from "@/server/actions/accounts";

/**
 * Criação de conta pelo painel (issue 32).
 *
 * Antes disso, conta só nascia pelo cadastro público (`AccountWebService`), sem
 * papel — e promover exigia `psql`. Aqui o admin já escolhe o papel e o motivo, e
 * a criação entra na mesma auditoria das alterações.
 */
export function CreateAccountDialog() {
  const router = useRouter();
  const toast = useToast();
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [form, setForm] = useState<{
    name: string;
    password: string;
    email: string;
    role: AccountRole;
    reason: string;
  }>({
    name: "",
    password: "",
    email: "",
    role: "player",
    reason: "",
  });

  function patch(values: Partial<typeof form>) {
    setForm((current) => ({ ...current, ...values }));
  }

  function validate(): string | null {
    return (
      nameError(form.name) ??
      passwordError(form.password) ??
      emailError(form.email) ??
      reasonError(form.reason)
    );
  }

  async function submit() {
    const invalid = validate();
    if (invalid) {
      setError(invalid);
      return;
    }
    setPending(true);
    const result = await createAccount(form);
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    toast(result.message ?? "conta criada", "ok");
    setOpen(false);
    setError(null);
    setForm({ name: "", password: "", email: "", role: "player", reason: "" });
    router.refresh();
  }

  return (
    <>
      <button className="ui-btn ui-btn--primary" onClick={() => setOpen(true)}>
        criar conta
      </button>

      <Dialog
        open={open}
        title="Criar conta"
        onClose={() => setOpen(false)}
        footer={
          <div style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}>
            <button className="ui-btn ui-btn--ghost" onClick={() => setOpen(false)} disabled={pending}>
              cancelar
            </button>
            <button className="ui-btn ui-btn--primary" onClick={submit} disabled={pending}>
              criar
            </button>
          </div>
        }
      >
        <div style={{ display: "grid", gap: 10 }}>
          <p className="ui-field__hint" style={{ margin: 0 }}>
            A senha é gravada só como argon2id. A conta já nasce com o papel escolhido — nenhuma
            escrita toca personagem vivo.
          </p>

          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 10 }}>
            <Field label="login" hint="4–12 letras/números, sem acento">
              <input
                className="ui-input"
                value={form.name}
                maxLength={12}
                autoComplete="off"
                onChange={(e) => patch({ name: e.target.value })}
              />
            </Field>
            <Field label="senha" hint="mínimo 4 caracteres">
              <input
                className="ui-input"
                type="password"
                value={form.password}
                autoComplete="new-password"
                onChange={(e) => patch({ password: e.target.value })}
              />
            </Field>
          </div>

          <Field label="e-mail" hint="opcional">
            <input
              className="ui-input"
              type="email"
              value={form.email}
              autoComplete="off"
              onChange={(e) => patch({ email: e.target.value })}
            />
          </Field>

          <Field label="papel" hint="admin pode tudo; moderador só bloqueia jogador">
            <select
              className="ui-input"
              value={form.role}
              onChange={(e) => patch({ role: e.target.value as AccountRole })}
            >
              {ACCOUNT_ROLES.map((role) => (
                <option key={role} value={role}>
                  {roleLabel(role)}
                </option>
              ))}
            </select>
          </Field>

          <Field label="motivo" hint="obrigatório — vai para a auditoria">
            <input
              className="ui-input"
              maxLength={200}
              value={form.reason}
              onChange={(e) => patch({ reason: e.target.value })}
            />
          </Field>

          {error && <p style={{ color: "var(--danger)", margin: 0 }}>{error}</p>}
        </div>
      </Dialog>
    </>
  );
}
