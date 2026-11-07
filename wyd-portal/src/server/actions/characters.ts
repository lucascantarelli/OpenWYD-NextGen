"use server";

/**
 * Concessão de item por `delivery_queue` (issue 43).
 *
 * A web **não** escreve o personagem: enfileira uma linha que o tmserver drena
 * no cargo da conta dentro do loop do mundo. Por isso funciona com o personagem
 * online — e o item aparece no próximo login/dreno, não na hora.
 *
 * Uma linha por unidade (o payload da fila não tem quantidade), com motivo
 * obrigatório: é o que a auditoria guarda junto do antes/depois.
 */
import { z } from "zod";

import { adminCall } from "@/server/admin";
import { AdminError, isInfraFailure } from "@/server/errors";
import { characterAdmin, type GrantCharacterItemResponse } from "@/server/grpc";

const grantSchema = z.object({
  character_id: z.number().int().positive(),
  item_index: z.number().int().min(1).max(65535),
  quantity: z.number().int().min(1).max(255),
  eff1: z.number().int().min(0).max(255),
  effv1: z.number().int().min(0).max(255),
  eff2: z.number().int().min(0).max(255),
  effv2: z.number().int().min(0).max(255),
  eff3: z.number().int().min(0).max(255),
  effv3: z.number().int().min(0).max(255),
  reason: z.string().trim().min(3, "descreva o motivo (mín. 3 letras)").max(200),
});

export type GrantInput = z.infer<typeof grantSchema>;

export type GrantResult =
  | { ok: true; deliveryIds: number[] }
  | { ok: false; error: string; status: number };

export async function grantCharacterItem(input: GrantInput): Promise<GrantResult> {
  const parsed = grantSchema.safeParse(input);
  if (!parsed.success) {
    const issue = parsed.error.issues[0];
    const field = issue?.path.join(".");
    return {
      ok: false,
      status: 422,
      error: field ? `${field}: ${issue?.message}` : "dados inválidos",
    };
  }

  try {
    const res = await adminCall<Omit<GrantInput, "character_id"> & { character_id: number }, GrantCharacterItemResponse>(
      characterAdmin.grantItem.bind(characterAdmin),
      parsed.data,
      { action: "character.grant_item", target: `character:${parsed.data.character_id}` },
    );
    return { ok: true, deliveryIds: (res.delivery_ids ?? []).map(Number) };
  } catch (error) {
    if (error instanceof AdminError) {
      return { ok: false, error: error.message, status: error.status };
    }
    if (isInfraFailure(error)) {
      return { ok: false, error: "serviço de administração indisponível", status: 502 };
    }
    console.error(
      "[character] falha na concessão:",
      error instanceof Error ? error.message : String(error),
    );
    return { ok: false, error: "erro interno do portal", status: 500 };
  }
}
