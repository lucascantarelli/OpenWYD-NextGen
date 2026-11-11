import { describe, expect, it } from "vitest";

import { splitPlayersMobs } from "./monitoring";

describe("players × mobs (estimativa)", () => {
  it("separa jogadores do banco e mobs por diferença", () => {
    expect(splitPlayersMobs(1281, 1)).toEqual({ players: 1, mobs: 1280 });
  });

  it("nunca retorna mob negativo", () => {
    expect(splitPlayersMobs(3, 10)).toEqual({ players: 10, mobs: 0 });
  });

  it("null sem as duas fontes", () => {
    expect(splitPlayersMobs(null, 1).mobs).toBeNull();
    expect(splitPlayersMobs(10, null)).toEqual({ players: null, mobs: null });
  });
});
