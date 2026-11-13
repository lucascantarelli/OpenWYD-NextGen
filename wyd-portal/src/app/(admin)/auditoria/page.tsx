import { ScrollText, ShieldAlert } from "lucide-react";

import { Badge } from "@/components/Badge";
import { AdminError } from "@/server/errors";
import {
  filterAuditTrail,
  getAuditTrail,
  type AuditEntryView,
} from "@/server/moderation";
import { getSession } from "@/server/session";

export const dynamic = "force-dynamic";

export const metadata = { title: "Auditoria · WYD2" };

const fmtWhen = (when: Date): string => {
  if (Number.isNaN(when.getTime())) return "—";
  return when.toLocaleString("pt-BR");
};

const one = (value: string | string[] | undefined): string =>
  (Array.isArray(value) ? (value[0] ?? "") : (value ?? "")).trim();

/** Nome de ação como o staff o entende; o identificador cru fica no title. */
const ACTION_LABEL: Record<string, string> = {
  "auth.login": "tentativa de login",
  "accounts.find": "busca de conta",
  "accounts.create": "criação de conta",
  "accounts.list": "lista de contas",
  "accounts.get": "ficha de conta",
  "accounts.set_role": "mudança de papel",
  "accounts.set_blocked": "bloqueio de conta",
  "catalog.list": "leitura do catálogo",
  "catalog.list_items": "leitura do catálogo",
  "config.rates": "leitura de taxas",
  "config.teleport": "leitura de teleportes",
  "config.drops": "leitura de drops",
  "config.npcs": "leitura de NPCs",
  "server.presence": "presença do servidor",
  "character.search": "busca de personagem",
  "character.grants": "fila do personagem",
  "character.grant_item": "concessão de item",
  "drop.list": "leitura de drops",
  "teleport.list": "leitura de teleportes",
  "npc.list": "leitura de NPCs",
  "mob.list": "leitura de monstros",
  "panel.channels": "canais do painel",
  "grants.request": "pedido de entrega",
  "audit.read": "leitura da trilha",
};

function actionLabel(name: string): string {
  if (name.startsWith("auth.denied")) return "recusa por permissão";
  return ACTION_LABEL[name] ?? name;
}

/**
 * Auditoria: últimas ações de staff, lidas de `audit_log`.
 *
 * O gateway não tem rota de busca na trilha (`audit_admin/recent` só pagina por
 * limite), então os filtros rodam sobre as últimas 200 linhas lidas — a tela
 * mostra o tamanho da janela. A trilha é append-only e a leitura é exclusiva
 * de admin; quem não é admin vê a recusa real do backend.
 */
