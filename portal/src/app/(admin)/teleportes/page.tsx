import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import { type TeleportRoute } from "@/lib/teleport";
import { zoneAt, type ZoneBox } from "@/lib/zone";
import { getWorldAtlas } from "@/server/atlas";
import { AdminError } from "@/server/errors";
import { listTeleportRoutes } from "@/server/teleports";

import { TeleportTable } from "./teleport-table";

export const dynamic = "force-dynamic";

export const metadata = { title: "Teleportes · WYD2" };

/**
 * Rotas de teleporte: origem → destino, custo e requisito (issue 42).
 *
 * A lista vem **mesclada** do web-api (tabela compilada + overlay), então o
 * moderador vê as 23 rotas mesmo sem o seed. O aviso de aplicação é fixo porque o
 * efeito depende de uma flag do tmserver, não do portal.
 */
export default async function TeleportsPage() {
  let routes: TeleportRoute[] = [];
  let error: string | null = null;

  try {
    routes = await listTeleportRoutes();
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível carregar as rotas de teleporte do web-api";
  }

  // Nome do lugar de origem/destino (issue 47): o atlas já publica as caixas das
  // cidades e das regiões, então a tela não precisa de tabela nova. Sem atlas
  // gerado, a tabela volta a mostrar só as coordenadas — nunca quebra.
  const atlas = await getWorldAtlas();
  if (atlas) {
    const boxes: ZoneBox[] = [
      ...atlas.cities.map((city) => ({
        name: city.name,
        x1: city.limit_x1,
        y1: city.limit_y1,
        x2: city.limit_x2,
        y2: city.limit_y2,
      })),
      ...atlas.regions.map((region) => ({
        name: region.name,
        x1: region.x1,
        y1: region.y1,
        x2: region.x2,
        y2: region.y2,
      })),
    ];
    routes = routes.map((route) => ({
      ...route,
      src_zone: zoneAt(boxes, route.src_x, route.src_y),
      dst_zone: zoneAt(boxes, route.dst_x, route.dst_y),
    }));
  }

  const overrides = routes.filter((route) => route.has_override).length;
  const disabled = routes.filter((route) => !route.enabled).length;

  return (
    <>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            Teleportes
          </h1>
          <p className="ui-page-sub" style={{ margin: "4px 0 0" }}>
            Origem → destino, custo e requisito. O servidor resolve pelo tile arredondado
            (<code>x &amp;^ 3</code>) e soma 0–2 de spread no destino. Os nomes de lugar vêm do
            atlas do mundo; as coordenadas ficam no detalhe.
          </p>
        </div>
      </div>

      <div className="ui-card" style={{ marginBottom: 12 }}>
        <strong>Aplicação</strong>
        <p style={{ margin: "4px 0 0" }}>
          Liga no tmserver com <code>W2PP_TELEPORT_EDITING=true</code>: o overlay é lido no boot
          e <strong>re-lido a cada 15 s</strong>, então uma edição chega ao mundo sem restart. Sem
          a flag, o mundo usa só a tabela compilada.
        </p>
      </div>

      {error && (
        <div className="ui-card ui-empty" style={{ marginBottom: 12 }}>
          <strong>Não deu para listar as rotas</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      )}

      {!error && (
        <>
          <p className="ui-field__hint" style={{ marginBottom: 10 }}>
            {routes.length} rotas · {overrides} com linha no banco · {disabled} desativadas ·{" "}
            <Badge tone="warn">conteúdo</Badge> não pode ser apagado, só desativado
          </p>
          <TeleportTable routes={routes} />
        </>
      )}
    </>
  );
}
