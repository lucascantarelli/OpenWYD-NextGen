"use server";

/**
 * Escrita do preço global de um item.
 *
 * `price >= 0` define o override; `price < 0` (sentinela `-1`) limpa e o item
 * volta ao preço do `ItemList.csv`. Não há preço por loja — a loja do NPC
 * (issue 23) não edita preço.
 */
import { CLEAR_PRICE, priceSchema, type PriceInput } from "@/lib/price";

import { adminCall } from "@/server/admin";
import { AdminError, isInfraFailure } from "@/server/errors";
import { npcAdmin, type AdminAck } from "@/server/admin-client";

export type PriceActionResult = { ok: true } | { ok: false; error: string; status: number };

export async function setItemPrice(input: PriceInput): Promise<PriceActionResult> {
  const parsed = priceSchema.safeParse(input);
  if (!parsed.success) {
    return { ok: false, status: 422, error: parsed.error.issues[0]?.message ?? "dados inválidos" };
  }

  const clearing = parsed.data.price === CLEAR_PRICE;

  try {
    await adminCall<{ item_index: number; price: number }, AdminAck>(
      npcAdmin.setItemPrice.bind(npcAdmin),
      { item_index: parsed.data.itemIndex, price: parsed.data.price },
      {
        action: clearing ? "item.price.clear" : "item.price.set",
        target: `item:${parsed.data.itemIndex}`,
      },
    );
    return { ok: true };
  } catch (error) {
    if (error instanceof AdminError) {
      return { ok: false, error: error.message, status: error.status };
    }
    if (isInfraFailure(error)) {
      return { ok: false, error: "serviço de administração indisponível", status: 502 };
    }
    console.error("[price] falha inesperada:", error instanceof Error ? error.message : String(error));
    return { ok: false, error: "erro interno do portal", status: 500 };
  }
}
