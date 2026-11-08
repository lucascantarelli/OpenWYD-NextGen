/**
 * Leituras de drop: relatórios item ↔ mob (issue 26) e tabela de taxa por slot
 * (issue 34).
 *
 * O relatório é caro — sem filtro o web-api devolve o catálogo inteiro de pares
 * (milhares de linhas sobre 3,3 mil itens e 2 mil templates). Por isso a busca é
 * server-side, exige filtro (a action cobra ≥2 caracteres) e o resultado fica em
 * cache por processo, com teto de 64 filtros distintos; o próximo boot do portal
 * esvazia.
 */
import type { DropRateSlot } from "@/lib/droprate";
import type { ItemDropEntry, MobDropEntry } from "@/lib/drop";

import { adminCall } from "./admin";
import {
  dropAdmin,
  npcAdmin,
  type ListDropItemsResponse,
  type ListDropRatesResponse,
  type ListMobDropsResponse,
} from "./admin-client";

/** Teto de filtros guardados: o suficiente para navegar e pequeno o bastante para não crescer. */
const CACHE_LIMIT = 64;

const itemCache = new Map<string, ItemDropEntry[]>();
const mobCache = new Map<string, MobDropEntry[]>();

function remember<T>(cache: Map<string, T[]>, key: string, value: T[]): T[] {
  cache.delete(key);
  cache.set(key, value);
  while (cache.size > CACHE_LIMIT) {
    const oldest = cache.keys().next().value;
    if (oldest === undefined) break;
    cache.delete(oldest);
  }
  return value;
}

export async function listItemDrops(input: {
  itemQuery: string;
  mobQuery?: string;
  includeZeroDropItems?: boolean;
}): Promise<ItemDropEntry[]> {
  const key = JSON.stringify([input.itemQuery, input.mobQuery ?? "", !!input.includeZeroDropItems]);
  const cached = itemCache.get(key);
  if (cached) return cached;

  const res = await adminCall<
    { item_index: number; item_query: string; mob_query: string; include_zero_drop_items: boolean },
    ListDropItemsResponse
  >(
    npcAdmin.listDropItems.bind(npcAdmin),
    {
      item_index: 0,
      item_query: input.itemQuery,
      mob_query: input.mobQuery ?? "",
      include_zero_drop_items: !!input.includeZeroDropItems,
    },
    { action: "drop.items", target: `drop:item:${input.itemQuery}` },
  );
  return remember(itemCache, key, res.items ?? []);
}

export async function listMobDrops(input: {
  mobQuery: string;
  itemIndex?: number;
}): Promise<MobDropEntry[]> {
  const itemIndex = input.itemIndex ?? 0;
  const key = JSON.stringify([input.mobQuery, itemIndex]);
  const cached = mobCache.get(key);
  if (cached) return cached;

  const res = await adminCall<
    { mob_query: string; item_index: number; item_query: string },
    ListMobDropsResponse
  >(
    npcAdmin.listMobDrops.bind(npcAdmin),
    { mob_query: input.mobQuery, item_index: itemIndex, item_query: "" },
    { action: "drop.mobs", target: `drop:mob:${input.mobQuery}` },
  );
  return remember(mobCache, key, res.mobs ?? []);
}

/** `ListDropRates` devolve os 64 slots com valor de código, efetivo e override. */
export async function listDropRates(): Promise<DropRateSlot[]> {
  const res = await adminCall<Record<string, never>, ListDropRatesResponse>(
    dropAdmin.listDropRates.bind(dropAdmin),
    {},
    { action: "drop.rates" },
  );
  return res.slots ?? [];
}
