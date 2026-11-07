import { describe, expect, it } from "vitest";

import { averageCents, centsToDecimal, fmtBRL, fmtCount, fmtPercent } from "./money";

describe("fmtBRL", () => {
  it("formata centavos inteiros sem float", () => {
    expect(fmtBRL(0)).toBe("R$ 0,00");
    expect(fmtBRL(5)).toBe("R$ 0,05");
    expect(fmtBRL(5000)).toBe("R$ 50,00");
    expect(fmtBRL(123456789)).toBe("R$ 1.234.567,89");
  });

  it("mantém o sinal antes do símbolo", () => {
    expect(fmtBRL(-2590)).toBe("-R$ 25,90");
  });

  it("trata valor inválido como zero", () => {
    expect(fmtBRL(Number.NaN)).toBe("R$ 0,00");
  });
});

describe("centsToDecimal", () => {
  it("devolve vírgula decimal sem símbolo", () => {
    expect(centsToDecimal(1234)).toBe("12,34");
    expect(centsToDecimal(50)).toBe("0,50");
    expect(centsToDecimal(-1234)).toBe("-12,34");
  });
});

describe("fmtCount", () => {
  it("separa milhar", () => {
    expect(fmtCount(1234567)).toBe("1.234.567");
  });
});

describe("averageCents", () => {
  it("arredonda o ticket e não divide por zero", () => {
    expect(averageCents(5000, 3)).toBe(1667);
    expect(averageCents(5000, 0)).toBe(0);
  });
});

describe("fmtPercent", () => {
  it("mostra uma casa decimal em pt-BR", () => {
    expect(fmtPercent(1, 4)).toBe("25%");
    expect(fmtPercent(1, 3)).toBe("33,3%");
    expect(fmtPercent(0, 0)).toBe("0%");
  });
});
