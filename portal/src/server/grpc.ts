/**
 * Cliente gRPC do web-api (`webserver`, porta 7600).
 *
 * Regra: este módulo é server-only. Nenhum componente client importa daqui —
 * o browser nunca fala gRPC nem vê certificado (AGENTS.md / skill
 * wyd-web-admin-ui §1).
 */
import fs from "node:fs";
import path from "node:path";

import * as grpc from "@grpc/grpc-js";
import * as protoLoader from "@grpc/proto-loader";

import type { MapZone, MerchantTemplate, NpcDefinition, NpcFormValues, NpcShopItem } from "@/lib/npc";
import type { Rate, ServerFlag } from "@/lib/rates";
import type { CastleQuest, Guild, GuildRelation, Kefra, KingdomZone } from "@/lib/worldstate";

import { promisify, type Unary } from "./promisify";

const PROTO_PATH =
  process.env.WEB_API_PROTO ??
  path.join(process.cwd(), "..", "api", "web", "v1", "web.proto");

const ADDR = process.env.WEB_API_ADDR ?? "127.0.0.1:7600";

function loadProto() {
  const definition = protoLoader.loadSync(PROTO_PATH, {
    keepCase: true,
    longs: Number,
    enums: Number,
    defaults: true,
    oneofs: true,
  });
  return grpc.loadPackageDefinition(definition) as unknown as {
    web: { v1: Record<string, new (addr: string, creds: grpc.ChannelCredentials) => unknown> };
  };
}

function buildCredentials(): grpc.ChannelCredentials {
  const { WEB_API_TLS_CA, WEB_API_TLS_CERT, WEB_API_TLS_KEY } = process.env;
  if (!WEB_API_TLS_CA || !WEB_API_TLS_CERT || !WEB_API_TLS_KEY) {
    // Dev local: o webserver sobe com credenciais insecure quando não há certs.
    return grpc.credentials.createInsecure();
  }
  return grpc.credentials.createSsl(
    fs.readFileSync(WEB_API_TLS_CA),
    fs.readFileSync(WEB_API_TLS_KEY),
    fs.readFileSync(WEB_API_TLS_CERT),
  );
}

const proto = loadProto();
const credentials = buildCredentials();

/** Cria um cliente gRPC tipado de forma frouxa (tipos de domínio em src/lib). */
function client<T>(service: string): T {
  const Ctor = proto.web.v1[service];
  if (!Ctor) throw new Error(`serviço gRPC desconhecido: ${service}`);
  return new Ctor(ADDR, credentials) as T;
}

export const itemCatalog = client<{
  listItems: Unary<Record<string, never>, ListItemsResponse>;
}>("ItemCatalogService");

/** `AccountWebService` — login web (`VerifyCredentials`), criar conta, play code. */
export const accountWeb = client<{
  verifyCredentials: Unary<{ name: string; password: string }, VerifyCredentialsResponse>;
}>("AccountWebService");

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

/** `ADMIN_RESULT_*` viaja no corpo de toda resposta admin (`web.proto:253`). */
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

/**
 * `NpcAdminService` — edição de NPC/mercador. Toda chamada passa por
 * `adminCall` (issue 21); nunca chame este cliente direto de uma page.
 */
export const npcAdmin = client<{
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
}>("NpcAdminService");

/** Filtros de `ListDropItems` (issue 26); tudo vazio devolve o catálogo com drop. */
export interface ListDropItemsInput {
  item_index: number;
  item_query: string;
  mob_query: string;
  include_zero_drop_items: boolean;
}

/** Filtros de `ListMobDrops` (issue 26). */
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

/**
 * `TeleportAdminService` — rotas de teleporte (issue 42). `ListRoutes` devolve a
 * visão mesclada: tabela compilada + overlay. Sempre via `adminCall`.
 */
/**
 * `ContentRatesService` — taxas de refino, combine e quests (issue 36). A chave
 * da célula é textual (`name`), porque o `CompRate.txt` tem várias chaves por
 * família. `GetRates` devolve valor do arquivo, valor efetivo e faixa.
 */
