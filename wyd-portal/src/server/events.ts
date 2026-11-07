/**
 * Leitura do `WorldEventAdminService` (issue 28).
 *
 * `version` só muda quando um moderador salva — o tmserver grava o progresso de
 * `current_index` sem incrementá-la. É por isso que a tela mostra a versão: se
 * ela não mudou, o config lido é o mesmo que o mundo está usando.
 */
import { emptyConfig, type WorldEventConfig } from "@/lib/worldevent";

import { adminCall } from "./admin";
import { worldEventAdmin, type GetWorldEventConfigResponse } from "./grpc";

export interface WorldEventSnapshot {
  version: number;
  config: WorldEventConfig;
}

function normalize(cfg: Partial<WorldEventConfig> | undefined): WorldEventConfig {
  const base = emptyConfig();
  if (!cfg) return base;
  return {
    enabled: Boolean(cfg.enabled),
    item_index: Number(cfg.item_index ?? 0),
    rate: Number(cfg.rate ?? 0),
    start_index: Number(cfg.start_index ?? 0),
    current_index: Number(cfg.current_index ?? 0),
    end_index: Number(cfg.end_index ?? 0),
    indexed: Boolean(cfg.indexed),
    notice_enabled: Boolean(cfg.notice_enabled),
    double_exp_enabled: Boolean(cfg.double_exp_enabled),
    newbie_event_enabled: Boolean(cfg.newbie_event_enabled),
  };
}

export async function getWorldEvent(): Promise<WorldEventSnapshot> {
  const res = await adminCall<Record<string, never>, GetWorldEventConfigResponse>(
    worldEventAdmin.getWorldEventConfig.bind(worldEventAdmin),
    {},
    { action: "world_event.get" },
  );
  return { version: Number(res.version ?? 0), config: normalize(res.config) };
}
