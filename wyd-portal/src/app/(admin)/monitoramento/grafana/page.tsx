import { BarChart3 } from "lucide-react";

import { Badge } from "@/components/Badge";
import { MonitorNav } from "@/components/MonitorNav";

import { GRAFANA_PUBLIC_URL } from "@/server/grafana";
import { getMonitoringSnapshot } from "@/server/monitoring";

export const dynamic = "force-dynamic";

export const metadata = { title: "Painéis · Monitoramento · WYD2" };

/**
 * Painéis: catálogo do Grafana e embed do dashboard escolhido.
 *
 * O portal não redesenha gráfico: os dashboards são provisionados por arquivo em
 * `monitoring/grafana/dashboards`, então uma métrica nova aparece aqui sem deploy.
 */
export default async function GrafanaMonitorPage({
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
  const health = snapshot.grafana.health;

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Operação · monitoramento</p>
          <h1 className="page-head__title">Painéis</h1>
          <p className="page-head__sub">Dashboards provisionados, com embed do Grafana.</p>
        </div>
      </div>

      <MonitorNav current="/monitoramento/grafana" />

      <section className="card" style={{ marginBottom: 16 }}>
        <div className="card__head">
          <div>
            <h2 className="card__title">
              <BarChart3 size={15} strokeWidth={1.8} aria-hidden="true" /> Grafana
            </h2>
            <p className="card__sub">
              {health.reachable
                ? `versão ${health.version} · banco ${health.database}`
                : "painéis indisponíveis"}
            </p>
          </div>
          <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
            <Badge tone={health.reachable ? "ok" : "danger"}>
              {health.reachable ? "no ar" : "fora"}
            </Badge>
            {health.reachable && (
              <a href={selected?.openUrl ?? GRAFANA_PUBLIC_URL} target="_blank" rel="noreferrer">
                Abrir em nova aba
              </a>
            )}
          </div>
        </div>
        <div className="card__body card__body--flush">
          {!health.reachable ? (
            <p className="ui-field__error" style={{ padding: 20, margin: 0 }}>
              {health.error}
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
                  href={`/monitoramento/grafana?dash=${encodeURIComponent(dashboard.uid)}`}
                  aria-current={dashboard.uid === selected?.uid ? "page" : undefined}
                >
                  {dashboard.title}
                </a>
              ))}
            </div>
          )}
        </div>
      </section>

      {selected && health.reachable && (
        <div className="embed">
          <iframe title={selected.title} src={selected.embedUrl} loading="lazy" />
        </div>
      )}
    </>
  );
}
