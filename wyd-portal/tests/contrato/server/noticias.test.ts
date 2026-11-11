import { readFileSync, readdirSync } from "node:fs";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

import { formatarData, getNoticia, listNoticias, renderNoticia } from "@/server/noticias";

/**
 * O JSON publicado é gerado de `docs/noticias/*.md` (`npm run noticias`). Sem este
 * teste, esquecer de rodar o gerador deixa uma notícia fora do site — ou um texto
 * antigo publicado depois de o markdown mudar.
 */
// `noticias.ts` está em `wyd-portal/src/server/`; o JSON nasce de `docs/noticias/`
// na raiz do repositório.
const raiz = fileURLToPath(new URL("../../..", import.meta.url));
const dir = `${raiz}/docs/noticias`;

interface DoDoc {
  slug: string;
  title: string;
  date: string;
  author: string;
  tag: string;
  summary: string;
}

function doDisco(): DoDoc[] {
  return readdirSync(dir)
    .filter((nome) => nome.endsWith(".md"))
    .map((nome) => {
      const texto = readFileSync(`${dir}/${nome}`, "utf8").replace(/\r\n/g, "\n");
      const match = texto.match(/^---\n([\s\S]*?)\n---\n/);
      if (!match) throw new Error(`${nome} sem frontmatter`);
      const meta = new Map<string, string>();
      for (const linha of match[1]!.split("\n")) {
        if (!linha.trim()) continue;
        const corte = linha.indexOf(":");
        meta.set(linha.slice(0, corte).trim(), linha.slice(corte + 1).trim());
      }
      return {
        slug: nome.replace(/\.md$/, ""),
        title: meta.get("title") ?? "",
        date: meta.get("date") ?? "",
        author: meta.get("author") ?? "",
        tag: meta.get("tag") ?? "",
        summary: meta.get("summary") ?? "",
      };
    });
}

describe("notícias publicadas", () => {
  it("o JSON está em dia com docs/noticias", () => {
    const disco = doDisco();
    expect(disco.length).toBeGreaterThan(0);

    const publicadas = listNoticias();
    expect(publicadas.map((noticia) => noticia.slug).sort()).toEqual(
      disco.map((noticia) => noticia.slug).sort(),
    );

    for (const doDoc of disco) {
      const noticia = getNoticia(doDoc.slug);
      expect(noticia, doDoc.slug).toBeDefined();
      expect(noticia?.title).toBe(doDoc.title);
      expect(noticia?.date).toBe(doDoc.date);
      expect(noticia?.summary).toBe(doDoc.summary);
    }
  });

  it("lista da mais recente para a mais antiga", () => {
    const datas = listNoticias().map((noticia) => noticia.date);
    expect([...datas].sort((a, b) => b.localeCompare(a))).toEqual(datas);
  });

  it("renderiza o corpo em HTML", () => {
    const noticia = listNoticias()[0]!;
    const html = renderNoticia(noticia);
    expect(html).toContain("<h2");
    expect(html).toContain("</p>");
  });

  it("formata a data em português", () => {
    expect(formatarData("2026-11-11")).toBe("11 de novembro de 2026");
  });
});
