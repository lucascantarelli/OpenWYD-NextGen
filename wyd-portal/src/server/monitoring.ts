/**
 * Foto do monitoramento para a área de admin.
 *
 * Uma página, três fontes, cada uma independente:
 *
 * - **Prometheus** (`PROMETHEUS_URL`): métricas do motor, do gateway, do host e os
 *   alertas avaliados pelas regras de `monitoring/prometheus/alerts.yml`;
 * - **Loki** (`LOKI_URL`): volume/erros de log por serviço (o pipeline que substituiu
 *   o sidecar caseiro de logs);
 * - **Gateway** (`server_admin/list_channel_presence`): presença viva por canal, lida
 *   de `character_session` — a única fonte que sobrevive a restart de processo;
 * - **Grafana** (`GRAFANA_URL`): saúde + catálogo de dashboards para embed.
 *
 * Regras: server-only (o browser não fala com Prometheus/Loki), staff obrigatório e
 * degradação por fonte — painel de monitoramento não pode cair junto com o monitorado.
 */
import { adminCall } from "./admin";
import { presenceAdmin } from "./admin-client";
import { getGrafanaEmbeds, type GrafanaEmbed, type GrafanaHealth } from "./grafana";
import { getLogStats, type LogStats } from "./loki";
import { getSession, isStaff } from "./session";

const PROM_URL = (process.env.PROMETHEUS_URL ?? "http://prometheus:9090").replace(/\/+$/, "");
/** URL que o browser usa só para "abrir no Prometheus" (loopback no dev). */
export const PROMETHEUS_PUBLIC_URL = (
  process.env.PROMETHEUS_PUBLIC_URL ?? "http://127.0.0.1:9090"
).replace(/\/+$/, "");
const TIMEOUT_MS = 3_000;

/** Amostra escalar de uma query instantânea (`/api/v1/query`). */
export interface Sample {
  labels: Record<string, string>;
  value: number;
}

export interface ChannelHealth {
  /** `canal` do rótulo do alvo (`1`, `2`…) ou o próprio target. */
  canal: string;
  /** `up` do alvo: 1 = o Prometheus consegue raspar `/metrics`. */
  up: boolean;
  handshakes: number;
  sessions: number;
  memoryBytes: number | null;
}

export interface JobHealth {
  job: string;
  up: boolean;
}

/** Alerta ativo/pendente vindo de `/api/v1/alerts`. */
export interface StackAlert {
  name: string;
  severity: string;
  state: string;
  summary: string;
  activeAt: string;
}

/** Presença viva por canal (banco, não métrica do processo). */
export interface ChannelPresenceView {
  channel: number;
  host: string;
  online: number;
}

export interface PresenceSnapshotView {
  reachable: boolean;
  error: string;
  total: number;
  channels: ChannelPresenceView[];
}

export interface MonitoringSnapshot {
  /** O Prometheus respondeu. `false` = números do motor ausentes, com `error`. */
  reachable: boolean;
  error: string;
  collectedAt: string;
  channels: ChannelHealth[];
  otherJobs: JobHealth[];
  gateway: { requests: number; uptimeSeconds: number | null };
  host: { cpuPercent: number | null; memoryPercent: number | null; load1: number | null };
  status: { requests: number; channelsPublished: number | null };
  alerts: StackAlert[];
  presence: PresenceSnapshotView;
  logs: LogStats;
  grafana: { health: GrafanaHealth; dashboards: GrafanaEmbed[] };
}

