import { Search, ShieldAlert, UserRound } from "lucide-react";

import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import { roleLabel, roleTone, type Role } from "@/lib/role";
import { findAccount, type AccountSummary } from "@/server/accounts";
import { AdminError } from "@/server/errors";
import { getSession } from "@/server/session";

import { CreateAccountDialog } from "./create-account-dialog";

export const dynamic = "force-dynamic";

export const metadata = { title: "Contas · WYD2" };

/**
 * Contas: busca exata por login e criação administrativa.
 *
 * O contrato do gateway expõe `accounts/find` e `accounts/create` — não há
 * listagem paginada. A tela mostra o que existe: uma conta por busca, com papel
 * lido do banco, e a criação com papel explícito.
 */
export default async function AccountsPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const params = await searchParams;
  const raw = params.login;
  const login = (Array.isArray(raw) ? raw[0] ?? "" : raw ?? "").trim();
  const session = await getSession();

  let account: AccountSummary | null = null;
  let error: string | null = null;
  if (login !== "") {
    try {
      account = await findAccount(login);
    } catch (err) {
      error =
        err instanceof AdminError ? err.message : "não foi possível consultar a conta no gateway";
    }
  }

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Operação</p>
          <h1 className="page-head__title">Contas</h1>
          <p className="page-head__sub">Busca exata por login. O papel e o bloqueio valem no jogo e são conferidos a cada login.
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
        <div className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">{account.login}</h2>
              <p className="card__sub">Conta #{account.id}</p>
            </div>
            <Badge tone={roleTone(account.role)}>{roleLabel(account.role)}</Badge>
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
                {["moderator", "admin"].includes(account.role as Role)
                  ? "painel administrativo"
                  : "jogo e área do jogador"}
              </dd>
            </dl>
          </div>
        </div>
      )}

      {!error && login === "" && (
        <div className="card">
          <div className="card__body">
            <div className="empty">
              <UserRound size={22} strokeWidth={1.6} aria-hidden="true" />
              <strong>Busque uma conta para começar</strong>
              <p>
                A criação administrativa gera a conta e o papel. A senha de jogo e a senha numérica
                são definidas pelo jogador no primeiro acesso, dentro do cliente.
              </p>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
