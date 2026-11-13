import { describe, expect, it } from "vitest";

import { GET } from "@/app/api/launcher/noticias/route";
import { listNoticias } from "@/server/noticias";

/**
 * Contrato público de notícias para o launcher: vitrine sem corpo markdown,
 * ordem igual à do site, URL pronta para abrir e cache curto no emissor.
 */
describe("GET /api/launcher/noticias", () => {
  it("devolve a vitrine na ordem do site, sem corpo", async () => {
    const res = await GET();
    expect(res.status).toBe(200);

    const corpo = (await res.json()) as {
      noticias: {
        slug: string;
        titulo: string;
        data: string;
        tag: string;
        resumo: string;
        url: string;
        body?: string;
      }[];
    };
    const doSite = listNoticias();

    expect(corpo.noticias).toHaveLength(doSite.length);
    expect(corpo.noticias.map((n) => n.slug)).toEqual(doSite.map((n) => n.slug));
    expect(corpo.noticias[0]).toEqual({
      slug: doSite[0]!.slug,
      titulo: doSite[0]!.title,
      data: doSite[0]!.date,
      tag: doSite[0]!.tag,
      resumo: doSite[0]!.summary,
      url: `/noticias/${doSite[0]!.slug}`,
    });
    expect(corpo.noticias.every((n) => n.body === undefined)).toBe(true);
  });

  it("manda cache público curto com revalidação em segundo plano", async () => {
    const cache = (await GET()).headers.get("cache-control") ?? "";
    expect(cache).toContain("public");
    expect(cache).toContain("max-age=60");
    expect(cache).toContain("stale-while-revalidate=300");
  });
});
