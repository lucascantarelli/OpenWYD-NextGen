import { describe, expect, it } from "vitest";

import { findCatalogItem } from "./catalog";

/**
 * A ficha do item (`?item=`) filtra a foto já lida do gateway: sem rota nova,
 * sem busca inventada — índice inválido vira \"não encontrado\".
 */
describe("ficha do item no catálogo", () => {
  const items = [
    { item_index: 3321, name: "espada_teste", display_name: "Espada" },
    { item_index: 540, name: "pocao_hp", display_name: "" },
  ];

  it("acha por índice exato", () => {
    expect(findCatalogItem(items, 3321)?.name).toBe("espada_teste");
  });

  it("índice ausente ou inválido vira nulo", () => {
    expect(findCatalogItem(items, 9999)).toBeNull();
    expect(findCatalogItem(items, 0)).toBeNull();
    expect(findCatalogItem(items, Number.NaN)).toBeNull();
  });
});
