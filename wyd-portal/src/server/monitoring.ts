/**
 * Leitura de métricas para a aba de monitoramento (Prometheus no mesmo compose).
 *
 * Duas regras da casa:
 *
 * 1. **server-only**: o browser nunca fala com o Prometheus (que só existe na
 *    rede `wyd`); a página renderiza no servidor e recebe números já prontos.
 * 2. **degrada sozinho**: Prometheus fora do ar devolve `reachable: false` com o
 *    motivo, nunca uma tela de erro — monitoramento quebrado não pode derrubar o
 *    painel que o administrador usa para descobrir que algo quebrou.
 */
import { isStaff } from "./session";
import { getSession } from "./session";

const PROM_URL = (process.env.PROMETHEUS_URL ?? "http://prometheus:9090").replace(/\/+$/, "");
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

export interface MonitoringSnapshot {
  /** O Prometheus respondeu. `false` = números ausentes, com `error` preenchido. */
  reachable: boolean;
  error: string;
  collectedAt: string;
  channels: ChannelHealth[];
  otherJobs: JobHealth[];
  gateway: { requests: number; uptimeSeconds: number | null };
  host: { cpuPercent: number | null; memoryPercent: number | null; load1: number | null };
  status: { requests: number; channelsPublished: number | null };
}

async function query(expr: string): Promise<Sample[]> {
  const url = `${PROM_URL}/api/v1/query?query=${encodeURIComponent(expr)}`;
  const response = await fetch(url, {
    signal: AbortSignal.timeout(TIMEOUT_MS),
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error(`prometheus ${response.status}`);
  }
  const body = (await response.json()) as {
    status?: string;
    data?: { result?: { metric?: Record<string, string>; value?: [number, string] }[] };
  };
  if (body.status !== "success") {
    throw new Error("prometheus recusou a consulta");
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

/**
 * Foto do monitoramento no momento da renderização.
 *
 * Cada família de métrica falha isolada: se só o node-exporter cair, os canais
 * continuam aparecendo e o campo do host fica `null`.
 */
export async function getMonitoringSnapshot(): Promise<MonitoringSnapshot> {
  const session = await getSession();
  const empty: MonitoringSnapshot = {
    reachable: false,
    error: "sessão de staff expirada",
    collectedAt: new Date().toISOString(),
    channels: [],
    otherJobs: [],
    gateway: { requests: 0, uptimeSeconds: null },
    host: { cpuPercent: null, memoryPercent: null, load1: null },
    status: { requests: 0, channelsPublished: null },
  };
  if (!session || !isStaff(session.role)) return empty;

  let up: Sample[];
  try {
    up = await query("up");
  } catch (error) {
    return {
      ...empty,
      error: error instanceof Error ? error.message : "prometheus indisponível",
    };
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
    collectedAt: new Date().toISOString(),
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
