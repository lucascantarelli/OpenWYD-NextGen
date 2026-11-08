"use server";

/**
 * Escritas do editor de teleportes (issue 42).
 *
 * Mesma regra do editor de NPC: valida localmente **e** deixa a decisão final com
 * o `AdminResult` do web-api. O alvo da auditoria é `route:<src_x>,<src_y>`, que
 * é a chave real da rota (não existe id estável entre compilado e overlay).
 */
import type { z } from "zod";

import { routeFormSchema, type RouteFormValues, type TeleportSimulation } from "@/lib/teleport";

import { adminCall } from "@/server/admin";
import { AdminError, isInfraFailure } from "@/server/errors";
import {
  teleportAdmin,
  type AdminAck,
  type SimulateTeleportRouteResponse,
  type UpsertTeleportRouteResponse,
} from "@/server/admin-client";

export type TeleportActionResult =
  | { ok: true; warnings: string[] }
  | { ok: false; error: string; status: number };

function failure(error: unknown): TeleportActionResult {
  if (error instanceof AdminError) {
    return { ok: false, error: error.message, status: error.status };
  }
  if (isInfraFailure(error)) {
    return { ok: false, error: "serviço de administração indisponível", status: 502 };
  }
  console.error(
    "[teleport] falha inesperada:",
    error instanceof Error ? error.message : String(error),
  );
  return { ok: false, error: "erro interno do portal", status: 500 };
}

function firstIssue(error: z.ZodError): string {
  const issue = error.issues[0];
  const field = issue?.path.join(".");
  return field ? `${field}: ${issue?.message}` : "dados inválidos";
}

export async function upsertRoute(input: RouteFormValues): Promise<TeleportActionResult> {
  const parsed = routeFormSchema.safeParse(input);
  if (!parsed.success) return { ok: false, status: 422, error: firstIssue(parsed.error) };

  try {
    const res = await adminCall<RouteFormValues, UpsertTeleportRouteResponse>(
      teleportAdmin.upsertRoute.bind(teleportAdmin),
      parsed.data,
      { action: "teleport.upsert", target: `route:${parsed.data.src_x},${parsed.data.src_y}` },
    );
    return { ok: true, warnings: res.warnings ?? [] };
  } catch (error) {
    return failure(error);
  }
}

export async function setRouteEnabled(input: {
  srcX: number;
  srcY: number;
  enabled: boolean;
}): Promise<TeleportActionResult> {
  try {
    await adminCall<{ src_x: number; src_y: number; enabled: boolean }, AdminAck>(
      teleportAdmin.setRouteEnabled.bind(teleportAdmin),
      { src_x: input.srcX, src_y: input.srcY, enabled: input.enabled },
      {
        action: input.enabled ? "teleport.enable" : "teleport.disable",
        target: `route:${input.srcX},${input.srcY}`,
      },
    );
    return { ok: true, warnings: [] };
  } catch (error) {
    return failure(error);
  }
}

/** Remove a linha do overlay: a rota volta ao que o código define (se existir). */
export async function deleteRoute(input: {
  srcX: number;
  srcY: number;
}): Promise<TeleportActionResult> {
  try {
    await adminCall<{ src_x: number; src_y: number }, AdminAck>(
      teleportAdmin.deleteRoute.bind(teleportAdmin),
      { src_x: input.srcX, src_y: input.srcY },
      { action: "teleport.delete", target: `route:${input.srcX},${input.srcY}` },
    );
    return { ok: true, warnings: [] };
  } catch (error) {
    return failure(error);
  }
}

/**
 * “Duplicar inverso”: cria a volta com o mesmo custo. Se a volta já existir, o
 * upsert simplesmente a sobrescreve — é o comportamento esperado de quem clica
 * duas vezes.
 */
export async function duplicateReverse(input: RouteFormValues): Promise<TeleportActionResult> {
  return upsertRoute({
    ...input,
    src_x: input.dst_x,
    src_y: input.dst_y,
    dst_x: input.src_x,
    dst_y: input.src_y,
    note: input.note ? `${input.note} (volta)` : "volta (duplicar inverso)",
  });
}

/**
 * Resolve uma origem sem consumir o RNG do mundo.
 *
 * É uma **action** (e não uma leitura de `@/server/teleports`) porque quem chama é
 * um client component: importar o módulo gRPC ali puxa `fs` para o bundle do
 * browser e quebra a página.
 */
export async function simulateRoute(input: {
  srcX: number;
  srcY: number;
}): Promise<{ ok: true; simulation: TeleportSimulation } | { ok: false; error: string }> {
  try {
    const res = await adminCall<{ src_x: number; src_y: number }, SimulateTeleportRouteResponse>(
      teleportAdmin.simulateRoute.bind(teleportAdmin),
      { src_x: input.srcX, src_y: input.srcY },
      { action: "teleport.simulate", target: `route:${input.srcX},${input.srcY}` },
    );
    return {
      ok: true,
      simulation: {
        ok: Boolean(res.ok),
        src_x: Number(res.src_x),
        src_y: Number(res.src_y),
        dst_x: Number(res.dst_x),
        dst_y: Number(res.dst_y),
        cost: Number(res.cost),
        requirement: res.requirement ?? "none",
        warnings: res.warnings ?? [],
      },
    };
  } catch (error) {
    const fail = failure(error);
    return { ok: false, error: fail.ok ? "não foi possível simular agora" : fail.error };
  }
}
