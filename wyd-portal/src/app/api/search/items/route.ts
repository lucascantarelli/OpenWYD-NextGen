import { fmtCoin } from "@/lib/format";
import { normalizeSearch } from "@/lib/itemsearch";
import { getItemCatalog } from "@/server/catalog";
import { getBasePrices } from "@/server/prices";
import { getSession } from "@/server/session";

export const dynamic = "force-dynamic";

const LIMIT = 20;

/**
 * Busca de itens para o `Combobox` (nome ou índice). Só com sessão: o catálogo
 * é grande e não precisa ficar aberto a qualquer visitante.
 */
export async function GET(request: Request) {
  const session = await getSession();
  if (!session) {
    return Response.json({ error: "sessão expirada" }, { status: 401 });
  }

  const { searchParams } = new URL(request.url);
  const query = normalizeSearch((searchParams.get("q") ?? "").trim());

  try {
    const catalog = await getItemCatalog();
    const base = await getBasePrices();
    const matches = catalog.items.filter((item) => {
      if (!query) return true;
      if (String(item.item_index) === query) return true;
      return (
        normalizeSearch(item.name).includes(query) ||
        normalizeSearch(item.display_name ?? "").includes(query)
      );
    });

    return Response.json(
      {
        catalogVersion: catalog.version,
        options: matches.slice(0, LIMIT).map((item) => ({
          value: String(item.item_index),
          label: item.display_name || item.name,
          hint:
            base && base[String(item.item_index)] !== undefined
              ? `#${item.item_index} · base ${fmtCoin(base[String(item.item_index)])}`
              : `#${item.item_index}`,
        })),
      },
      { headers: { "cache-control": "private, no-store" } },
    );
  } catch {
    return Response.json({ error: "catálogo indisponível" }, { status: 502 });
  }
}