export const contentRates = client<{
  getRates: Unary<Record<string, never>, GetRatesResponse>;
  upsertRate: Unary<
    { file: string; family: string; name: string; value: number; reason: string },
    SetRateResponse
  >;
  resetRate: Unary<
    { file: string; family: string; name: string; reason: string },
    SetRateResponse
  >;
  getServerFlags: Unary<Record<string, never>, GetServerFlagsResponse>;
}>("ContentRatesService");

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

/**
 * `DropAdminService` — divisor de drop por slot de `Carry` (issue 34).
 * `ListDropRates` devolve os 64 slots com valor de código, valor efetivo e se há
 * override; as escritas são um slot por vez, com motivo auditado.
 */
export const dropAdmin = client<{
  listDropRates: Unary<Record<string, never>, ListDropRatesResponse>;
  upsertDropRate: Unary<{ slot: number; value: number; reason: string }, AdminAck>;
  resetDropRate: Unary<{ slot: number; reason: string }, AdminAck>;
  listMobCarry: Unary<{ template_name: string }, ListMobCarryResponse>;
  setMobCarry: Unary<
    { template_name: string; slots: { slot: number; item_index: number }[]; reason: string },
    AdminAck
  >;
  resetMobCarry: Unary<{ template_name: string; reason: string }, AdminAck>;
}>("DropAdminService");

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

/**
 * `DailyRewardAdminService` — catálogo da recompensa diária (issue 30).
 * `ListRewardItems` devolve todas as ofertas (a tabela de moderação);
 * `upsertRewardItem` com `item.id == 0` cria. Sempre via `adminCall`.
 */
export const dailyRewardAdmin = client<{
  listRewardItems: Unary<Record<string, never>, ListRewardItemsResponse>;
  upsertRewardItem: Unary<
    { item: import("@/lib/daily").DailyRewardItem },
    UpsertRewardItemResponse
  >;
  setRewardItemEnabled: Unary<{ item_id: number; enabled: boolean }, AdminAck>;
  deleteRewardItem: Unary<{ item_id: number }, AdminAck>;
}>("DailyRewardAdminService");

export interface ListRewardItemsResponse {
  result: number;
  items?: import("@/lib/daily").DailyRewardItem[];
}

export interface UpsertRewardItemResponse {
  result: number;
  item_id: number;
}

/**
 * `DonateAdminService` — loja de donate (issue 29). `ListShopItems` devolve todas
 * as ofertas (a tabela de moderação); `CreditDonateBalance` é a operação mais
 * sensível do portal e sempre passa por `adminCall` com motivo auditado.
 */
export const donateAdmin = client<{
  listShopItems: Unary<Record<string, never>, ListShopItemsResponse>;
  upsertShopItem: Unary<
    { item: import("@/lib/donate").DonateShopItem },
    UpsertShopItemResponse
  >;
  setShopItemEnabled: Unary<{ item_id: number; enabled: boolean }, AdminAck>;
  deleteShopItem: Unary<{ item_id: number }, AdminAck>;
  creditDonateBalance: Unary<
    { account_id: number; amount: number; reason: string },
    CreditDonateBalanceResponse
  >;
}>("DonateAdminService");

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

/**
 * `DonateRevenueAdminService` — faturamento (issue 31). **Só leitura**: KPIs do
 * período, tabela de pedidos, ranking de compradores, extrato da carteira e a
 * busca de conta que o filtro usa. Dinheiro (`*_cents`) e donate (`credits`) são
 * unidades distintas; o painel nunca as soma. Sempre via `adminCall`.
 */
export const donateRevenueAdmin = client<{
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
}>("DonateRevenueAdminService");

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

/**
 * `WorldEventAdminService` — evento global de mundo (issue 28). `Set` substitui o
 * config inteiro; o tmserver relê por polling de 15 s, então a alteração chega ao
 * mundo sem restart. Sempre via `adminCall`.
 */
export const worldEventAdmin = client<{
  getWorldEventConfig: Unary<Record<string, never>, GetWorldEventConfigResponse>;
  setWorldEventConfig: Unary<{ config: import("@/lib/worldevent").WorldEventConfig }, AdminAck>;
}>("WorldEventAdminService");

