import { RetryButton } from "@/components/RetryButton";
import { getCatalog } from "@/server/catalog";
import { getDailyRewards } from "@/server/daily";
import { AdminError } from "@/server/errors";
import type { ItemEffectType } from "@/server/admin-client";

import { DailyTable, type DailyRewardRow } from "./daily-table";

export const dynamic = "force-dynamic";

export const metadata = { title: "Recompensa diária · WYD2" };

/**
 * Catálogo da recompensa diária (issue 30).
 *
 * Só o catálogo é editável aqui — resgate e entrega são do `DailyRewardService`
 * (account da sessão do jogador) e do `delivery_queue` drenado pelo tmserver.
 * O painel de "resgates de hoje" ficou para a issue 31: não existe RPC para
 * contar claims e o portal não fala Postgres direto.
 */
export default async function DailyRewardPage() {
  let rows: DailyRewardRow[] = [];
  let effectTypes: ItemEffectType[] = [];
  let iconPackVersion = "";
  let error: string | null = null;

  try {
    const rewards = await getDailyRewards();
    try {
      const catalog = await getCatalog();
      iconPackVersion = catalog.iconPackVersion;
      effectTypes = catalog.effectTypes;
      rows = rewards.map((reward) => ({
        ...reward,
        item: catalog.byIndex.get(reward.item_index) ?? null,
      }));
    } catch {
      // Catálogo fora não impede editar: a linha cai no `item #N`.
      rows = rewards.map((reward) => ({ ...reward, item: null }));
    }
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível ler o catálogo de recompensa diária do web-api";
  }

  return (
    <>
      <h1 className="ui-page-title">Recompensa diária</h1>

      {error ? (
        <div className="ui-card ui-empty">
          <strong>Recompensa diária indisponível</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      ) : (
        <DailyTable rows={rows} effectTypes={effectTypes} iconPackVersion={iconPackVersion} />
      )}
    </>
  );
}
