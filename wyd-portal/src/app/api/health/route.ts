import { itemCatalog, promisify, type ListItemsResponse } from "@/server/admin-client";

export const dynamic = "force-dynamic";

/**
 * Saúde do portal (liveness) e do caminho até o gateway.
 *
 * O endpoint nunca devolve 502 por causa do backend: portal no ar e gateway fora
 * são estados diferentes, e quem monitora precisa distinguir os dois.
 */
export async function GET() {
  let gateway: "up" | "down" = "down";
  let items: number | null = null;
  try {
    const res = await promisify<Record<string, never>, ListItemsResponse>(
      itemCatalog.listItems.bind(itemCatalog),
    )({});
    gateway = "up";
    items = res.items?.length ?? 0;
    return Response.json({ ok: true, gateway, items });
  } catch {
    return Response.json({ ok: true, gateway, items });
  }
}
