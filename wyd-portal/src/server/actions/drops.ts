"use server";

/**
 * Actions de drops: busca dos relatórios (issue 26) e escrita da taxa por slot
 * (issue 34).
 *
 * São **actions** (e não leituras de `@/server/drops`) porque quem chama é um
 * client component: importar o módulo gRPC ali puxaria `fs` para o bundle do
 * browser. A regra de sessão/papel continua sendo a do `adminCall`, que injeta
 * `moderator_id` da sessão e audita cada resultado.
 */
import { revalidatePath } from "next/cache";

import type { ItemDropEntry, MobDropEntry } from "@/lib/drop";
import { CARRY_SLOTS } from "@/lib/mobcarry";
import { DROP_RATE_MAX, DROP_RATE_MIN, hardcodedRate } from "@/lib/droprate";

import { adminCall } from "@/server/admin";
import { AdminError, isInfraFailure } from "@/server/errors";
import { dropAdmin, type AdminAck } from "@/server/admin-client";
import { listItemDrops, listMobDrops } from "@/server/drops";

export type DropSearchResult<T> =
  | { ok: true; entries: T[] }
  | { ok: false; error: string };

const MIN_QUERY = 2;

function failure(error: unknown): { ok: false; error: string } {
  if (error instanceof AdminError) return { ok: false, error: error.message };
  if (isInfraFailure(error)) {
    return { ok: false, error: "serviço de administração indisponível" };
  }
  console.error("[drops] falha inesperada:", error instanceof Error ? error.message : String(error));
  return { ok: false, error: "erro interno do portal" };
}

export async function searchItemDrops(input: {
  itemQuery: string;
  mobQuery?: string;
  includeZeroDropItems?: boolean;
}): Promise<DropSearchResult<ItemDropEntry>> {
  const itemQuery = input.itemQuery.trim();
  const mobQuery = (input.mobQuery ?? "").trim();
  if (itemQuery.length < MIN_QUERY && mobQuery.length < MIN_QUERY) {
    return { ok: false, error: `digite ao menos ${MIN_QUERY} caracteres do item ou do mob` };
  }
  try {
    const entries = await listItemDrops({
      itemQuery,
      mobQuery,
      includeZeroDropItems: input.includeZeroDropItems,
    });
    return { ok: true, entries };
  } catch (error) {
    return failure(error);
  }
}

export async function searchMobDrops(input: {
  mobQuery: string;
  itemIndex?: number;
}): Promise<DropSearchResult<MobDropEntry>> {
  const mobQuery = input.mobQuery.trim();
  if (mobQuery.length < MIN_QUERY) {
    return {
      ok: false,
      error: `digite ao menos ${MIN_QUERY} caracteres do template ou do nome do mob`,
    };
  }
  try {
    const entries = await listMobDrops({ mobQuery, itemIndex: input.itemIndex });
    return { ok: true, entries };
  } catch (error) {
    return failure(error);
  }
}

export type DropRateActionResult = { ok: true } | { ok: false; error: string };

/** Salva o divisor de um slot. O motivo entra na auditoria (`drop_rate_audit`). */
export async function saveDropRate(input: {
  slot: number;
  value: number;
  reason: string;
}): Promise<DropRateActionResult> {
  if (!Number.isInteger(input.slot) || input.slot < 0 || input.slot > 63) {
    return { ok: false, error: "posição inválida" };
  }
  const hard = hardcodedRate(input.slot);
  if (hard !== null) {
    return {
      ok: false,
      error: `esta posição é fixa no código (divisor ${hard}) e não aceita ajuste`,
    };
  }
  if (!Number.isInteger(input.value) || input.value < DROP_RATE_MIN || input.value > DROP_RATE_MAX) {
    return {
      ok: false,
      error: `o divisor precisa ser inteiro entre ${DROP_RATE_MIN} e ${DROP_RATE_MAX.toLocaleString("pt-BR")}`,
    };
  }
  try {
    await adminCall<{ slot: number; value: number; reason: string }, AdminAck>(
      dropAdmin.upsertDropRate.bind(dropAdmin),
      { slot: input.slot, value: input.value, reason: input.reason.trim() },
      { action: "drop.rate.upsert", target: `drop:slot:${input.slot}` },
    );
    revalidatePath("/drops/editar");
    return { ok: true };
  } catch (error) {
    return failure(error);
  }
}

/**
 * Salva a tabela `Carry[]` inteira de um template. Só `slot` e `item_index`
 * viajam: os efeitos do `STRUCT_ITEM` são preservados pelo web-api quando o item
 * não muda, e zero quando muda.
 */
export async function saveMobCarry(input: {
  templateName: string;
  slots: { slot: number; item_index: number }[];
  reason: string;
}): Promise<DropRateActionResult> {
  if (!input.templateName) return { ok: false, error: "escolha um template" };
  const seen = new Set<number>();
  for (const slot of input.slots) {
    if (!Number.isInteger(slot.slot) || slot.slot < 0 || slot.slot >= CARRY_SLOTS) {
      return { ok: false, error: "posição de Carry inválida" };
    }
    if (seen.has(slot.slot)) return { ok: false, error: "posição repetida na tabela" };
    seen.add(slot.slot);
    if (!Number.isInteger(slot.item_index) || slot.item_index <= 0) {
      return { ok: false, error: "toda posição ocupada precisa de um item" };
    }
  }
  try {
    await adminCall<
      { template_name: string; slots: { slot: number; item_index: number }[]; reason: string },
      AdminAck
    >(
      dropAdmin.setMobCarry.bind(dropAdmin),
      { template_name: input.templateName, slots: input.slots, reason: input.reason.trim() },
      { action: "drop.carry.set", target: `drop:carry:${input.templateName}` },
    );
    revalidatePath("/drops/carry");
    return { ok: true };
  } catch (error) {
    return failure(error);
  }
}

/** Remove o overlay do template; o `Carry[]` volta a ser o do arquivo. */
export async function resetMobCarry(input: {
  templateName: string;
  reason: string;
}): Promise<DropRateActionResult> {
  if (!input.templateName) return { ok: false, error: "escolha um template" };
  try {
    await adminCall<{ template_name: string; reason: string }, AdminAck>(
      dropAdmin.resetMobCarry.bind(dropAdmin),
      { template_name: input.templateName, reason: input.reason.trim() },
      { action: "drop.carry.reset", target: `drop:carry:${input.templateName}` },
    );
    revalidatePath("/drops/carry");
    return { ok: true };
  } catch (error) {
    return failure(error);
  }
}

/** Remove o override do slot; volta ao divisor do código. */
export async function resetDropRate(input: {
  slot: number;
  reason: string;
}): Promise<DropRateActionResult> {
  if (!Number.isInteger(input.slot) || input.slot < 0 || input.slot > 63) {
    return { ok: false, error: "posição inválida" };
  }
  try {
    await adminCall<{ slot: number; reason: string }, AdminAck>(
      dropAdmin.resetDropRate.bind(dropAdmin),
      { slot: input.slot, reason: input.reason.trim() },
      { action: "drop.rate.reset", target: `drop:slot:${input.slot}` },
    );
    revalidatePath("/drops/editar");
    return { ok: true };
  } catch (error) {
    return failure(error);
  }
}
