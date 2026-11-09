import { AlertTriangle, ScrollText } from "lucide-react";

import { Badge } from "@/components/Badge";
import { Stat } from "@/components/Stat";

import { GRAFANA_PUBLIC_URL } from "@/server/grafana";
import { getMonitoringSnapshot } from "@/server/monitoring";

export const dynamic = "force-dynamic";

export const metadata = { title: "Monitoramento · WYD2" };

const fmtInt = (value: number | null): string =>
  value === null ? "—" : value.toLocaleString("pt-BR", { maximumFractionDigits: 0 });

const fmtPercent = (value: number | null): string =>
  value === null ? "—" : `${value.toLocaleString("pt-BR", { maximumFractionDigits: 1 })}%`;

const fmtMb = (bytes: number | null): string =>
  bytes === null ? "—" : `${(bytes / 1024 / 1024).toLocaleString("pt-BR", { maximumFractionDigits: 1 })} MB`;

const fmtUptime = (seconds: number | null): string => {
  if (seconds === null) return "—";
  const total = Math.floor(seconds);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  return `${h}h ${m}m`;
};

const fmtWhen = (iso: string): string => {
  if (!iso) return "—";
  const when = new Date(iso);
  if (Number.isNaN(when.getTime())) return "—";
  return when.toLocaleString("pt-BR");
};

const severityTone = (severity: string): "danger" | "warn" | "neutral" =>
  severity === "critical" ? "danger" : severity === "warning" ? "warn" : "neutral";

/**
 * Monitoramento: porta de entrada da observabilidade.
 *
 * Prometheus responde pelas métricas e alertas, Loki pelos logs, o gateway pela
 * presença viva no banco e o Grafana renderiza os dashboards provisionados. Cada
 * fonte falha isolada.
 */
