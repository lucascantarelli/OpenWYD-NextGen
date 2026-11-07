"use server";

/**
 * Escrita do evento global de mundo (issue 28).
 *
 * Duas regras desta action:
 *
 * 1. `SetWorldEventConfig` **substitui o config inteiro**, então o formulário
 *    manda todos os campos e o BFF revalida a faixa antes de chamar o web-api
 *    (espelho de `worldevent.validConfig` — um "salvar" recusado sem motivo é
 *    o pior resultado possível);
 * 2. item fora do catálogo é recusado aqui, não no backend: o `world_event_config`
 *    aceita qualquer índice ≤ 32767 e o jogador receberia um slot vazio.
 *
 * Depois de salvar, a action **relê** o config: o estado que a tela mostra é o
 * que o servidor confirmou, não o que o browser achava que enviou.
 */
import { revalidatePath } from "next/cache";

import { validateWorldEvent, type WorldEventConfig } from "@/lib/worldevent";
import { adminCall } from "@/server/admin";
import { isOk } from "@/server/admin-result";
import { getCatalog } from "@/server/catalog";
import { AdminError, isInfraFailure } from "@/server/errors";
import { worldEventAdmin, type AdminAck, type GetWorldEventConfigResponse } from "@/server/grpc";

export type WorldEventActionResult =
  | { ok: true; message: string; config: WorldEventConfig; version: number }
  | { ok: false; error: string };

function toInt(value: unknown): number {
  const n = Math.trunc(Number(value));
  return Number.isFinite(n) ? n : 0;
}

function normalize(input: WorldEventConfig): WorldEventConfig {
  return {
    enabled: Boolean(input.enabled),
    item_index: toInt(input.item_index),
    rate: toInt(input.rate),
    start_index: toInt(input.start_index),
    current_index: toInt(input.current_index),
    end_index: toInt(input.end_index),
    indexed: Boolean(input.indexed),
    notice_enabled: Boolean(input.notice_enabled),
    double_exp_enabled: Boolean(input.double_exp_enabled),
    newbie_event_enabled: Boolean(input.newbie_event_enabled),
  };
}

function failure(error: unknown): WorldEventActionResult {
  if (error instanceof AdminError) return { ok: false, error: error.message };
  if (isInfraFailure(error)) return { ok: false, error: "serviço de administração indisponível" };
  console.error(
    "[world_event] falha inesperada:",
    error instanceof Error ? error.message : String(error),
  );
  return { ok: false, error: "erro interno do portal" };
}

export async function saveWorldEventConfig(
  input: WorldEventConfig,
): Promise<WorldEventActionResult> {
  const config = normalize(input);

  const errors = validateWorldEvent(config);
  const first = Object.values(errors)[0];
  if (first) return { ok: false, error: first };

  if (config.item_index > 0) {
    try {
      const catalog = await getCatalog();
      if (!catalog.byIndex.has(config.item_index)) {
        return {
          ok: false,
          error: `o item #${config.item_index} não existe no catálogo — escolha outro no seletor`,
        };
      }
    } catch {
      return {
        ok: false,
        error: "catálogo de itens indisponível: não dá para validar o item do evento",
      };
    }
  }

  try {
    const ack = await adminCall<{ config: WorldEventConfig }, AdminAck>(
      worldEventAdmin.setWorldEventConfig.bind(worldEventAdmin),
      { config },
      {
        action: "world_event.set",
        target: `item:${config.item_index}`,
      },
    );
    if (!isOk(ack.result)) {
      return { ok: false, error: "o web-api recusou a configuração do evento" };
    }

    const confirmed = await adminCall<Record<string, never>, GetWorldEventConfigResponse>(
      worldEventAdmin.getWorldEventConfig.bind(worldEventAdmin),
      {},
      { action: "world_event.get", target: "after-save" },
    );

    revalidatePath("/eventos");

    const saved = confirmed.config;
    return {
      ok: true,
      message: config.double_exp_enabled
        ? "evento salvo — EXP dobrada ativa no próximo ciclo do tmserver (15 s)"
        : "evento salvo — o tmserver relê em até 15 s, sem restart",
      version: Number(confirmed.version ?? 0),
      config: {
        enabled: Boolean(saved?.enabled),
        item_index: toInt(saved?.item_index),
        rate: toInt(saved?.rate),
        start_index: toInt(saved?.start_index),
        current_index: toInt(saved?.current_index),
        end_index: toInt(saved?.end_index),
        indexed: Boolean(saved?.indexed),
        notice_enabled: Boolean(saved?.notice_enabled),
        double_exp_enabled: Boolean(saved?.double_exp_enabled),
        newbie_event_enabled: Boolean(saved?.newbie_event_enabled),
      },
    };
  } catch (error) {
    return failure(error);
  }
}
