import { getWorldAtlas } from "@/server/atlas";
import { getSession } from "@/server/session";

export const dynamic = "force-dynamic";

const LAYERS = ["cities", "regions", "teleports", "static_items", "spawns", "all"] as const;
type Layer = (typeof LAYERS)[number];

function csv(rows: (string | number)[][]): string {
  return (
    rows
      .map((row) =>
        row
          .map((cell) => `"${String(cell).replace(/"/g, '""')}"`)
          .join(","),
      )
      .join("\r\n") + "\r\n"
  );
}

/** Export das camadas do atlas em CSV (por camada) ou JSON (tudo). */
export async function GET(request: Request) {
  const session = await getSession();
  if (!session) return Response.json({ error: "sessão expirada" }, { status: 401 });

  const atlas = await getWorldAtlas();
  if (!atlas) return Response.json({ error: "atlas não gerado" }, { status: 404 });

  const { searchParams } = new URL(request.url);
  const layer = (searchParams.get("layer") ?? "regions") as Layer;
  const format = searchParams.get("format") ?? "csv";
  if (!LAYERS.includes(layer)) {
    return Response.json({ error: `camada inválida: ${layer}` }, { status: 422 });
  }

  if (format === "json") {
    return Response.json(layer === "all" ? atlas : atlas[layer], {
      headers: { "cache-control": "private, no-store" },
    });
  }

  let rows: (string | number)[][] = [];
  switch (layer) {
    case "cities":
      rows = [
        ["name", "spawn_x", "spawn_y", "limit_x1", "limit_y1", "limit_x2", "limit_y2", "war_x1", "war_y1", "war_x2", "war_y2"],
        ...atlas.cities.map((c) => [
          c.name,
          c.spawn_x,
          c.spawn_y,
          c.limit_x1,
          c.limit_y1,
          c.limit_x2,
          c.limit_y2,
          c.war_x1,
          c.war_y1,
          c.war_x2,
          c.war_y2,
        ]),
      ];
      break;
    case "regions":
      rows = [
        ["name", "x1", "y1", "x2", "y2", "source"],
        ...atlas.regions.map((r) => [r.name, r.x1, r.y1, r.x2, r.y2, r.source]),
      ];
      break;
    case "teleports":
      rows = [
        ["src_x", "src_y", "dst_x", "dst_y", "cost", "requirement"],
        ...atlas.teleports.map((t) => [t.src_x, t.src_y, t.dst_x, t.dst_y, t.cost, t.requirement]),
      ];
      break;
    case "static_items":
      rows = [
        ["order", "item_index", "x", "y", "rotate"],
        ...atlas.static_items.map((i) => [i.order, i.item_index, i.x, i.y, i.rotate]),
      ];
      break;
    case "spawns":
      rows = [
        ["generator", "leader", "follower", "x", "y", "range", "max_mob", "route_type"],
        ...atlas.spawns.map((s) => [
          s.generator,
          s.leader,
          s.follower,
          s.x,
          s.y,
          s.range,
          s.max_mob,
          s.route_type,
        ]),
      ];
      break;
    default:
      return Response.json(
        { error: "para a camada all use format=json" },
        { status: 422 },
      );
  }

  return new Response(csv(rows), {
    headers: {
      "content-type": "text/csv; charset=utf-8",
      "content-disposition": `attachment; filename="atlas-${layer}.csv"`,
      "cache-control": "private, no-store",
    },
  });
}
