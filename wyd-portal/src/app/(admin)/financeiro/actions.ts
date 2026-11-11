"use server";

/**
 * Loja de donate e crédito de saldo (PORTAL-03, lado staff).
 *
 * Cada ação valida a forma (`checkShopItem`/`checkDonateCredit`), delega ao
 * `adminCall` (autor da sessão, auditoria) e confirma com a revisão ou o recibo
 * do gateway. Dinheiro real nunca passa aqui.
 */
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

import {
  checkDonateCredit,
  checkShopItem,
  creditDonateBalance,
  deleteShopItem,
  setShopItemEnabled,
  upsertShopItem,
} from "@/server/donate";
import { AdminError } from "@/server/errors";

const PATH = "/financeiro";

function fail(message: string): never {
  redirect(`${PATH}?erro=${encodeURIComponent(message)}`);
}

/** Cria ou substitui um item da vitrine. */
export async function saveShopItemAction(formData: FormData): Promise<void> {
  const rawId = String(formData.get("id") ?? "").trim();
  const durationRaw = String(formData.get("duration_days") ?? "").trim();
  const input = {
    ...(rawId ? { id: Number(rawId) } : {}),
    itemIndex: Number(formData.get("item_index")),
    price: Number(formData.get("price_credits")),
    ...(durationRaw ? { durationDays: Number(durationRaw) } : {}),
    enabled: String(formData.get("enabled")) === "true",
  };
  const invalid = checkShopItem(input);
  if (invalid) fail(invalid);
  try {
    await upsertShopItem(input);
  } catch (error) {
    fail(error instanceof AdminError ? error.message : "falha ao gravar o item da loja");
  }
  revalidatePath(PATH);
  redirect(`${PATH}?ok=loja`);
}

/** Liga/desliga ou remove um item da vitrine. */
export async function setShopItemAction(formData: FormData): Promise<void> {
  const id = Number(formData.get("id"));
  const intent = String(formData.get("intent") ?? "");
  if (!Number.isInteger(id) || id <= 0) fail("item da loja inválido");
  try {
    if (intent === "delete") {
      await deleteShopItem(id);
    } else if (intent === "enable" || intent === "disable") {
      await setShopItemEnabled(id, intent === "enable");
    } else {
      fail("intenção inválida");
    }
  } catch (error) {
    fail(error instanceof AdminError ? error.message : "falha ao atualizar o item da loja");
  }
  revalidatePath(PATH);
  redirect(`${PATH}?ok=loja`);
}

/** Credita saldo de donate na conta (idempotente por chave). */
export async function creditDonateAction(formData: FormData): Promise<void> {
  const input = {
    accountId: Number(formData.get("account_id")),
    credits: Number(formData.get("credits")),
    key: String(formData.get("key") ?? ""),
  };
  const invalid = checkDonateCredit(input);
  if (invalid) fail(invalid);
  try {
    const receipt = await creditDonateBalance({ ...input, key: input.key.trim() });
    revalidatePath(PATH);
    redirect(
      `${PATH}?ok=${encodeURIComponent(receipt.replay ? `replay #${receipt.credit_id}` : `crédito #${receipt.credit_id}`)}`,
    );
  } catch (error) {
    fail(error instanceof AdminError ? error.message : "falha ao creditar o saldo");
  }
}
