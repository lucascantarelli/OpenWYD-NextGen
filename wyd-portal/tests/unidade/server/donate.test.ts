import { describe, expect, it } from "vitest";

import { checkDonateCredit, checkShopItem } from "@/server/donate";

/**
 * As validações espelham o gateway (`api::donate::validar_edicao` e
 * `credit_balance`): erro aqui volta como mensagem de tela sem gastar a
 * chamada e sem tocar auditoria.
 */
describe("validação do item da vitrine", () => {
  it("aceita item permanente e com duração", () => {
    expect(checkShopItem({ itemIndex: 3321, price: 100 })).toBeNull();
    expect(checkShopItem({ itemIndex: 3321, price: 100, durationDays: 30 })).toBeNull();
  });

  it("recusa índice e preço inválidos", () => {
    expect(checkShopItem({ itemIndex: 0, price: 100 })).toBe("índice do item inválido");
    expect(checkShopItem({ itemIndex: 3321, price: 0 })).toBe(
      "preço em créditos precisa ser positivo",
    );
    expect(checkShopItem({ itemIndex: 3321, price: 1.5 })).toBe(
      "preço em créditos precisa ser positivo",
    );
  });

  it("recusa duração não-positiva, mas aceita ausência", () => {
    expect(checkShopItem({ itemIndex: 3321, price: 100, durationDays: 0 })).toBe(
      "duração precisa ser positiva (ou vazia, para permanente)",
    );
    expect(checkShopItem({ itemIndex: 3321, price: 100, durationDays: -7 })).toBe(
      "duração precisa ser positiva (ou vazia, para permanente)",
    );
  });
});

describe("validação do crédito de donate", () => {
  const base = { accountId: 9, credits: 500, key: "conciliacao-2026-001" };

  it("aceita a forma válida", () => {
    expect(checkDonateCredit(base)).toBeNull();
  });

  it("recusa conta e créditos inválidos", () => {
    expect(checkDonateCredit({ ...base, accountId: 0 })).toBe("conta inválida");
    expect(checkDonateCredit({ ...base, credits: -10 })).toBe("créditos precisam ser positivos");
  });

  it("recusa chave vazia ou longa demais", () => {
    expect(checkDonateCredit({ ...base, key: "   " })).toBe(
      "chave de idempotência obrigatória (até 120 caracteres)",
    );
    expect(checkDonateCredit({ ...base, key: `x${"y".repeat(120)}` })).toBe(
      "chave de idempotência obrigatória (até 120 caracteres)",
    );
  });
});
