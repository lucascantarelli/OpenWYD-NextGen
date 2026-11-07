import Link from "next/link";

import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import {
  ACCOUNT_PAGE_SIZE,
  BLOCKED_ANY,
  BLOCKED_NO,
  BLOCKED_YES,
  fmtUnix,
  roleLabel,
  roleTone,
  type AdminAccount,
} from "@/lib/account";
import { listAccounts } from "@/server/accounts";
import { AdminError } from "@/server/errors";
import { getSession } from "@/server/session";

import { CreateAccountDialog } from "./create-account-dialog";

export const dynamic = "force-dynamic";

export const metadata = { title: "Contas · WYD2" };

/**
 * Contas, papéis e banimento (issue 32).
 *
 * A lista é o estado **frio** do Postgres: papel, bloqueio e motivo. Mudar papel
 * ou bloquear vale a partir do **próximo login** — não há escrita no save nem no
 * personagem vivo, e a tela diz isso.
 *
 * O cursor é keyset (`apos` = último id da página), então a paginação continua
 * estável mesmo com contas criadas durante a navegação.
 */
export default async function AccountsPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const params = await searchParams;
  const first = (key: string) => {
    const value = params[key];
    return Array.isArray(value) ? value[0] ?? "" : value ?? "";
  };

  const query = first("q").trim().toLowerCase();
  const role = first("papel");
  const blockedParam = Number(first("bloqueio") || String(BLOCKED_ANY));
  const blocked = [BLOCKED_ANY, BLOCKED_NO, BLOCKED_YES].includes(blockedParam)
    ? blockedParam
    : BLOCKED_ANY;
  const afterId = Math.max(0, Number(first("apos")) || 0);

  const page = { query, role, blocked, afterId, limit: ACCOUNT_PAGE_SIZE };

  let accounts: AdminAccount[] = [];
  let total = 0;
  let nextId = 0;
  let error: string | null = null;
  try {
    const result = await listAccounts(page);
    accounts = result.accounts;
    total = result.total;
    nextId = result.nextId;
  } catch (err) {
    error =
      err instanceof AdminError ? err.message : "não foi possível listar as contas no web-api";
  }

  const session = await getSession();
  const isAdmin = session?.role === "admin";

  const nextParams = new URLSearchParams();
  if (query) nextParams.set("q", query);
  if (role) nextParams.set("papel", role);
  if (blocked !== BLOCKED_ANY) nextParams.set("bloqueio", String(blocked));
  nextParams.set("apos", String(nextId));

  return (
    <>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            Contas e papéis
          </h1>
          <p className="ui-page-sub" style={{ margin: "4px 0 0" }}>
            Papel e bloqueio valem no <strong>próximo login</strong>. Toda alteração exige motivo e
            entra na auditoria.
          </p>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 8 }}>
          {isAdmin ? <CreateAccountDialog /> : null}
        </div>
      </div>

      <form className="ui-filters" action="/contas" method="get" style={{ gap: 8, flexWrap: "wrap" }}>
        <input
          className="ui-input"
          type="search"
          name="q"
          defaultValue={query}
          placeholder="login (prefixo)"
          style={{ maxWidth: 220 }}
        />
        <select className="ui-input" name="papel" defaultValue={role} style={{ maxWidth: 160 }}>
          <option value="">todos os papéis</option>
          <option value="player">jogador</option>
          <option value="moderator">moderador</option>
          <option value="admin">admin</option>
        </select>
        <select
          className="ui-input"
          name="bloqueio"
          defaultValue={String(blocked)}
          style={{ maxWidth: 160 }}
        >
          <option value={BLOCKED_ANY}>todas</option>
          <option value={BLOCKED_NO}>ativas</option>
          <option value={BLOCKED_YES}>bloqueadas</option>
        </select>
        <button className="ui-btn ui-btn--primary" type="submit">
          filtrar
        </button>
        <Link href="/contas" className="ui-btn ui-btn--ghost">
          limpar
        </Link>
      </form>

      {error && (
        <div className="ui-card ui-empty" style={{ marginBottom: 12 }}>
          <strong>Não deu para listar</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      )}

      {!error && accounts.length === 0 && (
        <div className="ui-card ui-empty">
          <strong>Nenhuma conta encontrada</strong>
          <p>
            {query || role || blocked !== BLOCKED_ANY
              ? "Nenhuma conta combina com esses filtros."
              : "O banco ainda não tem contas."}
          </p>
        </div>
      )}

      {!error && accounts.length > 0 && (
        <>
          <p className="ui-field__hint" style={{ marginBottom: 10 }}>
            {total} {total === 1 ? "conta" : "contas"} no filtro ·{" "}
            <Badge tone="neutral">somente leitura nesta tela</Badge>
          </p>
          <div className="ui-table-wrap">
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Conta</th>
                    <th>Personagens</th>
                    <th>Papel</th>
                    <th>Situação</th>
                    <th>E-mail</th>
                    <th>Criada em</th>
                  </tr>
                </thead>
                <tbody>
                  {accounts.map((account) => (
                    <tr key={account.id}>
                      <td>
                        <Link href={`/contas/${account.id}`} className="ui-mono">
                          {account.name}
                        </Link>
                      </td>
                      <td>{account.characters}</td>
                      <td>
                        <Badge tone={roleTone(account.role)}>{roleLabel(account.role)}</Badge>
                      </td>
                      <td>
                        {account.is_blocked ? (
                          <Badge tone="danger" title={account.blocked_reason || "sem motivo registrado"}>
                            bloqueada
                          </Badge>
                        ) : (
                          <Badge tone="ok">ativa</Badge>
                        )}
                      </td>
                      <td>{account.email || "—"}</td>
                      <td>{fmtUnix(account.created_at_unix)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
          {nextId > 0 && (
            <div style={{ display: "flex", gap: 8, marginTop: 12 }}>
              <Link href={`/contas?${nextParams.toString()}`} className="ui-btn ui-btn--ghost">
                próxima página
              </Link>
            </div>
          )}
        </>
      )}
    </>
  );
}
