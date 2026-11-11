"use server";

/**
 * Ações da tela de configuração.
 *
 * Cada ação valida a forma, delega ao `adminCall` (autor da sessão, auditoria) e
 * confirma com a revisão devolvida pelo gateway. A procedência da edição é
 * derivada da sessão (`portal#<conta>`) — o staff não digita fonte em formulário,
 * e a trilha continua com autor e valor anterior.
 */
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

import {
  checkItemPrice,
  resetDrop,
  resetRate,
  upsertDrop,
  upsertItemPrice,
  upsertNpc,
  upsertRate,
  upsertRoute,
} from "@/server/config";
import { setWorldEventEnabled } from "@/server/events";
import { AdminError } from "@/server/errors";
import { getSession } from "@/server/session";

const PATH = "/configuracoes";

/** Procedência gravada junto do valor: quem editou pelo portal. */
async function procedencia(): Promise<string> {
  const session = await getSession();
  return session ? `portal#${session.accountId}` : "portal";
}

function back(tab: string, extra: Record<string, string> = {}): string {
  const query = new URLSearchParams({ aba: tab, ...extra });
  return `${PATH}?${query.toString()}`;
}

function fail(error: unknown, tab: string): never {
  const message = error instanceof AdminError ? error.message : "falha ao gravar a configuração";
  redirect(back(tab, { erro: message }));
}

/** Grava em lote só o que mudou (taxas de refino, SancRate, composição, acessório). */
export async function saveRateBatchAction(formData: FormData): Promise<void> {
  const tab = String(formData.get("aba") ?? "refino");
  const source = await procedencia();
  let changed = 0;
  try {
    for (const [name, raw] of formData.entries()) {
      if (!name.startsWith("rate:")) continue;
      const key = name.slice("rate:".length);
      const value = Number(raw);
      if (!Number.isFinite(value)) continue;
      const original = String(formData.get(`orig:${key}`) ?? "");
      if (String(value) === original) continue;
      const cut = key.indexOf(":");
      if (cut <= 0) continue;
      await upsertRate({
        domain: key.slice(0, cut),
        key: key.slice(cut + 1),
        value,
        source,
      });
      changed += 1;
    }
  } catch (error) {
    fail(error, tab);
  }
  if (changed === 0) redirect(back(tab, { erro: "nada mudou" }));
  revalidatePath(PATH);
  redirect(back(tab, { ok: String(changed) }));
}

/** Um peso de drop: salvar ou remover. */
export async function saveDropAction(formData: FormData): Promise<void> {
  const template = String(formData.get("template_name") ?? "").trim();
  const item = Number(formData.get("item_index"));
  const intent = String(formData.get("intent") ?? "save");
  try {
    if (intent === "reset") {
      await resetDrop(template, item);
    } else {
      await upsertDrop({
        template_name: template,
        item_index: item,
        weight: Number(formData.get("weight")),
        min_qty: Number(formData.get("min_qty")),
        max_qty: Number(formData.get("max_qty")),
        source: await procedencia(),
      });
    }
  } catch (error) {
    fail(error, "drops");
  }
  revalidatePath(PATH);
  redirect(back("drops", { ok: "1" }));
}

/** Move/ativa um NPC. */
export async function saveNpcAction(formData: FormData): Promise<void> {
  const id = String(formData.get("id") ?? "");
  try {
    await upsertNpc({
      id: Number(id),
      map_id: Number(formData.get("map_id")),
      pos_x: Number(formData.get("pos_x")),
      pos_y: Number(formData.get("pos_y")),
      enabled: String(formData.get("enabled")) === "true",
    });
  } catch (error) {
    fail(error, "npc");
  }
  revalidatePath(PATH);
  redirect(back("npc", { ok: "1", npc: id }));
}

/** Custo/nível/disponibilidade de uma rota de teleporte. */
export async function saveRouteAction(formData: FormData): Promise<void> {
  try {
    await upsertRoute({
      id: Number(formData.get("id")),
      cost_gold: Number(formData.get("cost_gold")),
      req_level: Number(formData.get("req_level")),
      enabled: String(formData.get("enabled")) === "true",
    });
  } catch (error) {
    fail(error, "teleportes");
  }
  revalidatePath(PATH);
  redirect(back("teleportes", { ok: "1" }));
}

/** Preço de NPC por (item, mercador): compra e venda. */
export async function saveItemPriceAction(formData: FormData): Promise<void> {
  const itemIndex = Number(formData.get("item_index"));
  const input = {
    itemIndex,
    npcId: Number(formData.get("npc_id")),
    buy: Number(formData.get("buy_price")),
    sell: Number(formData.get("sell_price")),
    source: await procedencia(),
  };
  const invalid = checkItemPrice(input);
  if (invalid) redirect(back("precos", { erro: invalid }));
  try {
    await upsertItemPrice(input);
  } catch (error) {
    fail(error, "precos");
  }
  revalidatePath(PATH);
  redirect(back("precos", { ok: "1", item: String(itemIndex) }));
}

/** Liga/desliga um evento de mundo (vale para o próximo leitor do dado). */
export async function setEventAction(formData: FormData): Promise<void> {
  const code = String(formData.get("code") ?? "").trim();
  if (code === "") redirect(back("eventos", { erro: "código de evento inválido" }));
  try {
    await setWorldEventEnabled(code, String(formData.get("enabled")) === "true");
  } catch (error) {
    fail(error, "eventos");
  }
  revalidatePath(PATH);
  redirect(back("eventos", { ok: "1" }));
}

/** Volta uma linha ao valor de fábrica. */
export async function resetRateAction(formData: FormData): Promise<void> {
  try {
    await resetRate(String(formData.get("domain") ?? ""), String(formData.get("key") ?? ""));
  } catch (error) {
    fail(error, "refino");
  }
  revalidatePath(PATH);
  redirect(back("refino", { ok: "1" }));
}
