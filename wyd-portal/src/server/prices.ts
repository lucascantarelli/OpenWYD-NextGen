/**
 * Preços globais: leitura dos overrides (`ListItemPrices`) e do preço base do
 * `ItemList.csv`.
 *
 * O preço base **não** vem por RPC — `ItemCatalogEntry` não tem preço. Ele é a
 * 6ª coluna do CSV, exatamente a que `content.ItemList.Prices()` usa
 * (`tmserver/internal/content/catalog.go:35`). Lemos o arquivo no host (o portal
 * roda ao lado do conteúdo), nunca escrevemos nele.
 */
import { readFile } from "node:fs/promises";
import path from "node:path";

import { unstable_cache } from "next/cache";

import { adminCall } from "./admin";
import { npcAdmin, type ItemPriceEntry, type ListItemPricesResponse } from "./grpc";

export const BASE_PRICE_TAG = "item-base-prices";

const CONTENT_DIR =
  process.env.W2PP_CONTENT ?? path.join(process.cwd(), "..", "tmserver", "content");

const loadBasePrices = unstable_cache(
  async (): Promise<Record<string, number>> => {
    const raw = await readFile(path.join(CONTENT_DIR, "Common", "ItemList.csv"), "utf8");
    const out: Record<string, number> = {};
    for (const line of raw.split(/\r?\n/)) {
      if (!line) continue;
      const fields = line.split(",");
      if (fields.length < 6) continue;
      const index = Number(fields[0]);
      const price = Number(fields[5]);
      if (Number.isInteger(index) && index > 0 && Number.isFinite(price)) {
        out[String(index)] = price;
      }
    }
    return out;
  },
  ["item-base-prices-v1"],
  { revalidate: 900, tags: [BASE_PRICE_TAG] },
);

/** Índice → preço base do CSV; `null` quando o conteúdo não está legível. */
export async function getBasePrices(): Promise<Record<string, number> | null> {
  try {
    return await loadBasePrices();
  } catch {
    return null;
  }
}

export async function listItemPrices(): Promise<ItemPriceEntry[]> {
  const res = await adminCall<Record<string, never>, ListItemPricesResponse>(
    npcAdmin.listItemPrices.bind(npcAdmin),
    {},
    { action: "item.price.list" },
  );
  return res.prices ?? [];
}
