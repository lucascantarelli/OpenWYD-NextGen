import { ATLAS_COMMAND, getWorldAtlas } from "@/server/atlas";
import { getCatalog } from "@/server/catalog";

import { AtlasView } from "./atlas-view";

export const dynamic = "force-dynamic";

export const metadata = { title: "Atlas do mundo · WYD2" };

/**
 * Mapa do mundo: cidades, regiões, teleportes, spawns e itens fixos, tudo vindo
 * do artefato gerado por `webserver/cmd/worldatlas` (mesmas constantes do
 * servidor). Sem o artefato, a tela ensina a gerar em vez de estourar erro.
 */
export default async function AtlasPage() {
  const atlas = await getWorldAtlas();

  // Itens fixos e portões vêm do InitItem.csv só como índice; o nome sai do
  // catálogo já em cache (mesmo caminho de `/itens` e `/precos`). Sem catálogo a
  // tabela cai no índice — o mapa nunca deixa de abrir por causa disso.
  const itemNames: Record<string, string> = {};
  if (atlas) {
    try {
      const catalog = await getCatalog();
      for (const item of atlas.static_items) {
        const entry = catalog.byIndex.get(item.item_index);
        if (entry) itemNames[String(item.item_index)] = entry.display_name || entry.name;
      }
    } catch {
      // catálogo indisponível: segue com o índice
    }
  }

  return (
    <>
      <h1 className="ui-page-title">Atlas do mundo</h1>
      <p className="ui-page-sub">
        Onde fica cada coisa: {atlas ? atlas.counts.cities : 0} cidades,{" "}
        {atlas ? atlas.regions.length : 0} regiões nomeadas, {atlas ? atlas.teleports.length : 0}{" "}
        rotas de teleporte e {atlas ? atlas.spawns.length : 0} geradores de spawn. Coordenadas no grid
        único de 4096×4096.
      </p>

      {atlas === null ? (
        <div className="ui-card ui-empty">
          <strong>Atlas não gerado</strong>
          <p>
            O portal lê <code>dist/world-atlas.json</code> e <code>world-map.png</code>, que
            ainda não existem. Gere a partir do código e do conteúdo:
          </p>
          <p className="ui-mono">{ATLAS_COMMAND}</p>
          <p>
            Rode na raiz do projeto (ou no container Go, ver skill{" "}
            <code>wyd-server-ops</code>). O arquivo é read-only: nada é escrito em <code>tmserver/content/</code>.
          </p>
        </div>
      ) : (
        <AtlasView atlas={atlas} itemNames={itemNames} />
      )}
    </>
  );
}
