/**
 * Catálogo de itens — leitura do gateway (`item_catalog/list_items`).
 *
 * A foto é carregada do Postgres no boot do gateway. Cache de processo por 60 s
 * evita refazer a mesma leitura a cada navegação; o conteúdo só muda com novo
 * boot, então não há invalidação a inventar.
 */
import { adminCall } from "./admin";
import { itemCatalog, type CatalogItem, type ListItemsResponse } from "./admin-client";

const TTL_MS = 60_000;

export interface Catalog {
  items: CatalogItem[];
  catalogVersion: string;
}

let cached: { value: Catalog; at: number } | null = null;

export async function getCatalog(): Promise<Catalog> {
  if (cached && Date.now() - cached.at < TTL_MS) return cached.value;

  const res = await adminCall<Record<string, never>, ListItemsResponse>(
    itemCatalog.listItems.bind(itemCatalog),
    {},
    { action: "catalog.list", target: "content" },
  );
  const value: Catalog = {
    items: res.items ?? [],
    catalogVersion: res.catalog_version ?? "",
  };
  cached = { value, at: Date.now() };
  return value;
}
