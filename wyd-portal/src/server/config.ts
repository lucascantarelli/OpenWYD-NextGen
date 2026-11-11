/**
 * Configuração do servidor (FASE-07-6) — taxas, drops, NPC e teleporte.
 *
 * Tudo passa por `adminCall`: sessão de staff, `moderator_id` da sessão, auditoria e
 * mapeamento `AdminResult` → HTTP. O portal não fala com o banco: lê e grava pelo
 * gateway, e cada escrita incrementa a revisão que o servidor de jogo observa.
 */
import { adminCall } from "./admin";
import {
  contentRates,
  dropAdmin,
  npcAdmin,
  teleportAdmin,
  type ItemPriceWire,
  type AccessoryRuleWire,
  type DropWire,
  type GetNpcResponse,
  type GetRatesResponse,
  type ListDropsResponse,
  type ListItemPricesResponse,
  type ListRoutesResponse,
  type NpcWire,
  type RateWire,
  type RevisionResponse,
  type RouteWire,
} from "./admin-client";

export type { AccessoryRuleWire, DropWire, ItemPriceWire, NpcWire, RateWire, RouteWire };

/** Teto de preços por leitura (o mesmo do gateway, `api::npc::MAX_PRICES`). */
export const MAX_PRICES = 200;

/**
 * Valida a forma de um preço de NPC antes do `adminCall`.
 *
 * Espelha o gateway (`api::npc::validar_preco`): item e mercador obrigatórios (a chave
 * é o par), preço não-negativo e fonte obrigatória — preço sem procedência não grava.
 */
export function checkItemPrice(input: {
  itemIndex: number;
  npcId: number;
  buy: number;
  sell: number;
  source: string;
}): string | null {
  if (!Number.isInteger(input.itemIndex) || input.itemIndex <= 0) {
    return "índice do item inválido";
  }
  if (!Number.isInteger(input.npcId) || input.npcId <= 0) {
    return "NPC do preço é obrigatório";
  }
  if (!Number.isInteger(input.buy) || input.buy < 0) return "preço de compra inválido";
  if (!Number.isInteger(input.sell) || input.sell < 0) return "preço de venda inválido";
  const source = input.source.trim();
  if (source === "") return "fonte do preço obrigatória";
  if (source.length > 160) return "fonte do preço longa demais";
  return null;
}

/**
 * Preços de NPC definidos pelo dono do servidor (`npc_admin/list_item_prices`).
 *
 * Lista vazia é resposta honesta: a tabela `item_price` nasce vazia e significa
 * "nenhum preço definido" — a tela mostra ausência, nunca zero.
 */
export async function listItemPrices(itemIndex?: number): Promise<ItemPriceWire[]> {
  const res = await adminCall<{ item_index?: number; limit: number }, ListItemPricesResponse>(
    npcAdmin.listItemPrices.bind(npcAdmin),
    {
      limit: MAX_PRICES,
      ...(Number.isInteger(itemIndex) && (itemIndex as number) > 0
        ? { item_index: itemIndex }
        : {}),
    },
    { action: "config.prices", target: itemIndex ? `item:${itemIndex}` : "prices" },
  );
  return res.prices ?? [];
}

/** Grava o preço de (item, NPC) e devolve a revisão. */
export async function upsertItemPrice(input: {
  itemIndex: number;
  npcId: number;
  buy: number;
  sell: number;
  source: string;
}): Promise<number> {
  const res = await adminCall<
    {
      item_index: number;
      npc_id: number;
      buy_price: number;
      sell_price: number;
      source: string;
    },
    RevisionResponse
  >(
    npcAdmin.setItemPrice.bind(npcAdmin),
    {
      item_index: input.itemIndex,
      npc_id: input.npcId,
      buy_price: input.buy,
      sell_price: input.sell,
      source: input.source.trim(),
    },
    {
      action: "config.price.set",
      target: `item${input.itemIndex}@npc${input.npcId}`,
    },
  );
  return res.revision ?? 0;
}

export interface RatesView {
  revision: number;
  rates: RateWire[];
  accessories: AccessoryRuleWire[];
}

/** Foto completa da configuração de taxas. */
export async function getRates(): Promise<RatesView> {
  const res = await adminCall<Record<string, never>, GetRatesResponse>(
    contentRates.getRates.bind(contentRates),
    {},
    { action: "config.rates", target: "rates" },
  );
  return {
    revision: res.revision ?? 0,
    rates: res.rates ?? [],
    accessories: res.accessories ?? [],
  };
}

