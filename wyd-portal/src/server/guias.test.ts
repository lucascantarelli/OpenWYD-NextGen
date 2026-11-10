import { describe, expect, it } from "vitest";

import { getGuia, groupGuias, listGuias } from "./guias";

/**
 * O JSON de guias é gerado a partir de `docs/` (wyd-portal/scripts/build-guias.mjs).
 * Estes testes cobrem o que quebra silenciosamente: página sem título, slug
 * repetido, link de markdown que o portal não sabe servir e seção vazia.
 *
 * A sincronia com `docs/` é cobrada pelo CI (`npm run guias` + `git diff`), não
 * aqui — teste que relê o disco esconderia a divergência até o build da imagem.
 */
describe("conteúdo de guias", () => {
  it("publica a seção do manual", () => {
    const ids = groupGuias().map((section) => section.id);

    expect(ids).toEqual(["manual"]);
  });

  it("toda página tem slug, título e corpo", () => {
    for (const guia of listGuias()) {
      expect(guia.slug, guia.slug).toMatch(/^[a-z0-9-]+$/);
      expect(guia.title.trim().length, guia.slug).toBeGreaterThan(2);
      expect(guia.body.trim().length, guia.slug).toBeGreaterThan(80);
      expect(guia.sectionLabel.trim().length, guia.slug).toBeGreaterThan(2);
    }
  });

  it("não repete slug", () => {
    const slugs = listGuias().map((guia) => guia.slug);

    expect(new Set(slugs).size).toBe(slugs.length);
  });

  it("não deixa link .md nem admonition crua no corpo", () => {
    for (const guia of listGuias()) {
      expect(guia.body, guia.slug).not.toMatch(/\]\([^)]*\.md(#|\))/);
      expect(guia.body, guia.slug).not.toMatch(/^!!!\s/m);
    }
  });

  it("todo link interno de guia aponta para página existente", () => {
    for (const guia of listGuias()) {
      for (const match of guia.body.matchAll(/\]\((\/guias\/[a-z0-9-]+)/g)) {
        const target = match[1].replace("/guias/", "");
        expect(getGuia(target), `${guia.slug} → ${target}`).toBeDefined();
      }
    }
  });

  it("o índice e a história existem com o título esperado", () => {
    expect(getGuia("manual")?.title).toMatch(/manual/i);
    expect(getGuia("manual-lore")?.title).toMatch(/hist[oó]ria/i);
  });
});

/**
 * Estrutura do manual: uma página por domínio, na ordem de leitura declarada em
 * `scripts/build-guias.mjs`.
 */
describe("manual de WYD", () => {
  const slugs = listGuias().map((guia) => guia.slug);

  it("publica os 23 documentos na ordem de leitura", () => {
    expect(slugs).toEqual([
      "manual",
      "manual-lore",
      "manual-classes",
      "manual-skills",
      "manual-progressao",
      "manual-itens",
      "manual-economia",
      "manual-sistemas",
      "manual-sistemas-refino",
      "manual-sistemas-drops",
      "manual-sistemas-guerras",
      "manual-sistemas-mundo",
      "manual-sistemas-quests",
      "manual-sistemas-eventos",
      "manual-fontes",
      "manual-referencia-externa-2026-11",
      "manual-pesquisa-internet-2026-11-10",
      "manual-canais-e-cidadania",
      "manual-decisoes-servidor",
      "manual-multiconta",
      "manual-senha-numerica",
      "manual-sistemas-battle-royale",
      "manual-sistemas-refino-auditoria",
    ]);
  });

  it("cada página traz tabela e subtítulo, não só prosa", () => {
    for (const guia of listGuias()) {
      expect(guia.body, `${guia.slug} sem tabela`).toContain("|---");
      expect(guia.body, `${guia.slug} sem H2`).toMatch(/^##\s/m);
    }
  });

  it("não cita arquivo Go: o projeto é 100% Rust desde 2026-11", () => {
    for (const guia of listGuias()) {
      expect(guia.body, guia.slug).not.toMatch(/`[^`]+\.go`/);
    }
  });
});
