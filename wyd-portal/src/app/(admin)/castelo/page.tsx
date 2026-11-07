import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import { formatUnix, type WorldState } from "@/lib/worldstate";
import { AdminError } from "@/server/errors";
import { getWorldState } from "@/server/worldstate";

import { CastlePanel } from "./castle-panel";

export const dynamic = "force-dynamic";

export const metadata = { title: "Castelo e Kefra · WYD2" };

/**
 * Cidades, torre, castelo e Kefra (issue 45).
 *
 * A taxa da cidade e o flag do Kefra são as duas escritas desta fatia. Ambas são
 * estado de banco que o tmserver lê no boot, então cada card diz isso na cara —
 * prometer efeito imediato seria mentir sobre o modelo de estado do servidor.
 */
export default async function CastlePage() {
  let state: WorldState | null = null;
  let error: string | null = null;

  try {
    state = await getWorldState();
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível carregar o estado do mundo no web-api";
  }

  return (
    <>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            Castelo e Kefra
          </h1>
          <p className="ui-page-sub" style={{ margin: "4px 0 0" }}>
            Cidades do reino, taxa da loja pessoal, dono da torre, quest do castelo e o ciclo do
            Kefra que destrava o portal do Deserto.
          </p>
        </div>
      </div>

      {error && (
        <div className="ui-card ui-empty" style={{ marginBottom: 12 }}>
          <strong>Não deu para carregar o estado do mundo</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      )}

      {!error && state && (
        <>
          <div className="ui-card" style={{ marginBottom: 12 }}>
            <strong>Aplicação</strong>
            <p style={{ margin: "4px 0 0" }}>
              Taxa da cidade e estado do Kefra vivem na memória do tmserver e voltam do banco no
              boot: uma edição só vale depois de <code>docker compose restart tmserver</code>. O
              portal avisa o moderador, não reinicia nada sozinho.
            </p>
          </div>

          <div className="ui-filters" style={{ marginBottom: 10 }}>
            <p className="ui-field__hint" style={{ margin: 0 }}>
              torre:{" "}
              {state.tower_owner_name ? (
                <Badge tone="accent">{state.tower_owner_name}</Badge>
              ) : (
                "sem dono"
              )}
              {state.tower_updated_unix > 0 && ` · desde ${formatUnix(state.tower_updated_unix)}`}
            </p>
          </div>

          <CastlePanel
            zones={state.zones}
            guilds={state.guilds}
            kefra={state.kefra}
            castle={state.castle}
            towerOwnerGuild={state.tower_owner_guild}
          />
        </>
      )}
    </>
  );
}
