/**
 * Preços globais de item (issue 24).
 *
 * Regra do backend (`§5.3`): `SetItemPrice(item_index, price)` com `price >= 0`
 * define o preço **global** — vale em todas as lojas que não fixam preço — e
 * `price < 0` limpa o override, voltando ao preço do `ItemList.csv`.
 *
 * Não existe preço por loja: `AdminNpcShopItem` não carrega preço.
 */
import { z } from "zod";

export const priceSchema = z.object({
  itemIndex: z.number().int().min(1, "índice de item inválido"),
  /** `-1` limpa o override; `0` é preço válido (item de graça). */
  price: z
    .number()
    .int("use um inteiro")
    .min(-1, "preço mínimo é 0 (ou -1 para limpar)")
    .max(2_000_000_000, "preço grande demais"),
});

export type PriceInput = z.infer<typeof priceSchema>;

/** `-1` é sentinela de limpeza, nunca um preço gravado. */
export const CLEAR_PRICE = -1;

export interface OverrideRow {
  index: number;
  price: number;
}