async function query(expr: string): Promise<Sample[]> {
  const url = `${PROM_URL}/api/v1/query?query=${encodeURIComponent(expr)}`;
  const response = await fetch(url, {
    signal: AbortSignal.timeout(TIMEOUT_MS),
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error(`métricas responderam ${response.status}`);
  }
  const body = (await response.json()) as {
    status?: string;
    data?: { result?: { metric?: Record<string, string>; value?: [number, string] }[] };
  };
  if (body.status !== "success") {
    throw new Error("consulta de métricas recusada");
  }
  return (body.data?.result ?? []).map((row) => ({
    labels: row.metric ?? {},
    value: Number(row.value?.[1] ?? "0"),
  }));
}

/** Soma todos os elementos do vetor (uma métrica por instância/canal). */
function total(samples: Sample[]): number {
  return samples.reduce((acc, sample) => acc + sample.value, 0);
}

function first(samples: Sample[]): number | null {
  return samples.length > 0 ? samples[0]!.value : null;
}

/** Alertas avaliados pelo Prometheus (firing + pending), mais graves primeiro. */
async function getAlerts(): Promise<StackAlert[]> {
  try {
    const response = await fetch(`${PROM_URL}/api/v1/alerts`, {
      signal: AbortSignal.timeout(TIMEOUT_MS),
      cache: "no-store",
    });
    if (!response.ok) return [];
    const body = (await response.json()) as {
      data?: {
        alerts?: {
          labels?: Record<string, string>;
          annotations?: Record<string, string>;
          state?: string;
          activeAt?: string;
        }[];
      };
    };
    const rank = (severity: string) => (severity === "critical" ? 0 : severity === "warning" ? 1 : 2);
    return (body.data?.alerts ?? [])
      .map((alert) => ({
        name: alert.labels?.alertname ?? "?",
        severity: alert.labels?.severity ?? "info",
        state: alert.state ?? "?",
        summary: alert.annotations?.summary ?? "",
        activeAt: alert.activeAt ?? "",
      }))
      .sort((a, b) => rank(a.severity) - rank(b.severity) || a.name.localeCompare(b.name));
  } catch {
    return [];
  }
}

/**
 * Presença por canal via gateway (leitura auditada).
 *
 * O `adminCall` injeta o `moderator_id` da sessão e audita a chamada; o gateway
 * responde `result` 1..5 como qualquer rota do contrato. Erro aqui não derruba a
 * página: `reachable:false` + motivo.
 */
async function getPresence(): Promise<PresenceSnapshotView> {
  try {
    const res = await adminCall(presenceAdmin.listChannelPresence, {}, { action: "server.presence" });
    const channels = (res.channels ?? []).map((row) => ({
      channel: row.channel,
      host: row.host,
      online: row.online,
    }));
    return {
      reachable: true,
      error: "",
      total: channels.reduce((acc, row) => acc + row.online, 0),
      channels: channels.sort((a, b) => a.channel - b.channel),
    };
  } catch (error) {
    return {
      reachable: false,
      error: error instanceof Error ? error.message : "gateway indisponível",
      total: 0,
      channels: [],
    };
  }
}

/** Métricas do Prometheus (motor, gateway, host e estado dos alvos). */
async function getPrometheusSnapshot(): Promise<
  Omit<MonitoringSnapshot, "collectedAt" | "alerts" | "presence" | "logs" | "grafana">
> {
  const empty = {
    reachable: false,
    error: "métricas indisponíveis",
    channels: [] as ChannelHealth[],
    otherJobs: [] as JobHealth[],
    gateway: { requests: 0, uptimeSeconds: null },
    host: { cpuPercent: null, memoryPercent: null, load1: null },
    status: { requests: 0, channelsPublished: null },
  };

  let up: Sample[];
  try {
    up = await query("up");
  } catch (error) {
    // O motivo técnico fica no log do servidor; a tela recebe texto legível.
    console.info("[monitoramento] métricas:", error instanceof Error ? error.message : error);
    return { ...empty, error: "métricas indisponíveis" };
  }

  /** Lê uma query opcional: falha vira valor neutro, sem derrubar o resto. */
  const optional = async (expr: string): Promise<Sample[]> => {
    try {
      return await query(expr);
    } catch {
      return [];
    }
  };

  const upServers = up.filter((sample) => sample.labels.job === "wyd-server");
  const handshakes = await optional("wyd_handshakes_total");
  const sessions = await optional("wyd_sessions_open");
  const memory = await optional('process_resident_memory_bytes{job="wyd-server"}');
  const statusRequests = await optional("wyd_status_requests_total");
  const channelsPublished = await optional("wyd_channels_published");
  const gatewayRequests = await optional('wyd_gateway_requests_total{job="wyd-gateway"}');
  const gatewayUptime = await optional('wyd_gateway_uptime_seconds{job="wyd-gateway"}');
  const cpu = await optional(
    '100 - (avg(rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)',
  );
  const memoryPercent = await optional(
    "100 * (1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)",
  );
  const load1 = await optional('node_load1{job="node"}');

  const canalOf = (labels: Record<string, string>, fallback: string): string =>
    labels.canal ?? labels.instance ?? fallback;

  const channels: ChannelHealth[] = upServers
    .map((sample) => {
      const canal = canalOf(sample.labels, sample.labels.target ?? "?");
      const pick = (samples: Sample[]): number =>
        total(samples.filter((s) => canalOf(s.labels, "") === canal));
      const mem = memory.find((s) => canalOf(s.labels, "") === canal);
      return {
        canal,
        up: sample.value === 1,
        handshakes: pick(handshakes),
        sessions: pick(sessions),
        memoryBytes: mem ? mem.value : null,
      };
    })
    .sort((a, b) => Number(a.canal) - Number(b.canal));

  return {
    reachable: true,
    error: "",
    channels,
    otherJobs: up
      .filter((sample) => sample.labels.job !== "wyd-server")
      .map((sample) => ({ job: sample.labels.job ?? "?", up: sample.value === 1 }))
      .sort((a, b) => a.job.localeCompare(b.job)),
    gateway: { requests: total(gatewayRequests), uptimeSeconds: first(gatewayUptime) },
    host: {
      cpuPercent: first(cpu),
      memoryPercent: first(memoryPercent),
      load1: first(load1),
    },
    status: {
      requests: total(statusRequests),
      channelsPublished: first(channelsPublished),
    },
  };
}

/**
 * Foto completa no momento da renderização.
 *
 * As quatro fontes falham isoladas: Prometheus fora não apaga presença do banco
 * nem o painel de logs, e vice-versa.
 */
export async function getMonitoringSnapshot(): Promise<MonitoringSnapshot> {
  const session = await getSession();
  if (!session || !isStaff(session.role)) {
    return {
      reachable: false,
      error: "sessão de staff expirada",
      collectedAt: new Date().toISOString(),
      channels: [],
      otherJobs: [],
      gateway: { requests: 0, uptimeSeconds: null },
      host: { cpuPercent: null, memoryPercent: null, load1: null },
      status: { requests: 0, channelsPublished: null },
      alerts: [],
      presence: { reachable: false, error: "sessão de staff expirada", total: 0, channels: [] },
      logs: { reachable: false, error: "sessão de staff expirada", services: [], totalLines15m: 0 },
      grafana: {
        health: { reachable: false, error: "sessão de staff expirada", version: "", database: "" },
        dashboards: [],
      },
    };
  }

  const [prometheus, alerts, presence, logs, grafana] = await Promise.all([
    getPrometheusSnapshot(),
    getAlerts(),
    getPresence(),
    getLogStats(),
    getGrafanaEmbeds(),
  ]);

  return {
    ...prometheus,
    collectedAt: new Date().toISOString(),
    alerts,
    presence,
    logs,
    grafana,
  };
}
