import { AlertTriangle, Gauge } from "lucide-react";

import { Badge } from "@/components/Badge";
import { MonitorNav } from "@/components/MonitorNav";
import { Stat } from "@/components/Stat";

import { PROMETHEUS_PUBLIC_URL, getMonitoringSnapshot } from "@/server/monitoring";

export const dynamic = "force-dynamic";

export const metadata = { title: "Métricas e alertas · Monitoramento · WYD2" };

const fmtInt = (value: number | null): string =>
  value === null ? "—" : value.toLocaleString("pt-BR", { maximumFractionDigits: 0 });

const fmtWhen = (iso: string): string => {
  if (!iso) return "—";
  const when = new Date(iso);
  return Number.isNaN(when.getTime()) ? "—" : when.toLocaleString("pt-BR");
};

/**
 * Métricas e alertas: o que o Prometheus raspa e o que está disparando.
 *
 * As regras vivem em `monitoring/prometheus/alerts.yml`; esta tela só lê o
 * estado (`/api/v1/alerts` e `up`), então regra nova aparece sem deploy do portal.
 */
export default async function PrometheusMonitorPage() {
  const snapshot = await getMonitoringSnapshot();
  const firing = snapshot.alerts.filter((alert) => alert.state === "firing");
  const down = snapshot.channels.filter((channel) => !channel.up);
  const jobsDown = snapshot.otherJobs.filter((job) => !job.up);

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Operação · monitoramento</p>
          <h1 className="page-head__title">Métricas e alertas</h1>
          <p className="page-head__sub">Alvos raspados, saúde do coletor e regras avaliadas.</p>
        </div>
      </div>

      <MonitorNav current="/monitoramento/prometheus" />

      <div className="stats">
        <Stat
          label="Coletor"
          value={snapshot.reachable ? "no ar" : "fora"}
          pending={!snapshot.reachable}
          hint={snapshot.reachable ? "Prometheus respondendo" : snapshot.error}
        />
        <Stat
          label="Canais fora"
          value={snapshot.reachable ? fmtInt(down.length) : "—"}
          pending={!snapshot.reachable}
          hint="scrape do motor falhando"
        />
        <Stat
          label="Alertas disparando"
          value={snapshot.reachable ? fmtInt(firing.length) : "—"}
          pending={!snapshot.reachable}
          hint={`${snapshot.alerts.length} avaliados`}
        />
        <Stat
          label="Outros alvos fora"
          value={snapshot.reachable ? fmtInt(jobsDown.length) : "—"}
          pending={!snapshot.reachable}
          hint="gateway, node, loki…"
        />
      </div>

      {!snapshot.reachable && (
        <div className="banner" style={{ marginBottom: 16 }}>
          <AlertTriangle size={16} strokeWidth={1.8} aria-hidden="true" />
          <span>
            <strong>Coletor fora.</strong> As regras e alvos abaixo ficam indisponíveis até o
            Prometheus voltar.
          </span>
        </div>
      )}

      <div className="monitor-grid">
        <section className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">
                <Gauge size={15} strokeWidth={1.8} aria-hidden="true" /> Alvos
              </h2>
              <p className="card__sub">`up` por job — 1 = scrape respondendo.</p>
            </div>
            <a href={PROMETHEUS_PUBLIC_URL} target="_blank" rel="noreferrer">
              Abrir Prometheus
            </a>
          </div>
          <div className="card__body card__body--flush">
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Job</th>
                    <th className="ui-align-right">Estado</th>
                  </tr>
                </thead>
                <tbody>
                  {snapshot.channels.map((channel) => (
                    <tr key={`server-${channel.canal}`}>
                      <td>
                        wyd-server · canal {channel.canal}
                      </td>
                      <td className="ui-align-right">
                        <Badge tone={channel.up ? "ok" : "danger"}>
                          {channel.up ? "up" : "down"}
                        </Badge>
                      </td>
                    </tr>
                  ))}
                  {snapshot.otherJobs.map((job) => (
                    <tr key={job.job}>
                      <td>{job.job}</td>
                      <td className="ui-align-right">
                        <Badge tone={job.up ? "ok" : "danger"}>{job.up ? "up" : "down"}</Badge>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </section>

        <section className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">Alertas avaliados</h2>
              <p className="card__sub">Disparando e pendentes, mais graves primeiro.</p>
            </div>
            <Badge tone={firing.length === 0 ? "ok" : "danger"}>
              {snapshot.reachable ? `${firing.length} disparando` : "indisponível"}
            </Badge>
          </div>
          <div className="card__body card__body--flush">
            {snapshot.alerts.length === 0 ? (
              <p className="muted small" style={{ padding: 20, margin: 0 }}>
                {snapshot.reachable ? "Nenhum alerta ativo ou pendente." : "Alertas indisponíveis."}
              </p>
            ) : (
              <ul className="alert-list">
                {snapshot.alerts.map((alert) => (
                  <li
                    key={`${alert.name}-${alert.activeAt}`}
                    className={
                      alert.severity === "critical"
                        ? "alert alert--critical"
                        : alert.state === "firing"
                          ? "alert"
                          : "alert alert--ok"
                    }
                  >
                    <span className="alert__bar" aria-hidden="true" />
                    <div>
                      <p className="alert__name">{alert.name}</p>
                      <p className="alert__summary">{alert.summary || "sem resumo"}</p>
                    </div>
                    <div style={{ display: "grid", gap: 4, justifyItems: "end" }}>
                      <Badge tone={alert.severity === "critical" ? "danger" : "warn"}>
                        {alert.severity}
                      </Badge>
                      <span className="muted small">{fmtWhen(alert.activeAt)}</span>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
          <div className="card__foot">
            Regras versionadas em <span className="mono">monitoring/prometheus/alerts.yml</span>.
          </div>
        </section>
      </div>
    </>
  );
}
