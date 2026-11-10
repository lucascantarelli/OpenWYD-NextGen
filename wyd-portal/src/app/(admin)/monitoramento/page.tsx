import { AlertTriangle, BarChart3, Gauge, ScrollText, Server } from "lucide-react";
import Link from "next/link";

import { Badge } from "@/components/Badge";
import { MonitorNav } from "@/components/MonitorNav";
import { Stat } from "@/components/Stat";

import { getMonitoringSnapshot } from "@/server/monitoring";

export const dynamic = "force-dynamic";

export const metadata = { title: "Monitoramento · WYD2" };

const fmtInt = (value: number | null): string =>
  value === null ? "—" : value.toLocaleString("pt-BR", { maximumFractionDigits: 0 });

const fmtMicros = (value: number | null): string =>
  value === null ? "—" : `${value.toLocaleString("pt-BR", { maximumFractionDigits: 0 })} µs`;

const SECTIONS = [
  {
    href: "/monitoramento/servidor",
    title: "Servidor de jogo",
    text: "Tick, P99 do motor, entidades e presença por canal.",
    icon: Server,
  },
  {
    href: "/monitoramento/grafana",
    title: "Painéis",
    text: "Dashboards provisionados e embed do Grafana.",
    icon: BarChart3,
  },
  {
    href: "/monitoramento/prometheus",
    title: "Métricas e alertas",
    text: "Alvos raspados, regras e alertas em disparo.",
    icon: Gauge,
  },
  {
    href: "/monitoramento/logs",
    title: "Logs",
    text: "Volume, erros e avisos por serviço via Loki.",
    icon: ScrollText,
  },
] as const;

/**
 * Monitoramento: resumo operacional.
 *
 * Dados enxutos de propósito — cada fonte tem a sua subpágina com o detalhe.
 * As fontes falham isoladas: Prometheus fora não apaga a presença do banco.
 */
