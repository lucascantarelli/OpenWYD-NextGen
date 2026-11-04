import { describe, expect, it } from "vitest";

import {
  CREDIT_MAX,
  DESCRIPTION_MAX,
  EXPIRES_MAX,
  PRICE_MAX,
  REASON_MIN,
  activeEffects,
  emptyShopItem,
  effectsSummary,
  expiryLabel,
  fmtDonate,
  priceLabel,
  validateCredit,
  validateShopItem,
  type CreditInput,
  type DonateShopItem,
} from "./donate";

function offer(overrides: Partial<DonateShopItem> = {}): DonateShopItem {
  return { ...emptyShopItem(), item_index: 400, price: 5000, title: "Poção de cura", ...overrides };
}

function credit(overrides: Partial<CreditInput> = {}): CreditInput {
  return {
    account_id: 1,
    amount: 100,
    reason: "bonificação do evento de lançamento",
    confirmed: true,
    ...overrides,
  };
}

describe("validateShopItem", () => {
  it("aceita uma oferta completa", () => {
    expect(validateShopItem(offer())).toEqual({});
  });

  it("exige item do catálogo", () => {
    expect(validateShopItem(offer({ item_index: 0 })).item_index).toMatch(/escolha o item/);
  });

  it("não deixa preço zerado — donate é a moeda da loja", () => {
    expect(validateShopItem(offer({ price: 0 })).price).toMatch(/maior que zero/);
  });

  it("recusa preço acima do teto e cita donate", () => {
    const error = validateShopItem(offer({ price: PRICE_MAX + 1 })).price ?? "";
    expect(error).toMatch(/máximo/);
    expect(error).toMatch(/donate/);
  });

  it("limita título, descrição e validade", () => {
    expect(validateShopItem(offer({ title: "ab" })).title).toMatch(/mínimo/);
    expect(validateShopItem(offer({ title: "a".repeat(61) })).title).toMatch(/máximo/);
    expect(validateShopItem(offer({ description: "x".repeat(DESCRIPTION_MAX + 1) })).description)
      .toMatch(/máximo/);
    expect(validateShopItem(offer({ expires_days: -1 })).expires_days).toMatch(/negativo/);
    expect(validateShopItem(offer({ expires_days: EXPIRES_MAX + 1 })).expires_days).toMatch(/máximo/);
  });

  it("mantém efeito e valor dentro do uint8 da STRUCT_ITEM", () => {
    const errors = validateShopItem(offer({ eff1: 256, effv1: -1 }));
    expect(errors.eff1).toMatch(/entre 0 e 255/);
    expect(errors.effv1).toMatch(/entre 0 e 255/);
  });
});

describe("validateCredit", () => {
  it("aceita crédito com motivo e confirmação", () => {
    expect(validateCredit(credit())).toEqual({});
  });

  it("exige id de conta positivo", () => {
    expect(validateCredit(credit({ account_id: 0 })).account_id).toMatch(/escolha a conta/);
  });

  it("exige motivo com tamanho de auditoria", () => {
    const error = validateCredit(credit({ reason: "ok" })).reason ?? "";
    expect(error).toMatch(new RegExp(`mínimo de ${REASON_MIN}`));
    expect(error).toMatch(/auditoria/);
  });

  it("exige a confirmação dupla", () => {
    expect(validateCredit(credit({ confirmed: false })).confirmed).toMatch(/confirme/);
  });

  it("limita o valor creditado", () => {
    expect(validateCredit(credit({ amount: 0 })).amount).toMatch(/maior que zero/);
    expect(validateCredit(credit({ amount: CREDIT_MAX + 1 })).amount).toMatch(/máximo/);
  });
});

describe("rótulos", () => {
  it("formata preço em donate com separador pt-BR", () => {
    expect(fmtDonate(5000)).toBe("5.000");
    expect(priceLabel(5000)).toBe("5.000 donate");
    expect(priceLabel(0)).toBe("0 donate");
  });

  it("traduz a validade como o cliente mostra", () => {
    expect(expiryLabel(0)).toBe("permanente");
    expect(expiryLabel(1)).toBe("1 dia");
    expect(expiryLabel(7)).toBe("7 dias");
  });

  it("resume apenas os adicionais preenchidos", () => {
    expect(activeEffects(offer())).toEqual([]);
    expect(effectsSummary(offer())).toBe("—");
    expect(effectsSummary(offer({ eff2: 5, effv2: 10 }))).toBe("1 adicional");
    expect(effectsSummary(offer({ eff1: 1, effv1: 1, eff3: 2, effv3: 3 }))).toBe("2 adicionais");
  });
});
