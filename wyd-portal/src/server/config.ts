/**
 * Configuração do servidor (FASE-07-6) — taxas, drops, NPC e teleporte.
 *
 * Tudo passa por `adminCall`: sessão de staff, `moderator_id` da sessão, auditoria e
 * mapeamento `AdminResult` → HTTP. O portal não fala com o banco: lê e grava pelo
 * gateway, e cada escrita incrementa a revisão que o servidor de jogo observa.
 */
import { adminCall } from "./admin";
import {
  contentRates,
  dropAdmin,
  npcAdmin,
  teleportAdmin,
  type AccessoryRuleWire,
  type DropWire,
  type GetNpcResponse,
  type GetRatesResponse,
  type ListDropsResponse,
  type ListRoutesResponse,
  type NpcWire,
  type RateWire,
  type RevisionResponse,
  type RouteWire,
} from "./admin-client";

export type { AccessoryRuleWire, DropWire, NpcWire, RateWire, RouteWire };

export interface RatesView {
  revision: number;
  rates: RateWire[];
  accessories: AccessoryRuleWire[];
}

/** Foto completa da configuração de taxas. */
export async function getRates(): Promise<RatesView> {
  const res = await adminCall<Record<string, never>, GetRatesResponse>(
    contentRates.getRates.bind(contentRates),
    {},
    { action: "config.rates", target: "rates" },
  );
  return {
    revision: res.revision ?? 0,
    rates: res.rates ?? [],
    accessories: res.accessories ?? [],
  };
}

/** Grava uma taxa; a fonte é a procedência que fica na trilha. */
export async function upsertRate(input: {
  domain: string;
  key: string;
  value: number;
  source: string;
}): Promise<number> {
  const res = await adminCall<
    { domain: string; key: string; value: number; source: string },
    RevisionResponse
  >(contentRates.upsertRate.bind(contentRates), input, {
    action: "config.rate.set",
    target: `${input.domain}:${input.key}`,
  });
  return res.revision ?? 0;
}

/** Remove o override e volta ao default documentado. */
export async function resetRate(domain: string, key: string): Promise<number> {
  const res = await adminCall<{ domain: string; key: string }, RevisionResponse>(
    contentRates.resetRate.bind(contentRates),
    { domain, key },
    { action: "config.rate.reset", target: `${domain}:${key}` },
  );
  return res.revision ?? 0;
}

/** Pesos de drop (filtro opcional por template). */
export async function listDrops(template?: string): Promise<DropWire[]> {
  const clean = template?.trim().slice(0, 120) ?? "";
  const res = await adminCall<{ template_name?: string; limit: number }, ListDropsResponse>(
    dropAdmin.listDropRates.bind(dropAdmin),
    { ...(clean ? { template_name: clean } : {}), limit: 50 },
    { action: "config.drops", target: clean || "*" },
  );
  return res.drops ?? [];
}

export async function upsertDrop(input: {
  template_name: string;
  item_index: number;
  weight: number;
  min_qty: number;
  max_qty: number;
  source: string;
}): Promise<number> {
  const res = await adminCall<
    {
      template_name: string;
      item_index: number;
      weight: number;
      min_qty: number;
      max_qty: number;
      source: string;
    },
    RevisionResponse
  >(dropAdmin.upsertDropRate.bind(dropAdmin), input, {
    action: "config.drop.set",
    target: `${input.template_name}#${input.item_index}`,
  });
  return res.revision ?? 0;
}

export async function resetDrop(template: string, itemIndex: number): Promise<number> {
  const res = await adminCall<
    { template_name: string; item_index: number },
    RevisionResponse
  >(
    dropAdmin.resetDropRate.bind(dropAdmin),
    { template_name: template, item_index: itemIndex },
    { action: "config.drop.reset", target: `${template}#${itemIndex}` },
  );
  return res.revision ?? 0;
}

/** Um NPC por id; ausência não é erro de tela. */
export async function getNpc(id: number): Promise<NpcWire | null> {
  try {
    const res = await adminCall<{ id: number }, GetNpcResponse>(
      npcAdmin.getNpc.bind(npcAdmin),
      { id },
      { action: "config.npc", target: `npc:${id}` },
    );
    return res.npc ?? null;
  } catch {
    return null;
  }
}

export async function upsertNpc(input: {
  id: number;
  map_id: number;
  pos_x: number;
  pos_y: number;
  enabled: boolean;
}): Promise<number> {
  const res = await adminCall<
    { id: number; map_id: number; pos_x: number; pos_y: number; enabled: boolean },
    RevisionResponse
  >(npcAdmin.upsertNpc.bind(npcAdmin), input, {
    action: "config.npc.set",
    target: `npc:${input.id}`,
  });
  return res.revision ?? 0;
}

/** Rotas de teleporte cadastradas. */
export async function listRoutes(): Promise<RouteWire[]> {
  const res = await adminCall<{ limit: number }, ListRoutesResponse>(
    teleportAdmin.listRoutes.bind(teleportAdmin),
    { limit: 200 },
    { action: "config.teleport", target: "routes" },
  );
  return res.routes ?? [];
}

export async function upsertRoute(input: {
  id: number;
  cost_gold?: number;
  req_level?: number;
  enabled?: boolean;
}): Promise<number> {
  const res = await adminCall<
    { id: number; cost_gold?: number; req_level?: number; enabled?: boolean },
    RevisionResponse
  >(teleportAdmin.upsertRoute.bind(teleportAdmin), input, {
    action: "config.teleport.set",
    target: `route:${input.id}`,
  });
  return res.revision ?? 0;
}
