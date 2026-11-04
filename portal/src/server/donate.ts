/**
 * Leitura do `DonateAdminService` (issue 29).
 *
 * `ListShopItems` devolve **todas** as ofertas (habilitadas ou não) — é a tabela
 * de moderação. A vitrine do jogador é do `DonateShopService`, que lê só as
 * habilitadas; o portal nunca escolhe o que o jogador vê por conta própria.
 */
import { emptyShopItem, type DonateShopItem } from "@/lib/donate";

import { adminCall } from "./admin";
import { donateAdmin, type ListShopItemsResponse } from "./grpc";

function normalize(item: Partial<DonateShopItem>): DonateShopItem {
  const base = emptyShopItem();
  return {
    id: Number(item.id ?? 0),
    item_index: Number(item.item_index ?? 0),
    eff1: Number(item.eff1 ?? base.eff1),
    effv1: Number(item.effv1 ?? base.effv1),
    eff2: Number(item.eff2 ?? base.eff2),
    effv2: Number(item.effv2 ?? base.effv2),
    eff3: Number(item.eff3 ?? base.eff3),
    effv3: Number(item.effv3 ?? base.effv3),
    price: Number(item.price ?? 0),
    title: item.title ?? "",
    description: item.description ?? "",
    enabled: Boolean(item.enabled),
    expires_days: Number(item.expires_days ?? 0),
  };
}

export async function getShopItems(): Promise<DonateShopItem[]> {
  const res = await adminCall<Record<string, never>, ListShopItemsResponse>(
    donateAdmin.listShopItems.bind(donateAdmin),
    {},
    { action: "donate.list" },
  );
  return (res.items ?? []).map(normalize);
}
