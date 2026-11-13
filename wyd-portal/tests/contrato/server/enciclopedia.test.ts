import { describe, expect, it } from "vitest";

import { ENCYCLOPEDIA_COUNT, getItem, itemKinds, searchItems } from "@/server/enciclopedia";

/**
 * A enciclopédia é gerada (`tools/content/gen-enciclopedia.mjs`) e versionada.
 * O que pode quebrar sem ninguém notar: JSON truncado (parser parando no meio),
 * busca sensível a acento e detalhe de item inexistente.
 */
describe("enciclopédia de itens", () => {
  it("publica o catálogo inteiro", () => {
    expect(ENCYCLOPEDIA_COUNT).toBeGreaterThan(3000);
  });

  it("acha item por nome sem acento", () => {
    const { items } = searchItems({ q: "antidoto" });

    expect(items.some((item) => item.name === "Antídoto")).toBe(true);
  });

  it("acha o mesmo item com acento e em caixa alta", () => {
    const semAcento = searchItems({ q: "antidoto" }).total;
    const comAcento = searchItems({ q: "ANTÍDOTO" }).total;

    expect(comAcento).toBe(semAcento);
    expect(comAcento).toBeGreaterThan(0);
  });

  it("detalhe devolve efeitos e metadados do item", () => {
    const { items } = searchItems({ q: "antidoto" });
    const found = items.find((item) => item.name === "Antídoto");

    expect(found).toBeDefined();
    expect(getItem(found!.index)?.name).toBe("Antídoto");
    expect(Array.isArray(found!.effects)).toBe(true);
  });

  it("todo item tem índice e nome", () => {
    const { items } = searchItems({ limit: 50 });

    expect(items.length).toBe(50);
    for (const item of items) {
      expect(item.index).toBeGreaterThan(0);
      expect(item.name.trim().length).toBeGreaterThan(0);
    }
  });

  it("filtro por tipo soma o total do catálogo", () => {
    const kinds = itemKinds();
    const sum = kinds.reduce((count, entry) => count + entry.count, 0);

    expect(sum).toBe(ENCYCLOPEDIA_COUNT);
  });
});
