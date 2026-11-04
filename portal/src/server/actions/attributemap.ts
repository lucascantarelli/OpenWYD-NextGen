"use server";

/**
 * Action de pré-visualização da transformação (issue 27).
 *
 * A transformação roda no web-api sobre o arquivo de conteúdo; esta action
 * devolve só o resumo (contagem, hashes e histogramas) e nunca o payload de
 * 1 MiB. O download é uma rota própria, que refaz a mesma transformação com os
 * mesmos parâmetros — por isso a tela só habilita o download do que já foi
 * pré-visualizado.
 */
import { transformError, type TransformParams, type TransformSummary } from "@/lib/attributemap";

import { AdminError, isInfraFailure } from "@/server/errors";
import { previewAttributeMapTransform } from "@/server/attributemap";

export type PreviewResult = { ok: true; summary: TransformSummary } | { ok: false; error: string };

export async function previewAttributeMap(input: TransformParams): Promise<PreviewResult> {
  const invalid = transformError(input);
  if (invalid) return { ok: false, error: invalid };

  try {
    const summary = await previewAttributeMapTransform(input);
    return { ok: true, summary };
  } catch (error) {
    if (error instanceof AdminError) return { ok: false, error: error.message };
    if (isInfraFailure(error)) {
      return {
        ok: false,
        error: "serviço de administração indisponível; verifique se o webserver está no ar",
      };
    }
    console.error(
      "[attributemap] falha na pré-visualização:",
      error instanceof Error ? error.message : String(error),
    );
    return { ok: false, error: "erro interno do portal ao pré-visualizar o mapa" };
  }
}
