import { Server } from "lucide-react";

import { Badge } from "@/components/Badge";
import { MonitorNav } from "@/components/MonitorNav";
import { Stat } from "@/components/Stat";

import { getMonitoringSnapshot } from "@/server/monitoring";

export const dynamic = "force-dynamic";

export const metadata = { title: "Servidor de jogo · Monitoramento · WYD2" };

const fmtInt = (value: number | null): string =>
  value === null ? "—" : value.toLocaleString("pt-BR", { maximumFractionDigits: 0 });

const fmtMicros = (value: number | null): string =>
  value === null ? "—" : `${value.toLocaleString("pt-BR", { maximumFractionDigits: 0 })} µs`;

const fmtMb = (bytes: number | null): string =>
  bytes === null ? "—" : `${(bytes / 1024 / 1024).toLocaleString("pt-BR", { maximumFractionDigits: 1 })} MB`;

const fmtPercent = (value: number | null): string =>
  value === null ? "—" : `${value.toLocaleString("pt-BR", { maximumFractionDigits: 1 })}%`;

const fmtBps = (value: number | null): string => {
  if (value === null) return "—";
  if (value >= 1_000_000) return `${(value / 1_000_000).toLocaleString("pt-BR", { maximumFractionDigits: 1 })} MB/s`;
  if (value >= 1_000) return `${(value / 1_000).toLocaleString("pt-BR", { maximumFractionDigits: 1 })} KB/s`;
  return `${value.toLocaleString("pt-BR", { maximumFractionDigits: 0 })} B/s`;
};

/**
 * Servidor de jogo: a tela que só existe por causa do motor.
 *
 * Um cartão por canal (processo) e o agregado do grupo. O orçamento do tick vem do
 * próprio processo (`wyd_tick_budget_microseconds = 1e6 / WYD_TICK_HZ`: 50 ms a
 * 20 Hz, 20 ms a 50 Hz); o P99 vem do histograma real
 * (`wyd_tick_duration_microseconds`), não de estimativa. Sem métrica, o campo fica
 * "—" em vez de zero.
 */
