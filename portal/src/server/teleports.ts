/**
 * Leituras do `TeleportAdminService` (issue 42).
 *
 * `ListRoutes` devolve a visão **mesclada** (tabela compilada + overlay), então
 * a tela nunca fica vazia só porque ninguém rodou o seed.
 */
import type { TeleportRoute, TeleportSimulation } from "@/lib/teleport";

import { adminCall } from "./admin";
import {
  teleportAdmin,
  type ListTeleportRoutesResponse,
  type SimulateTeleportRouteResponse,
} from "./grpc";

export async function listTeleportRoutes(): Promise<TeleportRoute[]> {
  const res = await adminCall<Record<string, never>, ListTeleportRoutesResponse>(
    teleportAdmin.listRoutes.bind(teleportAdmin),
    {},
    { action: "teleport.list" },
  );
  return res.routes ?? [];
}

/** Resolve uma origem sem consumir o RNG do mundo. */
export async function simulateTeleportRoute(
  srcX: number,
  srcY: number,
): Promise<TeleportSimulation> {
  const res = await adminCall<{ src_x: number; src_y: number }, SimulateTeleportRouteResponse>(
    teleportAdmin.simulateRoute.bind(teleportAdmin),
    { src_x: srcX, src_y: srcY },
    { action: "teleport.simulate", target: `route:${srcX},${srcY}` },
  );
  return {
    ok: Boolean(res.ok),
    src_x: Number(res.src_x),
    src_y: Number(res.src_y),
    dst_x: Number(res.dst_x),
    dst_y: Number(res.dst_y),
    cost: Number(res.cost),
    requirement: res.requirement ?? "none",
    warnings: res.warnings ?? [],
  };
}
