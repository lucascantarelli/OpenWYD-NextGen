"use server";

/**
 * Concessão de item pela tela de correio (WEB-09).
 *
 * A ação só valida a forma e delega ao `adminCall` + gateway
 * (`character_admin/grant_item`): o autor vem da sessão, a idempotência é a
 * chave informada e a entrega acontece no dreno do servidor — mochila cheia
 * deixa a linha pendente, nunca duplica nem perde.
 */
import { revalidatePath } from "next/cache";

import { checkItemGrant, requestItemGrant } from "@/server/deliveries";
import { AdminError } from "@/server/errors";

export type GrantItemState =
  | { status: "idle" }
  | { status: "ok"; message: string; grantId: number; replay: boolean }
  | { status: "error"; message: string };

export async function grantItemAction(
  _prev: GrantItemState,
  formData: FormData,
): Promise<GrantItemState> {
  const input = {
    characterId: Number(formData.get("character_id")),
    itemId: Number(formData.get("item_id")),
    qty: Number(formData.get("qty")),
    key: String(formData.get("key") ?? ""),
  };
  const invalid = checkItemGrant(input);
  if (invalid) return { status: "error", message: invalid };

  try {
    const receipt = await requestItemGrant({ ...input, key: input.key.trim() });
    revalidatePath("/correio");
    return {
      status: "ok",
      grantId: receipt.grant_id,
      replay: receipt.replay,
      message: receipt.replay
        ? `replay: entrega #${receipt.grant_id} já existia — nada foi duplicado`
        : `item enfileirado como entrega #${receipt.grant_id} (mochila cheia deixa pendente)`,
    };
  } catch (error) {
    if (error instanceof AdminError) return { status: "error", message: error.message };
    return { status: "error", message: "serviço de administração indisponível" };
  }
}
