/**
 * Leituras do `NpcAdminService`.
 *
 * Tudo passa por `adminCall` (issue 21): sessão de staff, `moderator_id` da
 * sessão, deadline e auditoria. `ListMerchantTemplates`/`ListMapZones` também
 * exigem `moderator_id` no proto, mesmo sendo leitura.
 */
import type { MapZone, MerchantTemplate, NpcDefinition } from "@/lib/npc";

import { adminCall } from "./admin";
import {
  npcAdmin,
  type GetNpcResponse,
  type ListMapZonesResponse,
  type ListMerchantTemplatesResponse,
  type ListNpcsResponse,
} from "./grpc";

export async function listNpcs(): Promise<NpcDefinition[]> {
  const res = await adminCall<Record<string, never>, ListNpcsResponse>(
    npcAdmin.listNpcs.bind(npcAdmin),
    {},
    { action: "npc.list" },
  );
  return res.npcs ?? [];
}

export async function getNpc(npcId: number): Promise<NpcDefinition | null> {
  if (!Number.isInteger(npcId) || npcId <= 0) return null;
  const res = await adminCall<{ npc_id: number }, GetNpcResponse>(
    npcAdmin.getNpc.bind(npcAdmin),
    { npc_id: npcId },
    { action: "npc.get", target: `npc:${npcId}` },
  );
  return res.npc ?? null;
}

/** Templates merchant do `tmserver/content/TMsrv/run/npc/`; `[]` quando o web-api sobe sem `-content`. */
export async function listMerchantTemplates(): Promise<MerchantTemplate[]> {
  const res = await adminCall<Record<string, never>, ListMerchantTemplatesResponse>(
    npcAdmin.listMerchantTemplates.bind(npcAdmin),
    {},
    { action: "npc.templates" },
  );
  return res.templates ?? [];
}

/** Zonas fixas (0 Armia … 4 Noatum) — não dependem de `-content`. */
export async function listMapZones(): Promise<MapZone[]> {
  const res = await adminCall<Record<string, never>, ListMapZonesResponse>(
    npcAdmin.listMapZones.bind(npcAdmin),
    {},
    { action: "npc.zones" },
  );
  return res.zones ?? [];
}

export async function loadNpcFormOptions(): Promise<{
  templates: MerchantTemplate[];
  zones: MapZone[];
}> {
  const [templates, zones] = await Promise.all([listMerchantTemplates(), listMapZones()]);
  return { templates, zones };
}
