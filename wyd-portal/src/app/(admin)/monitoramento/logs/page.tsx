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
 * Logs: volume por serviço nos últimos 15 min (Loki).
 *
 * O pipeline é Docker → Alloy → Loki; a tela não guarda log, só agrega. O texto
 * completo continua no Loki, alcançável pelo link do topo.
 */
export default async function LogsMonitorPage() {
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
      </div>

      <MonitorNav current="/monitoramento/logs" />

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
            <p className="card__sub">Consulta LogQL sobre o rótulo `container`.</p>
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
