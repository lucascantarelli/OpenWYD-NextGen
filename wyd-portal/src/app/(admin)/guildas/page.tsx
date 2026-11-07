import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import { formatUnix, type Guild, type GuildRelation, type WorldState } from "@/lib/worldstate";
import { AdminError } from "@/server/errors";
import { getWorldState } from "@/server/worldstate";

import { GuildsTable } from "./guilds-table";

export const dynamic = "force-dynamic";

export const metadata = { title: "Guildas · WYD2" };

/**
 * Guildas e relações (issue 45, fatia de leitura).
 *
 * O snapshot vem inteiro do web-api: cada guilda já chega com membros, mestre e
 * cidades. Guildas/relações são estado de banco carregado no boot do tmserver —
 * por isso o aviso fixo de restart, e por isso nenhuma ação aqui promete efeito
 * imediato no mundo.
 */
export default async function GuildsPage() {
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

  const guilds: Guild[] = state?.guilds ?? [];
  const relations: GuildRelation[] = state?.relations ?? [];

  return (
    <>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            Guildas
          </h1>
          <p className="ui-page-sub" style={{ margin: "4px 0 0" }}>
            Guildas, membros, mestre e relações (aliada/inimiga). Dono da torre e cidades em{" "}
            <code>/castelo</code>.
          </p>
        </div>
      </div>

      <div className="ui-card" style={{ marginBottom: 12 }}>
        <strong>Aplicação</strong>
        <p style={{ margin: "4px 0 0" }}>
          O mundo carrega guildas no boot: mudanças feitas aqui chegam ao jogo depois de{" "}
          <code>docker compose restart tmserver</code>. Com jogadores online, avise antes —
          o restart salva os personagens e derruba as sessões.
        </p>
      </div>

      {error && (
        <div className="ui-card ui-empty" style={{ marginBottom: 12 }}>
          <strong>Não deu para listar as guildas</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      )}

      {!error && state && (
        <>
          <p className="ui-field__hint" style={{ marginBottom: 10 }}>
            {guilds.length} guildas · {guilds.reduce((sum, g) => sum + g.members.length, 0)} membros ·{" "}
            {relations.length} relações · torre:{" "}
            {state.tower_owner_name ? (
              <Badge tone="accent">{state.tower_owner_name}</Badge>
            ) : (
              <span>sem dono</span>
            )}
            {state.tower_updated_unix > 0 && (
              <span> (desde {formatUnix(state.tower_updated_unix)})</span>
            )}
          </p>
          <GuildsTable guilds={guilds} relations={relations} />
        </>
      )}
    </>
  );
}
