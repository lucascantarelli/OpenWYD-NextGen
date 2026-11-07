import { describe, expect, it } from "vitest";

import {
  barHeight,
  describeRect,
  parseTransformQuery,
  rectError,
  transformError,
  transformKey,
  transformQuery,
  worldToCell,
  type TransformParams,
} from "./attributemap";

describe("attribute map", () => {
  it("converte coordenada de mundo em célula com divisão inteira", () => {
    expect(worldToCell(0)).toBe(0);
    expect(worldToCell(3)).toBe(0);
    expect(worldToCell(4)).toBe(1);
    expect(worldToCell(100)).toBe(25);
    expect(worldToCell(4095)).toBe(1023);
  });

  it("descreve o retângulo nas duas unidades", () => {
    expect(describeRect({ min_x: 100, min_y: 100, max_x: 199, max_y: 199 })).toBe(
      "mundo 100,100–199,199 → células 25,25–49,49",
    );
    expect(describeRect(null)).toContain("mapa inteiro");
  });

  it("recusa retângulo invertido ou fora do mundo", () => {
    expect(rectError({ min_x: 0, min_y: 0, max_x: 4095, max_y: 4095 })).toBeNull();
    expect(rectError({ min_x: 10, min_y: 0, max_x: 9, max_y: 4095 })).toMatch(/x mínimo/);
    expect(rectError({ min_x: 0, min_y: 0, max_x: 4096, max_y: 4095 })).toMatch(/x máximo/);
    expect(rectError({ min_x: -1, min_y: 0, max_x: 10, max_y: 10 })).toMatch(/x mínimo/);
  });

  it("operando zero só vale para definir valor", () => {
    const base: TransformParams = { operation: 3, operand: 0, rect: null, filter: null };
    expect(transformError(base)).toMatch(/operando 0/);
    expect(transformError({ ...base, operation: 2 })).toBeNull();
    expect(transformError({ ...base, operation: 1 })).toBeNull();
    expect(transformError({ ...base, operand: 256 })).toMatch(/operando/);
  });

  it("serializa e relê os parâmetros sem perder retângulo nem filtro", () => {
    const params: TransformParams = {
      operation: 3,
      operand: 64,
      rect: { min_x: 4, min_y: 8, max_x: 12, max_y: 16 },
      filter: { enabled: true, exact_value: 3, mask: 0, match_value: 0 },
    };
    const parsed = parseTransformQuery(new URLSearchParams(transformQuery(params)));
    expect(parsed).toEqual(params);
    expect(transformQuery({ ...params, rect: null, filter: null })).not.toContain("min_x");
    expect(transformKey(parsed)).toBe(transformKey(params));
  });

  it("barra vazia não ganha altura e a maior ocupa 100%", () => {
    expect(barHeight(0, 100)).toBe(0);
    expect(barHeight(100, 100)).toBe(100);
    expect(barHeight(1, 1000)).toBe(2);
  });
});
