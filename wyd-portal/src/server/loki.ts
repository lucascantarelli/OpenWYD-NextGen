/**
 * Logs da stack, lidos do **Loki** (padrão Grafana), nunca do Docker.
 *
 * Caminho: `containers → Alloy (socket read-only) → Loki (volume, 14 d) → BFF do portal`.
 * O stdout do Docker é por container (`json-file`) e morre na recriação — justamente no deploy em
 * que o log mais importa; o Loki guarda o histórico em volume.
 *
 * Este módulo guarda **só o agregado** que o painel de monitoramento mostra: linhas de erro/aviso
 * por container nos últimos 15 min e o volume total. O visualizador linha a linha (`/logs`, SSE
 * `/api/logs/<container>` e `/api/logs/status`) foi removido em 2026-11-09; para ler linha a linha
 * use o Grafana (datasource Loki provisionado, dashboard `wyd-logs`).
 *
 * Regras da casa que continuam valendo:
 *
 * 1. server-only: o browser nunca fala com o Loki (ele só existe na rede `wyd`);
 * 2. as consultas são constantes deste arquivo — nenhum texto do browser entra em LogQL.
 */
export const LOKI_URL = (process.env.LOKI_URL ?? "http://loki:3100").replace(/\/+$/, "");

const TIMEOUT_MS = 4_000;

/**
 * Nome curto de cada container para a UI (`wyd-canal-1` → `canal 1`).
 *
 * As métricas do Loki são agrupadas por `container`: o rótulo `service` é o nome do serviço no
 * compose (`server`, `portal`...) e não identifica o canal.
 */
const LOG_LABELS: Record<string, string> = {
  "wyd-canal-1": "canal 1",
  "wyd-canal-2": "canal 2",
  "wyd-gateway": "gateway",
  "wyd-db": "postgres",
  "wyd-portal": "portal",
  "wyd-prometheus": "prometheus",
  "wyd-grafana": "grafana",
  "wyd-loki": "loki",
  "wyd-alloy": "alloy",
  "wyd-node-exporter": "node-exporter",
};

/** Nome curto de um container (`wyd-canal-1` → `canal 1`); desconhecido volta cru. */
export function logLabel(container: string): string {
  return LOG_LABELS[container] ?? container;
}

interface LokiMetricRow {
  metric?: Record<string, string>;
  value?: [number, string];
}

interface LokiQueryResponse {
  status?: string;
  data?: { resultType?: string; result?: LokiMetricRow[] };
}

async function lokiFetch(path: string): Promise<LokiQueryResponse> {
  const response = await fetch(`${LOKI_URL}${path}`, {
    signal: AbortSignal.timeout(TIMEOUT_MS),
    cache: "no-store",
  });
  if (!response.ok) throw new Error(`logs responderam ${response.status}`);
  const body = (await response.json()) as LokiQueryResponse;
  if (body.status !== "success") throw new Error("consulta de logs recusada");
  return body;
}

function metricSamples(body: LokiQueryResponse): { labels: Record<string, string>; value: number }[] {
  const result = body.data?.result ?? [];
  return result.map((row) => ({
    labels: row.metric ?? {},
    value: Number(row.value?.[1] ?? "0"),
  }));
}

export interface LogStats {
  /** Loki respondeu. `false` = pipeline fora, com motivo. */
  reachable: boolean;
  error: string;
  services: { service: string; errors: number; warnings: number }[];
  totalLines15m: number;
}

/**
 * Foto do pipeline para o monitoramento: por serviço, quantas linhas de
 * erro/aviso nos últimos 15 min. Uma consulta métrica (LogQL sobre range) por
 * família — barato e sem varrer tudo.
 */
export async function getLogStats(): Promise<LogStats> {
  const empty: LogStats = { reachable: false, error: "", services: [], totalLines15m: 0 };
  try {
    const [errors, warnings, volume] = await Promise.all([
      lokiFetch(
        `/loki/api/v1/query?query=${encodeURIComponent('sum by (container) (count_over_time({job="docker"} |~ "(?i)(level=error|panic|fatal)" [15m]))')}`,
      ),
      lokiFetch(
        `/loki/api/v1/query?query=${encodeURIComponent('sum by (container) (count_over_time({job="docker"} |~ "(?i)(level=warn|warn(ing)?)" [15m]))')}`,
      ),
      lokiFetch(`/loki/api/v1/query?query=${encodeURIComponent('sum(count_over_time({job="docker"} [15m]))')}`),
    ]);

    const byService = new Map<string, { errors: number; warnings: number }>();
    const read = (body: LokiQueryResponse, key: "errors" | "warnings") => {
      for (const sample of metricSamples(body)) {
        const service = logLabel(sample.labels.container ?? "?");
        const row = byService.get(service) ?? { errors: 0, warnings: 0 };
        row[key] = sample.value;
        byService.set(service, row);
      }
    };
    read(errors, "errors");
    read(warnings, "warnings");

    const totalLines15m = metricSamples(volume).reduce((acc, sample) => acc + sample.value, 0);
    return {
      reachable: true,
      error: "",
      totalLines15m,
      services: [...byService.entries()]
        .map(([service, row]) => ({ service, errors: row.errors, warnings: row.warnings }))
        .sort((a, b) => b.errors - a.errors || b.warnings - a.warnings || a.service.localeCompare(b.service)),
    };
  } catch (error) {
    console.info("[monitoramento] logs:", error instanceof Error ? error.message : error);
    return { ...empty, error: "logs indisponíveis" };
  }
}
