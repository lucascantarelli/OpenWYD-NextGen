"use client";

import { UserPlus } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { Dialog } from "@/components/Dialog";
import { Field } from "@/components/Field";
import { useToast } from "@/components/Toast";
import { ROLES, roleLabel, type Role } from "@/lib/role";

import { createAccountAction } from "./actions";

/**
 * Criação de conta pelo painel.
 *
 * A conta nasce **sem credencial**: a senha de jogo é definida no primeiro
 * acesso, e a senha numérica na tela de personagem do cliente. Aqui se escolhe
 * só o login e o papel — o resto é do fluxo de entrada.
 */
export function CreateAccountDialog({ canCreateStaff }: { canCreateStaff: boolean }) {
  const router = useRouter();
  const toast = useToast();
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [login, setLogin] = useState("");
  const [role, setRole] = useState<Role>("player");

  async function submit() {
    if (login.trim() === "") {
      setError("informe o login");
      return;
    }
    setPending(true);
    const result = await createAccountAction({ login, role });
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    toast(result.message, "ok");
    setOpen(false);
    setError(null);
    setLogin("");
    setRole("player");
    router.refresh();
  }

  return (
    <>
      <button className="ui-btn ui-btn--primary" onClick={() => setOpen(true)}>
        <UserPlus size={15} strokeWidth={1.8} aria-hidden="true" />
        Criar conta
      </button>

      <Dialog
        open={open}
        title="Criar conta"
        description="A conta nasce sem senha de jogo; o jogador define as credenciais no primeiro acesso."
        onClose={() => setOpen(false)}
        footer={
          <>
            <button className="ui-btn ui-btn--ghost" onClick={() => setOpen(false)} disabled={pending}>
              Cancelar
            </button>
            <button className="ui-btn ui-btn--primary" onClick={submit} disabled={pending}>
              {pending ? "Criando…" : "Criar conta"}
            </button>
          </>
        }
      >
        <div className="ui-form" style={{ maxWidth: "none" }}>
          <Field label="Login" hint="4 a 12 letras ou números, sem espaço">
            <input
              className="ui-input"
              value={login}
              maxLength={12}
              autoComplete="off"
              autoFocus
              onChange={(event) => setLogin(event.target.value)}
            />
          </Field>

          <Field
            label="Papel"
            hint={canCreateStaff ? "administrador pode criar qualquer papel" : "você só cria jogador"}
          >
            <select
              className="ui-select"
              value={role}
              onChange={(event) => setRole(event.target.value as Role)}
              disabled={!canCreateStaff}
            >
              {ROLES.filter((option) => canCreateStaff || option === "player").map((option) => (
                <option key={option} value={option}>
                  {roleLabel(option)}
                </option>
              ))}
            </select>
          </Field>

          {error ? (
            <p className="ui-field__error" role="alert">
              {error}
            </p>
          ) : null}
        </div>
      </Dialog>
    </>
  );
}
