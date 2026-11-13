/**
 * Loja de donate e crédito de saldo (WEB-09, WEB-06).
 *
 * Tudo passa por `adminCall` (sessão de staff + auditoria) e chega ao gateway
 * pelo contrato real (`donate_admin/*`). Dinheiro real nunca passa aqui: o
 * crédito parte de saldo já conciliado fora do portal e cai em `account_wallet`
 * (saldo de **conta**, nunca personagem vivo), com idempotência por chave.
 */
import { adminCall } from "./admin";
import {
  donateAdmin,
  type DonateCreditWire,
  type ListShopResponse,
  type RevisionResponse,
  type ShopItemWire,
} from "./admin-client";

export type { ShopItemWire };

/** Teto de itens por leitura (o mesmo do gateway, `api::donate::MAX_LIMIT`). */
export const MAX_SHOP_PAGE = 200;

export interface ShopItemInput {
  id?: number;
  itemIndex: number;
  price: number;
  durationDays?: number;
  enabled: boolean;
}

export interface DonateCreditInput {
  accountId: number;
  credits: number;
  key: string;
}

/**
 * Valida a forma de um item da vitrine antes do `adminCall`.
 *
 * Espelha o gateway (`api::donate::validar_edicao`): índice e preço positivos,
 * duração opcional positiva. Erro volta como mensagem de tela sem gastar a chamada.
 */
export function checkShopItem(input: {
  itemIndex: number;
  price: number;
  durationDays?: number;
}): string | null {
  if (!Number.isInteger(input.itemIndex) || input.itemIndex <= 0) {
    return "índice do item inválido";
  }
  if (!Number.isInteger(input.price) || input.price <= 0) {
    return "preço em créditos precisa ser positivo";
  }
  if (
    input.durationDays !== undefined &&
    (!Number.isInteger(input.durationDays) || input.durationDays <= 0)
  ) {
    return "duração precisa ser positiva (ou vazia, para permanente)";
  }
  return null;
}

/**
 * Valida a forma de um crédito antes do `adminCall`.
 *
 * Espelha o gateway (`api::donate::credit_balance`): conta positiva, créditos
 * positivos, chave 1..120.
 */
export function checkDonateCredit(input: {
  accountId: number;
  credits: number;
  key: string;
}): string | null {
  if (!Number.isInteger(input.accountId) || input.accountId <= 0) {
    return "conta inválida";
  }
  if (!Number.isInteger(input.credits) || input.credits <= 0) {
    return "créditos precisam ser positivos";
  }
  const key = input.key.trim();
  if (key.length === 0 || key.length > 120) {
    return "chave de idempotência obrigatória (até 120 caracteres)";
  }
  return null;
}

/** Vitrine em páginas por id, mais recente por último. */
export async function listShopItems(afterId?: number): Promise<ShopItemWire[]> {
  const res = await adminCall<{ limit: number; after_id?: number }, ListShopResponse>(
    donateAdmin.listShopItems.bind(donateAdmin),
    { limit: MAX_SHOP_PAGE, ...(afterId ? { after_id: afterId } : {}) },
    { action: "donate.shop.list", target: "shop" },
  );
  return res.shop_items ?? [];
}

/** Cria (`id` ausente) ou substitui um item da vitrine; devolve a revisão. */
export async function upsertShopItem(input: ShopItemInput): Promise<number> {
  const res = await adminCall<
    {
      id?: number;
      item_index: number;
      price_credits: number;
      duration_days?: number;
      enabled: boolean;
    },
    RevisionResponse
  >(
    donateAdmin.upsertShopItem.bind(donateAdmin),
    {
      ...(input.id ? { id: input.id } : {}),
      item_index: input.itemIndex,
      price_credits: input.price,
      ...(input.durationDays !== undefined ? { duration_days: input.durationDays } : {}),
      enabled: input.enabled,
    },
    {
      action: "donate.shop.set",
      target: input.id ? `shop:${input.id}` : "shop:novo",
    },
  );
  return res.revision ?? 0;
}

/** Liga/desliga um item da vitrine; devolve a revisão. */
export async function setShopItemEnabled(id: number, enabled: boolean): Promise<number> {
  const res = await adminCall<{ id: number; enabled: boolean }, RevisionResponse>(
    donateAdmin.setShopItemEnabled.bind(donateAdmin),
    { id, enabled },
    { action: "donate.shop.set", target: `shop:${id}` },
  );
  return res.revision ?? 0;
}

/** Remove um item da vitrine (não apaga compra já feita); devolve a revisão. */
export async function deleteShopItem(id: number): Promise<number> {
  const res = await adminCall<{ id: number }, RevisionResponse>(
    donateAdmin.deleteShopItem.bind(donateAdmin),
    { id },
    { action: "donate.shop.delete", target: `shop:${id}` },
  );
  return res.revision ?? 0;
}

/**
 * Credita saldo de donate na conta, idempotente por chave.
 *
 * Mesma chave devolve o mesmo `credit_id` com `replay: true`, sem creditar duas
 * vezes (`donate_credit`).
 */
export async function creditDonateBalance(input: DonateCreditInput): Promise<DonateCreditWire> {
  const res = await adminCall<
    { account_id: number; credits: number; idempotency_key: string },
    DonateCreditWire
  >(
    donateAdmin.creditDonateBalance.bind(donateAdmin),
    {
      account_id: input.accountId,
      credits: input.credits,
      idempotency_key: input.key.trim(),
    },
    { action: "donate.credit", target: `account:${input.accountId}` },
  );
  return res;
}
