"use server";

/**
 * CRUD da loja de donate + crédito manual de saldo (issue 29).
 *
 * Duas coisas aqui são deliberadamente mais rígidas que o backend:
 *
 * - **item do catálogo**: o `donateshop.Upsert` aceita qualquer `item_index > 0`
 *   e o jogador compraria um slot vazio; a action confere no catálogo.
 * - **apagar**: exige o título digitado. O backend só pede o id; uma oferta de
 *   donate apagada por engano é dinheiro real de alguém.
 *
 * O crédito é a operação mais sensível: `CreditDonateBalance` cria saldo do
 * nada. Motivo obrigatório (>= 10 chars) e confirmação explícita são validados
 * no servidor também, não só na UI, e cada crédito entra na auditoria com
 * `account:<id>`, valor e motivo — nunca o saldo inteiro de terceiros.
 */
import { revalidatePath } from "next/cache";

import {
  validateCredit,
  validateShopItem,
  type CreditInput,
  type DonateShopItem,
} from "@/lib/donate";
import { adminCall } from "@/server/admin";
import { isOk } from "@/server/admin-result";
import { getCatalog } from "@/server/catalog";
import { AdminError, isInfraFailure } from "@/server/errors";
import {
  donateAdmin,
  type AdminAck,
  type CreditDonateBalanceResponse,
  type ListShopItemsResponse,
  type UpsertShopItemResponse,
} from "@/server/admin-client";

export type DonateActionResult =
  | { ok: true; message: string; id?: number; new_balance?: number }
  | { ok: false; error: string };

function toInt(value: unknown): number {
  const n = Math.trunc(Number(value));
  return Number.isFinite(n) ? n : 0;
}

function normalize(input: DonateShopItem): DonateShopItem {
  return {
    id: toInt(input.id),
    item_index: toInt(input.item_index),
    eff1: toInt(input.eff1),
    effv1: toInt(input.effv1),
    eff2: toInt(input.eff2),
    effv2: toInt(input.effv2),
    eff3: toInt(input.eff3),
    effv3: toInt(input.effv3),
    price: toInt(input.price),
    title: String(input.title ?? "").trim(),
    description: String(input.description ?? ""),
    enabled: Boolean(input.enabled),
    expires_days: toInt(input.expires_days),
  };
}

function failure(error: unknown): DonateActionResult {
  if (error instanceof AdminError) return { ok: false, error: error.message };
  if (isInfraFailure(error)) return { ok: false, error: "serviço de administração indisponível" };
  console.error("[donate] falha inesperada:", error instanceof Error ? error.message : String(error));
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

export async function saveShopItem(input: DonateShopItem): Promise<DonateActionResult> {
  const item = normalize(input);

  const errors = validateShopItem(item);
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
    const res = await adminCall<{ item: DonateShopItem }, UpsertShopItemResponse>(
      donateAdmin.upsertShopItem.bind(donateAdmin),
      { item },
      { action: item.id === 0 ? "donate.create" : "donate.update", target: `shop:${item.id || "new"}` },
    );
    if (!isOk(res.result)) return { ok: false, error: "o web-api recusou a oferta" };
    revalidatePath("/loja");
    return {
      ok: true,
      id: Number(res.item_id ?? 0),
      message:
        item.id === 0
          ? `oferta criada por ${item.price} donate — a entrega é no próximo login do personagem`
          : "oferta salva — vale para as próximas compras",
    };
  } catch (error) {
    return failure(error);
  }
}

export async function setShopItemEnabled(input: {
  id: number;
  enabled: boolean;
}): Promise<DonateActionResult> {
  try {
    const res = await adminCall<{ item_id: number; enabled: boolean }, AdminAck>(
      donateAdmin.setShopItemEnabled.bind(donateAdmin),
      { item_id: input.id, enabled: input.enabled },
      { action: "donate.set_enabled", target: `shop:${input.id}` },
    );
    if (!isOk(res.result)) return { ok: false, error: "o web-api recusou a alteração" };
    revalidatePath("/loja");
    return {
      ok: true,
      message: input.enabled ? "oferta habilitada na vitrine" : "oferta desabilitada (sai da vitrine)",
    };
  } catch (error) {
    return failure(error);
  }
}

/**
 * Apagar exige o título digitado: a comparação acontece **no servidor** contra o
 * catálogo atual, então um clique errado na UI não basta para remover a oferta.
 */
export async function deleteShopItem(input: {
  id: number;
  title: string;
}): Promise<DonateActionResult> {
  try {
    const list = await adminCall<Record<string, never>, ListShopItemsResponse>(
      donateAdmin.listShopItems.bind(donateAdmin),
      {},
      { action: "donate.list", target: `shop:${input.id}` },
    );
    const target = (list.items ?? []).find((item) => Number(item.id) === input.id);
    if (!target) return { ok: false, error: "oferta não encontrada — recarregue a lista" };

    const typed = input.title.trim();
    const expected = String(target.title ?? "").trim();
    if (!typed || typed.localeCompare(expected, "pt-BR", { sensitivity: "accent" }) !== 0) {
      return { ok: false, error: "o título digitado não confere com a oferta" };
    }

    const res = await adminCall<{ item_id: number }, AdminAck>(
      donateAdmin.deleteShopItem.bind(donateAdmin),
      { item_id: input.id },
      { action: "donate.delete", target: `shop:${input.id}` },
    );
    if (!isOk(res.result)) return { ok: false, error: "o web-api recusou a remoção" };
    revalidatePath("/loja");
    return { ok: true, message: "oferta apagada — compras já feitas continuam entregues" };
  } catch (error) {
    return failure(error);
  }
}

export async function creditDonateBalance(input: CreditInput): Promise<DonateActionResult> {
  const errors = validateCredit(input);
  const first = Object.values(errors)[0];
  if (first) return { ok: false, error: first };

  const reason = input.reason.trim().slice(0, 280);

  try {
    const res = await adminCall<
      { account_id: number; amount: number; reason: string },
      CreditDonateBalanceResponse
    >(
      donateAdmin.creditDonateBalance.bind(donateAdmin),
      { account_id: input.account_id, amount: input.amount, reason },
      { action: "donate.credit", target: `account:${input.account_id}` },
    );
    if (!isOk(res.result)) {
      return {
        ok: false,
        error:
          res.result === 4 // NOT_FOUND no AdminResult
            ? "conta não encontrada"
            : "o web-api recusou o crédito",
      };
    }
    revalidatePath("/loja");
    return {
      ok: true,
      new_balance: Number(res.new_balance ?? 0),
      message: `crédito de ${input.amount} donate aplicado — saldo agora ${res.new_balance}`,
    };
  } catch (error) {
    return failure(error);
  }
}

