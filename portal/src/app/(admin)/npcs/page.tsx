import Link from "next/link";

import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import { NpcTable } from "@/components/npc/npc-table";
import type { MapZone, NpcDefinition } from "@/lib/npc";
import { AdminError } from "@/server/errors";
import { listMapZones, listNpcs } from "@/server/npcs";

export const dynamic = "force-dynamic";

export const metadata = { title: "NPCs e lojas · WYD2" };

/**
 * Lista de definições de NPC/mercador.
 *
 * O banner de lista vazia cobre o caso de catálogo não semeado: `npc_definition`
 * é a única fonte de spawn do tmserver (issue 57), então um banco vazio = mundo
 * sem NPCs — aqui o operador vê o que falta fazer, em vez de uma tela vazia.
 */
export default async function NpcsPage() {
  let npcs: NpcDefinition[] = [];
  let zones: MapZone[] = [];
  let error: string | null = null;

  try {
    [npcs, zones] = await Promise.all([listNpcs(), listMapZones()]);
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível carregar as definições de NPC do web-api";
  }

  const hidden = npcs.filter((npc) => !npc.enabled).length;
  const merchants = npcs.filter((npc) => npc.merchant > 0).length;

  return (
    <>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            NPCs e lojas
          </h1>
          <p className="ui-page-sub" style={{ margin: "4px 0 0" }}>
            Definições em Postgres (overlay), aplicadas pelo tmserver. Não toca em{" "}
            <code>NPCGener.txt</code>.
          </p>
        </div>
        <Link href="/npcs/novo" className="ui-btn ui-btn--primary" style={{ marginLeft: "auto" }}>
          novo NPC
        </Link>
      </div>

      {error && (
        <div className="ui-card ui-empty" style={{ marginBottom: 12 }}>
          <strong>Não deu para listar os NPCs</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      )}

      {!error && npcs.length === 0 && (
        <div className="ui-card ui-empty" style={{ marginBottom: 12 }}>
          <strong>Nenhuma definição no banco</strong>
          <p>
            O catálogo está vazio. Rode o seed e reinicie o tmserver — sem definições o mundo
            sobe sem NPCs:
          </p>
          <p className="ui-mono">
            docker compose run --rm openwyd-dbserver import-npcs -content /content -dsn
            &quot;postgres://postgres:dev@db:5432/postgres&quot;
          </p>
        </div>
      )}

      {!error && npcs.length > 0 && (
        <>
          <p className="ui-field__hint" style={{ marginBottom: 10 }}>
            {npcs.length} definições · {merchants} mercadores · {hidden} ocultos ·{" "}
            <Badge tone="warn">conteúdo</Badge> não pode ser apagado, só ocultado
          </p>
          <NpcTable npcs={npcs} zones={zones} />
        </>
      )}
    </>
  );
}