export default async function MonitoramentoPage() {
  const snapshot = await getMonitoringSnapshot();

  const onlineChannels = snapshot.channels.filter((channel) => channel.up).length;
  const presenceTotal = snapshot.presence.reachable ? snapshot.presence.total : null;
  const firing = snapshot.alerts.filter((alert) => alert.state === "firing");
  const errors15m = snapshot.logs.services.reduce((acc, row) => acc + row.errors, 0);

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Operação</p>
          <h1 className="page-head__title">Monitoramento</h1>
          <p className="page-head__sub">
            Resumo da stack; cada seção aprofunda uma fonte.
          </p>
        </div>
      </div>

      <MonitorNav current="/monitoramento" />

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
          Servidor <strong>{snapshot.presence.reachable ? "respondendo" : "indisponível"}</strong>
        </span>
        <span className="status" role="listitem">
          <span
            className={`dot ${snapshot.grafana.health.reachable ? "dot--ok" : "dot--danger"}`}
            aria-hidden="true"
          />
          Painéis <strong>{snapshot.grafana.health.reachable ? "no ar" : "fora"}</strong>
        </span>
      </div>

      {!snapshot.reachable && (
        <div className="banner" style={{ marginBottom: 16 }}>
          <AlertTriangle size={16} strokeWidth={1.8} aria-hidden="true" />
          <span>
            <strong>Métricas indisponíveis.</strong> Presença e logs continuam sendo lidos das
            respectivas fontes.
          </span>
        </div>
      )}

      <div className="stats">
        <Stat label="Jogadores online" value={fmtInt(presenceTotal)} hint="sessões ativas agora" />
        <Stat
          label="Canais no ar"
          value={snapshot.reachable ? `${onlineChannels}/${snapshot.channels.length}` : "—"}
          pending={!snapshot.reachable}
          hint={snapshot.reachable ? `${fmtInt(snapshot.engine.ticksPerSecond)} ticks/s` : "métricas indisponíveis"}
        />
        <Stat
          label="Tick P99"
          value={fmtMicros(snapshot.engine.tickP99Us)}
          pending={!snapshot.reachable}
          hint={
            snapshot.engine.tickBudgetPct === null
              ? "sem amostra do motor"
              : `${snapshot.engine.tickBudgetPct.toLocaleString("pt-BR", { maximumFractionDigits: 1 })}% do orçamento de ${fmtMicros(snapshot.engine.tickBudgetUs)}`
          }
        />
        <Stat
          label="Entidades no mundo"
          value={fmtInt(snapshot.engine.entities)}
          pending={!snapshot.reachable}
          hint="jogadores + NPCs"
        />
      </div>

      <div className="monitor-grid">
        <section className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">Motor por canal</h2>
              <p className="card__sub">Ticks, latência e entidades de cada processo.</p>
            </div>
            <Badge tone={snapshot.reachable ? "ok" : "neutral"}>
              {snapshot.reachable ? `${snapshot.channels.length} alvos` : "indisponível"}
            </Badge>
          </div>
          <div className="card__body card__body--flush">
            {snapshot.channels.length === 0 ? (
              <p className="muted small" style={{ padding: 20, margin: 0 }}>
                {snapshot.reachable ? "Nenhum canal raspado." : "Métricas indisponíveis."}
              </p>
            ) : (
              <div className="ui-table-scroll">
                <table className="ui-table">
                  <thead>
                    <tr>
                      <th>Canal</th>
                      <th className="ui-align-right">Ticks/s</th>
                      <th className="ui-align-right">P99</th>
                      <th className="ui-align-right">Entidades</th>
                    </tr>
                  </thead>
                  <tbody>
                    {snapshot.channels.map((channel) => (
                      <tr key={channel.canal}>
                        <td>
                          canal {channel.canal}{" "}
                          <Badge tone={channel.up ? "ok" : "danger"}>
                            {channel.up ? "no ar" : "fora"}
                          </Badge>
                        </td>
                        <td className="ui-align-right">{fmtInt(channel.ticksPerSecond)}</td>
                        <td className="ui-align-right">{fmtMicros(channel.tickP99Us)}</td>
                        <td className="ui-align-right">{fmtInt(channel.entities)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
          <div className="card__foot">
            <Link href="/monitoramento/servidor">Ver detalhe do servidor</Link>
          </div>
        </section>

        <section className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">Alertas</h2>
              <p className="card__sub">Regras avaliadas pelo Prometheus.</p>
            </div>
            <Badge tone={!snapshot.reachable ? "neutral" : firing.length === 0 ? "ok" : "danger"}>
              {!snapshot.reachable ? "indisponível" : firing.length === 0 ? "estável" : `${firing.length} ativos`}
            </Badge>
          </div>
          <div className="card__body card__body--flush">
            {snapshot.alerts.length === 0 ? (
              <p className="muted small" style={{ padding: 20, margin: 0 }}>
                {snapshot.reachable ? "Nenhum alerta ativo." : "Alertas indisponíveis."}
              </p>
            ) : (
              <ul className="alert-list">
                {snapshot.alerts.slice(0, 4).map((alert) => (
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
                    <Badge tone={alert.severity === "critical" ? "danger" : "warn"}>
                      {alert.severity}
                    </Badge>
                  </li>
                ))}
              </ul>
            )}
          </div>
          <div className="card__foot">
            <Link href="/monitoramento/prometheus">Todas as regras e alvos</Link>
          </div>
        </section>
      </div>

      <div className="stats" style={{ marginTop: 16 }}>
        <Stat
          label="Logs em 15 min"
          value={fmtInt(snapshot.logs.reachable ? snapshot.logs.totalLines15m : null)}
          pending={!snapshot.logs.reachable}
          hint={`${fmtInt(errors15m)} erros`}
        />
      </div>

      <div className="monitor-grid">
        {SECTIONS.map((section) => {
          const Icon = section.icon;
          return (
            <Link key={section.href} href={section.href} className="card" style={{ textDecoration: "none" }}>
              <div className="card__head">
                <div>
                  <h2 className="card__title">
                    <Icon size={15} strokeWidth={1.8} aria-hidden="true" /> {section.title}
                  </h2>
                  <p className="card__sub">{section.text}</p>
                </div>
              </div>
            </Link>
          );
        })}
      </div>
    </>
  );
}
