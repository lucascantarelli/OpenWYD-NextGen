import Link from "next/link";
import { notFound } from "next/navigation";

import { SiteFooter } from "@/components/SiteFooter";
import { SiteHeader } from "@/components/SiteHeader";
import {
  formatarData,
  getNoticia,
  listNoticias,
  noticiaStaticParams,
  renderNoticia,
} from "@/server/noticias";

export function generateStaticParams() {
  return noticiaStaticParams();
}

export async function generateMetadata({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const noticia = getNoticia(slug);
  if (!noticia) return { title: "Notícia não encontrada · WYD2" };
  return { title: `${noticia.title} · WYD2`, description: noticia.summary };
}

export default async function NoticiaPage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const noticia = getNoticia(slug);
  if (!noticia) notFound();

  const html = renderNoticia(noticia);
  const outras = listNoticias()
    .filter((candidate) => candidate.slug !== noticia.slug)
    .slice(0, 3);

  return (
    <>
      <SiteHeader current="/noticias" />

      <main className="site site--narrow">
        <article className="news-article">
          <Link className="news-article__back" href="/noticias">
            ← Todas as notícias
          </Link>
          <span className="news-card__tag">{noticia.tag}</span>
          <h1>{noticia.title}</h1>
          <p className="news-article__meta">
            {formatarData(noticia.date)} · {noticia.author}
          </p>
          <div className="news-article__body" dangerouslySetInnerHTML={{ __html: html }} />
        </article>

        {outras.length > 0 && (
          <aside className="news-article__more">
            <h2>Continue lendo</h2>
            <ul>
              {outras.map((item) => (
                <li key={item.slug}>
                  <Link href={`/noticias/${item.slug}`}>{item.title}</Link>
                  <span>{formatarData(item.date)}</span>
                </li>
              ))}
            </ul>
          </aside>
        )}
      </main>

      <SiteFooter />
    </>
  );
}
