/**
 * Leitura do `WorldStateAdminService` (issue 45).
 *
 * Uma chamada devolve o snapshot inteiro: guildas já com membros/mestre/cidades,
 * relações, as cinco zonas do reino, a torre, a quest do castelo e o Kefra. A
 * página não faz N+1 no web-api — quem agrupa é o serviço Go.
 */
import type { WorldState } from "@/lib/worldstate";

import { adminCall } from "./admin";
import { worldStateAdmin, type GetWorldStateResponse } from "./admin-client";

export async function getWorldState(): Promise<WorldState> {
  const res = await adminCall<Record<string, never>, GetWorldStateResponse>(
    worldStateAdmin.getWorldState.bind(worldStateAdmin),
    {},
    { action: "world_state.get" },
  );
  return {
    guilds: res.guilds ?? [],
    relations: res.relations ?? [],
    zones: res.zones ?? [],
    tower_owner_guild: Number(res.tower_owner_guild ?? 0),
    tower_owner_name: res.tower_owner_name ?? "",
    tower_updated_unix: Number(res.tower_updated_unix ?? 0),
    castle: res.castle ?? { level: 0, time_left: 0, clear: false, leader_name: "" },
    kefra: res.kefra ?? {
      initialized: false,
      defeated: false,
      next_spawn_unix: 0,
      last_spawn_unix: 0,
      revision: 0,
    },
  };
}
