import { ScrollText } from "lucide-react";

import { Badge } from "@/components/Badge";
import { MonitorNav } from "@/components/MonitorNav";
import { Stat } from "@/components/Stat";

import { getMonitoringSnapshot } from "@/server/monitoring";

export const dynamic = "force-dynamic";

export const metadata = { title: "Logs · Monitoramento · WYD2" };

const fmtInt = (value: number | null): string =>
  value === null ? "—" : value.toLocaleString("pt-BR", { maximumFractionDigits: 0 });

/**
 * Logs: volume por serviço nos últimos 15 min (Loki) + exportação filtrada.
 *
 * O pipeline é Docker → Alloy → Loki; a tela não guarda log, só agrega. O texto
 * completo sai por `/api/logs/export` (staff, NDJSON, última 1h, até 1000 linhas).
 */
export default async function LogsMonitorPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const params = await searchParams;
  const pick = (key: string): string => {
    const raw = params[key];
    return (Array.isArray(raw) ? raw[0] : raw ?? "").trim();
  };
  const container = pick("container") || "all";
  const level = pick("level") || "all";
  const q = pick("q");
  const snapshot = await getMonitoringSnapshot();
  const logs = snapshot.logs;
  const errors = logs.services.reduce((acc, row) => acc + row.errors, 0);
  const warnings = logs.services.reduce((acc, row) => acc + row.warnings, 0);

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Operação · monitoramento</p>
          <h1 className="page-head__title">Logs</h1>
          <p className="page-head__sub">Volume, erros e avisos por serviço nos últimos 15 minutos.</p>
        </div>
        <div className="page-head__actions">
          <a
            className="ui-btn ui-btn--secondary"
            href={`/api/logs/export?container=${encodeURIComponent(container)}&level=${encodeURIComponent(level)}&q=${encodeURIComponent(q)}&limit=500`}
          >
            Baixar NDJSON (1h)
          </a>
        </div>
      </div>

      <MonitorNav current="/monitoramento/logs" />

      <form className="ui-filters" action="/monitoramento/logs" method="get" role="search">
        <select className="ui-input" name="container" defaultValue={container} aria-label="Serviço">
          {["all", "wyd-canal-1", "wyd-canal-2", "wyd-gateway", "wyd-db", "wyd-portal", "wyd-loki", "wyd-alloy"].map((c) => (
            <option key={c} value={c}>
              {c === "all" ? "todos os serviços" : c}
            </option>
          ))}
        </select>
        <select className="ui-input" name="level" defaultValue={level} aria-label="Nível">
          <option value="all">todos os níveis</option>
          <option value="error">erros</option>
          <option value="warn">avisos</option>
        </select>
        <input
          className="ui-input"
          type="search"
          name="q"
          defaultValue={q}
          placeholder="texto exato (máx. 64, sem LogQL)"
          aria-label="Texto exato"
        />
        <button className="ui-btn ui-btn--primary" type="submit">
          Aplicar no download
        </button>
      </form>

      <div className="stats">
        <Stat
          label="Linhas em 15 min"
          value={fmtInt(logs.reachable ? logs.totalLines15m : null)}
          pending={!logs.reachable}
          hint="todos os containers"
        />
        <Stat
          label="Erros"
          value={fmtInt(logs.reachable ? errors : null)}
          pending={!logs.reachable}
          hint="level=error, panic, fatal"
        />
        <Stat
          label="Avisos"
          value={fmtInt(logs.reachable ? warnings : null)}
          pending={!logs.reachable}
          hint="level=warn"
        />
        <Stat
          label="Serviços"
          value={fmtInt(logs.reachable ? logs.services.length : null)}
          pending={!logs.reachable}
          hint="containers com linha no período"
        />
      </div>

      <section className="card">
        <div className="card__head">
          <div>
            <h2 className="card__title">
              <ScrollText size={15} strokeWidth={1.8} aria-hidden="true" /> Por serviço
            </h2>
            <p className="card__sub">Linhas recentes de cada serviço.</p>
          </div>
          <Badge tone={logs.reachable ? "ok" : "danger"}>{logs.reachable ? "no ar" : "fora"}</Badge>
        </div>
        <div className="card__body card__body--flush">
          {!logs.reachable ? (
            <p className="ui-field__error" style={{ padding: 20, margin: 0 }}>
              {logs.error}
            </p>
          ) : logs.services.length === 0 ? (
            <p className="muted small" style={{ padding: 20, margin: 0 }}>
              Nenhuma linha nos últimos 15 minutos.
            </p>
          ) : (
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Serviço</th>
                    <th className="ui-align-right">Erros</th>
                    <th className="ui-align-right">Avisos</th>
                  </tr>
                </thead>
                <tbody>
                  {logs.services.map((row) => (
                    <tr key={row.service}>
                      <td>
                        <span className="mono">{row.service}</span>
                      </td>
                      <td className="ui-align-right">
                        <strong style={{ color: row.errors > 0 ? "var(--danger)" : undefined }}>
                          {fmtInt(row.errors)}
                        </strong>
                      </td>
                      <td className="ui-align-right">
                        <strong style={{ color: row.warnings > 0 ? "var(--warn)" : undefined }}>
                          {fmtInt(row.warnings)}
                        </strong>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
        <div className="card__foot">
          Pipeline Docker → Alloy → Loki; retenção definida em{" "}
          <span className="mono">monitoring/loki/loki.yml</span>.
        </div>
      </section>
    </>
  );
}
