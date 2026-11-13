import Link from "next/link";

import { SiteFooter } from "@/components/SiteFooter";
import { SiteHeader } from "@/components/SiteHeader";
import { formatarData, listNoticias } from "@/server/noticias";

export const metadata = {
  title: "Notícias · WYD2",
  description: "O que muda no servidor, entrega por entrega, em linguagem de jogador.",
};

export default function NoticiasPage() {
  const noticias = listNoticias();

  return (
    <>
      <SiteHeader current="/noticias" />

      <main className="site site--narrow">
        <header className="site-page-head">
          <p className="site-section__eyebrow">Notícias</p>
          <h1>O servidor, em texto</h1>
          <p>
            Cada entrega do time vira uma notícia curta: o que mudou, por que mudou e o que o
            jogador sente no jogo. Sem changelog técnico e sem promessa.
          </p>
        </header>

        <div className="news-list">
          {noticias.map((noticia) => (
            <article className="news-row" key={noticia.slug}>
              <div className="news-row__date">
                <time dateTime={noticia.date}>{formatarData(noticia.date)}</time>
                <span className="news-card__tag">{noticia.tag}</span>
              </div>
              <div className="news-row__body">
                <h2>
                  <Link href={`/noticias/${noticia.slug}`}>{noticia.title}</Link>
                </h2>
                <p>{noticia.summary}</p>
              </div>
              <Link className="ui-btn ui-btn--ghost ui-btn--sm" href={`/noticias/${noticia.slug}`}>
                Ler
              </Link>
            </article>
          ))}
        </div>
      </main>

      <SiteFooter />
    </>
  );
}
