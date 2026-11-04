import { RetryButton } from "@/components/RetryButton";
import { getCatalog } from "@/server/catalog";
import { AdminError } from "@/server/errors";
import { getWorldEvent, type WorldEventSnapshot } from "@/server/events";

import { EventPanel, type EventItem } from "./event-panel";

export const dynamic = "force-dynamic";

export const metadata = { title: "Eventos globais · WYD2" };

/**
 * Evento global de mundo (issue 28).
 *
 * A tela é config fria: `WorldEventAdminService` grava em `world_event_config` e
 * o tmserver relê por polling (15 s) e aplica dentro do loop — sem restart, ao
 * contrário das flags `-double-exp`/`-newbie-event`, que são de boot.
 *
 * O nome do item é resolvido aqui, no BFF: o browser nunca vê 3,2 mil linhas de
 * catálogo nem recebe um índice cru como rótulo (issue 47).
 */
export default async function EventsPage() {
  let snapshot: WorldEventSnapshot | null = null;
  let item: EventItem | null = null;
  let error: string | null = null;

  try {
    snapshot = await getWorldEvent();
    if (snapshot.config.item_index > 0) {
      try {
        const catalog = await getCatalog();
        const found = catalog.byIndex.get(snapshot.config.item_index);
        item = found
          ? { index: found.item_index, name: found.display_name || found.name }
          : null;
      } catch {
        // Catálogo fora não impede editar o evento: a tela cai no índice com `#`.
        item = null;
      }
    }
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível ler o evento global do web-api";
  }

  return (
    <>
      <h1 className="ui-page-title">Eventos globais</h1>
      <p className="ui-page-sub">
        Evento de drop com item serializado, anúncio no chat e os multiplicadores de EXP. O portal
        grava configuração fria no Postgres e o tmserver relê a cada 15 segundos — nenhuma alteração
        aqui pede restart.
      </p>

      {error ? (
        <div className="ui-card ui-empty">
          <strong>Evento indisponível</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      ) : (
        snapshot && (
          <EventPanel
            key={`v${snapshot.version}`}
            config={snapshot.config}
            version={snapshot.version}
            item={item}
          />
        )
      )}
    </>
  );
}
