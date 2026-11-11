"use server";

/**
 * Ações da tela de configuração.
 *
 * Cada ação valida a forma, delega ao `adminCall` (autor da sessão, auditoria) e
 * confirma com a revisão devolvida pelo gateway. O servidor de jogo não é acionado
 * por HTTP: ele relê a configuração quando `config_revision` muda, fora do tick.
 */
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

import {
  resetDrop,
  resetRate,
  upsertDrop,
  upsertNpc,
  upsertRate,
  upsertRoute,
} from "@/server/config";
import { setWorldEventEnabled } from "@/server/events";
import { AdminError } from "@/server/errors";

const PATH = "/configuracoes";

function fail(error: unknown): never {
  const message = error instanceof AdminError ? error.message : "falha ao gravar a configuração";
  redirect(`${PATH}?erro=${encodeURIComponent(message)}`);
}

/** Grava em lote só o que mudou (tbody de refino, SancRate, composição, acessório). */
export async function saveRateBatchAction(formData: FormData): Promise<void> {
  const source = String(formData.get("source") ?? "").trim();
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
    fail(error);
  }
  if (changed === 0) redirect(`${PATH}?erro=${encodeURIComponent("nada mudou")}`);
  revalidatePath(PATH);
  redirect(`${PATH}?ok=${changed}`);
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
      const weight = Number(formData.get("weight"));
      const min = Number(formData.get("min_qty"));
      const max = Number(formData.get("max_qty"));
      await upsertDrop({
        template_name: template,
        item_index: item,
        weight,
        min_qty: min,
        max_qty: max,
        source: String(formData.get("source") ?? "").trim(),
      });
    }
  } catch (error) {
    fail(error);
  }
  revalidatePath(PATH);
  redirect(`${PATH}?ok=1`);
}

/** Move/ativa um NPC. */
export async function saveNpcAction(formData: FormData): Promise<void> {
  try {
    await upsertNpc({
      id: Number(formData.get("id")),
      map_id: Number(formData.get("map_id")),
      pos_x: Number(formData.get("pos_x")),
      pos_y: Number(formData.get("pos_y")),
      enabled: String(formData.get("enabled")) === "true",
    });
  } catch (error) {
    fail(error);
  }
  revalidatePath(PATH);
  redirect(`${PATH}?ok=1&npc=${encodeURIComponent(String(formData.get("id") ?? ""))}`);
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
    fail(error);
  }
  revalidatePath(PATH);
  redirect(`${PATH}?ok=1`);
}

/** Liga/desliga um evento de mundo (vale para o próximo leitor do dado). */
export async function setEventAction(formData: FormData): Promise<void> {
  const code = String(formData.get("code") ?? "").trim();
  if (code === "") redirect(`${PATH}?erro=${encodeURIComponent("código de evento inválido")}`);
  try {
    await setWorldEventEnabled(code, String(formData.get("enabled")) === "true");
  } catch (error) {
    fail(error instanceof AdminError ? error.message : "falha ao gravar o evento");
  }
  revalidatePath(PATH);
  redirect(`${PATH}?ok=1`);
}

/** Limpa um override de taxa (botão por linha). */
export async function resetRateAction(formData: FormData): Promise<void> {
  try {
    await resetRate(
      String(formData.get("domain") ?? ""),
      String(formData.get("key") ?? ""),
    );
  } catch (error) {
    fail(error);
  }
  revalidatePath(PATH);
  redirect(`${PATH}?ok=1`);
}
