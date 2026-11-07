import { describe, expect, it } from "vitest";

import {
  EFFECT_MAX,
  EXPIRES_MAX,
  activeEffects,
  effectsSummary,
  emptyDailyReward,
  expiryLabel,
  validateDailyReward,
  type DailyRewardItem,
} from "./daily";

function offer(overrides: Partial<DailyRewardItem> = {}): DailyRewardItem {
  return { ...emptyDailyReward(), item_index: 400, title: "Poção diária", ...overrides };
}

describe("validateDailyReward", () => {
  it("aceita uma oferta completa", () => {
    expect(validateDailyReward(offer())).toEqual({});
  });

  it("exige título entre 3 e 60 caracteres (sem contar espaço nas pontas)", () => {
    expect(validateDailyReward(offer({ title: "ab" })).title).toBeDefined();
    expect(validateDailyReward(offer({ title: "  ab  " })).title).toBeDefined();
    expect(validateDailyReward(offer({ title: "abc" })).title).toBeUndefined();
    expect(validateDailyReward(offer({ title: "x".repeat(61) })).title).toBeDefined();
  });

  it("limita a descrição a 280 caracteres", () => {
    expect(validateDailyReward(offer({ description: "x".repeat(281) })).description).toBeDefined();
    expect(validateDailyReward(offer({ description: "x".repeat(280) })).description).toBeUndefined();
  });

  it("exige item e recusa validade fora da faixa", () => {
    expect(validateDailyReward(offer({ item_index: 0 })).item_index).toBeDefined();
    expect(validateDailyReward(offer({ expires_days: -1 })).expires_days).toBeDefined();
    expect(validateDailyReward(offer({ expires_days: EXPIRES_MAX + 1 })).expires_days).toBeDefined();
    expect(validateDailyReward(offer({ expires_days: EXPIRES_MAX })).expires_days).toBeUndefined();
  });

  it("mantém efeito e valor dentro do uint8 da STRUCT_ITEM", () => {
    expect(validateDailyReward(offer({ eff1: EFFECT_MAX + 1 })).eff1).toBeDefined();
    expect(validateDailyReward(offer({ effv2: -1 })).effv2).toBeDefined();
    expect(validateDailyReward(offer({ eff1: 8, effv1: 20 }))).toEqual({});
  });
});

describe("rótulos", () => {
  it("validade 0 é permanente", () => {
    expect(expiryLabel(0)).toBe("permanente");
    expect(expiryLabel(1)).toBe("1 dia");
    expect(expiryLabel(7)).toBe("7 dias");
  });

  it("resumo ignora pares zerados", () => {
    expect(effectsSummary(offer())).toBe("—");
    expect(effectsSummary(offer({ eff1: 8, effv1: 20 }))).toBe("1 adicional");
    expect(effectsSummary(offer({ eff1: 8, effv1: 20, eff2: 9, effv2: 1 }))).toBe("2 adicionais");
  });

  it("activeEffects preserva a ordem do proto", () => {
    const effects = activeEffects(offer({ eff3: 7, effv3: 3, eff1: 8, effv1: 20 }));
    expect(effects).toEqual([
      { effect: 8, value: 20 },
      { effect: 7, value: 3 },
    ]);
  });
});
