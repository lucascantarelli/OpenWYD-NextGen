import { fmtCoin } from "@/lib/format";
import { getCatalog } from "@/server/catalog";
import { AdminError } from "@/server/errors";
import { getBasePrices, listItemPrices } from "@/server/prices";

import { PriceTable, type PriceRow } from "./price-table";

export const dynamic = "force-dynamic";

export const metadata = { title: "Preços globais · WYD2" };

/**
 * Overrides de preço global (`item_price`).
 *
 * Fonte primária é `ListItemPrices` — ela só devolve quem tem override, então a
 * tabela não precisa carregar 3.2k itens; o catálogo entra apenas para nome e
 * ícone, do cache já existente.
 */
export default async function PricesPage() {
  let rows: PriceRow[] = [];
  let catalogVersion = "";
  let baseAvailable = true;
  let error: string | null = null;

  try {
    const [prices, catalog] = await Promise.all([listItemPrices(), getCatalog()]);
    catalogVersion = catalog.catalogVersion;
    const base = await getBasePrices();
    baseAvailable = base !== null;

    rows = prices
      .map((entry) => ({
        index: entry.item_index,
        price: Number(entry.price),
        base: base ? (base[String(entry.item_index)] ?? null) : null,
        item: catalog.byIndex.get(entry.item_index) ?? null,
      }))
      .sort((a, b) => a.index - b.index);
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível ler os preços globais do web-api";
  }

  const above = rows.filter((row) => row.base !== null && row.price > row.base).length;

  return (
    <>
      <h1 className="ui-page-title">Preços globais</h1>
      <p className="ui-page-sub">
        O preço é <strong>global por item</strong>, não por loja: vale em todos os mercadores que não
        fixam preço. Item sem override usa a base do <code>ItemList.csv</code>.
      </p>

      {!baseAvailable && (
        <p className="ui-banner" style={{ marginBottom: 12 }} role="note">
          O preço base não pôde ser lido: o portal não encontrou{" "}
          <code>tmserver/content/Common/ItemList.csv</code>. Ajuste <code>W2PP_CONTENT</code> se o conteúdo
          estiver em outro caminho — editar preço continua funcionando.
        </p>
      )}

      {error ? (
        <div className="ui-card ui-empty">
          <strong>Preços indisponíveis</strong>
          <p>{error}</p>
        </div>
      ) : (
        <>
          {rows.length > 0 && (
            <p className="ui-field__hint" style={{ marginBottom: 10 }}>
              {rows.length} overrides · {above} acima da base · total em uso:{" "}
              {fmtCoin(rows.reduce((sum, row) => sum + row.price, 0))}
            </p>
          )}
          <PriceTable rows={rows} catalogVersion={catalogVersion} />
        </>
      )}
    </>
  );
}
