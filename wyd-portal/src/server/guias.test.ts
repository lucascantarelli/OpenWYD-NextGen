import { existsSync } from "node:fs";
import path from "node:path";

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
  it("tem as quatro seções publicadas", () => {
    const ids = groupGuias().map((section) => section.id);

    expect(ids).toEqual(["jogador", "admin", "sistemas", "lore"]);
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

  it("o guia do jogador e o do administrador existem", () => {
    expect(getGuia("comecando")?.title).toMatch(/jogador/i);
    expect(getGuia("portal")?.title).toMatch(/pap[eé]is/i);
  });
});

/**
 * Estrutura da seção de sistemas: uma página por sistema catalogado, na ordem
 * de leitura declarada em `scripts/build-guias.mjs`.
 */
describe("sistemas de jogo", () => {
  const slugs = listGuias()
    .filter((guia) => guia.section === "sistemas")
    .map((guia) => guia.slug);

  it("publica o índice e uma página por sistema, na ordem de leitura", () => {
    expect(slugs).toEqual([
      "sistemas",
      "personagem",
      "combate",
      "skills",
      "drops",
      "refino",
      "quests",
      "comercio",
      "guildas",
      "castelos",
      "eventos",
      "canais",
      "mapa",
    ]);
  });

  it("cada página, inclusive o índice, cita módulo Rust existente", () => {
    // O vitest roda com cwd em wyd-portal/; o repositório é o diretório pai.
    // Fonte canônica Rust (arquivo Go removido em 2026-11): payloads em
    // crates/wyd-protocol/src/payload/ e despacho em crates/wyd-net/src/dispatch/.
    const payloadDir = path.resolve(process.cwd(), "..", "crates", "wyd-protocol", "src", "payload");
    const paginas = listGuias().filter((guia) => guia.section === "sistemas");

    for (const guia of paginas) {
      const citados = [...guia.body.matchAll(/`[^`]*?([a-z0-9_]+\.rs)`/g)].map((match) => match[1]);

      expect(citados.length, `${guia.slug} não cita nenhum módulo Rust`).toBeGreaterThan(0);
      for (const arquivo of citados) {
        expect(
          existsSync(path.join(payloadDir, arquivo)),
          `${guia.slug} cita payload/${arquivo}, que não existe`,
        ).toBe(true);
      }
    }
  });
});
