"use server";

/**
 * Concessão de ouro pela tela de entregas (FASE-07-4).
 *
 * A ação só valida a forma e delega ao `adminCall` + gateway: o autor vem da
 * sessão, a conta vem do personagem e a idempotência é a chave informada. O
 * status exibido é o recibo real (`enfileirada` × `replay`), não uma suposição.
 */
import { revalidatePath } from "next/cache";

import { requestGoldGrant } from "@/server/deliveries";
import { AdminError } from "@/server/errors";

export type GrantGoldState =
  | { status: "idle" }
  | { status: "ok"; message: string; grantId: number; replay: boolean }
  | { status: "error"; message: string };

const MIN_KEY = 8;
const MAX_KEY = 64;
const MAX_AMOUNT = 1_000_000;

export async function grantGoldAction(
  _prev: GrantGoldState,
  formData: FormData,
): Promise<GrantGoldState> {
  const characterId = Number(formData.get("character_id"));
  const amount = Number(formData.get("amount"));
  const key = String(formData.get("key") ?? "").trim();

  if (!Number.isInteger(characterId) || characterId <= 0) {
    return { status: "error", message: "personagem inválido" };
  }
  if (!Number.isInteger(amount) || amount <= 0 || amount > MAX_AMOUNT) {
    return { status: "error", message: `informe um valor entre 1 e ${MAX_AMOUNT}` };
  }
  if (key.length < MIN_KEY || key.length > MAX_KEY) {
    return {
      status: "error",
      message: `a chave de idempotência aceita de ${MIN_KEY} a ${MAX_KEY} caracteres`,
    };
  }

  try {
    const receipt = await requestGoldGrant({ characterId, amount, key });
    revalidatePath("/entregas");
    return {
      status: "ok",
      grantId: receipt.grant_id,
      replay: receipt.replay,
      message: receipt.replay
        ? `replay: entrega #${receipt.grant_id} já existia — nada foi duplicado`
        : `entrega #${receipt.grant_id} enfileirada`,
    };
  } catch (error) {
    if (error instanceof AdminError) return { status: "error", message: error.message };
    return { status: "error", message: "serviço de administração indisponível" };
  }
}
