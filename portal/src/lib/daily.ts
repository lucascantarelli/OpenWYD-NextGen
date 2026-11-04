/**
 * Catálogo da recompensa diária (issue 30 / `DailyRewardAdminService`).
 *
 * A oferta é **grátis** e sem preço — o que a limita é a unicidade de
 * `daily_reward_claim (account_id, claim_date)` no banco (dia UTC) e o
 * `delivery_queue` que o tmserver drena no próximo login. Ou seja: duas ofertas
 * habilitadas **não** significam dois resgates no mesmo dia; a tela diz isso.
 *
 * O backend (`webserver/internal/dailyreward`) só valida `item_index > 0` e
 * `expires_days >= 0`; título/descrição são limites de UX e valem aqui e na
 * action do BFF, para a oferta não nascer ilegível.
 */
export interface DailyRewardItem {
  /** 0 = cria; qualquer outro valor = atualiza aquela oferta. */
  id: number;
  item_index: number;
  eff1: number;
  effv1: number;
  eff2: number;
  effv2: number;
  eff3: number;
  effv3: number;
  title: string;
  description: string;
  enabled: boolean;
  /** 0 = item permanente; > 0 = item com prazo em dias. */
  expires_days: number;
}

export const TITLE_MIN = 3;
export const TITLE_MAX = 60;
export const DESCRIPTION_MAX = 280;
export const EXPIRES_MAX = 3650;
/** `STRUCT_ITEM` guarda efeito e valor em `uint8`. */
export const EFFECT_MAX = 255;

export function emptyDailyReward(): DailyRewardItem {
  return {
    id: 0,
    item_index: 0,
    eff1: 0,
    effv1: 0,
    eff2: 0,
    effv2: 0,
    eff3: 0,
    effv3: 0,
    title: "",
    description: "",
    enabled: true,
    expires_days: 0,
  };
}

export type DailyErrors = Partial<Record<keyof DailyRewardItem, string>>;

export function validateDailyReward(item: DailyRewardItem): DailyErrors {
  const errors: DailyErrors = {};

  const title = item.title.trim();
  if (title.length < TITLE_MIN) errors.title = `mínimo de ${TITLE_MIN} caracteres`;
  else if (title.length > TITLE_MAX) errors.title = `máximo de ${TITLE_MAX} caracteres`;

  if (item.description.length > DESCRIPTION_MAX)
    errors.description = `máximo de ${DESCRIPTION_MAX} caracteres`;

  if (!Number.isInteger(item.item_index) || item.item_index <= 0)
    errors.item_index = "escolha o item entregue";

  if (!Number.isInteger(item.expires_days) || item.expires_days < 0)
    errors.expires_days = "não pode ser negativo";
  else if (item.expires_days > EXPIRES_MAX)
    errors.expires_days = `máximo de ${EXPIRES_MAX} dias`;

  const pairs: [keyof DailyRewardItem, keyof DailyRewardItem][] = [
    ["eff1", "effv1"],
    ["eff2", "effv2"],
    ["eff3", "effv3"],
  ];
  for (const [effect, value] of pairs) {
    // `keyof` mistura string/bool; estes seis campos são numéricos por contrato.
    const code = Number(item[effect]);
    const amount = Number(item[value]);
    if (!Number.isInteger(code) || code < 0 || code > EFFECT_MAX)
      errors[effect] = `código entre 0 e ${EFFECT_MAX}`;
    if (!Number.isInteger(amount) || amount < 0 || amount > EFFECT_MAX)
      errors[value] = `valor entre 0 e ${EFFECT_MAX}`;
  }

  return errors;
}

/** `0` é permanente; qualquer outro valor é prazo em dias (como no cliente). */
export function expiryLabel(days: number): string {
  if (days <= 0) return "permanente";
  if (days === 1) return "1 dia";
  return `${days} dias`;
}

/** Pares não zerados, na ordem do proto (para resumo na tabela). */
export function activeEffects(item: DailyRewardItem): { effect: number; value: number }[] {
  const pairs: [number, number][] = [
    [item.eff1, item.effv1],
    [item.eff2, item.effv2],
    [item.eff3, item.effv3],
  ];
  return pairs
    .filter(([effect, value]) => effect !== 0 || value !== 0)
    .map(([effect, value]) => ({ effect, value }));
}

/** Resumo curto da coluna "adicionais". */
export function effectsSummary(item: DailyRewardItem): string {
  const count = activeEffects(item).length;
  if (count === 0) return "—";
  return count === 1 ? "1 adicional" : `${count} adicionais`;
}
