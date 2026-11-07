import { RetryButton } from "@/components/RetryButton";
import { getCatalog } from "@/server/catalog";
import { getShopItems } from "@/server/donate";
import { AdminError } from "@/server/errors";
import type { ItemEffectType } from "@/server/grpc";

import { ShopTable, type ShopRow } from "./shop-table";

export const dynamic = "force-dynamic";

export const metadata = { title: "Loja de donate · WYD2" };

/**
 * Catálogo da loja de donate (issue 29).
 *
 * O portal edita o catálogo frio (`donate_shop_item`) e credita saldo manual
 * auditado. Comprar é do `DonateShopService`, com o `account_id` da sessão do
 * jogador; a entrega vai por `delivery_queue` e só acontece no próximo login.
 * Faturamento (pedidos, compradores, KPIs) é a issue 31 e não mora aqui.
 */
export default async function DonateShopPage() {
  let rows: ShopRow[] = [];
  let effectTypes: ItemEffectType[] = [];
  let iconPackVersion = "";
  let error: string | null = null;

  try {
    const items = await getShopItems();
    try {
      const catalog = await getCatalog();
      iconPackVersion = catalog.iconPackVersion;
      effectTypes = catalog.effectTypes;
      rows = items.map((item) => ({
        ...item,
        item: catalog.byIndex.get(item.item_index) ?? null,
      }));
    } catch {
      // Catálogo fora não impede editar: a linha cai no `item #N`.
      rows = items.map((item) => ({ ...item, item: null }));
    }
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível ler o catálogo da loja de donate do web-api";
  }

  return (
    <>
      <h1 className="ui-page-title">Loja de donate</h1>

      {error ? (
        <div className="ui-card ui-empty">
          <strong>Loja de donate indisponível</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      ) : (
        <ShopTable rows={rows} effectTypes={effectTypes} iconPackVersion={iconPackVersion} />
      )}
    </>
  );
}
