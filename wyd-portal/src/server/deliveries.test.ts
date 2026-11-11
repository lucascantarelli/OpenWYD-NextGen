import { describe, expect, it } from "vitest";

import { checkItemGrant } from "./deliveries";

/**
 * A validação do grant de item espelha o gateway (`validate_request`):
 * personagem/item positivos, quantidade 1..999 (cabe no slot), chave 8..120.
 * Erro aqui volta como mensagem de tela sem gastar a chamada.
 */
describe("validação do grant de item", () => {
  const base = { characterId: 9, itemId: 3321, qty: 1, key: "correio-9-3321-x" };

  it("aceita a forma válida", () => {
    expect(checkItemGrant(base)).toBeNull();
  });

  it("recusa personagem e item inválidos", () => {
    expect(checkItemGrant({ ...base, characterId: 0 })).toBe("personagem inválido");
    expect(checkItemGrant({ ...base, itemId: -1 })).toBe("índice do item inválido");
  });

  it("recusa quantidade fora de 1..999", () => {
    expect(checkItemGrant({ ...base, qty: 0 })).toBe("quantidade precisa ser 1..999");
    expect(checkItemGrant({ ...base, qty: 1000 })).toBe("quantidade precisa ser 1..999");
    expect(checkItemGrant({ ...base, qty: 1.5 })).toBe("quantidade precisa ser 1..999");
  });

  it("recusa chave curta ou longa demais", () => {
    expect(checkItemGrant({ ...base, key: "curta" })).toBe(
      "chave de idempotência precisa de 8..120 caracteres",
    );
    expect(checkItemGrant({ ...base, key: `x${"y".repeat(120)}` })).toBe(
      "chave de idempotência precisa de 8..120 caracteres",
    );
  });
});
