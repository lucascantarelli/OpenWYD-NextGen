import { listNoticias } from "@/server/noticias";

export const dynamic = "force-dynamic";

/**
 * Notícias públicas para o launcher (LAUNCHER-01).
 *
 * Reusa o JSON que o site publica (`docs/noticias/*.md` → `npm run noticias`):
 * o launcher recebe só o recorte de vitrine (slug, título, data, tag, resumo e
 * URL), nunca o corpo markdown. Notícia é dado de jogo, não de conta: nenhuma
 * sessão é necessária.
 *
 * Cache de 60 s no emissor, com `stale-while-revalidate`: o launcher abre mesmo
 * com publicação recém-feita, e o portal não é lido a cada clique.
 */
export function GET() {
  const noticias = listNoticias().map((noticia) => ({
    slug: noticia.slug,
    titulo: noticia.title,
    data: noticia.date,
    tag: noticia.tag,
    resumo: noticia.summary,
    url: `/noticias/${encodeURIComponent(noticia.slug)}`,
  }));

  return Response.json(
    { noticias },
    {
      headers: {
        "cache-control": "public, max-age=60, stale-while-revalidate=300",
      },
    },
  );
}
