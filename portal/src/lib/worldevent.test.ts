import { describe, expect, it } from "vitest";

import {
  configChanged,
  emptyConfig,
  rateLabel,
  remainingDrops,
  validateWorldEvent,
  worldEventStatus,
  type WorldEventConfig,
} from "./worldevent";

function active(overrides: Partial<WorldEventConfig> = {}): WorldEventConfig {
  return {
    ...emptyConfig(),
    enabled: true,
    item_index: 777,
    rate: 10,
    start_index: 100,
    current_index: 104,
    end_index: 200,
    ...overrides,
  };
}

describe("validateWorldEvent", () => {
  it("aceita o evento desligado com números zerados", () => {
    expect(validateWorldEvent(emptyConfig())).toEqual({});
  });

  it("aceita uma faixa coerente", () => {
    expect(validateWorldEvent(active())).toEqual({});
  });

  it("exige item, divisor e índice inicial quando ligado", () => {
    const errors = validateWorldEvent(active({ item_index: 0, rate: 0, start_index: 0 }));
    expect(errors.item_index).toBeDefined();
    expect(errors.rate).toBeDefined();
    expect(errors.start_index).toBeDefined();
  });

  it("recusa faixa invertida, progresso fora da faixa e item acima do limite", () => {
    expect(validateWorldEvent(active({ end_index: 100 })).end_index).toBeDefined();
    expect(validateWorldEvent(active({ current_index: 99 })).current_index).toBeDefined();
    expect(validateWorldEvent(active({ current_index: 201 })).current_index).toBeDefined();
    expect(validateWorldEvent(active({ item_index: 40000 })).item_index).toBeDefined();
  });

  it("recusa negativos mesmo com o evento desligado", () => {
    expect(validateWorldEvent(emptyConfig()).end_index).toBeUndefined();
    expect(validateWorldEvent({ ...emptyConfig(), end_index: -1 }).end_index).toBeDefined();
  });

  it("aceita current == end (evento esgotado é config válida)", () => {
    expect(validateWorldEvent(active({ current_index: 200 }))).toEqual({});
  });
});

describe("worldEventStatus", () => {
  it("desligado / ativo / esgotado / incompleto", () => {
    expect(worldEventStatus(emptyConfig()).status).toBe("off");
    expect(worldEventStatus(active()).status).toBe("active");
    expect(worldEventStatus(active({ current_index: 200 })).status).toBe("exhausted");
    expect(worldEventStatus(active({ rate: 0 })).status).toBe("incomplete");
  });
});

describe("rateLabel e progresso", () => {
  it("divisor 1 é sempre, e os demais são 1 em N", () => {
    expect(rateLabel(1)).toContain("sempre");
    expect(rateLabel(10)).toContain("10");
    expect(rateLabel(0)).toContain("sem chance");
  });

  it("remainingDrops para em zero no fim da faixa", () => {
    expect(remainingDrops(active())).toBe(96);
    expect(remainingDrops(active({ current_index: 200 }))).toBe(0);
    expect(remainingDrops(active({ current_index: 260 }))).toBe(0);
  });
});

describe("configChanged", () => {
  it("detecta qualquer campo alterado", () => {
    expect(configChanged(active(), active())).toBe(false);
    expect(configChanged(active({ notice_enabled: true }), active())).toBe(true);
  });
});
