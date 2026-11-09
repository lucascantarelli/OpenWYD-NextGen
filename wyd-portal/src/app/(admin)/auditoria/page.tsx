import { ScrollText, ShieldAlert } from "lucide-react";

import { Badge } from "@/components/Badge";
import { AdminError } from "@/server/errors";
import { getAuditTrail, type AuditEntryView } from "@/server/moderation";
import { getSession } from "@/server/session";

export const dynamic = "force-dynamic";

export const metadata = { title: "Auditoria · WYD2" };

const fmtWhen = (when: Date): string => {
  if (Number.isNaN(when.getTime())) return "—";
  return when.toLocaleString("pt-BR");
};

/**
 * Auditoria: últimas ações de staff, lidas de `audit_log`.
 *
 * A trilha é append-only e gravada pelo gateway a cada chamada autenticada —
 * inclusive as recusas (`auth.denied`) e as escritas de moderação. Leitura
 * exclusiva de admin; quem não é admin vê a recusa real do backend, sem tabela
 * vazia fingindo ausência de registros.
 */
export default async function AuditoriaPage() {
  const session = await getSession();
  let entries: AuditEntryView[] = [];
  let error: string | null = null;

  try {
    entries = await getAuditTrail(100);
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "trilha de auditoria indisponível no momento";
  }

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
          <Badge tone="neutral">{session?.role === "admin" ? "admin" : "leitura restrita"}</Badge>
        </div>
      </div>

      <section className="card">
        <div className="card__head">
          <div>
            <h2 className="card__title">
              <ScrollText size={15} strokeWidth={1.8} aria-hidden="true" /> Últimas 100 ações
            </h2>
            <p className="card__sub">Fonte: tabela audit_log (Postgres), lida pelo gateway.</p>
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
              Nenhuma ação registrada ainda. A primeira chamada admin autenticada já aparece aqui.
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
                      <td className="mono">{entry.action}</td>
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
          Recusas por papel aparecem como <span className="mono">auth.denied:&lt;ação&gt;</span>.
        </div>
      </section>
    </>
  );
}