/** Grava uma taxa; a fonte é a procedência que fica na trilha. */
export async function upsertRate(input: {
  domain: string;
  key: string;
  value: number;
  source: string;
}): Promise<number> {
  const res = await adminCall<
    { domain: string; key: string; value: number; source: string },
    RevisionResponse
  >(contentRates.upsertRate.bind(contentRates), input, {
    action: "config.rate.set",
    target: `${input.domain}:${input.key}`,
  });
  return res.revision ?? 0;
}

/** Remove o override e volta ao default documentado. */
export async function resetRate(domain: string, key: string): Promise<number> {
  const res = await adminCall<{ domain: string; key: string }, RevisionResponse>(
    contentRates.resetRate.bind(contentRates),
    { domain, key },
    { action: "config.rate.reset", target: `${domain}:${key}` },
  );
  return res.revision ?? 0;
}

/** Pesos de drop (filtro opcional por template). */
export async function listDrops(template?: string): Promise<DropWire[]> {
  const clean = template?.trim().slice(0, 120) ?? "";
  const res = await adminCall<{ template_name?: string; limit: number }, ListDropsResponse>(
    dropAdmin.listDropRates.bind(dropAdmin),
    { ...(clean ? { template_name: clean } : {}), limit: 50 },
    { action: "config.drops", target: clean || "*" },
  );
  return res.drops ?? [];
}

export async function upsertDrop(input: {
  template_name: string;
  item_index: number;
  weight: number;
  min_qty: number;
  max_qty: number;
  source: string;
}): Promise<number> {
  const res = await adminCall<
    {
      template_name: string;
      item_index: number;
      weight: number;
      min_qty: number;
      max_qty: number;
      source: string;
    },
    RevisionResponse
  >(dropAdmin.upsertDropRate.bind(dropAdmin), input, {
    action: "config.drop.set",
    target: `${input.template_name}#${input.item_index}`,
  });
  return res.revision ?? 0;
}

export async function resetDrop(template: string, itemIndex: number): Promise<number> {
  const res = await adminCall<
    { template_name: string; item_index: number },
    RevisionResponse
  >(
    dropAdmin.resetDropRate.bind(dropAdmin),
    { template_name: template, item_index: itemIndex },
    { action: "config.drop.reset", target: `${template}#${itemIndex}` },
  );
  return res.revision ?? 0;
}

/** Um NPC por id; ausência não é erro de tela. */
export async function getNpc(id: number): Promise<NpcWire | null> {
  try {
    const res = await adminCall<{ id: number }, GetNpcResponse>(
      npcAdmin.getNpc.bind(npcAdmin),
      { id },
      { action: "config.npc", target: `npc:${id}` },
    );
    return res.npc ?? null;
  } catch {
    return null;
  }
}

export async function upsertNpc(input: {
  id: number;
  map_id: number;
  pos_x: number;
  pos_y: number;
  enabled: boolean;
}): Promise<number> {
  const res = await adminCall<
    { id: number; map_id: number; pos_x: number; pos_y: number; enabled: boolean },
    RevisionResponse
  >(npcAdmin.upsertNpc.bind(npcAdmin), input, {
    action: "config.npc.set",
    target: `npc:${input.id}`,
  });
  return res.revision ?? 0;
}

/** Rotas de teleporte cadastradas. */
export async function listRoutes(): Promise<RouteWire[]> {
  const res = await adminCall<{ limit: number }, ListRoutesResponse>(
    teleportAdmin.listRoutes.bind(teleportAdmin),
    { limit: 200 },
    { action: "config.teleport", target: "routes" },
  );
  return res.routes ?? [];
}

export async function upsertRoute(input: {
  id: number;
  cost_gold?: number;
  req_level?: number;
  enabled?: boolean;
}): Promise<number> {
  const res = await adminCall<
    { id: number; cost_gold?: number; req_level?: number; enabled?: boolean },
    RevisionResponse
  >(teleportAdmin.upsertRoute.bind(teleportAdmin), input, {
    action: "config.teleport.set",
    target: `route:${input.id}`,
  });
  return res.revision ?? 0;
}
