import Link from "next/link";
import { notFound } from "next/navigation";

import { Badge } from "@/components/Badge";
import {
  actionLabel,
  fmtUnix,
  roleLabel,
  roleTone,
  type AdminAccount,
  type AccountAudit,
} from "@/lib/account";
import { getAccount } from "@/server/accounts";
import { AdminError } from "@/server/errors";
import { getSession } from "@/server/session";

import { AccountEditor } from "./account-editor";

export const dynamic = "force-dynamic";

export const metadata = { title: "Conta · WYD2" };

/**
 * Detalhe de conta (issue 32): papel, bloqueio, motivo e auditoria.
 *
 * O formulário é o único lugar do portal que escreve `account.role` /
 * `account.is_blocked`, e faz isso por action → `adminCall` → gRPC. O efeito é no
 * próximo login; a página não promete mais do que isso.
 */
export default async function AccountPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const accountId = Number(id);
  if (!Number.isInteger(accountId) || accountId <= 0) notFound();

  let detail: { account: AdminAccount; audit: AccountAudit[] } | null = null;
  let error: string | null = null;
  try {
    detail = await getAccount(accountId);
  } catch (err) {
    error =
      err instanceof AdminError ? err.message : "não foi possível carregar a conta no web-api";
  }

  if (error) {
    return (
      <div className="ui-card ui-empty">
        <strong>Não deu para carregar a conta</strong>
        <p>{error}</p>
        <Link href="/contas">voltar</Link>
      </div>
    );
  }
  if (!detail) notFound();

  const { account, audit } = detail;
  const session = await getSession();
  const actorRole = session?.role ?? "player";

  return (
    <>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            {account.name}
          </h1>
          <p className="ui-page-sub" style={{ margin: "4px 0 0" }}>
            conta <span className="ui-mono">#{account.id}</span> · {account.characters}{" "}
            {account.characters === 1 ? "personagem" : "personagens"} ·{" "}
            <Badge tone={roleTone(account.role)}>{roleLabel(account.role)}</Badge>{" "}
            {account.is_blocked ? (
              <Badge tone="danger">bloqueada</Badge>
            ) : (
              <Badge tone="ok">ativa</Badge>
            )}
          </p>
        </div>
        <div style={{ marginLeft: "auto" }}>
          <Link href="/contas" className="ui-btn ui-btn--ghost">
            voltar às contas
          </Link>
        </div>
      </div>

      {account.is_blocked && account.blocked_reason && (
        <div className="ui-card" style={{ marginBottom: 12, borderColor: "var(--danger)" }}>
          <strong>Motivo do bloqueio</strong>
          <p style={{ margin: "4px 0 0" }}>{account.blocked_reason}</p>
        </div>
      )}

      <div className="ui-card" style={{ marginBottom: 12 }}>
        <dl
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(150px, 1fr))",
            gap: 10,
            margin: 0,
          }}
        >
          <Stat label="E-mail" value={account.email || "—"} />
          <Stat label="Saldo de donate" value={`${account.donate_balance} créditos`} />
          <Stat label="Criada em" value={fmtUnix(account.created_at_unix)} />
          <Stat
            label="Papel alterado em"
            value={account.role_changed_at_unix ? fmtUnix(account.role_changed_at_unix) : "nunca pela web"}
          />
          <Stat
            label="Alterado por"
            value={account.role_changed_by ? `conta #${account.role_changed_by}` : "—"}
          />
        </dl>
        <p className="ui-field__hint" style={{ marginBottom: 0 }}>
          Papel e bloqueio são lidos no <strong>próximo login</strong>: um personagem online continua
          na sessão atual até reconectar.
        </p>
      </div>

      <AccountEditor account={account} actorRole={actorRole} actorId={session?.accountId ?? 0} />

      <h2 className="ui-page-title" style={{ fontSize: 15, marginTop: 20 }}>
        Auditoria ({audit.length})
      </h2>
      {audit.length === 0 ? (
        <div className="ui-card ui-empty">
          <p style={{ margin: 0 }}>Nenhuma alteração registrada pela web.</p>
        </div>
      ) : (
        <div className="ui-table-wrap">
          <div className="ui-table-scroll">
            <table className="ui-table">
              <thead>
                <tr>
                  <th>Quando</th>
                  <th>Quem</th>
                  <th>Ação</th>
                  <th>Antes → depois</th>
                  <th>Motivo</th>
                </tr>
              </thead>
              <tbody>
                {audit.map((entry) => (
                  <tr key={entry.id}>
                    <td>{fmtUnix(entry.created_at_unix)}</td>
                    <td className="ui-mono">#{entry.actor_id}</td>
                    <td>{actionLabel(entry.action)}</td>
                    <td className="ui-mono">
                      {entry.old_value || "—"} → {entry.new_value || "—"}
                    </td>
                    <td>{entry.reason || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </>
  );
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <dt className="ui-field__label" style={{ margin: 0 }}>
        {label}
      </dt>
      <dd style={{ margin: 0 }} className="ui-mono">
        {value}
      </dd>
    </div>
  );
}
