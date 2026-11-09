/**
 * Cliente do control plane `wyd-admin` — **HTTP/1.1 + JSON em loopback**
 * (ADR-0008).
 *
 * Só declara rotas que existem no contrato do gateway
 * (`crates/wyd-admin/src/transport/mod.rs`). Rota fora daqui não é "planejada":
 * é ausência de contrato, e o portal não inventa chamada.
 *
 * Regras:
 *
 * 1. server-only: nenhum componente client importa daqui — o browser nunca vê o
 *    token de staff nem fala com o backend.
 * 2. o corpo enviado **não** carrega `moderator_id`: o autor vem da sessão no
 *    servidor Rust, nunca do browser.
 * 3. toda chamada leva `x-trace-id` e deadline de 10 s.
 * 4. recusa de negócio viaja em `200` com `result` 1..5 (`AdminResult`); só
 *    falha de transporte vira exceção (`AdminError`/`AdminUnavailableError`).
 */
import { AdminError, AdminUnavailableError } from "./errors";
import { promisify, type Unary } from "./promisify";
import { newTraceId, traceHeaders } from "./trace";

/** Endereço do `wyd-admin` (loopback por padrão; ADR-0008). */
const ADMIN_URL = (process.env.WEB_API_ADDR ?? "http://127.0.0.1:7600").replace(/\/+$/, "");

/** Token de staff processo→processo. Sem ele o backend responde 401. */
const ADMIN_TOKEN = process.env.WEB_API_TOKEN ?? "";

const DEFAULT_TIMEOUT_MS = 10_000;

/** Resposta do envelope: `result` sempre presente (1 = OK). */
interface AdminEnvelope {
  result: number;
  message?: string;
  trace_id?: string;
}

/** `camelCase` do método → `snake_case` da rota (ADR-0008 §1). */
function toSnake(method: string): string {
  return method.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`);
}

function withoutModerator(params: unknown): unknown {
  if (typeof params !== "object" || params === null) return params ?? {};
  const { moderator_id: _snake, moderatorId: _camel, ...rest } = params as Record<string, unknown>;
  void _snake;
  void _camel;
  return rest;
}

async function call<Res>(service: string, method: string, params: unknown, traceId?: string): Promise<Res> {
  const id = traceId && traceId.trim() !== "" ? traceId : newTraceId();
  const url = `${ADMIN_URL}/admin/v1/${service}/${method}`;
  let response: Response;
  try {
    response = await fetch(url, {
      method: "POST",
      headers: {
        "content-type": "application/json",
        ...traceHeaders(id),
        ...(ADMIN_TOKEN ? { authorization: `Bearer ${ADMIN_TOKEN}` } : {}),
      },
      body: JSON.stringify(withoutModerator(params)),
      signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
      cache: "no-store",
    });
  } catch {
    throw new AdminUnavailableError("serviço de administração indisponível");
  }

  const text = await response.text();
  let body: unknown = null;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    body = null;
  }
  const envelope = (body ?? {}) as Partial<AdminEnvelope>;
  const message = typeof envelope.message === "string" ? envelope.message : "";

  if (!response.ok) {
    if (response.status >= 500) {
      throw new AdminUnavailableError("serviço de administração indisponível");
    }
    throw new AdminError(response.status, message || "chamada administrativa recusada");
  }
  if (typeof envelope.result !== "number") {
    throw new AdminUnavailableError("resposta inválida do serviço de administração");
  }
  return envelope as Res;
}

type Handler = (req: unknown, traceId?: string) => Promise<unknown>;

/** Cria o objeto de serviço com um método por rota do contrato. */
function svc<T extends object>(service: string, methods: readonly string[]): T {
  const built: Record<string, Handler> = {};
  for (const method of methods) {
    built[method] = (req, traceId) => call(service, toSnake(method), req, traceId);
  }
  return built as unknown as T;
}

/* ------------------------------- contas ------------------------------- */

/** Conta vista por staff: id, login e papel. Nunca hash, e-mail ou PIN. */
export interface AccountSummary {
  id: number;
  login: string;
  role: string;
}

export interface FindAccountResponse extends AdminEnvelope {
  account?: AccountSummary;
}

export interface CreateAccountResponse extends AdminEnvelope {
  account?: AccountSummary;
}

/** `accounts/*` — busca exata e criação administrativa. */
export const accountsAdmin = svc<{
  find: Unary<{ login: string }, FindAccountResponse>;
  create: Unary<{ login: string; role: string }, CreateAccountResponse>;
}>("accounts", ["find", "create"]);

/* ------------------------------- catálogo ------------------------------ */

/** Item como o gateway o expõe hoje (`item_catalog/list_items`, ADR-0022). */
export interface CatalogItem {
  item_index: number;
  name: string;
  display_name: string;
}

export interface ListItemsResponse extends AdminEnvelope {
  items?: CatalogItem[];
  catalog_version?: string;
  icon_pack_version?: string;
}

/** `item_catalog/list_items` — foto do conteúdo carregado no boot do gateway. */
export const itemCatalog = svc<{
  listItems: Unary<Record<string, never>, ListItemsResponse>;
}>("item_catalog", ["listItems"]);

/* ------------------------------ sessão web ----------------------------- */

export interface VerifyCredentialsResponse {
  ok: boolean;
  account_id: number;
  blocked: boolean;
  role: string;
}

/** `account_web/*` — verificação de credencial e troca de senha numérica. */
export const accountWeb = svc<{
  verifyCredentials: Unary<{ name: string; password: string }, VerifyCredentialsResponse>;
}>("account_web", ["verifyCredentials"]);

/* ------------------------------- presença ------------------------------ */

/** Presença viva de um canal (`character_session`, lida pelo gateway). */
export interface ChannelPresenceWire {
  group_id: number;
  channel: number;
  host: string;
  online: number;
}

export interface ListPresenceResponse extends AdminEnvelope {
  channels?: ChannelPresenceWire[];
}

/**
 * `server_admin/list_channel_presence` — estado **somente leitura** dos canais.
 *
 * A fonte é `character_session`: o portal nunca escreve estado vivo, só enxerga
 * a foto e audita a leitura.
 */
export const presenceAdmin = svc<{
  listChannelPresence: Unary<Record<string, never>, ListPresenceResponse>;
}>("server_admin", ["listChannelPresence"]);

export { promisify };
export type { Unary };
