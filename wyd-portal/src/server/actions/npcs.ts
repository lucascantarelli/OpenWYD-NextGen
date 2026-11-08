"use server";

/**
 * Escritas do editor de NPC. Cada ação valida localmente **e** deixa a decisão
 * final com o `AdminResult` do web-api — nenhuma regra de negócio é duplicada
 * como se fosse garantia.
 *
 * Devolvem objeto (não `Response`): quem chama é um client component que precisa
 * mostrar o erro no formulário. O que o cliente recebe é só a mensagem segura
 * que o `adminCall` produziu.
 */
import type { z } from "zod";

import { npcFormSchema, shopSchema, type NpcFormValues } from "@/lib/npc";

import { adminCall } from "@/server/admin";
import { AdminError, isInfraFailure } from "@/server/errors";
import { npcAdmin, type AdminAck, type UpsertNpcResponse } from "@/server/admin-client";
import { getNpc } from "@/server/npcs";

export type NpcActionResult =
  | { ok: true; npcId?: number }
  | { ok: false; error: string; status: number; contentOwned?: boolean };

function failure(error: unknown): NpcActionResult {
  if (error instanceof AdminError) {
    return {
      ok: false,
      error: error.message,
      status: error.status,
      contentOwned: error.status === 409,
    };
  }
  if (isInfraFailure(error)) {
    return { ok: false, error: "serviço de administração indisponível", status: 502 };
  }
  console.error("[npc] falha inesperada:", error instanceof Error ? error.message : String(error));
  return { ok: false, error: "erro interno do portal", status: 500 };
}

function firstIssue(error: z.ZodError): string {
  const issue = error.issues[0];
  const field = issue?.path.join(".");
  return field ? `${field}: ${issue?.message}` : "dados inválidos";
}

export async function upsertNpc(input: NpcFormValues): Promise<NpcActionResult> {
  const parsed = npcFormSchema.safeParse(input);
  if (!parsed.success) return { ok: false, status: 422, error: firstIssue(parsed.error) };

  try {
    const res = await adminCall<NpcFormValues, UpsertNpcResponse>(
      npcAdmin.upsertNpc.bind(npcAdmin),
      parsed.data,
      { action: "npc.upsert", target: `npc:${parsed.data.slug}` },
    );
    return { ok: true, npcId: Number(res.npc_id) };
  } catch (error) {
    return failure(error);
  }
}

export async function setNpcVisibility(input: {
  npcId: number;
  enabled: boolean;
}): Promise<NpcActionResult> {
  try {
    await adminCall<{ npc_id: number; enabled: boolean }, AdminAck>(
      npcAdmin.setNpcVisibility.bind(npcAdmin),
      { npc_id: input.npcId, enabled: input.enabled },
      { action: input.enabled ? "npc.show" : "npc.hide", target: `npc:${input.npcId}` },
    );
    return { ok: true };
  } catch (error) {
    return failure(error);
  }
}

/** Envia a loja **inteira**: o que não estiver no array deixa de existir (`§5.2`). */
export async function saveNpcShop(input: unknown): Promise<NpcActionResult> {
  const parsed = shopSchema.safeParse(input);
  if (!parsed.success) return { ok: false, status: 422, error: firstIssue(parsed.error) };

  try {
    await adminCall<{ npc_id: number; items: typeof parsed.data.items }, AdminAck>(
      npcAdmin.setNpcShop.bind(npcAdmin),
      { npc_id: parsed.data.npcId, items: parsed.data.items },
      { action: "npc.shop", target: `npc:${parsed.data.npcId}` },
    );
    return { ok: true };
  } catch (error) {
    return failure(error);
  }
}

export async function deleteNpc(input: {
  npcId: number;
  slug: string;
}): Promise<NpcActionResult> {
  try {
    await adminCall<{ npc_id: number }, AdminAck>(
      npcAdmin.deleteNpc.bind(npcAdmin),
      { npc_id: input.npcId },
      { action: "npc.delete", target: `npc:${input.slug}` },
    );
    return { ok: true };
  } catch (error) {
    return failure(error);
  }
}

/**
 * Duplica um NPC com sufixo `-copy`. A loja também é copiada quando existe:
 * duplicar um mercador sem estoque não serve para nada.
 */
export async function duplicateNpc(input: { npcId: number }): Promise<NpcActionResult> {
  try {
    const source = await getNpc(input.npcId);
    if (!source) return { ok: false, status: 404, error: "o registro não existe mais" };

    const slug = `${source.slug.slice(0, 34)}-copy`;
    const created = await adminCall<NpcFormValues, UpsertNpcResponse>(
      npcAdmin.upsertNpc.bind(npcAdmin),
      {
        slug,
        display_name: source.display_name.slice(0, 36) + " (copy)",
        template_name: source.template_name,
        enabled: source.enabled,
        map_id: source.map_id,
        pos_x: source.pos_x,
        pos_y: source.pos_y,
        route_type: source.route_type,
        merchant: source.merchant,
      },
      { action: "npc.duplicate", target: `npc:${source.slug}` },
    );

    const npcId = Number(created.npc_id);
    if (source.shop?.length) {
      await adminCall<{ npc_id: number; items: typeof source.shop }, AdminAck>(
        npcAdmin.setNpcShop.bind(npcAdmin),
        { npc_id: npcId, items: source.shop },
        { action: "npc.shop", target: `npc:${slug}` },
      );
    }

    return { ok: true, npcId };
  } catch (error) {
    return failure(error);
  }
}
