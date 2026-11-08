/**
 * Cliente do control plane `wyd-admin` — **HTTP/1.1 + JSON em loopback**
 * (ADR-0008). Substitui o antigo cliente gRPC mantendo as mesmas exportações,
 * para que os módulos do BFF não mudem de contrato.
 *
 * Regras:
 *
 * 1. server-only: nenhum componente client importa daqui — o browser nunca vê
 *    o token de staff nem fala com o backend (AGENTS.md / skill wyd-web-admin-ui).
 * 2. o corpo enviado **não** carrega `moderator_id`: autor vem da sessão no
 *    servidor Rust (ADR-0008 §2), nunca do browser.
 * 3. toda chamada leva `x-trace-id` e um deadline de 10 s (`AbortSignal.timeout`).
 * 4. recusa de negócio viaja em `200` com `result` 1..5 (`AdminResult`); só
 *    falha de transporte vira exceção (`AdminError`/`AdminUnavailableError`).
 */
import type { MapZone, MerchantTemplate, NpcDefinition, NpcFormValues, NpcShopItem } from "@/lib/npc";
import type { Rate, ServerFlag } from "@/lib/rates";
import type { CastleQuest, Guild, GuildRelation, Kefra, KingdomZone } from "@/lib/worldstate";

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
      },
      body: JSON.stringify(withoutModerator(params)),
      signal: AbortSignal.timeout(DEFAULT_TIMEOUT_MS),
      cache: "no-store",
    });
  } catch {
    // Conexão recusada, DNS, timeout do AbortSignal: backend indisponível.
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

/**
 * Cria o objeto de serviço com um método por rota. A tipagem exata vem do tipo
 * declarado em cada `svc<T>(...)`; o corpo é o mesmo `call` para todos.
 */
function svc<T extends object>(service: string, methods: readonly string[]): T {
  const built: Record<string, Handler> = {};
  for (const method of methods) {
    built[method] = (req, traceId) => call(service, toSnake(method), req, traceId);
  }
  return built as unknown as T;
}

export interface ItemEffect {
  code: number;
  /** Rótulo em português, "—" quando o código está fora do ItemEffect.h. */
  label: string;
  value: number;
}

export interface ItemEffectType {
  code: number;
  name: string;
  label: string;
}

export interface CatalogItem {
  item_index: number;
  /** Adds base do catálogo (issue 44). */
  effects?: ItemEffect[];
  name: string;
  icon_key: string;
  display_name: string;
  slot_mask: number;
  slots: string[];
  grade: number;
  mesh: number;
  texture: number;
  icon_url: string;
}

export interface ListItemsResponse {
  items: CatalogItem[];
  catalog_version: string;
  icon_pack_version: string;
  /** Vocabulário de ItemEffect.h para os seletores de efeito. */
  effect_types?: ItemEffectType[];
}

export interface VerifyCredentialsResponse {
  ok: boolean;
  account_id: number;
  blocked: boolean;
  role: string;
}

/** `ADMIN_RESULT_*` viaja no corpo de toda resposta admin. */
export interface AdminAck {
  result: number;
}

export interface ListNpcsResponse {
  result: number;
  npcs: NpcDefinition[];
}

export interface GetNpcResponse {
  result: number;
  npc?: NpcDefinition;
}

export interface UpsertNpcResponse {
  result: number;
  npc_id: number;
}

export interface ListMerchantTemplatesResponse {
  result: number;
  templates: MerchantTemplate[];
}

export interface ListMapZonesResponse {
  result: number;
  zones: MapZone[];
}

export interface ListDropItemsInput {
  item_index: number;
  item_query: string;
  mob_query: string;
  include_zero_drop_items: boolean;
}

export interface ListMobDropsInput {
  mob_query: string;
  item_index: number;
  item_query: string;
}

export interface ListDropItemsResponse {
  result: number;
  items?: import("@/lib/drop").ItemDropEntry[];
}

export interface ListMobDropsResponse {
  result: number;
  mobs?: import("@/lib/drop").MobDropEntry[];
}

export interface GetServerFlagsResponse {
  result: number;
  flags?: ServerFlag[];
  available: boolean;
  captured_at: string;
  message: string;
}

export interface GetRatesResponse {
  result: number;
  rates?: Rate[];
  content_ok: boolean;
  warning: string;
}

export interface SetRateResponse {
  result: number;
  message: string;
}

export interface ListDropRatesResponse {
  result: number;
  slots?: import("@/lib/droprate").DropRateSlot[];
}

export interface ListMobCarryResponse {
  result: number;
  template_name?: string;
  display_name?: string;
  level?: number;
  overridden?: boolean;
  slots?: import("@/lib/mobcarry").MobCarrySlot[];
}

export interface ListRewardItemsResponse {
  result: number;
  items?: import("@/lib/daily").DailyRewardItem[];
}

export interface UpsertRewardItemResponse {
  result: number;
  item_id: number;
}

export interface ListShopItemsResponse {
  result: number;
  items?: import("@/lib/donate").DonateShopItem[];
}

export interface UpsertShopItemResponse {
  result: number;
  item_id: number;
}

export interface CreditDonateBalanceResponse {
  result: number;
  new_balance: number;
}

export interface RevenueSummaryRequest {
  window: import("@/lib/revenue").RevenueWindowInput;
  bucket: number;
  account_id: number;
}

export interface RevenueSummaryResponse {
  result: number;
  totals?: import("@/lib/revenue").RevenueTotalsWire;
  by_method?: import("@/lib/revenue").RevenueByMethodWire[];
  series?: import("@/lib/revenue").RevenuePointWire[];
  from_unix: number;
  to_unix: number;
}

export interface ListTopupOrdersResponse {
  result: number;
  orders?: import("@/lib/revenue").TopupOrderRowWire[];
  total_count: number;
  from_unix: number;
  to_unix: number;
}

export interface ListTopBuyersResponse {
  result: number;
  buyers?: import("@/lib/revenue").TopBuyerRowWire[];
  total_count: number;
  from_unix: number;
  to_unix: number;
}

export interface ListDonateSpendResponse {
  result: number;
  entries?: import("@/lib/revenue").DonateLedgerRowWire[];
  total_count: number;
  from_unix: number;
  to_unix: number;
}

export interface SearchAccountsResponse {
  result: number;
  accounts?: import("@/lib/revenue").AccountSummaryWire[];
}

export interface GetWorldEventConfigResponse {
  result: number;
  version: number;
  config?: import("@/lib/worldevent").WorldEventConfig;
}

export interface GetWorldStateResponse {
  result: number;
  guilds?: Guild[];
  relations?: GuildRelation[];
  zones?: KingdomZone[];
  tower_owner_guild: number;
  tower_owner_name: string;
  tower_updated_unix: number;
  castle?: CastleQuest;
  kefra?: Kefra;
}

export interface SetWorldStateResponse {
  result: number;
  message: string;
}

export interface ListMobTemplatesResponse {
  result: number;
  templates?: import("@/lib/mob").MobTemplateFile[];
}

export interface GetMobTemplateStatResponse {
  result: number;
  stat?: import("@/lib/mob").MobStat;
}

export interface UpsertMobTemplateStatResponse {
  result: number;
}

export interface AttributeMapTransformRequest {
  operation: number;
  operand: number;
  rect?: import("@/lib/attributemap").AttributeMapRect;
  filter?: import("@/lib/attributemap").AttributeMapFilter;
}

export interface AttributeMapTransformResponse {
  result: number;
  changed_count?: number;
  before_histogram?: import("@/lib/attributemap").AttributeMapValueCount[];
  after_histogram?: import("@/lib/attributemap").AttributeMapValueCount[];
  original_sha256?: string;
  new_sha256?: string;
  filename?: string;
  data?: Uint8Array | string;
}

export interface ItemEffectEntry {
  /** Código de ItemEffect.h — é a chave do overlay. */
  effect: number;
  value: number;
  /** Constante do ItemEffect.h sem o prefixo EF_, ex. "CRITICAL". */
  name: string;
  /** Rótulo em português, ex. "Crítico". */
  label: string;
  /** O CSV declara este efeito. */
  from_content: boolean;
  /** O valor veio da web. */
  has_override: boolean;
  updated_by: number;
  updated_at: string;
}

export interface ListItemEffectsResponse {
  result: number;
  effects: ItemEffectEntry[];
  effect_types: ItemEffectType[];
  reason: string;
}

export interface ItemEffectWriteResponse {
  result: number;
  /** Explicação para o moderador quando `result` não é OK. */
  message: string;
}

export type UpsertItemEffectInput = {
  item_index: number;
  effect: number;
  value: number;
  reason: string;
};

export type TeleportRouteInput = {
  src_x: number;
  src_y: number;
  dst_x: number;
  dst_y: number;
  cost: number;
  requirement: string;
  enabled: boolean;
  note: string;
};

export interface AdminTeleportRoute extends TeleportRouteInput {
  updated_by: number;
  updated_at: string;
  from_content: boolean;
  has_override: boolean;
}

export interface ListTeleportRoutesResponse {
  result: number;
  routes: AdminTeleportRoute[];
}

export interface UpsertTeleportRouteResponse {
  result: number;
  warnings: string[];
  src_x: number;
  src_y: number;
}

export interface SimulateTeleportRouteResponse {
  result: number;
  ok: boolean;
  src_x: number;
  src_y: number;
  dst_x: number;
  dst_y: number;
  cost: number;
  requirement: string;
  warnings: string[];
}

export interface ListAccountsResponse {
  result: number;
  accounts: import("@/lib/account").AdminAccount[];
  /** último id da página; 0 = sem próxima. */
  next_id: number;
  total_count: number;
}

export interface GetAccountAdminResponse {
  result: number;
  account?: import("@/lib/account").AdminAccount;
  audit: import("@/lib/account").AccountAudit[];
}

export interface AccountWriteResponse {
  result: number;
  message?: string;
  account?: import("@/lib/account").AdminAccount;
}

export interface AdminCreateAccountResponse {
  result: number;
  account_id: number;
  message?: string;
}

export type GrantCharacterItemInput = {
  character_id: number;
  item_index: number;
  quantity: number;
  eff1: number;
  effv1: number;
  eff2: number;
  effv2: number;
  eff3: number;
  effv3: number;
  reason: string;
};

export interface GrantCharacterItemResponse {
  result: number;
  delivery_ids: number[];
}

export interface AdminCharacterGrant {
  delivery_id: number;
  item_index: number;
  status: string;
  source: string;
  created_at: string;
}

export interface ListCharacterGrantsResponse {
  result: number;
  grants: AdminCharacterGrant[];
}

export interface SearchCharactersResponse {
  result: number;
  characters: import("@/lib/character").CharacterSummary[];
}

export interface GetCharacterAdminResponse {
  result: number;
  character?: import("@/lib/character").CharacterDetail;
  /** índice da skill → nome do SkillData.csv (issue 47). */
  skill_names?: Record<string, string>;
  /** código de efeito (eff1..3) → rótulo, do vocabulário de ItemEffect.h. */
  add_labels?: Record<string, string>;
}

/** Item com preço global definido; ausente da lista = usa a base do catálogo. */
export interface ItemPriceEntry {
  item_index: number;
  price: number;
}

export interface ListItemPricesResponse {
  result: number;
  prices: ItemPriceEntry[];
}

/** `ItemCatalogService` — catálogo de itens e vocabulário de efeitos. */
export const itemCatalog = svc<{
  listItems: Unary<Record<string, never>, ListItemsResponse>;
}>("item_catalog", ["listItems"]);

/** `AccountWebService` — login web (`verifyCredentials`) e checagem de bloqueio. */
export const accountWeb = svc<{
  verifyCredentials: Unary<{ name: string; password: string }, VerifyCredentialsResponse>;
}>("account_web", ["verifyCredentials"]);

/** `NpcAdminService` — NPC/mercador, preços, drops e zonas. Sempre via `adminCall`. */
export const npcAdmin = svc<{
  listNpcs: Unary<Record<string, never>, ListNpcsResponse>;
  getNpc: Unary<{ npc_id: number }, GetNpcResponse>;
  upsertNpc: Unary<NpcFormValues, UpsertNpcResponse>;
  setNpcVisibility: Unary<{ npc_id: number; enabled: boolean }, AdminAck>;
  setNpcShop: Unary<{ npc_id: number; items: NpcShopItem[] }, AdminAck>;
  deleteNpc: Unary<{ npc_id: number }, AdminAck>;
  listMerchantTemplates: Unary<Record<string, never>, ListMerchantTemplatesResponse>;
  listMapZones: Unary<Record<string, never>, ListMapZonesResponse>;
  listItemPrices: Unary<Record<string, never>, ListItemPricesResponse>;
  setItemPrice: Unary<{ item_index: number; price: number }, AdminAck>;
  listDropItems: Unary<ListDropItemsInput, ListDropItemsResponse>;
  listMobDrops: Unary<ListMobDropsInput, ListMobDropsResponse>;
}>("npc_admin", [
  "listNpcs",
  "getNpc",
  "upsertNpc",
  "setNpcVisibility",
  "setNpcShop",
  "deleteNpc",
  "listMerchantTemplates",
  "listMapZones",
  "listItemPrices",
  "setItemPrice",
  "listDropItems",
  "listMobDrops",
]);

/** `ContentRatesService` — refino, combine e quests (chave textual por família). */
export const contentRates = svc<{
  getRates: Unary<Record<string, never>, GetRatesResponse>;
  upsertRate: Unary<
    { file: string; family: string; name: string; value: number; reason: string },
    SetRateResponse
  >;
  resetRate: Unary<{ file: string; family: string; name: string; reason: string }, SetRateResponse>;
  getServerFlags: Unary<Record<string, never>, GetServerFlagsResponse>;
}>("content_rates", ["getRates", "upsertRate", "resetRate", "getServerFlags"]);

/** `DropAdminService` — divisor de drop por slot de `Carry`. */
export const dropAdmin = svc<{
  listDropRates: Unary<Record<string, never>, ListDropRatesResponse>;
  upsertDropRate: Unary<{ slot: number; value: number; reason: string }, AdminAck>;
  resetDropRate: Unary<{ slot: number; reason: string }, AdminAck>;
  listMobCarry: Unary<{ template_name: string }, ListMobCarryResponse>;
  setMobCarry: Unary<
    { template_name: string; slots: { slot: number; item_index: number }[]; reason: string },
    AdminAck
  >;
  resetMobCarry: Unary<{ template_name: string; reason: string }, AdminAck>;
}>("drop_admin", [
  "listDropRates",
  "upsertDropRate",
  "resetDropRate",
  "listMobCarry",
  "setMobCarry",
  "resetMobCarry",
]);

/** `DailyRewardAdminService` — catálogo da recompensa diária. */
export const dailyRewardAdmin = svc<{
  listRewardItems: Unary<Record<string, never>, ListRewardItemsResponse>;
  upsertRewardItem: Unary<{ item: import("@/lib/daily").DailyRewardItem }, UpsertRewardItemResponse>;
  setRewardItemEnabled: Unary<{ item_id: number; enabled: boolean }, AdminAck>;
  deleteRewardItem: Unary<{ item_id: number }, AdminAck>;
}>("daily_reward_admin", [
  "listRewardItems",
  "upsertRewardItem",
  "setRewardItemEnabled",
  "deleteRewardItem",
]);

/** `DonateAdminService` — loja de donate; crédito é a operação mais sensível. */
export const donateAdmin = svc<{
  listShopItems: Unary<Record<string, never>, ListShopItemsResponse>;
  upsertShopItem: Unary<{ item: import("@/lib/donate").DonateShopItem }, UpsertShopItemResponse>;
  setShopItemEnabled: Unary<{ item_id: number; enabled: boolean }, AdminAck>;
  deleteShopItem: Unary<{ item_id: number }, AdminAck>;
  creditDonateBalance: Unary<
    { account_id: number; amount: number; reason: string },
    CreditDonateBalanceResponse
  >;
}>("donate_admin", [
  "listShopItems",
  "upsertShopItem",
  "setShopItemEnabled",
  "deleteShopItem",
  "creditDonateBalance",
]);

/** `DonateRevenueAdminService` — faturamento (só leitura). */
export const donateRevenueAdmin = svc<{
  getRevenueSummary: Unary<RevenueSummaryRequest, RevenueSummaryResponse>;
  listTopupOrders: Unary<
    {
      window: import("@/lib/revenue").RevenueWindowInput;
      status: number;
      payment_method: number;
      account_id: number;
      limit: number;
      offset: number;
    },
    ListTopupOrdersResponse
  >;
  listTopBuyers: Unary<
    { window: import("@/lib/revenue").RevenueWindowInput; limit: number; offset: number },
    ListTopBuyersResponse
  >;
  listDonateSpend: Unary<
    {
      window: import("@/lib/revenue").RevenueWindowInput;
      action: number;
      account_id: number;
      limit: number;
      offset: number;
    },
    ListDonateSpendResponse
  >;
  searchAccounts: Unary<{ name_prefix: string; limit: number }, SearchAccountsResponse>;
}>("donate_revenue_admin", [
  "getRevenueSummary",
  "listTopupOrders",
  "listTopBuyers",
  "listDonateSpend",
  "searchAccounts",
]);

/** `WorldEventAdminService` — evento global de mundo. */
export const worldEventAdmin = svc<{
  getWorldEventConfig: Unary<Record<string, never>, GetWorldEventConfigResponse>;
  setWorldEventConfig: Unary<{ config: import("@/lib/worldevent").WorldEventConfig }, AdminAck>;
}>("world_event_admin", ["getWorldEventConfig", "setWorldEventConfig"]);

/** `WorldStateAdminService` — guildas, zonas do reino, castelo e Kefra. */
export const worldStateAdmin = svc<{
  getWorldState: Unary<Record<string, never>, GetWorldStateResponse>;
  setCityTax: Unary<{ zone: number; city_tax: number }, SetWorldStateResponse>;
  setKefraDefeated: Unary<{ defeated: boolean }, SetWorldStateResponse>;
  setGuildMaster: Unary<
    { guild_id: number; character_id: number; reason: string },
    SetWorldStateResponse
  >;
  removeGuildMember: Unary<
    { guild_id: number; character_id: number; reason: string },
    SetWorldStateResponse
  >;
  setGuildRelation: Unary<
    { guild_id: number; target_guild_id: number; kind: number; reason: string },
    SetWorldStateResponse
  >;
  setZoneOwner: Unary<{ zone: number; guild_id: number; reason: string }, SetWorldStateResponse>;
  setTowerOwner: Unary<{ guild_id: number; reason: string }, SetWorldStateResponse>;
}>("world_state_admin", [
  "getWorldState",
  "setCityTax",
  "setKefraDefeated",
  "setGuildMaster",
  "removeGuildMember",
  "setGuildRelation",
  "setZoneOwner",
  "setTowerOwner",
]);

/** `MobTemplateAdminService` — stats de qualquer template de NPC. */
export const mobAdmin = svc<{
  listMobTemplates: Unary<Record<string, never>, ListMobTemplatesResponse>;
  getMobTemplateStat: Unary<{ template_name: string }, GetMobTemplateStatResponse>;
  upsertMobTemplateStat: Unary<
    {
      stat: import("@/lib/mob").MobStatValues & {
        equip?: import("@/lib/mob").MobEquipSlot[];
      };
    },
    UpsertMobTemplateStatResponse
  >;
  setMobTemplateEquip: Unary<
    { template_name: string; items: import("@/lib/mob").MobEquipSlot[] },
    AdminAck
  >;
  deleteMobTemplateStat: Unary<{ template_name: string }, AdminAck>;
}>("mob_admin", [
  "listMobTemplates",
  "getMobTemplateStat",
  "upsertMobTemplateStat",
  "setMobTemplateEquip",
  "deleteMobTemplateStat",
]);

/** `AttributeMapAdminService` — leitura e transformação em massa do `AttributeMap.dat`. */
export const attributeMapAdmin = svc<{
  getAttributeMapInfo: Unary<
    Record<string, never>,
    { result: number; info?: import("@/lib/attributemap").AttributeMapInfo }
  >;
  transformAttributeMap: Unary<AttributeMapTransformRequest, AttributeMapTransformResponse>;
}>("attribute_map_admin", ["getAttributeMapInfo", "transformAttributeMap"]);

/** `TeleportAdminService` — rotas de teleporte (tabela compilada + overlay). */
export const teleportAdmin = svc<{
  listRoutes: Unary<Record<string, never>, ListTeleportRoutesResponse>;
  upsertRoute: Unary<TeleportRouteInput, UpsertTeleportRouteResponse>;
  setRouteEnabled: Unary<{ src_x: number; src_y: number; enabled: boolean }, AdminAck>;
  deleteRoute: Unary<{ src_x: number; src_y: number }, AdminAck>;
  simulateRoute: Unary<{ src_x: number; src_y: number }, SimulateTeleportRouteResponse>;
}>("teleport_admin", [
  "listRoutes",
  "upsertRoute",
  "setRouteEnabled",
  "deleteRoute",
  "simulateRoute",
]);

/** `ItemEffectAdminService` — efeitos base do item (overlay no Postgres). */
export const itemEffectAdmin = svc<{
  listEffects: Unary<{ item_index: number }, ListItemEffectsResponse>;
  upsertEffect: Unary<UpsertItemEffectInput, ItemEffectWriteResponse>;
  deleteEffect: Unary<
    { item_index: number; effect: number; reason: string },
    ItemEffectWriteResponse
  >;
}>("item_effect_admin", ["listEffects", "upsertEffect", "deleteEffect"]);

/** `AccountAdminService` — contas, papéis e banimento. */
export const accountAdmin = svc<{
  listAccounts: Unary<
    { query: string; role: string; blocked: number; limit: number; after_id: number },
    ListAccountsResponse
  >;
  getAccount: Unary<{ account_id: number }, GetAccountAdminResponse>;
  setRole: Unary<{ account_id: number; role: string; reason: string }, AccountWriteResponse>;
  setBlocked: Unary<{ account_id: number; blocked: boolean; reason: string }, AccountWriteResponse>;
  createAccount: Unary<
    { name: string; password: string; email: string; role: string; reason: string },
    AdminCreateAccountResponse
  >;
}>("account_admin", ["listAccounts", "getAccount", "setRole", "setBlocked", "createAccount"]);

/** `CharacterAdminService` — inspeção de personagem e grants. */
export const characterAdmin = svc<{
  searchCharacters: Unary<{ query: string }, SearchCharactersResponse>;
  getCharacter: Unary<{ character_id: number }, GetCharacterAdminResponse>;
  grantItem: Unary<GrantCharacterItemInput, GrantCharacterItemResponse>;
  listCharacterGrants: Unary<{ character_id: number }, ListCharacterGrantsResponse>;
}>("character_admin", [
  "searchCharacters",
  "getCharacter",
  "grantItem",
  "listCharacterGrants",
]);

/**
 * `promisify`/`Unary` moravam no cliente antigo; agora vivem em
 * `./promisify` e são reexportados para os imports existentes continuarem
 * válidos.
 */
export { promisify };
export type { Unary };
