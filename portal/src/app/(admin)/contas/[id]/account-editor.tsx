"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { Field } from "@/components/Field";
import { useToast } from "@/components/Toast";
import {
  ACCOUNT_ROLES,
  isAccountRole,
  MIN_REASON,
  reasonError,
  roleLabel,
  type AccountRole,
  type AdminAccount,
} from "@/lib/account";
import { changeAccountBlocked, changeAccountRole } from "@/server/actions/accounts";

/**
 * Editor de papel/bloqueio de uma conta (issue 32).
 *
 * Dois formulários separados, cada um com o **próprio motivo**: a auditoria
 * guarda um texto por ação, e reaproveitar o campo faria o motivo de um vazamento
 * de bloqueio virar o de uma promoção.
 *
 * O que a tela desabilita é espelho de UX; o web-api e o banco recusam de novo —
 * inclusive "não altere a si mesmo" e "última conta admin ativa".
 */
export function AccountEditor({
  account,
  actorRole,
  actorId,
}: {
  account: AdminAccount;
  actorRole: string;
  actorId: number;
}) {
  const router = useRouter();
  const toast = useToast();

  const isAdmin = actorRole === "admin";
  const isStaff = actorRole === "admin" || actorRole === "moderator";
  const isSelf = account.id === actorId;
  const canBlockTarget = isStaff && !isSelf && (isAdmin || account.role === "player");

  const [role, setRole] = useState<AccountRole>(
    isAccountRole(account.role) ? account.role : "player",
  );
  const [roleReason, setRoleReason] = useState("");
  const [roleError, setRoleError] = useState<string | null>(null);
  const [rolePending, setRolePending] = useState(false);

  const [blockReason, setBlockReason] = useState("");
  const [blockError, setBlockError] = useState<string | null>(null);
  const [blockPending, setBlockPending] = useState(false);

  async function submitRole() {
    const invalid = reasonError(roleReason);
    if (invalid) {
      setRoleError(invalid);
      return;
    }
    setRolePending(true);
    const result = await changeAccountRole({ accountId: account.id, role, reason: roleReason });
    setRolePending(false);
    if (!result.ok) {
      setRoleError(result.error);
      return;
    }
    setRoleError(null);
    setRoleReason("");
    toast(result.message ?? "papel atualizado", "ok");
    router.refresh();
  }

  async function submitBlock(next: boolean) {
    const invalid = reasonError(blockReason);
    if (invalid) {
      setBlockError(invalid);
      return;
    }
    setBlockPending(true);
    const result = await changeAccountBlocked({
      accountId: account.id,
      blocked: next,
      reason: blockReason,
    });
    setBlockPending(false);
    if (!result.ok) {
      setBlockError(result.error);
      return;
    }
    setBlockError(null);
    setBlockReason("");
    toast(result.message ?? (next ? "conta bloqueada" : "conta desbloqueada"), "ok");
    router.refresh();
  }

  return (
    <div style={{ display: "grid", gap: 12 }}>
      <div className="ui-card">
        <strong>Papel</strong>
        <p className="ui-field__hint">
          {isAdmin
            ? "Promover a admin dá acesso total ao painel. Rebaixar vale no próximo login."
            : "Só um admin altera papéis. Você pode bloquear contas de jogador."}
        </p>
        <div style={{ display: "grid", gridTemplateColumns: "1fr 2fr", gap: 10 }}>
          <Field label="novo papel">
            <select
              className="ui-input"
              value={role}
              disabled={!isAdmin || isSelf}
              onChange={(e) => setRole(e.target.value as AccountRole)}
            >
              {ACCOUNT_ROLES.map((value) => (
                <option key={value} value={value}>
                  {roleLabel(value)}
                </option>
              ))}
            </select>
          </Field>
          <Field label="motivo" hint={`mínimo ${MIN_REASON} caracteres, vai para a auditoria`}>
            <input
              className="ui-input"
              maxLength={200}
              value={roleReason}
              disabled={!isAdmin || isSelf}
              onChange={(e) => setRoleReason(e.target.value)}
            />
          </Field>
        </div>
        {roleError && <p style={{ color: "var(--danger)", margin: "6px 0 0" }}>{roleError}</p>}
        <div style={{ marginTop: 10 }}>
          <button
            className="ui-btn ui-btn--primary"
            onClick={submitRole}
            disabled={!isAdmin || isSelf || rolePending || role === account.role}
          >
            salvar papel
          </button>
          {isSelf && (
            <span className="ui-field__hint" style={{ marginLeft: 8 }}>
              não é possível alterar o próprio papel
            </span>
          )}
        </div>
      </div>

      <div className="ui-card">
        <strong>{account.is_blocked ? "Desbloquear" : "Bloquear"}</strong>
        <p className="ui-field__hint">
          Bloquear impede login no jogo e no portal. O motivo fica visível na tela e na auditoria;
          desbloquear limpa o motivo.
        </p>
        <Field label="motivo" hint={`mínimo ${MIN_REASON} caracteres`}>
          <input
            className="ui-input"
            maxLength={200}
            value={blockReason}
            disabled={!canBlockTarget}
            onChange={(e) => setBlockReason(e.target.value)}
          />
        </Field>
        {blockError && <p style={{ color: "var(--danger)", margin: "6px 0 0" }}>{blockError}</p>}
        <div style={{ marginTop: 10, display: "flex", gap: 8, alignItems: "center" }}>
          <button
            className={account.is_blocked ? "ui-btn ui-btn--primary" : "ui-btn ui-btn--danger"}
            onClick={() => submitBlock(!account.is_blocked)}
            disabled={!canBlockTarget || blockPending}
          >
            {account.is_blocked ? "desbloquear conta" : "bloquear conta"}
          </button>
          {!canBlockTarget && (
            <span className="ui-field__hint">
              {isSelf
                ? "não é possível bloquear a própria conta"
                : "moderador só bloqueia contas de jogador"}
            </span>
          )}
        </div>
      </div>
    </div>
  );
}