export default async function AuditoriaPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const params = await searchParams;
  const actor = one(params.ator);
  const action = one(params.acao);
  const target = one(params.alvo);
  const dias = Number(one(params.dias));
  const sinceMs =
    Number.isInteger(dias) && dias > 0 && dias <= 90
      ? Date.now() - dias * 86_400_000
      : undefined;

  const session = await getSession();
  let entries: AuditEntryView[] = [];
  let error: string | null = null;

  try {
    entries = filterAuditTrail(await getAuditTrail(200), {
      ...(actor ? { actor } : {}),
      ...(action ? { action } : {}),
      ...(target ? { target } : {}),
      ...(sinceMs !== undefined ? { sinceMs } : {}),
    });
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "trilha de auditoria indisponível no momento";
  }

  const denied = entries.filter((entry) => entry.action.startsWith("auth.denied:")).length;
  const byAction = new Map<string, number>();
  for (const entry of entries) byAction.set(entry.action, (byAction.get(entry.action) ?? 0) + 1);
  const topActions = [...byAction.entries()].sort((a, b) => b[1] - a[1]).slice(0, 5);

  const exportHref =
    `/api/auditoria/export?` +
    new URLSearchParams({
      ...(actor ? { ator: actor } : {}),
      ...(action ? { acao: action } : {}),
      ...(target ? { alvo: target } : {}),
      ...(sinceMs !== undefined ? { dias: String(dias) } : {}),
    }).toString();

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Governança</p>
          <h1 className="page-head__title">Auditoria</h1>
          <p className="page-head__sub">
            Quem fez o quê, quando e em qual alvo. A trilha é append-only: nada é editado nem
            apagado por aqui.
          </p>
        </div>
        <div className="page-head__actions">
          <Badge tone={session?.role === "admin" ? "warn" : "neutral"}>
            {session?.role === "admin" ? "admin" : "leitura restrita"}
          </Badge>
        </div>
      </div>

      <form className="ui-filters" action="/auditoria" method="get" role="search">
        <label className="login-field">
          <span>Autor (id ou papel)</span>
          <input className="ui-input" name="ator" defaultValue={actor} maxLength={32} />
        </label>
        <label className="login-field">
          <span>Ação</span>
          <input
            className="ui-input"
            name="acao"
            defaultValue={action}
            maxLength={64}
            placeholder="accounts.block, grants.request…"
          />
        </label>
        <label className="login-field">
          <span>Alvo</span>
          <input className="ui-input" name="alvo" defaultValue={target} maxLength={64} />
        </label>
        <label className="login-field">
          <span>Período</span>
          <select className="ui-select" name="dias" defaultValue={one(params.dias) || ""}>
            <option value="">tudo (janela de 200)</option>
            <option value="1">último dia</option>
            <option value="7">últimos 7 dias</option>
            <option value="30">últimos 30 dias</option>
          </select>
        </label>
        <button className="ui-btn ui-btn--primary" type="submit">
          Filtrar
        </button>
        {!error && entries.length > 0 && (
          <a className="ui-btn" href={exportHref} download>
            Exportar CSV
          </a>
        )}
      </form>

      {!error && entries.length > 0 && (
        <div className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">Resumo da janela</h2>
              <p className="card__sub">
                Quantas vezes cada ação aparece no filtro atual.
              </p>
            </div>
            <div className="page-head__actions">
              <Badge tone={denied > 0 ? "warn" : "ok"}>{denied} recusada(s)</Badge>
            </div>
          </div>
          <div className="card__body card__body--flush">
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Ação</th>
                    <th>Linhas</th>
                  </tr>
                </thead>
                <tbody>
                  {topActions.map(([name, count]) => (
                    <tr key={name}>
                      <td title={name}>{actionLabel(name)}</td>
                      <td>{count}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
          <div className="card__foot">
            Recusadas: tentativas sem permissão.
          </div>
        </div>
      )}

      <section className="card">
        <div className="card__head">
          <div>
            <h2 className="card__title">
              <ScrollText size={15} strokeWidth={1.8} aria-hidden="true" />{" "}
              {entries.length} {entries.length === 1 ? "ação" : "ações"}
            </h2>
            <p className="card__sub">
              Últimas 200 ações registradas; use os filtros para recortar.
            </p>
          </div>
          <Badge tone={error ? "warn" : "ok"}>{error ? "indisponível" : "no ar"}</Badge>
        </div>

        {error ? (
          <div className="card__body">
            <div className="empty">
              <ShieldAlert size={22} strokeWidth={1.6} aria-hidden="true" />
              <strong>Trilha indisponível</strong>
              <p>{error}</p>
            </div>
          </div>
        ) : entries.length === 0 ? (
          <div className="card__body">
            <p className="muted small" style={{ margin: 0 }}>
              Nenhuma ação combina com o filtro na janela lida.
            </p>
          </div>
        ) : (
          <div className="card__body card__body--flush">
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Quando</th>
                    <th>Autor</th>
                    <th>Ação</th>
                    <th>Alvo</th>
                    <th>Detalhe</th>
                  </tr>
                </thead>
                <tbody>
                  {entries.map((entry, index) => (
                    <tr key={`${entry.at.toISOString()}-${index}`}>
                      <td style={{ whiteSpace: "nowrap" }}>{fmtWhen(entry.at)}</td>
                      <td>
                        <span className="mono">#{entry.actorId}</span>{" "}
                        <span className="muted small">{entry.actorRole}</span>
                      </td>
                      <td title={entry.action}>{actionLabel(entry.action)}</td>
                      <td className="mono">{entry.target || "—"}</td>
                      <td>{entry.detail || "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
        <div className="card__foot">
          Tentativas sem permissão aparecem como “recusa por permissão”.
        </div>
      </section>
    </>
  );
}
