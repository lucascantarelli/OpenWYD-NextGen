/**
 * Fonte dos logs da stack para o portal (issue 40).
 *
 * Duas implementações atrás da mesma interface:
 *
 * 1. **logpipe** (`LOG_PIPE_URL`, padrão no compose): o sidecar `logpipe` é o
 *    único processo com o socket do Docker — read-only, sem escrita, servindo
 *    `/status` e `/logs/<alvo>`. É o modo do portal containerizado.
 * 2. **Docker CLI do host** (sem `LOG_PIPE_URL`): `docker compose logs -f`, para
 *    quem roda `make portal-dev` na máquina. `spawn` sem `shell`, whitelist
 *    fixa, nenhuma flag vinda do browser.
 *
 * Nos dois casos o portal autentica a sessão antes de repassar; o socket nunca é
 * montado no container do portal (AGENTS.md / issue 40).
 */
import { execFile, spawn, type ChildProcessByStdio } from "node:child_process";
import path from "node:path";
import type { Readable } from "node:stream";
import { promisify } from "node:util";

const run = promisify(execFile);

/** Whitelist única: nada de serviço vindo do cliente sem validação. */
export const SERVICES = [
  "tmserver",
  "tmserver2",
  "dbserver",
  "binserver",
  "webserver",
  "db",
  "portal",
] as const;
export type ServiceName = (typeof SERVICES)[number];

/** Alvo especial da aba "Tudo" — agrega todos os serviços. */
export const ALL = "all";
export type LogTarget = ServiceName | typeof ALL;

const TARGETS: readonly string[] = [...SERVICES, ALL];

export function isLogTarget(value: string): value is LogTarget {
  return TARGETS.includes(value);
}

export function isService(value: string): value is ServiceName {
  return (SERVICES as readonly string[]).includes(value);
}

/** Coletor HTTP (compose). Vazio = modo CLI do host. */
export const LOG_PIPE_URL = (process.env.LOG_PIPE_URL ?? "").replace(/\/+$/, "");

/**
 * Raiz do repositório (onde vive o `docker-compose.yaml`). Só o modo CLI usa
 * isto; `W2PP_COMPOSE_DIR` cobre instalações fora do layout do repo.
 */
export function composeCwd(): string {
  return process.env.W2PP_COMPOSE_DIR ?? path.join(process.cwd(), "..");
}

const ANSI = /\x1b\[[0-9;]*m/g;

/** Segredos que já vazaram para log em algum momento — nunca para a tela. */
const REDACTIONS: readonly [RegExp, string][] = [
  [/postgres(?:ql)?:\/\/\S+/gi, "postgres://***"],
  [/\b(password|passwd|pwd|secret|token)\s*[:=]\s*\S+/gi, "$1=***"],
  [/W2PP_PLAY_CODE_SECRET\s*[:=]\s*\S+/gi, "W2PP_PLAY_CODE_SECRET=***"],
  [/SESSION_SECRET\s*[:=]\s*\S+/gi, "SESSION_SECRET=***"],
];

/** Remove ANSI e mascara DSN/segredo antes de a linha chegar ao browser. */
export function redact(line: string): string {
  let out = line.replace(ANSI, "");
  for (const [pattern, replacement] of REDACTIONS) out = out.replace(pattern, replacement);
  return out;
}

export interface SpawnOptions {
  tail: number;
  /** `10m`, `1h`, `2s` — só o formato é aceito, nunca a string crua do cliente. */
  since?: string;
}

export interface ServiceStatus {
  service: ServiceName;
  state: string;
  health: string;
}

export interface ComposeStatus {
  ok: boolean;
  services: ServiceStatus[];
  error?: string;
}

const CACHE_MS = 5_000;
let cache: { at: number; value: ComposeStatus } | null = null;

function unknownServices(): ServiceStatus[] {
  return SERVICES.map((service) => ({ service, state: "desconhecido", health: "" }));
}

/** `GET /status` do logpipe, já normalizado para a whitelist do portal. */
async function pipeStatus(): Promise<ComposeStatus> {
  const res = await fetch(`${LOG_PIPE_URL}/status`, {
    cache: "no-store",
    signal: AbortSignal.timeout(8_000),
  });
  if (!res.ok) throw new Error(`coletor de logs respondeu ${res.status}`);

  const payload = (await res.json()) as {
    ok?: boolean;
    services?: { service?: string; state?: string; health?: string }[];
    error?: string;
  };

  const byService = new Map<string, ServiceStatus>();
  for (const row of payload.services ?? []) {
    if (!row.service || !isService(row.service)) continue;
    byService.set(row.service, {
      service: row.service,
      state: row.state ?? "unknown",
      health: row.health ?? "",
    });
  }

  return {
    ok: payload.ok !== false,
    services: SERVICES.map(
      (service) => byService.get(service) ?? { service, state: "ausente", health: "" },
    ),
    error: payload.error,
  };
}

/** `docker compose ps --format json` no host, cacheado por 5 s. */
async function cliStatus(): Promise<ComposeStatus> {
  try {
    const { stdout } = await run("docker", ["compose", "ps", "--format", "json"], {
      cwd: composeCwd(),
      timeout: 8000,
      maxBuffer: 4 * 1024 * 1024,
      windowsHide: true,
    });

    const byService = new Map<string, ServiceStatus>();
    for (const line of stdout.split(/\r?\n/)) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      try {
        const row = JSON.parse(trimmed) as { Service?: string; State?: string; Health?: string };
        if (!row.Service || !isService(row.Service)) continue;
        byService.set(row.Service, {
          service: row.Service,
          state: row.State ?? "unknown",
          health: row.Health ?? "",
        });
      } catch {
        // Linha não-JSON não derruba o status inteiro.
      }
    }

    return {
      ok: true,
      services: SERVICES.map(
        (service) => byService.get(service) ?? { service, state: "ausente", health: "" },
      ),
    };
  } catch (error) {
    // Sem Docker CLI no host (portal em container) ou daemon parado: a UI mostra
    // o motivo, sem inventar estado.
    return {
      ok: false,
      services: unknownServices(),
      error:
        (error as { code?: string }).code === "ENOENT"
          ? "o Docker CLI não está disponível neste processo — no container o portal usa o logpipe (LOG_PIPE_URL)"
          : "não foi possível consultar o docker compose",
    };
  }
}

