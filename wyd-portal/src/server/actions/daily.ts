"use server";

/**
 * CRUD da recompensa diária (issue 30).
 *
 * O BFF **não** reimplementa o resgate: claim é do `DailyRewardService`, com o
 * `account_id` da sessão do jogador e a unicidade por dia no banco. Aqui só se
 * edita o catálogo frio (`daily_reward_item`), com duas checagens que o backend
 * não faz: limites de título/descrição e existência do item no catálogo — o
 * backend aceita qualquer `item_index > 0` e o jogador receberia slot vazio.
 */
import { revalidatePath } from "next/cache";

import {
  TITLE_MAX,
  validateDailyReward,
  type DailyRewardItem,
} from "@/lib/daily";
import { adminCall } from "@/server/admin";
import { isOk } from "@/server/admin-result";
import { getCatalog } from "@/server/catalog";
import { AdminError, isInfraFailure } from "@/server/errors";
import {
  dailyRewardAdmin,
  type AdminAck,
  type ListRewardItemsResponse,
  type UpsertRewardItemResponse,
} from "@/server/grpc";

export type DailyActionResult =
  | { ok: true; message: string; id?: number }
  | { ok: false; error: string };

function toInt(value: unknown): number {
  const n = Math.trunc(Number(value));
  return Number.isFinite(n) ? n : 0;
}

function normalize(input: DailyRewardItem): DailyRewardItem {
  return {
    id: toInt(input.id),
    item_index: toInt(input.item_index),
    eff1: toInt(input.eff1),
    effv1: toInt(input.effv1),
    eff2: toInt(input.eff2),
    effv2: toInt(input.effv2),
    eff3: toInt(input.eff3),
    effv3: toInt(input.effv3),
    title: String(input.title ?? "").trim(),
    description: String(input.description ?? ""),
    enabled: Boolean(input.enabled),
    expires_days: toInt(input.expires_days),
  };
}

function failure(error: unknown): DailyActionResult {
  if (error instanceof AdminError) return { ok: false, error: error.message };
  if (isInfraFailure(error)) return { ok: false, error: "serviço de administração indisponível" };
  console.error("[daily] falha inesperada:", error instanceof Error ? error.message : String(error));
  return { ok: false, error: "erro interno do portal" };
}

async function itemExists(itemIndex: number): Promise<boolean | null> {
  try {
    const catalog = await getCatalog();
    return catalog.byIndex.has(itemIndex);
  } catch {
    return null;
  }
}

export async function saveReward(input: DailyRewardItem): Promise<DailyActionResult> {
  const item = normalize(input);

  const errors = validateDailyReward(item);
  const first = Object.values(errors)[0];
  if (first) return { ok: false, error: first };

  const exists = await itemExists(item.item_index);
  if (exists === null) {
    return { ok: false, error: "catálogo de itens indisponível: não dá para validar o item" };
  }
  if (!exists) {
    return {
      ok: false,
      error: `o item #${item.item_index} não existe no catálogo — escolha outro no seletor`,
    };
  }

  try {
    const res = await adminCall<{ item: DailyRewardItem }, UpsertRewardItemResponse>(
      dailyRewardAdmin.upsertRewardItem.bind(dailyRewardAdmin),
      { item },
      { action: item.id === 0 ? "daily.create" : "daily.update", target: `reward:${item.id || "new"}` },
    );
    if (!isOk(res.result)) return { ok: false, error: "o web-api recusou a oferta" };
    revalidatePath("/diaria");
    return {
      ok: true,
      id: Number(res.item_id ?? 0),
      message:
        item.id === 0
          ? "oferta criada — os jogadores resgatam no portal do jogador"
          : "oferta salva — vale para os próximos resgates",
    };
  } catch (error) {
    return failure(error);
  }
}

export async function setRewardEnabled(input: {
  id: number;
  enabled: boolean;
}): Promise<DailyActionResult> {
  try {
    const res = await adminCall<{ item_id: number; enabled: boolean }, AdminAck>(
      dailyRewardAdmin.setRewardItemEnabled.bind(dailyRewardAdmin),
      { item_id: input.id, enabled: input.enabled },
      { action: "daily.set_enabled", target: `reward:${input.id}` },
    );
    if (!isOk(res.result)) return { ok: false, error: "o web-api recusou a alteração" };
    revalidatePath("/diaria");
    return {
      ok: true,
      message: input.enabled ? "oferta habilitada" : "oferta desabilitada (deixa de aparecer)",
    };
  } catch (error) {
    return failure(error);
  }
}

export async function deleteReward(input: { id: number }): Promise<DailyActionResult> {
  try {
    const res = await adminCall<{ item_id: number }, AdminAck>(
      dailyRewardAdmin.deleteRewardItem.bind(dailyRewardAdmin),
      { item_id: input.id },
      { action: "daily.delete", target: `reward:${input.id}` },
    );
    if (!isOk(res.result)) return { ok: false, error: "o web-api recusou a remoção" };
    revalidatePath("/diaria");
    return { ok: true, message: "oferta apagada — os resgates já feitos ficam no histórico" };
  } catch (error) {
    return failure(error);
  }
}

/**
 * Duplica uma oferta. A cópia nasce **desabilitada**: duplicar é ponto de
 * partida para editar, e duas ofertas habilitadas não dão dois resgates por dia
 * — habilitar por engano só confunde quem lê a vitrine.
 */
export async function duplicateReward(input: { id: number }): Promise<DailyActionResult> {
  try {
    const list = await adminCall<Record<string, never>, ListRewardItemsResponse>(
      dailyRewardAdmin.listRewardItems.bind(dailyRewardAdmin),
      {},
      { action: "daily.list", target: `copy-of:${input.id}` },
    );
    const source = (list.items ?? []).find((item) => Number(item.id) === input.id);
    if (!source) return { ok: false, error: "oferta não encontrada — recarregue a lista" };

    const copy = normalize({
      ...(source as DailyRewardItem),
      id: 0,
      enabled: false,
      title: `${source.title} (cópia)`.slice(0, TITLE_MAX),
    });

    const res = await adminCall<{ item: DailyRewardItem }, UpsertRewardItemResponse>(
      dailyRewardAdmin.upsertRewardItem.bind(dailyRewardAdmin),
      { item: copy },
      { action: "daily.duplicate", target: `reward:${input.id}` },
    );
    if (!isOk(res.result)) return { ok: false, error: "o web-api recusou a cópia" };
    revalidatePath("/diaria");
    return {
      ok: true,
      id: Number(res.item_id ?? 0),
      message: "oferta duplicada desabilitada — edite e habilite quando estiver pronta",
    };
  } catch (error) {
    return failure(error);
  }
}
