import { fmtCoin } from "@/lib/format";
import { getCatalog, type Catalog } from "@/server/catalog";

import { ItemBrowser } from "./item-browser";

export const dynamic = "force-dynamic";

export const metadata = { title: "Catálogo de itens · WYD2" };

/**
 * Catálogo somente leitura dos 3.220 itens do `ItemList.csv`.
 *
 * Dois estados de exceção são distintos de propósito: web-api fora (não há
 * como ler) e catálogo **vazio** (serviço no ar sem `-content`). O segundo é
 * erro de configuração do servidor e diz exatamente o que subir.
 */
export default async function ItemsPage() {
  let catalog: Catalog | null = null;
  try {
    catalog = await getCatalog();
  } catch {
    catalog = null;
  }

  return (
    <>
      <h1 className="ui-page-title">Catálogo de itens</h1>
      <p className="ui-page-sub">
        Leitura de <code>Release/Common/ItemList.csv</code> via{" "}
        <code>ItemCatalogService.ListItems</code>. Somente leitura: preços são a issue 24 e a edição
        do catálogo é a 33.
      </p>

      {catalog === null && (
        <div className="ui-card ui-empty">
          <strong>Catálogo indisponível</strong>
          <p>
            O portal não conseguiu falar com o web-api (<code>:7600</code>). Confira se o serviço
            está no ar: <code>docker compose ps webserver</code> e{" "}
            <code>docker compose logs --tail 20 webserver</code>.
          </p>
        </div>
      )}

      {catalog !== null && catalog.items.length === 0 && (
        <div className="ui-card ui-empty">
          <strong>O web-api está no ar, mas sem catálogo</strong>
          <p>
            O serviço sobe sem itens quando não recebe a árvore de conteúdo. Suba com{" "}
            <code>-content /Release</code> (o <code>docker-compose.yaml</code> já faz isso) e
            reinicie o <code>webserver</code>. Enquanto o <code>ItemList.csv</code> não for lido, a
            lista fica vazia — não é erro do portal.
          </p>
        </div>
      )}

      {catalog !== null && catalog.items.length > 0 && (
        <>
          <p className="ui-field__hint" style={{ marginBottom: 10 }}>
            {fmtCoin(catalog.items.length)} itens · pack de ícones{" "}
            {catalog.iconPackVersion ? catalog.iconPackVersion.slice(0, 8) : "ausente (fallback)"}
          </p>
          <ItemBrowser
            items={catalog.items}
            iconPackVersion={catalog.iconPackVersion}
            catalogVersion={catalog.catalogVersion}
          />
        </>
      )}
    </>
  );
}