/**
 * Estado dos serviços, cacheado por 5 s — os chips consultam a cada 10 s e não
 * fazem sentido batendo no Docker a cada requisição.
 */
export async function composeStatus(): Promise<ComposeStatus> {
  const now = Date.now();
  if (cache && now - cache.at < CACHE_MS) return cache.value;

  let value: ComposeStatus;
  if (LOG_PIPE_URL) {
    try {
      value = await pipeStatus();
    } catch (error) {
      value = {
        ok: false,
        services: unknownServices(),
        error: `o coletor de logs (logpipe) não respondeu: ${(error as Error).message}`,
      };
    }
  } else {
    value = await cliStatus();
  }

  cache = { at: now, value };
  return value;
}

export interface LogStreamHandle {
  /** Texto já multiplexado (stdout+stderr) na ordem em que o Docker entregou. */
  stream: ReadableStream<string>;
  close(): void;
}

/** Abre o stream de log: logpipe quando configurado, senão o CLI do host. */
export function openContainerLogs(
  target: LogTarget,
  options: SpawnOptions & { signal: AbortSignal },
): LogStreamHandle {
  return LOG_PIPE_URL ? pipeLogs(target, options) : cliLogs(target, options);
}

function pipeLogs(
  target: LogTarget,
  { tail, since, signal }: SpawnOptions & { signal: AbortSignal },
): LogStreamHandle {
  const abortController = new AbortController();
  const abort = () => abortController.abort();
  if (signal.aborted) abort();
  else signal.addEventListener("abort", abort, { once: true });

  const params = new URLSearchParams({ tail: String(tail) });
  if (since) params.set("since", since);

  const stream = new ReadableStream<string>({
    async start(controller) {
      try {
        const res = await fetch(`${LOG_PIPE_URL}/logs/${target}?${params}`, {
          cache: "no-store",
          signal: abortController.signal,
        });
        if (!res.ok || !res.body) {
          controller.enqueue(`[portal] o coletor de logs respondeu ${res.status}`);
          return;
        }
        const reader = res.body.pipeThrough(new TextDecoderStream()).getReader();
        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          if (value) controller.enqueue(value);
        }
      } catch (error) {
        if (!abortController.signal.aborted) {
          controller.enqueue(`[portal] falha no coletor de logs: ${(error as Error).message}`);
        }
      } finally {
        signal.removeEventListener("abort", abort);
        try {
          controller.close();
        } catch {
          // já fechado pelo consumidor
        }
      }
    },
    cancel() {
      abortController.abort();
    },
  });

  return { stream, close: abort };
}

function cliLogs(
  target: LogTarget,
  { tail, since, signal }: SpawnOptions & { signal: AbortSignal },
): LogStreamHandle {
  const args = ["compose", "logs", "-f", "--no-color", "--tail", String(tail)];
  if (since) args.push("--since", since);
  if (target !== ALL) args.push(target);

  const child: ChildProcessByStdio<null, Readable, Readable> = spawn("docker", args, {
    cwd: composeCwd(),
    stdio: ["ignore", "pipe", "pipe"],
    windowsHide: true,
  });

  let closed = false;
  const stop = () => {
    if (closed) return;
    closed = true;
    child.kill("SIGTERM");
    // Sem SIGKILL depois de 2 s, um `docker compose logs` travado ficaria órfão.
    setTimeout(() => {
      if (child.exitCode === null && child.signalCode === null) child.kill("SIGKILL");
    }, 2000).unref?.();
  };

  const stream = new ReadableStream<string>({
    start(controller) {
      const send = (text: string) => {
        if (closed) return;
        try {
          controller.enqueue(text);
        } catch {
          closed = true;
        }
      };

      child.stdout.setEncoding("utf8");
      child.stderr.setEncoding("utf8");
      child.stdout.on("data", (chunk: string) => send(chunk));
      child.stderr.on("data", (chunk: string) => send(chunk));

      child.on("error", (error: NodeJS.ErrnoException) => {
        send(
          `[portal] falha ao executar o docker CLI: ${
            error.code === "ENOENT" ? "docker não encontrado neste processo" : error.message
          }`,
        );
        try {
          controller.close();
        } catch {
          // já fechado
        }
      });

      child.on("close", (code) => {
        send(`[portal] o stream do serviço terminou (exit ${code ?? "?"})`);
        try {
          controller.close();
        } catch {
          // já fechado
        }
      });

      signal.addEventListener("abort", stop, { once: true });
    },
    cancel() {
      stop();
    },
  });

  return { stream, close: stop };
}

/** Só o formato `10m`/`1h`/`2s` passa; qualquer outra coisa é ignorada. */
export function parseSince(raw: string | null): string | undefined {
  if (!raw) return undefined;
  return /^\d{1,4}[smh]$/.test(raw) ? raw : undefined;
}

/** `tail` entre 1 e 2000, default 200. */
export function parseTail(raw: string | null): number {
  const n = Number(raw ?? 200);
  if (!Number.isFinite(n)) return 200;
  return Math.min(Math.max(Math.trunc(n), 1), 2000);
}
