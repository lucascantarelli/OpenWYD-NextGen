import { describe, expect, it } from "vitest";

import { MAX_PRICES, checkItemPrice } from "@/server/config";

/**
 * A validação de preço espelha o gateway (`api::npc::validar_preco`): item e
 * mercador obrigatórios (a chave de `item_price` é o par), preço não-negativo e
 * fonte obrigatória — mesmo rigor das taxas, onde preço sem procedência não grava.
 */
describe("validação do preço de NPC", () => {
  const base = { itemIndex: 3321, npcId: 1212, buy: 5000, sell: 1250, source: "dono" };

  it("aceita a forma válida, inclusive preço zero", () => {
    expect(checkItemPrice(base)).toBeNull();
    expect(checkItemPrice({ ...base, buy: 0, sell: 0 })).toBeNull();
  });

  it("recusa item e NPC ausentes", () => {
    expect(checkItemPrice({ ...base, itemIndex: 0 })).toBe("índice do item inválido");
    expect(checkItemPrice({ ...base, npcId: 0 })).toBe("NPC do preço é obrigatório");
  });

  it("recusa preço negativo ou fracionário", () => {
    expect(checkItemPrice({ ...base, buy: -1 })).toBe("preço de compra inválido");
    expect(checkItemPrice({ ...base, sell: -1 })).toBe("preço de venda inválido");
    expect(checkItemPrice({ ...base, buy: 1.5 })).toBe("preço de compra inválido");
  });

  it("recusa fonte vazia ou longa demais", () => {
    expect(checkItemPrice({ ...base, source: "   " })).toBe("fonte do preço obrigatória");
    expect(checkItemPrice({ ...base, source: "x".repeat(161) })).toBe(
      "fonte do preço longa demais",
    );
  });

  it("expõe o teto do gateway para a tela limitar a leitura", () => {
    expect(MAX_PRICES).toBe(200);
  });
});
