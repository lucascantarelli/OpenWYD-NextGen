/**
 * Leitura do `DailyRewardAdminService` (issue 30).
 *
 * `ListRewardItems` devolve **todas** as ofertas (habilitadas ou não) — é a
 * tabela de moderação, não a vitrine do jogador. Quem decide resgatar é o
 * `DailyRewardService` com o `account_id` da sessão do jogador, nunca o portal.
 */
import { emptyDailyReward, type DailyRewardItem } from "@/lib/daily";

import { adminCall } from "./admin";
import { dailyRewardAdmin, type ListRewardItemsResponse } from "./grpc";

function normalize(item: Partial<DailyRewardItem>): DailyRewardItem {
  const base = emptyDailyReward();
  return {
    id: Number(item.id ?? 0),
    item_index: Number(item.item_index ?? 0),
    eff1: Number(item.eff1 ?? base.eff1),
    effv1: Number(item.effv1 ?? base.effv1),
    eff2: Number(item.eff2 ?? base.eff2),
    effv2: Number(item.effv2 ?? base.effv2),
    eff3: Number(item.eff3 ?? base.eff3),
    effv3: Number(item.effv3 ?? base.effv3),
    title: item.title ?? "",
    description: item.description ?? "",
    enabled: Boolean(item.enabled),
    expires_days: Number(item.expires_days ?? 0),
  };
}

export async function getDailyRewards(): Promise<DailyRewardItem[]> {
  const res = await adminCall<Record<string, never>, ListRewardItemsResponse>(
    dailyRewardAdmin.listRewardItems.bind(dailyRewardAdmin),
    {},
    { action: "daily.list" },
  );
  return (res.items ?? []).map(normalize);
}
