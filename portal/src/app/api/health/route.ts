import { itemCatalog, promisify, type ListItemsResponse } from "@/server/grpc";

export const dynamic = "force-dynamic";

/**
 * Prova viva do caminho portal → gRPC → web-api.
 * 200 com a contagem real do catálogo; 502 quando o web-api não responde.
 */
export async function GET() {
  try {
    const res = await promisify<Record<string, never>, ListItemsResponse>(
      itemCatalog.listItems.bind(itemCatalog),
    )({});
    return Response.json({
      ok: true,
      items: res.items?.length ?? 0,
      catalogVersion: res.catalog_version ?? "",
      iconPackVersion: res.icon_pack_version ?? "",
    });
  } catch {
    return Response.json(
      { ok: false, error: "web-api indisponível" },
      { status: 502 },
    );
  }
}
