/**
 * Saúde do banco via gateway (`GET /health` → `SELECT 1` no Postgres).
 *
 * O portal nunca fala SQL (invariante 7): a única fonte é o gateway, que já
 * faz `SELECT 1` no `/health` e devolve `{ ok, database, schema }`. Aqui só
 * medimos latência e degradamos para indisponível sem derrubar a página.
 *
 * Server-only: o browser nunca vê este endereço (loopback na v1).
 */
const GATEWAY_URL = (process.env.WEB_API_ADDR ?? "http://127.0.0.1:7600").replace(/\/+$/, "");

const TIMEOUT_MS = 3_000;

export interface DbHealth {
  reachable: boolean;
  error: string;
  latencyMs: number | null;
  database: string | null;
  schema: number | null;
}

/** `GET /health` do gateway com cronômetro; falha vira objeto neutro. */
export async function getDbHealth(): Promise<DbHealth> {
  const started = Date.now();
  try {
    const response = await fetch(`${GATEWAY_URL}/health`, {
      signal: AbortSignal.timeout(TIMEOUT_MS),
      cache: "no-store",
      headers: { accept: "application/json" },
    });
    const latencyMs = Date.now() - started;
    if (!response.ok) {
      return { reachable: false, error: `saúde respondeu ${response.status}`, latencyMs, database: null, schema: null };
    }
    const body = (await response.json()) as { database?: string; schema?: number };
    return {
      reachable: true,
      error: "",
      latencyMs,
      database: typeof body.database === "string" ? body.database : null,
      schema: typeof body.schema === "number" ? body.schema : null,
    };
  } catch (error) {
    return {
      reachable: false,
      error: "banco indisponível",
      latencyMs: null,
      database: null,
      schema: null,
    };
  }
}