export interface GetWorldEventConfigResponse {
  result: number;
  version: number;
  config?: import("@/lib/worldevent").WorldEventConfig;
}

/**
 * `WorldStateAdminService` — guildas, zonas do reino, castelo e Kefra (issue 45).
 * `GetWorldState` devolve tudo num só snapshot; as duas escritas são taxa da
 * cidade e o flag do Kefra, ambas aplicadas no próximo boot do tmserver.
 */
export const worldStateAdmin = client<{
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
}>("WorldStateAdminService");

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

/**
 * `MobTemplateAdminService` — stats de QUALQUER template de `Release/TMsrv/run/npc/`
 * (issue 25). `GetMobTemplateStat` faz read-through quando não há override; o
 * upsert substitui o override inteiro, **equip incluso** — por isso o BFF relê o
 * equip do servidor antes de gravar stats e o editor de equip usa
 * `SetMobTemplateEquip` (que exige override existente).
 */
export const mobAdmin = client<{
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
}>("MobTemplateAdminService");

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

/**
 * `AttributeMapAdminService` — leitura e transformação em massa do
 * `AttributeMap.dat` (issue 27). A transformação é pura: devolve os bytes
 * novos para download, nunca escreve o arquivo de conteúdo.
 */
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

export const attributeMapAdmin = client<{
  getAttributeMapInfo: Unary<
    Record<string, never>,
    { result: number; info?: import("@/lib/attributemap").AttributeMapInfo }
  >;
  transformAttributeMap: Unary<AttributeMapTransformRequest, AttributeMapTransformResponse>;
}>("AttributeMapAdminService");

export const teleportAdmin = client<{
  listRoutes: Unary<Record<string, never>, ListTeleportRoutesResponse>;
  upsertRoute: Unary<TeleportRouteInput, UpsertTeleportRouteResponse>;
  setRouteEnabled: Unary<{ src_x: number; src_y: number; enabled: boolean }, AdminAck>;
  deleteRoute: Unary<{ src_x: number; src_y: number }, AdminAck>;
  simulateRoute: Unary<{ src_x: number; src_y: number }, SimulateTeleportRouteResponse>;
}>("TeleportAdminService");

/**
 * `ItemEffectAdminService` — efeitos base do item (issue 44, metade de escrita).
 * A edição vive no overlay do Postgres; o `ItemList.csv` não é reescrito.
 */
export const itemEffectAdmin = client<{
  listEffects: Unary<{ item_index: number }, ListItemEffectsResponse>;
  upsertEffect: Unary<UpsertItemEffectInput, ItemEffectWriteResponse>;
  deleteEffect: Unary<{ item_index: number; effect: number; reason: string }, ItemEffectWriteResponse>;
}>("ItemEffectAdminService");

export type UpsertItemEffectInput = {
  item_index: number;
  effect: number;
  value: number;
  reason: string;
};

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

/**
 * `CharacterAdminService` — inspeção de personagem (issue 43). Só leitura nesta
 * fase: busca e detalhe vêm do estado frio em Postgres. Sempre via `adminCall`.
 */
/**
 * `AccountAdminService` — contas, papéis e banimento (issue 32).
 *
 * Leitura para staff; `setRole`/`createAccount` só para admin e `setBlocked`
 * também para moderador (alvo jogador). O `message` das respostas de escrita é a
 * recusa de negócio ("última conta admin ativa") que o `adminCall` repassa.
 */
export const accountAdmin = client<{
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
}>("AccountAdminService");

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

export const characterAdmin = client<{
  searchCharacters: Unary<{ query: string }, SearchCharactersResponse>;
  getCharacter: Unary<{ character_id: number }, GetCharacterAdminResponse>;
  grantItem: Unary<GrantCharacterItemInput, GrantCharacterItemResponse>;
  listCharacterGrants: Unary<{ character_id: number }, ListCharacterGrantsResponse>;
}>("CharacterAdminService");

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

/**
 * `promisify`/`Unary` moraram aqui até a issue 21; agora vivem em
 * `src/server/promisify.ts` (deadline via `CallOptions`) e são reexportados
 * para os imports existentes (`@/server/grpc`) continuarem válidos.
 */
export { promisify };
export type { Unary };
