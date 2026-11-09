/**
 * Integração com o Grafana (dashboards prontos), sem reinventar painel.
 *
 * Decisão de arquitetura: **não** reimplementamos gráficos de série temporal no
 * portal. Prometheus/Loki guardam os dados, o Grafana rende os dashboards
 * (provisionados por arquivo em `monitoring/grafana/`) e o portal é a porta de
 * entrada: mostra saúde, catálogo de dashboards e embute o painel escolhido em
 * iframe. Assim uma métrica nova aparece no Grafana sem deploy do portal.
 *
 * Server-only: o browser só recebe a URL pública (loopback) do iframe.
 */
const GRAFANA_URL = (process.env.GRAFANA_URL ?? "http://grafana:3000").replace(/\/+$/, "");
/** URL que o BROWSER usa no iframe. Só loopback: o Grafana não é serviço público. */
export const GRAFANA_PUBLIC_URL = (process.env.GRAFANA_PUBLIC_URL ?? "http://127.0.0.1:3001").replace(
  /\/+$/,
  "",
);

const TIMEOUT_MS = 3_000;

export interface GrafanaHealth {
  reachable: boolean;
  error: string;
  version: string;
  database: string;
}

export interface GrafanaDashboard {
  uid: string;
  title: string;
  url: string;
  folder: string;
  tags: string[];
}

export interface GrafanaEmbed {
  uid: string;
  title: string;
  /** URL de embed (kiosk, tema do portal, janela padrão). */
  embedUrl: string;
  /** URL "abrir no Grafana" (mesma dashboard com o chrome completo). */
  openUrl: string;
}

async function grafanaGet<T>(path: string): Promise<T> {
  const response = await fetch(`${GRAFANA_URL}${path}`, {
    signal: AbortSignal.timeout(TIMEOUT_MS),
    cache: "no-store",
    headers: { accept: "application/json" },
  });
  if (!response.ok) throw new Error(`painéis responderam ${response.status}`);
  return (await response.json()) as T;
}

/** `GET /api/health` — o Grafana responde antes de o banco dele estar pronto. */
export async function getGrafanaHealth(): Promise<GrafanaHealth> {
  try {
    const body = await grafanaGet<{ database?: string; version?: string }>("/api/health");
    return {
      reachable: true,
      error: "",
      version: body.version ?? "?",
      database: body.database ?? "?",
    };
  } catch (error) {
    console.info("[monitoramento] painéis:", error instanceof Error ? error.message : error);
    return {
      reachable: false,
      error: "painéis indisponíveis",
      version: "",
      database: "",
    };
  }
}

/** Dashboards provisionados (`/api/search`), somente leitura. */
export async function listGrafanaDashboards(): Promise<GrafanaDashboard[]> {
  try {
    const rows = await grafanaGet<
      { uid?: string; title?: string; url?: string; folderTitle?: string; tags?: string[] }[]
    >("/api/search?type=dash-db&limit=50");
    return rows
      .filter((row) => typeof row.uid === "string" && row.uid.length > 0)
      .map((row) => ({
        uid: row.uid as string,
        title: row.title ?? "sem título",
        url: row.url ?? `/d/${row.uid}`,
        folder: row.folderTitle ?? "",
        tags: row.tags ?? [],
      }));
  } catch {
    return [];
  }
}

/**
 * Query string do embed: kiosk de TV, tema escuro, sem nenhum controle do Grafana.
 *
 * Desde o Grafana 11.3 (dashboards em Scenes) o `kiosk` **não** esconde mais o
 * seletor de tempo nem as variáveis: sem `_dash.hideTimePicker`/`_dash.hideVariables`/
 * `_dash.hideLinks` o iframe desenha a barra do dashboard e a tela parece "outra
 * página" dentro do portal. `kiosk=tv` também remove a barra de navegação do topo.
 */
export function embedUrlFor(uid: string, from = "now-1h"): string {
  const params = new URLSearchParams({
    orgId: "1",
    // `kiosk=1` (não `kiosk=tv`): medido no Grafana 12.2.0 desta stack, só o
    // primeiro esconde menu lateral, navbar e breadcrumb dentro do iframe.
    kiosk: "true",
    theme: "dark",
    from,
    to: "now",
    refresh: "30s",
    "_dash.hideTimePicker": "true",
    "_dash.hideVariables": "true",
    "_dash.hideLinks": "true",
  });
  return `${GRAFANA_PUBLIC_URL}/d/${encodeURIComponent(uid)}?${params}`;
}

/** URL "abrir no Grafana" (sem kiosk). */
export function openUrlFor(uid: string): string {
  return `${GRAFANA_PUBLIC_URL}/d/${encodeURIComponent(uid)}`;
}

/** Junta catálogo + URLs para a página de monitoramento. */
export async function getGrafanaEmbeds(limit = 4): Promise<{
  health: GrafanaHealth;
  dashboards: GrafanaEmbed[];
}> {
  const [health, dashboards] = await Promise.all([getGrafanaHealth(), listGrafanaDashboards()]);
  return {
    health,
    dashboards: dashboards.slice(0, limit).map((dashboard) => ({
      uid: dashboard.uid,
      title: dashboard.title,
      embedUrl: embedUrlFor(dashboard.uid),
      openUrl: openUrlFor(dashboard.uid),
    })),
  };
}
