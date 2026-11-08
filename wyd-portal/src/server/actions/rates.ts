"use server";

/**
 * Escrita das taxas de conteúdo (issue 36).
 *
 * O web-api valida a célula contra o arquivo (quais células existem e a faixa de
 * cada uma) e registra before/after em `content_rate_audit`. A taxa só chega ao
 * jogo no próximo boot do tmserver com `-rates-editing`, e a tela diz isso.
 */
import { revalidatePath } from "next/cache";

import { adminCall } from "@/server/admin";
import { isOk } from "@/server/admin-result";
import { AdminError, isInfraFailure } from "@/server/errors";
import { contentRates, type SetRateResponse } from "@/server/admin-client";

export type RateActionResult = { ok: true; message?: string } | { ok: false; error: string };

function failure(error: unknown): RateActionResult {
  if (error instanceof AdminError) return { ok: false, error: error.message };
  if (isInfraFailure(error)) return { ok: false, error: "serviço de administração indisponível" };
  console.error("[rates] falha inesperada:", error instanceof Error ? error.message : String(error));
  return { ok: false, error: "erro interno do portal" };
}

function outcome(res: SetRateResponse, okMessage: string): RateActionResult {
  if (isOk(res.result)) return { ok: true, message: okMessage };
  return { ok: false, error: res.message || "o web-api recusou a alteração" };
}

export async function upsertRate(input: {
  file: string;
  family: string;
  name: string;
  value: number;
  reason: string;
}): Promise<RateActionResult> {
  try {
    const res = await adminCall<
      { file: string; family: string; name: string; value: number; reason: string },
      SetRateResponse
    >(
      contentRates.upsertRate.bind(contentRates),
      {
        file: input.file,
        family: input.family,
        name: input.name,
        value: input.value,
        reason: input.reason,
      },
      { action: "rates.upsert", target: `${input.file}:${input.family}/${input.name}` },
    );
    revalidatePath("/config");
    return outcome(res, "taxa salva — vale no próximo restart do tmserver");
  } catch (error) {
    return failure(error);
  }
}

export async function resetRate(input: {
  file: string;
  family: string;
  name: string;
  reason: string;
}): Promise<RateActionResult> {
  try {
    const res = await adminCall<
      { file: string; family: string; name: string; reason: string },
      SetRateResponse
    >(
      contentRates.resetRate.bind(contentRates),
      { file: input.file, family: input.family, name: input.name, reason: input.reason },
      { action: "rates.reset", target: `${input.file}:${input.family}/${input.name}` },
    );
    revalidatePath("/config");
    return outcome(res, "célula restaurada para o valor do arquivo");
  } catch (error) {
    return failure(error);
  }
}
