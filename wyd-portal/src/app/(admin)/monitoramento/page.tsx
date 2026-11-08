import { Stat } from "@/components/Stat";

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

/**
 * Monitoramento da stack, lido do Prometheus (mesmo compose, rede interna).
 *
 * A página é servidor puro: o browser não conhece `prometheus:9090`. Quando o
 * Prometheus está fora, o cartão de aviso explica — e o resto do painel continua
 * funcionando, porque quem monitora não pode depender de o monitorado estar vivo.
 */
export default async function MonitoramentoPage() {
  const snapshot = await getMonitoringSnapshot();

  return (
    <>
      <h1 className="ui-page-title">Monitoramento</h1>
      <p className="ui-page-sub">
        Métricas raspadas do Prometheus da própria stack. Cada canal publica em{" "}
        <code>:9898/metrics</code>; o gateway, em <code>:7600/metrics</code>. Os números são a
        última amostra (15 s de intervalo).
      </p>

      {!snapshot.reachable ? (
        <div className="ui-card" style={{ marginBottom: 20 }}>
          <strong>Prometheus indisponível</strong>
          <p style={{ margin: "6px 0 0", color: "var(--muted)", fontSize: 13 }}>
            {snapshot.error}. Confira <code>docker compose ps wyd-prometheus</code> e{" "}
            <code>PROMETHEUS_URL</code>. Sem ele esta página não inventa número.
          </p>
        </div>
      ) : (
        <>
          <div className="ui-stats">
            <Stat
              label="Canais no ar"
              value={`${snapshot.channels.filter((c) => c.up).length}/${snapshot.channels.length}`}
              hint={
                snapshot.channels.length === 0
                  ? "nenhum alvo wyd-server registrado"
                  : snapshot.channels.map((c) => `canal ${c.canal}`).join(", ")
              }
            />
            <Stat
              label="Handshakes (total)"
              value={fmtInt(snapshot.channels.reduce((acc, c) => acc + c.handshakes, 0))}
              hint="conexões CPSock aceitas desde o boot"
            />
            <Stat
              label="Sessões abertas"
              value={fmtInt(snapshot.channels.reduce((acc, c) => acc + c.sessions, 0))}
              hint="somatório dos canais"
            />
            <Stat
              label="Pedidos de status"
              value={fmtInt(snapshot.status.requests)}
              hint={`coluna Canal · ${fmtInt(snapshot.status.channelsPublished)} canais publicados`}
            />
          </div>

          <div className="ui-stats">
            <Stat
              label="CPU do host"
              value={fmtPercent(snapshot.host.cpuPercent)}
              hint="média de 5 min (node-exporter)"
            />
            <Stat
              label="Memória em uso"
              value={fmtPercent(snapshot.host.memoryPercent)}
              hint="disponível / total do nó"
            />
            <Stat label="Load 1 min" value={fmtInt(snapshot.host.load1)} hint="fila de execução" />
            <Stat
              label="Gateway"
              value={fmtInt(snapshot.gateway.requests)}
              hint={`requisições · uptime ${fmtUptime(snapshot.gateway.uptimeSeconds)}`}
            />
          </div>

          <div className="ui-card" style={{ marginBottom: 20 }}>
            <h2 style={{ fontSize: 14, margin: "0 0 10px" }}>Canais</h2>
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Canal</th>
                    <th>Alvo</th>
                    <th className="ui-align-right">Handshakes</th>
                    <th className="ui-align-right">Sessões</th>
                    <th className="ui-align-right">Memória</th>
                  </tr>
                </thead>
                <tbody>
                  {snapshot.channels.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="ui-table__empty">
                        Nenhum alvo <code>wyd-server</code> em <code>targets/wyd-server.json</code>.
                      </td>
                    </tr>
                  ) : (
                    snapshot.channels.map((channel) => (
                      <tr key={channel.canal}>
                        <td>canal {channel.canal}</td>
                        <td>
                          {channel.up ? (
                            "no ar"
                          ) : (
                            <span style={{ color: "var(--danger, #c0392b)" }}>fora do ar</span>
                          )}
                        </td>
                        <td className="ui-align-right">{fmtInt(channel.handshakes)}</td>
                        <td className="ui-align-right">{fmtInt(channel.sessions)}</td>
                        <td className="ui-align-right">{fmtMb(channel.memoryBytes)}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>

          <div className="ui-card">
            <h2 style={{ fontSize: 14, margin: "0 0 10px" }}>Demais alvos</h2>
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Job</th>
                    <th>Estado</th>
                  </tr>
                </thead>
                <tbody>
                  {snapshot.otherJobs.map((job) => (
                    <tr key={job.job}>
                      <td>{job.job}</td>
                      <td>{job.up ? "no ar" : "fora do ar"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p style={{ margin: "10px 0 0", color: "var(--muted)", fontSize: 12 }}>
              Coletado em {new Date(snapshot.collectedAt).toLocaleString("pt-BR")}. Prometheus fica só
              na rede <code>wyd</code>; o Grafana (dashboards prontos) escuta em{" "}
              <code>127.0.0.1:3001</code> na máquina host.
            </p>
          </div>
        </>
      )}
    </>
  );
}
