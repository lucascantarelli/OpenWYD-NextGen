/**
 * Cache do catálogo de itens (3.2k entradas), compartilhado por processo/rota.
 *
 * O catálogo é conteúdo **imutável** no web-api, versionado por
 * `catalog_version`: uma chamada `ListItems` por versão basta. `unstable_cache`
 * guarda a resposta serializada na tag `item-catalog` (5 min de revalidação);
 * o índice `byIndex` é derivado depois, porque `Map` não é serializável.
 */
import { unstable_cache } from "next/cache";

import {
  itemCatalog,
  promisify,
  type CatalogItem,
  type ItemEffectType,
  type ListItemsResponse,
} from "./grpc";

export const CATALOG_TAG = "item-catalog";

export interface CatalogSnapshot {
  version: string;
  iconPackVersion: string;
  items: CatalogItem[];
  fetchedAt: number;
}

export interface Catalog {
  items: CatalogItem[];
  byIndex: Map<number, CatalogItem>;
  catalogVersion: string;
  iconPackVersion: string;
  /** Vocabulário de efeitos, para os seletores não reinventarem a tabela. */
  effectTypes: ItemEffectType[];
}

interface CatalogPayload {
  catalogVersion: string;
  iconPackVersion: string;
  items: CatalogItem[];
  effectTypes: ItemEffectType[];
}

const fetchCatalog = unstable_cache(
  async (): Promise<CatalogPayload> => {
    const res = await promisify<Record<string, never>, ListItemsResponse>(
      itemCatalog.listItems.bind(itemCatalog),
    )({});
    return {
      catalogVersion: res.catalog_version ?? "",
      iconPackVersion: res.icon_pack_version ?? "",
      items: res.items ?? [],
      effectTypes: res.effect_types ?? [],
    };
  },
  ["item-catalog-v1"],
  { revalidate: 300, tags: [CATALOG_TAG] },
);

/** Catálogo completo + índice por `item_index`. Erros do web-api sobem. */
export async function getCatalog(): Promise<Catalog> {
  const payload = await fetchCatalog();
  const byIndex = new Map<number, CatalogItem>();
  for (const item of payload.items) byIndex.set(item.item_index, item);
  return { ...payload, byIndex };
}

/** Formato usado pelo painel (nome/versão). */
export async function getItemCatalog(): Promise<CatalogSnapshot> {
  const catalog = await getCatalog();
  return {
    version: catalog.catalogVersion,
    iconPackVersion: catalog.iconPackVersion,
    items: catalog.items,
    fetchedAt: Date.now(),
  };
}

/** Um item por índice, ou `null` quando não existe no `ItemList.csv`. */
export async function getCatalogItem(itemIndex: number): Promise<CatalogItem | null> {
  if (!Number.isInteger(itemIndex)) return null;
  const { byIndex } = await getCatalog();
  return byIndex.get(itemIndex) ?? null;
}
