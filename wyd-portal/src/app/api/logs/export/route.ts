import { LOKI_URL, logLabel } from "@/server/loki";
import { getSession, isStaff } from "@/server/session";

export const dynamic = "force-dynamic";

const NO_STORE = { "cache-control": "no-store" } as const;
const TIMEOUT_MS = 10_000;

/** Containers que o portal permite exportar (`all` = todos). */
const CONTAINERS = new Set([
  "all",
  "wyd-canal-1",
  "wyd-canal-2",
  "wyd-gateway",
  "wyd-db",
  "wyd-portal",
  "wyd-prometheus",
  "wyd-grafana",
  "wyd-loki",
  "wyd-alloy",
  "wyd-node-exporter",
]);

type Level = "all" | "error" | "warn";

function levelFilter(level: Level): string {
  if (level === "error") return ` |~ "(?i)(level=error|panic|fatal)"`;
  if (level === "warn") return ` |~ "(?i)(level=warn|warn(ing?))"`;
  return "";
}

/** Texto livre vira match literal; qualquer metacaractere LogQL recusa. */
function cleanQuery(raw: string | null): string | null {
  if (!raw) return null;
  const q = raw.trim().slice(0, 64);
  if (q === "") return null;
  if (/["`{}|\\]/.test(q)) return null;
  return q;
}

/**
 * Exporta logs do Loki em NDJSON (última 1h, até `limit` linhas).
 *
 * Server-only + staff: o browser nunca fala com o Loki; os filtros são
 * allowlist (container, nível) e o texto livre só entra como match literal.
 */
export async function GET(req: Request) {
  const session = await getSession();
  if (!session) {
    return Response.json({ error: "sessão expirada: entre novamente" }, { status: 401, headers: NO_STORE });
  }
  if (!isStaff(session.role)) {
    return Response.json({ error: "sua conta não tem permissão para esta ação" }, { status: 403, headers: NO_STORE });
  }

  const url = new URL(req.url);
  const container = (url.searchParams.get("container") ?? "all").trim();
  const levelRaw = (url.searchParams.get("level") ?? "all").trim().toLowerCase();
  const limitRaw = Number(url.searchParams.get("limit") ?? "500");
  if (!CONTAINERS.has(container)) {
    return Response.json({ error: "container inválido" }, { status: 422, headers: NO_STORE });
  }
  if (levelRaw !== "all" && levelRaw !== "error" && levelRaw !== "warn") {
    return Response.json({ error: "nível inválido (all, error, warn)" }, { status: 422, headers: NO_STORE });
  }
  if (!Number.isInteger(limitRaw) || limitRaw < 1 || limitRaw > 1000) {
    return Response.json({ error: "limit deve ser 1..1000" }, { status: 422, headers: NO_STORE });
  }
  const q = cleanQuery(url.searchParams.get("q"));
  if (url.searchParams.get("q") && !q) {
    return Response.json({ error: "texto com caractere inválido para LogQL" }, { status: 422, headers: NO_STORE });
  }

  const level: Level = levelRaw as Level;
  const selector = container === "all" ? `{job="docker"}` : `{job="docker",container="${container}"}`;
  const query = `${selector}${levelFilter(level)}${q ? ` |= ${JSON.stringify(q)}` : ""}`;
  const endNs = String(Date.now() * 1_000_000);
  const startNs = String(Date.now() * 1_000_000 - 3_600_000_000_000);
  const path =
    `/loki/api/v1/query_range?query=${encodeURIComponent(query)}` +
    `&start=${startNs}&end=${endNs}&limit=${limitRaw}&direction=backward`;

  let body: unknown;
  try {
    const response = await fetch(`${LOKI_URL}${path}`, {
      signal: AbortSignal.timeout(TIMEOUT_MS),
      cache: "no-store",
    });
    if (!response.ok) throw new Error(`logs responderam ${response.status}`);
    body = await response.json();
  } catch {
    return Response.json({ error: "logs indisponíveis" }, { status: 502, headers: NO_STORE });
  }

  const streams = (body as { data?: { result?: { stream?: Record<string, string>; values?: [string, string][] }[] } })
    .data?.result ?? [];
  const lines: string[] = [];
  for (const stream of streams) {
    const name = logLabel(stream.stream?.container ?? "?");
    for (const [ts, text] of stream.values ?? []) {
      lines.push(JSON.stringify({ service: name, ts, text }));
    }
  }

  const stamp = new Date().toISOString().replace(/[:.]/g, "-");
  return new Response(`${lines.join("\n")}${lines.length > 0 ? "\n" : ""}`, {
    headers: {
      ...NO_STORE,
      "content-type": "application/x-ndjson; charset=utf-8",
      "content-disposition": `attachment; filename="logs-${stamp}.ndjson"`,
    },
  });
}
