"use client";

import { Ban, ShieldCheck, UserCog } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { Dialog } from "@/components/Dialog";
import { Field } from "@/components/Field";
import { useToast } from "@/components/Toast";
import { ROLES, roleLabel, type Role } from "@/lib/role";

import { moderateAccountAction } from "./moderation-actions";

/**
 * Moderação de uma conta com confirmação explícita.
 *
 * Nada é aplicado no clique do botão: cada ação abre um diálogo que diz o que
 * vai acontecer, e o bloqueio exige motivo (vai para a trilha de auditoria).
 * O painel não decide alcance — o gateway relê o papel da sessão; aqui só se
 * escolhe o que *tentar*.
 */
export function ModerationPanel({
  login,
  role,
  blocked,
  canManageRoles,
  targetIsStaff,
  actorRole,
}: {
  login: string;
  role: Role;
  blocked: boolean;
  canManageRoles: boolean;
  targetIsStaff: boolean;
  actorRole: Role;
}) {
  const router = useRouter();
  const toast = useToast();
  const [mode, setMode] = useState<null | "role" | "block" | "unblock">(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [nextRole, setNextRole] = useState<Role>(role);
  const [reason, setReason] = useState("");

  const canBlockStaff = canManageRoles;

  function close() {
    setMode(null);
    setError(null);
    setReason("");
    setNextRole(role);
  }

  async function submit() {
    if (mode === null) return;
    setPending(true);
    const result = await moderateAccountAction(
      mode === "role"
        ? { login, action: "role", role: nextRole }
        : { login, action: mode, reason: mode === "unblock" ? "" : reason },
    );
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      return;
    }
    toast(result.message, "ok");
    close();
    router.refresh();
  }

  const runtimeRole = actorRole === "admin" ? "administrador" : "moderador";
  const blockDisabled = targetIsStaff && !canBlockStaff;

  return (
    <section className="card">
      <div className="card__head">
        <div>
          <h2 className="card__title">Moderação</h2>
          <p className="card__sub">
            Ação de staff, confirmada e registrada na trilha com autor, alvo e motivo.
          </p>
        </div>
        <span className="muted small">você age como {runtimeRole}</span>
      </div>
      <div className="card__body">
        <div className="ui-actions" style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
          <button
            className="ui-btn ui-btn--ghost"
            onClick={() => setMode("role")}
            disabled={!canManageRoles}
            title={canManageRoles ? "Trocar o papel da conta" : "só administrador troca papel"}
          >
            <UserCog size={15} strokeWidth={1.8} aria-hidden="true" />
            Alterar papel
          </button>

          {blocked ? (
            <button className="ui-btn ui-btn--primary" onClick={() => setMode("unblock")}>
              <ShieldCheck size={15} strokeWidth={1.8} aria-hidden="true" />
              Liberar acesso
            </button>
          ) : (
            <button
              className="ui-btn ui-btn--danger"
              onClick={() => setMode("block")}
              disabled={blockDisabled}
              title={blockDisabled ? "só administrador bloqueia staff" : "Bloquear o acesso da conta"}
            >
              <Ban size={15} strokeWidth={1.8} aria-hidden="true" />
              Bloquear acesso
            </button>
          )}
        </div>

        {blocked && (
          <p className="muted small" style={{ marginTop: 12, marginBottom: 0 }}>
            Bloqueada: {reason.trim() === "" ? "motivo não informado" : reason}
          </p>
        )}
      </div>

      <Dialog
        open={mode !== null}
        title={
          mode === "role"
            ? "Alterar papel"
            : mode === "block"
              ? "Bloquear conta"
              : "Liberar conta"
        }
        description={
          mode === "role"
            ? `O novo papel vale no jogo e no portal e é conferido a cada login de ${login}.`
            : mode === "block"
              ? `${login} perde o acesso ao jogo e ao portal até ser liberada. O motivo fica na trilha.`
              : `${login} volta a poder entrar no jogo e no portal.`
        }
        onClose={close}
        footer={
          <>
            <button className="ui-btn ui-btn--ghost" onClick={close} disabled={pending}>
              Cancelar
            </button>
            <button
              className={mode === "block" ? "ui-btn ui-btn--danger" : "ui-btn ui-btn--primary"}
              onClick={submit}
              disabled={pending || (mode === "block" && reason.trim() === "")}
            >
              {pending ? "Aplicando…" : "Confirmar"}
            </button>
          </>
        }
      >
        <div className="ui-form" style={{ maxWidth: "none" }}>
          {mode === "role" ? (
            <Field label="Novo papel" hint="moderador e admin abrem o painel administrativo">
              <select
                className="ui-select"
                value={nextRole}
                onChange={(event) => setNextRole(event.target.value as Role)}
                autoFocus
              >
                {ROLES.map((option) => (
                  <option key={option} value={option}>
                    {roleLabel(option)}
                  </option>
                ))}
              </select>
            </Field>
          ) : null}

          {mode === "block" ? (
            <Field label="Motivo do bloqueio" hint="obrigatório; vai para a trilha de auditoria">
              <textarea
                className="ui-input"
                rows={3}
                maxLength={200}
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                autoFocus
              />
            </Field>
          ) : null}

          {error ? (
            <p className="ui-field__error" role="alert">
              {error}
            </p>
          ) : null}
        </div>
      </Dialog>
    </section>
  );
}
