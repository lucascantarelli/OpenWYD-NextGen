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

async function call<Res>(
  service: string,
  method: string,
  params: unknown,
  traceId?: string,
  actor?: string,
): Promise<Res> {
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
        // Quem age: o gateway autoriza pelo papel deste ator, não pelo token.
        ...(actor ? { "x-wyd-actor": actor } : {}),
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

type Handler = (req: unknown, traceId?: string, actor?: string) => Promise<unknown>;

/** Cria o objeto de serviço com um método por rota do contrato. */
function svc<T extends object>(service: string, methods: readonly string[]): T {
  const built: Record<string, Handler> = {};
  for (const method of methods) {
    built[method] = (req, traceId, actor) => call(service, toSnake(method), req, traceId, actor);
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

/** Estado de moderação de uma conta (bloqueio + motivo + instante). */
export interface AccountModerationWire {
  account: AccountSummary;
  blocked: boolean;
  reason: string;
  blocked_at_ms: number | null;
}

export interface ModerationResponse extends AdminEnvelope {
  moderation?: AccountModerationWire;
}

/**
 * `accounts/*` — busca, criação e moderação.
 *
 * `setRole`/`setBlocked` exigem staff com alcance (admin para papel e para
 * staff); o backend relê o papel da sessão e audita toda escrita.
 */
export const accountsAdmin = svc<{
  find: Unary<{ login: string }, FindAccountResponse>;
  create: Unary<{ login: string; role: string }, CreateAccountResponse>;
  getModeration: Unary<{ login: string }, ModerationResponse>;
  setRole: Unary<{ login: string; role: string }, ModerationResponse>;
  setBlocked: Unary<{ login: string; blocked: boolean; reason: string }, ModerationResponse>;
}>("accounts", ["find", "create", "getModeration", "setRole", "setBlocked"]);

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

/* ----------------------------- auditoria ------------------------------ */

/** Linha da trilha append-only (`audit_log`, migration 0005). */
export interface AuditRowWire {
  at_ms: number;
  actor_id: number;
  actor_role: string;
  action: string;
  target: string;
  detail: string;
}

export interface ListAuditResponse extends AdminEnvelope {
  entries?: AuditRowWire[];
}

/** `audit_admin/recent` — últimas linhas da trilha (leitura de admin). */
export const auditAdmin = svc<{
  recent: Unary<{ limit: number }, ListAuditResponse>;
}>("audit_admin", ["recent"]);

/* ------------------------------ economia ------------------------------ */

/** Imposto somado do livro-caixa por cidade. */
export interface CityTaxWire {
  city_code: string;
  total: number;
}

/** Concessões por tipo na fila de entrega. */
export interface GrantTotalWire {
  kind: string;
  count: number;
  gold: number;
}

export interface EconomySummaryWire {
  tax_by_city: CityTaxWire[];
  grants: GrantTotalWire[];
}

export interface EconomyResponse extends AdminEnvelope {
  economy?: EconomySummaryWire;
}

/** `economy_admin/summary` — números reais (livro-caixa + fila de entrega). */
export const economyAdmin = svc<{
  summary: Unary<Record<string, never>, EconomyResponse>;
}>("economy_admin", ["summary"]);

/* ---------------------------- personagens ----------------------------- */

/** Personagem na tela de entregas: identidade + ouro atual (lido da fonte). */
export interface CharacterWire {
  id: number;
  account_id: number;
  name: string;
  level: number;
  gold: number;
  class: number;
  slot: number;
}

export interface SearchCharactersResponse extends AdminEnvelope {
  characters?: CharacterWire[];
}

/** Linha da `delivery_queue` como o staff a confere (somente leitura). */
export interface GrantRowWire {
  id: number;
  kind: string;
  gold: number;
  status: string;
  idempotency_key: string | null;
  created_by: number | null;
  at_ms: number;
}

export interface ListCharacterGrantsResponse extends AdminEnvelope {
  grants?: GrantRowWire[];
}

/** Recibo de `grants/request`: mesma chave devolve o mesmo `grant_id` com replay. */
export interface GrantReceiptWire extends AdminEnvelope {
  idempotency_key: string;
  grant_id: number;
  replay: boolean;
}

/** `character_admin/*` — busca de personagem e fila de entregas (staff). */
export const characterAdmin = svc<{
  searchCharacters: Unary<{ term: string; limit: number }, SearchCharactersResponse>;
  listCharacterGrants: Unary<{ character_id: number; limit: number }, ListCharacterGrantsResponse>;
}>("character_admin", ["searchCharacters", "listCharacterGrants"]);

/** `grants/request` — enfileira o grant com chave de idempotência. */
export const grantsAdmin = svc<{
  request: Unary<
    {
      idempotency_key: string;
      character_id: number;
      kind: { type: string; amount: number };
    },
    GrantReceiptWire
  >;
}>("grants", ["request"]);

/* --------------------------- configuração ---------------------------- */

/** Linha de taxa com o override já aplicado (`content_rates/get_rates`). */
export interface RateWire {
  domain: string;
  key: string;
  value: number;
  unit: string;
  source: string;
  overridden: boolean;
}

/** Regra de refino de acessório (brinco/colar/cinto/…), por passo e receita. */
export interface AccessoryRuleWire {
  kind: string;
  recipe: string;
  step_from: number;
  step_to: number;
  rate_bps: number;
  fail_mode: string;
  fail_bonus_bps: number;
  gold_cost: number;
  material_name: string;
  material_qty: number;
  source: string;
  notes: string;
}

export interface GetRatesResponse extends AdminEnvelope {
  revision?: number;
  rates?: RateWire[];
  accessories?: AccessoryRuleWire[];
}

export interface RevisionResponse extends AdminEnvelope {
  revision?: number;
}

export interface ServerFlagsResponse extends AdminEnvelope {
  revision?: number;
  flags?: Record<string, number>;
}

/** `content_rates/*` — taxas de refino, SancRate, composição e acessório. */
export const contentRates = svc<{
  getRates: Unary<Record<string, never>, GetRatesResponse>;
  upsertRate: Unary<
    { domain: string; key: string; value: number; source: string },
    RevisionResponse
  >;
  resetRate: Unary<{ domain: string; key: string }, RevisionResponse>;
  getServerFlags: Unary<Record<string, never>, ServerFlagsResponse>;
}>("content_rates", ["getRates", "upsertRate", "resetRate", "getServerFlags"]);

/** Peso de drop por template de monstro. */
export interface DropWire {
  template_name: string;
  item_index: number;
  item_name: string;
  weight: number;
  min_qty: number;
  max_qty: number;
  source: string;
}

export interface ListDropsResponse extends AdminEnvelope {
  drops?: DropWire[];
}

export const dropAdmin = svc<{
  listDropRates: Unary<{ template_name?: string; limit: number }, ListDropsResponse>;
  upsertDropRate: Unary<
    {
      template_name: string;
      item_index: number;
      weight: number;
      min_qty: number;
      max_qty: number;
      source: string;
    },
    RevisionResponse
  >;
  resetDropRate: Unary<{ template_name: string; item_index: number }, RevisionResponse>;
}>("drop_admin", ["listDropRates", "upsertDropRate", "resetDropRate"]);

/** NPC com posição editável. */
export interface NpcWire {
  id: number;
  slug: string;
  display_name: string;
  map_id: number;
  pos_x: number;
  pos_y: number;
  enabled: boolean;
  merchant: number;
}

export interface GetNpcResponse extends AdminEnvelope {
  npc?: NpcWire;
}

export const npcAdmin = svc<{
  getNpc: Unary<{ id: number }, GetNpcResponse>;
  upsertNpc: Unary<
    { id: number; map_id: number; pos_x: number; pos_y: number; enabled: boolean },
    RevisionResponse
  >;
}>("npc_admin", ["getNpc", "upsertNpc"]);

/** Rota de teleporte: política editável (custo, nível, disponibilidade). */
export interface RouteWire {
  id: number;
  from_map_id: number;
  from_x: number;
  from_y: number;
  to_map_id: number;
  to_x: number;
  to_y: number;
  npc_id: number | null;
  cost_gold: number;
  req_level: number;
  enabled: boolean;
}

export interface ListRoutesResponse extends AdminEnvelope {
  routes?: RouteWire[];
}

export const teleportAdmin = svc<{
  listRoutes: Unary<{ limit: number }, ListRoutesResponse>;
  upsertRoute: Unary<
    { id: number; cost_gold?: number; req_level?: number; enabled?: boolean },
    RevisionResponse
  >;
  setRouteEnabled: Unary<{ id: number; enabled: boolean }, RevisionResponse>;
}>("teleport_admin", ["listRoutes", "upsertRoute", "setRouteEnabled"]);

export { promisify };
export type { Unary };
