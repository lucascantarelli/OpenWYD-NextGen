import { describe, expect, it } from "vitest";

import { MAX_ACCOUNT_PAGE, parseAccountCursor } from "@/server/accounts";

/**
 * A paginação de contas usa cursor opaco (`after_id`): a query string pode
 * trazer lixo, e lixo vira \"primeira página\", nunca chamada inválida ao
 * gateway (que recusaria `after_id` negativo com 422).
 */
describe("cursor de paginação de contas", () => {
  it("aceita inteiro positivo", () => {
    expect(parseAccountCursor("42")).toBe(42);
    expect(parseAccountCursor(["7"])).toBe(7);
  });

  it("rejeita ausência, lixo e não-positivo", () => {
    expect(parseAccountCursor(undefined)).toBeUndefined();
    expect(parseAccountCursor("")).toBeUndefined();
    expect(parseAccountCursor("abc")).toBeUndefined();
    expect(parseAccountCursor("0")).toBeUndefined();
    expect(parseAccountCursor("-3")).toBeUndefined();
    expect(parseAccountCursor("4.5")).toBeUndefined();
  });

  it("expõe o teto do gateway para a tela limitar a página", () => {
    expect(MAX_ACCOUNT_PAGE).toBe(200);
  });
});
