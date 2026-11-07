import { describe, expect, it } from "vitest";

import { effectiveDropRate, hardcodedRate, oddsLabel, slotGroup } from "./droprate";

/** Mesma tabela de `internal/droprate/droprate.go` (Basedef.cpp:222). */
const BASE = (() => {
  const out = new Array<number>(64).fill(0);
  const fill = (from: number, to: number, value: number) => {
    for (let i = from; i <= to; i++) out[i] = value;
  };
  fill(0, 7, 900);
  fill(8, 11, 4);
  fill(12, 15, 900);
  fill(16, 23, 20000);
  fill(24, 47, 2000);
  fill(48, 55, 3000);
  out[56] = 1;
  [35, 500, 2500, 5000, 5000, 10000, 20000].forEach((value, i) => {
    out[57 + i] = value;
  });
  return out;
})();

describe("effectiveDropRate (espelho de loot.go)", () => {
  // Os casos foram conferidos contra o Go: divisões inteiras, mesmos cortes.
  it("aplica o corte por nível do bloco 0–2", () => {
    expect(effectiveDropRate(12, 0, 50, BASE)).toBe(72); // 900 * 8%
    expect(effectiveDropRate(0, 0, 5, BASE)).toBe(36); // 900 * 4%
  });

  it("aplica o corte do bloco ultra raro em nível alto", () => {
    expect(effectiveDropRate(16, 0, 250, BASE)).toBe(19800); // 20000 * 99%
  });

  it("multiplica pelo bônus do matador antes do nível", () => {
    expect(effectiveDropRate(12, 200, 50, BASE)).toBe(23); // 900 -> 297 -> 8%
  });

  it("respeita os slots fixos no código", () => {
    expect(effectiveDropRate(11, 0, 250, BASE)).toBe(1);
    expect(effectiveDropRate(8, 500, 5, BASE)).toBe(4);
    expect(hardcodedRate(9)).toBe(4);
    expect(hardcodedRate(12)).toBeNull();
  });

  it("limita o resultado a 32.000", () => {
    expect(effectiveDropRate(56, 0, 250, BASE)).toBeLessThanOrEqual(32000);
  });
});

describe("oddsLabel", () => {
  it("não inventa porcentagem nem '1 em 0'", () => {
    expect(oddsLabel(1)).toBe("sempre cai");
    expect(oddsLabel(0)).toBe("sempre cai");
    expect(oddsLabel(72)).toBe("≈ 1 em 72");
  });
});

describe("slotGroup", () => {
  it("nomeia o slot pelo papel, não pelo número", () => {
    expect(slotGroup(2).label).toBe("Equipamento comum");
    expect(slotGroup(10).label).toBe("Ouro e poção");
    expect(slotGroup(56).label).toBe("Sempre cai");
    expect(slotGroup(63).label).toBe("Raros progressivos");
  });
});