export default async function MonitoramentoPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const snapshot = await getMonitoringSnapshot();
  const params = await searchParams;
  const dashParam =
    typeof params.dash === "string"
      ? params.dash
      : Array.isArray(params.dash)
        ? params.dash[0]
        : undefined;
  const dashboards = snapshot.grafana.dashboards;
  const selected = dashboards.find((dashboard) => dashboard.uid === dashParam) ?? dashboards[0];

  const onlineChannels = snapshot.channels.filter((channel) => channel.up).length;
  const presenceTotal = snapshot.presence.reachable ? snapshot.presence.total : null;
  const firing = snapshot.alerts.filter((alert) => alert.state === "firing");
  const errors15m = snapshot.logs.services.reduce((acc, row) => acc + row.errors, 0);
  const warnings15m = snapshot.logs.services.reduce((acc, row) => acc + row.warnings, 0);

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Operação</p>
          <h1 className="page-head__title">Monitoramento</h1>
          <p className="page-head__sub">
            Saúde da stack, presença por canal e logs recentes.
          </p>
        </div>
      </div>

      <div className="monitor-strip" role="list" aria-label="Estado das fontes">
        <span className="status" role="listitem">
          <span className={`dot ${snapshot.reachable ? "dot--ok" : "dot--danger"}`} aria-hidden="true" />
          Métricas <strong>{snapshot.reachable ? "no ar" : "fora"}</strong>
        </span>
        <span className="status" role="listitem">
          <span className={`dot ${snapshot.logs.reachable ? "dot--ok" : "dot--danger"}`} aria-hidden="true" />
          Logs <strong>{snapshot.logs.reachable ? "ingerindo" : "fora"}</strong>
        </span>
        <span className="status" role="listitem">
          <span className={`dot ${snapshot.presence.reachable ? "dot--ok" : "dot--warn"}`} aria-hidden="true" />
          Gateway <strong>{snapshot.presence.reachable ? "respondendo" : "indisponível"}</strong>
        </span>
        <span className="status" role="listitem">
          <span
            className={`dot ${snapshot.grafana.health.reachable ? "dot--ok" : "dot--danger"}`}
            aria-hidden="true"
          />
          Painéis <strong>{snapshot.grafana.health.reachable ? "no ar" : "fora"}</strong>
        </span>
        <span className="status" role="listitem">
          <span
            className={`dot ${!snapshot.reachable ? "dot--warn" : firing.length === 0 ? "dot--ok" : "dot--danger"}`}
            aria-hidden="true"
          />
          Alertas <strong>{snapshot.reachable ? `${firing.length} disparando` : "indisponível"}</strong>
        </span>
      </div>

      {!snapshot.reachable && (
        <div className="banner" style={{ marginBottom: 16 }}>
          <AlertTriangle size={16} strokeWidth={1.8} aria-hidden="true" />
          <span>
            <strong>Métricas indisponíveis.</strong> A presença por canal continua sendo lida do
            servidor.
          </span>
        </div>
      )}

      <div className="stats">
        <Stat label="Jogadores online" value={fmtInt(presenceTotal)} hint="sessões ativas agora" />
        <Stat
          label="Canais no ar"
          value={snapshot.reachable ? `${onlineChannels}/${snapshot.channels.length}` : "—"}
          pending={!snapshot.reachable}
          hint={
            snapshot.reachable
              ? `${fmtInt(snapshot.channels.reduce((acc, c) => acc + c.handshakes, 0))} handshakes`
              : "métricas indisponíveis"
          }
        />
        <Stat
          label="Alertas ativos"
          value={snapshot.reachable ? fmtInt(firing.length) : "—"}
          pending={!snapshot.reachable}
          hint={
            snapshot.reachable
              ? `${snapshot.alerts.length - firing.length} pendentes`
              : "métricas indisponíveis"
          }
        />
        <Stat
          label="Logs em 15 min"
          value={fmtInt(snapshot.logs.reachable ? snapshot.logs.totalLines15m : null)}
          pending={!snapshot.logs.reachable}
          hint={
            snapshot.logs.reachable
              ? `${fmtInt(errors15m)} erros · ${fmtInt(warnings15m)} avisos`
              : "logs indisponíveis"
          }
        />
      </div>

      <div className="stats">
        <Stat label="CPU do host" value={fmtPercent(snapshot.host.cpuPercent)} hint="média de 5 min" />
        <Stat
          label="Memória do host"
          value={fmtPercent(snapshot.host.memoryPercent)}
          hint={`load 1 min: ${fmtInt(snapshot.host.load1)}`}
        />
        <Stat
          label="Requisições ao gateway"
          value={snapshot.reachable ? fmtInt(snapshot.gateway.requests) : "—"}
          pending={!snapshot.reachable}
          hint={snapshot.reachable ? `uptime ${fmtUptime(snapshot.gateway.uptimeSeconds)}` : "métricas indisponíveis"}
        />
        <Stat
          label="Pedidos de status"
          value={snapshot.reachable ? fmtInt(snapshot.status.requests) : "—"}
          pending={!snapshot.reachable}
          hint={
            snapshot.reachable
              ? `${fmtInt(snapshot.status.channelsPublished)} canais publicados`
              : "métricas indisponíveis"
          }
        />
      </div>

      <div className="monitor-grid">
        <section className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">Presença por canal</h2>
              <p className="card__sub">Sessões ativas em cada canal.</p>
            </div>
            <Badge tone={snapshot.presence.reachable ? "ok" : "warn"}>
              {snapshot.presence.reachable ? "banco" : "indisponível"}
            </Badge>
          </div>
          <div className="card__body card__body--flush">
            {!snapshot.presence.reachable ? (
              <p className="ui-field__error" style={{ padding: 20, margin: 0 }}>
                {snapshot.presence.error}
              </p>
            ) : snapshot.presence.channels.length === 0 ? (
              <p className="muted small" style={{ padding: 20, margin: 0 }}>
                Nenhum canal habilitado.
              </p>
            ) : (
              <div className="ui-table-scroll">
                <table className="ui-table">
                  <thead>
                    <tr>
                      <th>Canal</th>
                      <th>Host do cliente</th>
                      <th className="ui-align-right">Online</th>
                      <th className="ui-align-right">Memória</th>
                    </tr>
                  </thead>
                  <tbody>
                    {snapshot.presence.channels.map((channel) => {
                      const metric = snapshot.channels.find(
                        (candidate) => candidate.canal === String(channel.channel),
                      );
                      return (
                        <tr key={channel.channel}>
                          <td>canal {channel.channel}</td>
                          <td>
                            <span className="mono">{channel.host}</span>
                          </td>
                          <td className="ui-align-right">{fmtInt(channel.online)}</td>
                          <td className="ui-align-right">{fmtMb(metric?.memoryBytes ?? null)}</td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>
          <div className="card__foot">
            Cada canal é um processo e enxerga só a própria fatia.
          </div>
        </section>

        <section className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">Alertas</h2>
              <p className="card__sub">Regras avaliadas continuamente.</p>
            </div>
            <Badge tone={!snapshot.reachable ? "neutral" : firing.length === 0 ? "ok" : "danger"}>
              {!snapshot.reachable
                ? "indisponível"
                : firing.length === 0
                  ? "estável"
                  : `${firing.length} ativos`}
            </Badge>
          </div>
          <div className="card__body card__body--flush">
            {snapshot.alerts.length === 0 ? (
              <p className="muted small" style={{ padding: 20, margin: 0 }}>
                {snapshot.reachable
                  ? "Nenhum alerta ativo ou pendente."
                  : "Alertas indisponíveis no momento."}
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
                      <Badge tone={severityTone(alert.severity)}>{alert.severity}</Badge>
                      <span className="muted small">{fmtWhen(alert.activeAt)}</span>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </section>
      </div>

      <section className="card" style={{ marginBottom: 16 }}>
        <div className="card__head">
          <div>
            <h2 className="card__title">
              <ScrollText size={15} strokeWidth={1.8} aria-hidden="true" /> Logs por serviço
            </h2>
            <p className="card__sub">Volume, erros e avisos nos últimos 15 minutos.</p>
          </div>
          <Badge tone={snapshot.logs.reachable ? "ok" : "danger"}>
            {snapshot.logs.reachable ? "no ar" : "fora"}
          </Badge>
        </div>
        <div className="card__body card__body--flush">
          {!snapshot.logs.reachable ? (
            <p className="ui-field__error" style={{ padding: 20, margin: 0 }}>
              {snapshot.logs.error}
            </p>
          ) : snapshot.logs.services.length === 0 ? (
            <p className="muted small" style={{ padding: 20, margin: 0 }}>
              Nenhuma linha nos últimos 15 minutos.
            </p>
          ) : (
            snapshot.logs.services.map((row) => (
              <div className="log-row" key={row.service}>
                <span className="mono">{row.service}</span>
                <span className="log-row__counts">
                  <span>
                    <strong style={{ color: "var(--danger)" }}>{fmtInt(row.errors)}</strong> erros
                  </span>
                  <span>
                    <strong style={{ color: "var(--warn)" }}>{fmtInt(row.warnings)}</strong> avisos
                  </span>
                </span>
              </div>
            ))
          )}
        </div>
      </section>

      <section className="card" style={{ marginBottom: 12 }}>
        <div className="card__head">
          <div>
            <h2 className="card__title">Dashboards</h2>
            <p className="card__sub">Painéis operacionais embutidos nesta tela.</p>
          </div>
          {snapshot.grafana.health.reachable ? (
            <a href={selected?.openUrl ?? GRAFANA_PUBLIC_URL} target="_blank" rel="noreferrer">
              Abrir em nova aba
            </a>
          ) : null}
        </div>
        <div className="card__body card__body--flush">
          {!snapshot.grafana.health.reachable ? (
            <p className="ui-field__error" style={{ padding: 20, margin: 0 }}>
              {snapshot.grafana.health.error}
            </p>
          ) : dashboards.length === 0 ? (
            <p className="muted small" style={{ padding: 20, margin: 0 }}>
              Nenhum dashboard provisionado.
            </p>
          ) : (
            <div className="embed-tabs" role="tablist" aria-label="Dashboards">
              {dashboards.map((dashboard) => (
                <a
                  key={dashboard.uid}
                  className="chip"
                  href={`/monitoramento?dash=${encodeURIComponent(dashboard.uid)}`}
                  aria-current={dashboard.uid === selected?.uid ? "page" : undefined}
                >
                  {dashboard.title}
                </a>
              ))}
            </div>
          )}
        </div>
      </section>

      {selected && snapshot.grafana.health.reachable && (
        <div className="embed">
          <iframe title={selected.title} src={selected.embedUrl} loading="lazy" />
        </div>
      )}
    </>
  );
}
