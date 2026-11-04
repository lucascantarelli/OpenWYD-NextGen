/**
 * Loja de donate (issue 29 / `DonateAdminService`).
 *
 * O preço é em **donate**, não em coin: donate é a moeda da loja, creditada na
 * carteira da conta (`account.donate_balance`) e nunca confundida com o ouro do
 * personagem (issue 24). A compra do jogador debita a carteira e enfileira o item
 * em `delivery_queue`, drenado no **próximo login** do personagem — nada aparece
 * em tempo real, e a tela avisa isso em toda oferta.
 *
 * O backend (`webserver/internal/donateshop`) valida `item_index > 0`,
 * `price > 0` e `expires_days >= 0`; título/descrição e o teto de preço são
 * limites de UX aplicados aqui e na action do BFF.
 */
export interface DonateShopItem {
  /** 0 = cria; qualquer outro valor = atualiza aquela oferta. */
  id: number;
  item_index: number;
  eff1: number;
  effv1: number;
  eff2: number;
  effv2: number;
  eff3: number;
  effv3: number;
  /** Preço em unidades de donate (> 0). */
  price: number;
  title: string;
  description: string;
  enabled: boolean;
  /** 0 = item permanente; > 0 = item com prazo em dias. */
  expires_days: number;
}

export const TITLE_MIN = 3;
export const TITLE_MAX = 60;
export const DESCRIPTION_MAX = 280;
export const PRICE_MIN = 1;
export const PRICE_MAX = 1_000_000;
export const EXPIRES_MAX = 3650;
/** `STRUCT_ITEM` guarda efeito e valor em `uint8`. */
export const EFFECT_MAX = 255;
/** Motivo do crédito manual: curto demais não explica nada na auditoria. */
export const REASON_MIN = 10;
export const REASON_MAX = 280;
export const CREDIT_MIN = 1;
export const CREDIT_MAX = 1_000_000;

export function emptyShopItem(): DonateShopItem {
  return {
    id: 0,
    item_index: 0,
    eff1: 0,
    effv1: 0,
    eff2: 0,
    effv2: 0,
    eff3: 0,
    effv3: 0,
    price: 0,
    title: "",
    description: "",
    enabled: false,
    expires_days: 0,
  };
}

export type ShopErrors = Partial<Record<keyof DonateShopItem, string>>;

export function validateShopItem(item: DonateShopItem): ShopErrors {
  const errors: ShopErrors = {};

  const title = item.title.trim();
  if (title.length < TITLE_MIN) errors.title = `mínimo de ${TITLE_MIN} caracteres`;
  else if (title.length > TITLE_MAX) errors.title = `máximo de ${TITLE_MAX} caracteres`;

  if (item.description.length > DESCRIPTION_MAX)
    errors.description = `máximo de ${DESCRIPTION_MAX} caracteres`;

  if (!Number.isInteger(item.item_index) || item.item_index <= 0)
    errors.item_index = "escolha o item vendido";

  if (!Number.isInteger(item.price) || item.price < PRICE_MIN)
    errors.price = "o preço é em donate e precisa ser maior que zero";
  else if (item.price > PRICE_MAX) errors.price = `máximo de ${fmtDonate(PRICE_MAX)} donate`;

  if (!Number.isInteger(item.expires_days) || item.expires_days < 0)
    errors.expires_days = "não pode ser negativo";
  else if (item.expires_days > EXPIRES_MAX)
    errors.expires_days = `máximo de ${EXPIRES_MAX} dias`;

  const pairs: [keyof DonateShopItem, keyof DonateShopItem][] = [
    ["eff1", "effv1"],
    ["eff2", "effv2"],
    ["eff3", "effv3"],
  ];
  for (const [effect, value] of pairs) {
    // `keyof` mistura string/bool/numérico; estes seis campos são números por contrato.
    const code = Number(item[effect]);
    const amount = Number(item[value]);
    if (!Number.isInteger(code) || code < 0 || code > EFFECT_MAX)
      errors[effect] = `código entre 0 e ${EFFECT_MAX}`;
    if (!Number.isInteger(amount) || amount < 0 || amount > EFFECT_MAX)
      errors[value] = `valor entre 0 e ${EFFECT_MAX}`;
  }

  return errors;
}

export interface CreditInput {
  account_id: number;
  amount: number;
  reason: string;
  /** Confirmação dupla: o crédito não tem rollback automático. */
  confirmed: boolean;
}

export type CreditErrors = Partial<Record<keyof CreditInput, string>>;

/**
 * Crédito manual é a operação mais sensível da loja: cria saldo do nada. Motivo
 * obrigatório e confirmação explícita são pré-requisitos, não decoração.
 */
export function validateCredit(input: CreditInput): CreditErrors {
  const errors: CreditErrors = {};

  if (!Number.isInteger(input.account_id) || input.account_id <= 0)
    errors.account_id = "escolha a conta que vai receber o crédito";

  if (!Number.isInteger(input.amount) || input.amount < CREDIT_MIN)
    errors.amount = "o valor precisa ser maior que zero";
  else if (input.amount > CREDIT_MAX) errors.amount = `máximo de ${fmtDonate(CREDIT_MAX)} donate`;

  const reason = input.reason.trim();
  if (reason.length < REASON_MIN)
    errors.reason = `mínimo de ${REASON_MIN} caracteres — o motivo vai para a auditoria`;
  else if (reason.length > REASON_MAX) errors.reason = `máximo de ${REASON_MAX} caracteres`;

  if (!input.confirmed) errors.confirmed = "confirme que o crédito é manual e auditado";

  return errors;
}

const donateFormat = new Intl.NumberFormat("pt-BR");

/** `5.000 donate` — o rótulo nunca é só o número, para não virar coin. */
export function fmtDonate(amount: number): string {
  return donateFormat.format(Number.isFinite(amount) ? amount : 0);
}

export function priceLabel(price: number): string {
  return `${fmtDonate(price)} donate`;
}

/** `0` é permanente; qualquer outro valor é prazo em dias (como no cliente). */
export function expiryLabel(days: number): string {
  if (days <= 0) return "permanente";
  if (days === 1) return "1 dia";
  return `${days} dias`;
}

/** Pares não zerados, na ordem do proto (para resumo na tabela). */
export function activeEffects(item: DonateShopItem): { effect: number; value: number }[] {
  const pairs: [number, number][] = [
    [item.eff1, item.effv1],
    [item.eff2, item.effv2],
    [item.eff3, item.effv3],
  ];
  return pairs
    .filter(([effect, value]) => effect !== 0 || value !== 0)
    .map(([effect, value]) => ({ effect, value }));
}

export function effectsSummary(item: DonateShopItem): string {
  const count = activeEffects(item).length;
  if (count === 0) return "—";
  return count === 1 ? "1 adicional" : `${count} adicionais`;
}
