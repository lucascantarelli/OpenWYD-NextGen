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
import { getDbHealth, type DbHealth } from "./db-health";
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
  /** Ticks por segundo medidos no processo (FASE-03-7). */
  ticksPerSecond: number | null;
  /** P99 do tick em µs (histograma do motor). */
  tickP99Us: number | null;
  /** Entidades vivas no mundo (jogadores + NPCs). */
  entities: number | null;
  /** Jogadores (presença do banco por canal); null sem presença. */
  players: number | null;
  /** Mobs estimados (entidades − jogadores); null sem as duas fontes. */
  mobs: number | null;
  /** Orçamento do tick deste processo em µs (`1e6 / WYD_TICK_HZ`). */
  tickBudgetUs: number;
}

/**
 * Orçamento de fallback (20 Hz) quando o processo ainda não publica o próprio
 * orçamento. O valor real vem de `wyd_tick_budget_microseconds` (`1e6 / WYD_TICK_HZ`).
 */
export const TICK_BUDGET_US = 50_000;

/** Agregado do motor entre canais (pior P99, soma de ticks/entidades). */
export interface EngineHealth {
  ticksPerSecond: number | null;
  tickP99Us: number | null;
  tickBudgetUs: number;
  /** P99 como percentual do orçamento (null sem amostra). */
  tickBudgetPct: number | null;
  entities: number | null;
  /** Jogadores (presença do banco); null sem presença. */
  players: number | null;
  /** Mobs estimados (entidades − jogadores); null sem as duas fontes. */
  mobs: number | null;
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
  engine: EngineHealth;
  host: {
    cpuPercent: number | null;
    memoryPercent: number | null;
    load1: number | null;
    networkRxBps: number | null;
    networkTxBps: number | null;
  };
  status: { requests: number; channelsPublished: number | null };
  alerts: StackAlert[];
  presence: PresenceSnapshotView;
  logs: LogStats;
  grafana: { health: GrafanaHealth; dashboards: GrafanaEmbed[] };
  db: DbHealth;
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

/** Divide entidades em jogadores (banco) e mobs (estimativa); null sem as fontes. */
export function splitPlayersMobs(
  entities: number | null,
  players: number | null,
): { players: number | null; mobs: number | null } {
  if (entities === null || players === null) return { players, mobs: null };
  return { players, mobs: Math.max(entities - players, 0) };
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
  Omit<MonitoringSnapshot, "collectedAt" | "alerts" | "presence" | "logs" | "grafana" | "db">
> {
  const empty = {
    reachable: false,
    error: "métricas indisponíveis",
    channels: [] as ChannelHealth[],
    otherJobs: [] as JobHealth[],
    gateway: { requests: 0, uptimeSeconds: null },
    engine: {
      ticksPerSecond: null,
      tickP99Us: null,
      tickBudgetUs: TICK_BUDGET_US,
      tickBudgetPct: null,
      entities: null,
      players: null,
      mobs: null,
    },
    host: { cpuPercent: null, memoryPercent: null, load1: null, networkRxBps: null, networkTxBps: null },
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
  const channelsPublished = await optional("wyd_server_channels_published");
  const ticks = await optional("sum by (canal) (rate(wyd_ticks_total[1m]))");
  const tickP99 = await optional(
    "histogram_quantile(0.99, sum by (le, canal) (rate(wyd_tick_duration_microseconds_bucket[5m])))",
  );
  const tickBudgets = await optional("wyd_tick_budget_microseconds");
  const entities = await optional("sum by (canal) (wyd_entities_active)");
  const gatewayRequests = await optional('wyd_gateway_requests_total{job="wyd-gateway"}');
  const gatewayUptime = await optional('wyd_gateway_uptime_seconds{job="wyd-gateway"}');
  const cpu = await optional(
    '100 - (avg(rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)',
  );
  const memoryPercent = await optional(
    "100 * (1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)",
  );
  const load1 = await optional('node_load1{job="node"}');
  const networkRx = await optional('sum(rate(node_network_receive_bytes_total{device!~"lo"}[5m]))');
  const networkTx = await optional('sum(rate(node_network_transmit_bytes_total{device!~"lo"}[5m]))');

  const canalOf = (labels: Record<string, string>, fallback: string): string =>
    labels.canal ?? labels.instance ?? fallback;

  /** Orçamento do canal; sem a métrica (imagem antiga), cai no default de 20 Hz. */
  const budgetOf = (labels: Record<string, string>): number => {
    const canal = canalOf(labels, "");
    const found = tickBudgets.find((sample) => canalOf(sample.labels, "") === canal);
    return found && found.value > 0 ? found.value : TICK_BUDGET_US;
  };

  const channels: ChannelHealth[] = upServers
    .map((sample) => {
      const canal = canalOf(sample.labels, sample.labels.target ?? "?");
      const pick = (samples: Sample[]): number =>
        total(samples.filter((s) => canalOf(s.labels, "") === canal));
      const mem = memory.find((s) => canalOf(s.labels, "") === canal);
      const tickRate = ticks.find((s) => canalOf(s.labels, "") === canal);
      const tickP = tickP99.find((s) => canalOf(s.labels, "") === canal);
      const alive = entities.find((s) => canalOf(s.labels, "") === canal);
      return {
        canal,
        up: sample.value === 1,
        handshakes: pick(handshakes),
        sessions: pick(sessions),
        memoryBytes: mem ? mem.value : null,
        ticksPerSecond: tickRate ? tickRate.value : null,
        tickP99Us: tickP ? tickP.value : null,
        entities: alive ? alive.value : null,
        players: null,
        mobs: null,
        tickBudgetUs: budgetOf(sample.labels),
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
    engine: (() => {
      // Pior canal manda: o orçamento é por processo, não a média do grupo.
      const budgetPct = tickP99.map(
        (sample) => (sample.value / budgetOf(sample.labels)) * 100,
      );
      const worst = budgetPct.length > 0 ? Math.max(...budgetPct) : null;
      return {
        ticksPerSecond: ticks.length > 0 ? total(ticks) : null,
        tickP99Us:
          tickP99.length > 0 ? Math.max(...tickP99.map((sample) => sample.value)) : null,
        tickBudgetUs:
          tickP99.length > 0 && channels.length > 0
            ? Math.min(...channels.map((channel) => channel.tickBudgetUs))
            : TICK_BUDGET_US,
        tickBudgetPct: worst,
        entities: entities.length > 0 ? total(entities) : null,
        players: null,
        mobs: null,
      };
    })(),
    host: {
      cpuPercent: first(cpu),
      memoryPercent: first(memoryPercent),
      load1: first(load1),
      networkRxBps: first(networkRx),
      networkTxBps: first(networkTx),
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
      engine: {
        ticksPerSecond: null,
        tickP99Us: null,
        tickBudgetUs: TICK_BUDGET_US,
        tickBudgetPct: null,
        entities: null,
        players: null,
        mobs: null,
      },
      host: { cpuPercent: null, memoryPercent: null, load1: null, networkRxBps: null, networkTxBps: null },
      status: { requests: 0, channelsPublished: null },
      alerts: [],
      presence: { reachable: false, error: "sessão de staff expirada", total: 0, channels: [] },
      logs: { reachable: false, error: "sessão de staff expirada", services: [], totalLines15m: 0 },
      grafana: {
        health: { reachable: false, error: "sessão de staff expirada", version: "", database: "" },
        dashboards: [],
      },
      db: { reachable: false, error: "sessão de staff expirada", latencyMs: null, database: null, schema: null },
    };
  }

  const [prometheus, alerts, presence, logs, grafana, db] = await Promise.all([
    getPrometheusSnapshot(),
    getAlerts(),
    getPresence(),
    getLogStats(),
    getGrafanaEmbeds(),
    getDbHealth(),
  ]);

  const presenceByChannel = new Map(presence.channels.map((row) => [String(row.channel), row.online]));
  const channels = prometheus.channels.map((channel) => {
    const players = presence.reachable ? (presenceByChannel.get(channel.canal) ?? 0) : null;
    const { mobs } = splitPlayersMobs(channel.entities, players);
    return { ...channel, players, mobs };
  });
  const players = presence.reachable ? presence.total : null;
  const { mobs } = splitPlayersMobs(prometheus.engine.entities, players);

  return {
    ...prometheus,
    collectedAt: new Date().toISOString(),
    alerts,
    presence,
    logs,
    grafana,
    db,
    channels,
    engine: { ...prometheus.engine, players, mobs },
  };
}
