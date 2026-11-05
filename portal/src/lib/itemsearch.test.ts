import { describe, expect, it } from "vitest";

import { normalizeSearch } from "./itemsearch";

/** "Poção" em NFD: `o` + cedilha combinante, `a` + til combinante. */
const NFD = "Poc\u0327a\u0303o";
const NFC = "Poção";

describe("normalizeSearch", () => {
  it("aproxima NFD e NFC do mesmo termo", () => {
    expect(normalizeSearch(NFD)).toBe(normalizeSearch(NFC));
  });

  it("encontra no nome do catálogo a consulta digitada em NFD", () => {
    expect(normalizeSearch("Poção_Combatente")).toContain(normalizeSearch(NFD));
  });

  it("não confunde termo sem acento com termo acentuado", () => {
    expect(normalizeSearch("Pocao")).not.toBe(normalizeSearch(NFC));
  });
});
