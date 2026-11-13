import { Search, ShieldAlert, UserRound } from "lucide-react";
import Link from "next/link";

import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import { ROLES, roleLabel, roleTone, type Role } from "@/lib/role";
import {
  findAccount,
  getAccountModeration,
  listAccounts,
  parseAccountCursor,
  type AccountModerationView,
  type AccountRow,
  type AccountSummary,
} from "@/server/accounts";
import { AdminError } from "@/server/errors";
import { getAuditTrail, loginHistoryFor, type AuditEntryView } from "@/server/moderation";
import { getSession } from "@/server/session";

import { CreateAccountDialog } from "./create-account-dialog";
import { ModerationPanel } from "./moderation-panel";

export const dynamic = "force-dynamic";

export const metadata = { title: "Contas · WYD2" };

/**
 * Contas: busca exata, criação e moderação.
 *
 * O contrato expõe `accounts/find`, `accounts/create` e a moderação
 * (`get_moderation`/`set_role`/`set_blocked`). A tela mostra o que existe: uma
 * conta por busca, com papel e bloqueio lidos do banco, e ações confirmadas.
 */
export default async function AccountsPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const params = await searchParams;
  const raw = params.login;
  const login = (Array.isArray(raw) ? raw[0] ?? "" : raw ?? "").trim();
  const after = parseAccountCursor(params.after);
  const papelRaw = params.papel;
  const papel = (Array.isArray(papelRaw) ? papelRaw[0] ?? "" : papelRaw ?? "").trim();
  const papelFiltro = (ROLES as readonly string[]).includes(papel) ? papel : "";
  const session = await getSession();

  let account: AccountSummary | null = null;
  let moderation: AccountModerationView | null = null;
  let error: string | null = null;
  let history: AuditEntryView[] | null = null;
  if (login !== "") {
    try {
      account = await findAccount(login);
      if (account) {
        moderation = await getAccountModeration(login);
        try {
          history = loginHistoryFor(await getAuditTrail(200), login);
        } catch {
          // Trilha exclusiva de admin: moderador vê a ficha sem o histórico.
          history = null;
        }
      }
    } catch (err) {
      error =
        err instanceof AdminError ? err.message : "não foi possível consultar a conta agora";
    }
  }

  let rows: AccountRow[] = [];
  let nextAfter: number | undefined;
  let listError: string | null = null;
  if (login === "") {
    try {
      const page = await listAccounts({ limit: 50, afterId: after });
      rows = papelFiltro
        ? page.rows.filter((row) => row.role === papelFiltro)
        : page.rows;
      nextAfter = page.nextAfterId;
    } catch (err) {
      listError =
        err instanceof AdminError ? err.message : "não foi possível listar as contas agora";
    }
  }

  const targetRole = (moderation?.account.role ?? account?.role ?? "player") as Role;
  const targetIsStaff = targetRole === "moderator" || targetRole === "admin";

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Operação</p>
          <h1 className="page-head__title">Contas</h1>
          <p className="page-head__sub">
            Busca exata por login. O papel e o bloqueio valem no jogo e no portal e são conferidos a
            cada login.
          </p>
        </div>
        <div className="page-head__actions">
          <CreateAccountDialog canCreateStaff={session?.role === "admin"} />
        </div>
      </div>

      <form className="ui-filters" action="/contas" method="get" role="search">
        <div className="search">
          <Search size={16} strokeWidth={1.8} aria-hidden="true" />
          <input
            className="ui-input"
            type="search"
            name="login"
            defaultValue={login}
            placeholder="Login exato da conta"
            aria-label="Login exato da conta"
            maxLength={16}
            autoFocus
          />
        </div>
        <button className="ui-btn ui-btn--primary" type="submit">
          Buscar
        </button>
      </form>

      {error && (
        <div className="card">
          <div className="card__body">
            <div className="empty">
              <ShieldAlert size={22} strokeWidth={1.6} aria-hidden="true" />
              <strong>Não foi possível consultar</strong>
              <p>{error}</p>
              <RetryButton />
            </div>
          </div>
        </div>
      )}

      {!error && login !== "" && account === null && (
        <div className="card">
          <div className="card__body">
            <div className="empty">
              <UserRound size={22} strokeWidth={1.6} aria-hidden="true" />
              <strong>Nenhuma conta com esse login</strong>
              <p>Confira a grafia. A busca é exata — não aceita prefixo nem parte do nome.</p>
            </div>
          </div>
        </div>
      )}

      {!error && account !== null && (
        <>
          <div className="card">
            <div className="card__head">
              <div>
                <h2 className="card__title">{account.login}</h2>
                <p className="card__sub">Conta #{account.id}</p>
              </div>
              <div className="ui-actions" style={{ display: "flex", gap: 8 }}>
                {moderation?.blocked ? <Badge tone="danger">bloqueada</Badge> : null}
                <Badge tone={roleTone(account.role)}>{roleLabel(account.role)}</Badge>
              </div>
            </div>
            <div className="card__body">
              <dl className="kv">
                <dt>Login</dt>
                <dd className="mono">{account.login}</dd>
                <dt>Identificador</dt>
                <dd className="mono">{account.id}</dd>
                <dt>Papel</dt>
                <dd>{roleLabel(account.role)}</dd>
                <dt>Acesso</dt>
                <dd>
                  {moderation?.blocked
                    ? "bloqueado pela moderação"
                    : ["moderator", "admin"].includes(account.role as Role)
                      ? "painel administrativo"
                      : "jogo e área do jogador"}
                </dd>
                {moderation?.blocked ? (
                  <>
                    <dt>Motivo</dt>
                    <dd>{moderation.reason || "não informado"}</dd>
                    <dt>Bloqueada em</dt>
                    <dd>
                      {moderation.blockedAt
                        ? new Date(moderation.blockedAt).toLocaleString("pt-BR")
                        : "—"}
                    </dd>
                  </>
                ) : null}
              </dl>
            </div>
          </div>

          {history !== null && (
            <div className="card">
              <div className="card__head">
                <div>
                  <h2 className="card__title">Histórico de login</h2>
                  <p className="card__sub">
                    Tentativas de login com este login como alvo, dentro das últimas 200 ações
                    registradas.
                  </p>
                </div>
              </div>
              {history.length === 0 ? (
                <div className="card__body">
                  <p className="muted small" style={{ margin: 0 }}>
                    Nenhuma tentativa registrada na janela lida.
                  </p>
                </div>
              ) : (
                <div className="card__body card__body--flush">
                  <div className="ui-table-scroll">
                    <table className="ui-table">
                      <thead>
                        <tr>
                          <th>Quando</th>
                          <th>Resultado</th>
                        </tr>
                      </thead>
                      <tbody>
                        {history.map((entry, index) => (
                          <tr key={`${entry.at.toISOString()}-${index}`}>
                            <td style={{ whiteSpace: "nowrap" }}>
                              {Number.isNaN(entry.at.getTime())
                                ? "—"
                                : entry.at.toLocaleString("pt-BR")}
                            </td>
                            <td>{entry.detail || "—"}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          )}

          {session ? (
            <ModerationPanel
              login={account.login}
              role={targetRole}
              blocked={moderation?.blocked ?? false}
              canManageRoles={session.role === "admin"}
              targetIsStaff={targetIsStaff}
              actorRole={session.role}
            />
          ) : null}
        </>
      )}

      {!error && login === "" && (
        <div className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">Todas as contas</h2>
              <p className="card__sub">
                Lista paginada. O filtro por papel vale para a página exibida; bloqueio,
                histórico e moderação ficam na ficha de cada conta.
              </p>
            </div>
          </div>
          <div className="card__body">
            <form className="ui-filters" action="/contas" method="get">
              <label className="login-field">
                <span>Papel</span>
                <select className="ui-select" name="papel" defaultValue={papelFiltro}>
                  <option value="">todos</option>
                  <option value="player">jogador</option>
                  <option value="moderator">moderador</option>
                  <option value="admin">administrador</option>
                </select>
              </label>
              <button className="ui-btn" type="submit">
                Filtrar
              </button>
            </form>
          </div>
          {listError ? (
            <div className="card__body">
              <div className="empty">
                <ShieldAlert size={22} strokeWidth={1.6} aria-hidden="true" />
                <strong>Lista indisponível</strong>
                <p>{listError}</p>
                <RetryButton />
              </div>
            </div>
          ) : rows.length === 0 ? (
            <div className="card__body">
              <div className="empty">
                <UserRound size={22} strokeWidth={1.6} aria-hidden="true" />
                <strong>Nenhuma conta nesta página</strong>
                <p>
                  {after
                    ? "Fim da lista — volte ao início."
                    : "Nenhuma conta com este filtro."}
                </p>
                {after ? (
                  <p>
                    <Link href="/contas">Voltar ao início</Link>
                  </p>
                ) : null}
              </div>
            </div>
          ) : (
            <>
              <div className="card__body card__body--flush">
                <div className="ui-table-scroll">
                  <table className="ui-table">
                    <thead>
                      <tr>
                        <th>Login</th>
                        <th>Papel</th>
                        <th className="ui-align-right">Ação</th>
                      </tr>
                    </thead>
                    <tbody>
                      {rows.map((row) => (
                        <tr key={row.id}>
                          <td>
                            <span className="mono">{row.login}</span>{" "}
                            <span className="muted small">#{row.id}</span>
                          </td>
                          <td>
                            <Badge tone={roleTone(row.role)}>{roleLabel(row.role)}</Badge>
                          </td>
                          <td className="ui-align-right">
                            <Link href={`/contas?login=${encodeURIComponent(row.login)}`}>
                              ver ficha
                            </Link>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
              <div className="card__foot">
                {nextAfter ? (
                  <Link
                    href={`/contas?after=${nextAfter}${papelFiltro ? `&papel=${papelFiltro}` : ""}`}
                  >
                    Carregar mais (a partir de #{nextAfter})
                  </Link>
                ) : (
                  <span className="muted small">Fim da lista.</span>
                )}{" "}
                {after ? (
                  <Link href="/contas" style={{ marginLeft: 12 }}>
                    Voltar ao início
                  </Link>
                ) : null}
              </div>
            </>
          )}
        </div>
      )}
    </>
  );
}