export default async function ServidorMonitorPage() {
  const snapshot = await getMonitoringSnapshot();
  const engine = snapshot.engine;
  const tickAlerts = snapshot.alerts.filter((alert) => alert.name.startsWith("WydTick") || alert.name === "WydMotorSemTick");

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Operação · monitoramento</p>
          <h1 className="page-head__title">Servidor de jogo</h1>
          <p className="page-head__sub">
            Tick, entidades e presença de cada canal — números medidos no processo.
          </p>
        </div>
      </div>

      <MonitorNav current="/monitoramento/servidor" />

      <div className="stats">
        <Stat
          label="Ticks por segundo"
          value={fmtInt(engine.ticksPerSecond)}
          pending={!snapshot.reachable}
          hint="soma dos canais"
        />
        <Stat
          label="Tick P99"
          value={fmtMicros(engine.tickP99Us)}
          pending={!snapshot.reachable}
          hint={`orçamento ${fmtMicros(engine.tickBudgetUs)}`}
        />
        <Stat
          label="Orçamento usado"
          value={fmtPercent(engine.tickBudgetPct)}
          pending={!snapshot.reachable}
          hint="P99 ÷ orçamento do processo"
        />
        <Stat
          label="Entidades"
          value={fmtInt(engine.entities)}
          pending={!snapshot.reachable}
          hint={
            engine.players !== null && engine.mobs !== null
              ? `${fmtInt(engine.players)} jogadores · ${fmtInt(engine.mobs)} mobs (est.)`
              : "jogadores + NPCs"
          }
        />
        <Stat
          label="Banco"
          value={snapshot.db.reachable ? `${snapshot.db.latencyMs ?? "—"} ms` : "fora"}
          pending={!snapshot.db.reachable}
          hint={
            snapshot.db.reachable
              ? `database=${snapshot.db.database ?? "?"} schema=${snapshot.db.schema ?? "?"}`
              : snapshot.db.error
          }
        />
        <Stat label="Rede RX" value={fmtBps(snapshot.host.networkRxBps)} pending={!snapshot.reachable} hint="node_exporter, sem lo" />
        <Stat label="Rede TX" value={fmtBps(snapshot.host.networkTxBps)} pending={!snapshot.reachable} hint="node_exporter, sem lo" />
      </div>

      <section className="card" style={{ marginBottom: 16 }}>
        <div className="card__head">
          <div>
            <h2 className="card__title">
              <Server size={15} strokeWidth={1.8} aria-hidden="true" /> Canais
            </h2>
            <p className="card__sub">Um processo por canal; cada um tem o próprio motor e métricas.</p>
          </div>
          <Badge tone={snapshot.reachable ? "ok" : "neutral"}>
            {snapshot.reachable ? `${snapshot.channels.length} canais` : "métricas indisponíveis"}
          </Badge>
        </div>
        <div className="card__body card__body--flush">
          {snapshot.channels.length === 0 ? (
            <p className="muted small" style={{ padding: 20, margin: 0 }}>
              {snapshot.reachable ? "Nenhum canal raspado pelo Prometheus." : "Métricas indisponíveis."}
            </p>
          ) : (
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Canal</th>
                    <th>Estado</th>
                    <th className="ui-align-right">Handshakes</th>
                    <th className="ui-align-right">Sessões</th>
                    <th className="ui-align-right">Ticks/s</th>
                    <th className="ui-align-right">P99</th>
                    <th className="ui-align-right">Entidades</th>
                    <th className="ui-align-right">Jogadores</th>
                    <th className="ui-align-right">Mobs (est.)</th>
                    <th className="ui-align-right">RSS</th>
                  </tr>
                </thead>
                <tbody>
                  {snapshot.channels.map((channel) => (
                    <tr key={channel.canal}>
                      <td>canal {channel.canal}</td>
                      <td>
                        <Badge tone={channel.up ? "ok" : "danger"}>{channel.up ? "no ar" : "fora"}</Badge>
                      </td>
                      <td className="ui-align-right">{fmtInt(channel.handshakes)}</td>
                      <td className="ui-align-right">{fmtInt(channel.sessions)}</td>
                      <td className="ui-align-right">{fmtInt(channel.ticksPerSecond)}</td>
                      <td className="ui-align-right">{fmtMicros(channel.tickP99Us)}</td>
                      <td className="ui-align-right">{fmtInt(channel.entities)}</td>
                      <td className="ui-align-right">{fmtInt(channel.players)}</td>
                      <td className="ui-align-right">{fmtInt(channel.mobs)}</td>
                      <td className="ui-align-right">{fmtMb(channel.memoryBytes)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
        <div className="card__foot">
          Presença lida de <span className="mono">character_session</span>; o resto vem de{" "}
          <span className="mono">/metrics</span>.
        </div>
      </section>

      <div className="monitor-grid">
        <section className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">Presença por canal</h2>
              <p className="card__sub">Sessões vivas no banco (sobrevive a restart).</p>
            </div>
            <Badge tone={snapshot.presence.reachable ? "ok" : "warn"}>
              {snapshot.presence.reachable ? "servidor" : "indisponível"}
            </Badge>
          </div>
          <div className="card__body card__body--flush">
            {!snapshot.presence.reachable ? (
              <p className="ui-field__error" style={{ padding: 20, margin: 0 }}>
                {snapshot.presence.error}
              </p>
            ) : (
              <div className="ui-table-scroll">
                <table className="ui-table">
                  <thead>
                    <tr>
                      <th>Canal</th>
                      <th>Host do cliente</th>
                      <th className="ui-align-right">Online</th>
                    </tr>
                  </thead>
                  <tbody>
                    {snapshot.presence.channels.map((channel) => (
                      <tr key={channel.channel}>
                        <td>canal {channel.channel}</td>
                        <td>
                          <span className="mono">{channel.host}</span>
                        </td>
                        <td className="ui-align-right">{fmtInt(channel.online)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </section>

        <section className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">Alertas do motor</h2>
              <p className="card__sub">Tick acima do orçamento, motor parado, memória.</p>
            </div>
            <Badge tone={tickAlerts.some((a) => a.state === "firing") ? "danger" : "ok"}>
              {tickAlerts.filter((a) => a.state === "firing").length} disparando
            </Badge>
          </div>
          <div className="card__body card__body--flush">
            {tickAlerts.length === 0 ? (
              <p className="muted small" style={{ padding: 20, margin: 0 }}>
                Nenhum alerta do motor ativo.
              </p>
            ) : (
              <ul className="alert-list">
                {tickAlerts.map((alert) => (
                  <li
                    key={`${alert.name}-${alert.activeAt}`}
                    className={alert.state === "firing" ? "alert" : "alert alert--ok"}
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
        </section>
      </div>
    </>
  );
}
