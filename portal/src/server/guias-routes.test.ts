import { existsSync, readFileSync } from "node:fs";
import path from "node:path";

import { describe, expect, it } from "vitest";

import { getGuia, guiaStaticParams, listGuias, renderGuia } from "./guias";

/**
 * A ponte guias → rota não pode depender de inspeção manual.
 *
 * O teste anterior (`guias.test.ts`) garante que o JSON está íntegro; este
 * garante que **cada página do índice tem rota estática e corpo renderizável**.
 * Sem isso, uma página nova entrava no índice e virava 404 (ou um cartão sem
 * HTML) sem nenhum teste vermelho.
 */
const APP = path.resolve(process.cwd(), "src", "app", "(public)", "guias");
const ROUTE_FILE = path.join(APP, "[slug]", "page.tsx");
const INDEX_FILE = path.join(APP, "page.tsx");

describe("rotas públicas de guias", () => {
  it("todo guia do índice tem parâmetro de rota estática", () => {
    const doIndice = listGuias()
      .map((guia) => guia.slug)
      .sort();
    const dasRotas = guiaStaticParams()
      .map((param) => param.slug)
      .sort();

    expect(dasRotas).toEqual(doIndice);
  });

  it("a rota [slug] existe e continua estática", () => {
    expect(existsSync(ROUTE_FILE), ROUTE_FILE).toBe(true);

    const fonte = readFileSync(ROUTE_FILE, "utf8");
    expect(fonte).toContain("generateStaticParams");
    // Se alguém trocar a lista compartilhada por um filtro próprio, o teste de
    // cima deixa de valer — então a fonte precisa continuar usando a função.
    expect(fonte).toContain("guiaStaticParams");
    expect(fonte).toContain("notFound");
  });

  it("cada guia renderiza corpo, não só título", () => {
    for (const guia of listGuias()) {
      const { html } = renderGuia(guia);

      expect(html.length, `${guia.slug} renderizou ${html.length} bytes`).toBeGreaterThan(
        guia.title.length + 200,
      );
      expect(html, guia.slug).toMatch(/<(p|h2|table|ul)\b/);
    }
  });

  it("as 12 páginas de sistema renderizam tabela, handler e subtítulo", () => {
    const sistemas = listGuias().filter(
      (guia) => guia.section === "sistemas" && guia.slug !== "sistemas",
    );

    expect(sistemas).toHaveLength(12);
    for (const guia of sistemas) {
      const { html } = renderGuia(guia);

      expect(html, guia.slug).toContain("<table");
      expect(html, guia.slug).toContain("<h2");
      expect(html, guia.slug).toMatch(/\.go/);
    }
  });

  it("a navegação de seção alcança todos os sistemas a partir de um deles", () => {
    const combate = getGuia("combate");
    expect(combate, "guia combate ausente").toBeDefined();

    const { siblings } = renderGuia(combate!);
    const slugs = siblings.map((item) => item.slug);

    expect(siblings).toHaveLength(12);
    expect(slugs).toContain("personagem");
    expect(slugs).toContain("mapa");
  });

  it("slug fora do índice não renderiza (caminho de 404)", () => {
    expect(getGuia("nao-existe")).toBeUndefined();
    expect(guiaStaticParams().map((param) => param.slug)).not.toContain("nao-existe");
  });

  it("a página de índice agrupa por seção a partir do JSON", () => {
    expect(existsSync(INDEX_FILE), INDEX_FILE).toBe(true);
    expect(readFileSync(INDEX_FILE, "utf8")).toContain("groupGuias");
  });
});
