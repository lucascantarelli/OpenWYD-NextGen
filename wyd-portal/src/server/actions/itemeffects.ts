"use server";

/**
 * Escrita dos efeitos base de item (issue 44, metade de escrita).
 *
 * A chave é o código do efeito (`ItemEffect.h`), não uma posição: a quantidade de
 * pares `EF_*` varia por item e o servidor os indexa por código. A validação de
 * código/valor é do web-api (`itemeffects.Known`); o portal só garante que a
 * chamada chega íntegra e mostra o motivo devolvido.
 */
import { revalidatePath } from "next/cache";

import { adminCall } from "@/server/admin";
import { isOk } from "@/server/admin-result";
import { AdminError, isInfraFailure } from "@/server/errors";
import {
  itemEffectAdmin,
  type ItemEffectWriteResponse,
  type UpsertItemEffectInput,
} from "@/server/admin-client";

export type ItemEffectActionResult =
  | { ok: true; message?: string }
  | { ok: false; error: string };

function failure(error: unknown): ItemEffectActionResult {
  if (error instanceof AdminError) return { ok: false, error: error.message };
  if (isInfraFailure(error)) return { ok: false, error: "serviço de administração indisponível" };
  console.error(
    "[item_effect] falha inesperada:",
    error instanceof Error ? error.message : String(error),
  );
  return { ok: false, error: "erro interno do portal" };
}

/**
 * `ADMIN_RESULT.OK` é **1**, não 0 (`admin-result.ts`) — comparar com zero fazia
 * uma escrita bem-sucedida aparecer como recusada. Qualquer outro valor traz a
 * explicação em `message` (código fora do ItemEffect.h, valor fora de short, item
 * fora do catálogo), e é ela que a tela mostra.
 */
function outcome(res: ItemEffectWriteResponse, okMessage: string): ItemEffectActionResult {
  if (isOk(res.result)) return { ok: true, message: okMessage };
  return { ok: false, error: res.message || "o web-api recusou a alteração" };
}

export async function upsertEffect(input: {
  itemIndex: number;
  effect: number;
  value: number;
  reason: string;
}): Promise<ItemEffectActionResult> {
  const payload: UpsertItemEffectInput = {
    item_index: input.itemIndex,
    effect: input.effect,
    value: input.value,
    reason: input.reason,
  };
  try {
    const res = await adminCall<UpsertItemEffectInput, ItemEffectWriteResponse>(
      itemEffectAdmin.upsertEffect.bind(itemEffectAdmin),
      payload,
      { action: "item_effect.upsert", target: `item:${input.itemIndex}:ef${input.effect}` },
    );
    revalidatePath(`/itens/${input.itemIndex}`);
    return outcome(res, "efeito salvo na web");
  } catch (error) {
    return failure(error);
  }
}

/** Remove a linha do overlay: o efeito volta a valer o que o CSV declara. */
export async function deleteEffect(input: {
  itemIndex: number;
  effect: number;
  reason: string;
}): Promise<ItemEffectActionResult> {
  try {
    const res = await adminCall<
      { item_index: number; effect: number; reason: string },
      ItemEffectWriteResponse
    >(
      itemEffectAdmin.deleteEffect.bind(itemEffectAdmin),
      { item_index: input.itemIndex, effect: input.effect, reason: input.reason },
      { action: "item_effect.delete", target: `item:${input.itemIndex}:ef${input.effect}` },
    );
    revalidatePath(`/itens/${input.itemIndex}`);
    return outcome(res, "efeito restaurado do conteúdo");
  } catch (error) {
    return failure(error);
  }
}
