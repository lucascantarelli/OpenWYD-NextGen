import Link from "next/link";
import { notFound } from "next/navigation";

import { getGuia, guiaStaticParams, renderGuia } from "@/server/guias";

import "../guias.css";

interface Params {
  params: Promise<{ slug: string }>;
}

/** Todas as páginas são estáticas: o conteúdo só muda quando `npm run guias` roda. */
export function generateStaticParams() {
  return guiaStaticParams();
}

export async function generateMetadata({ params }: Params) {
  const { slug } = await params;
  const guia = getGuia(slug);

  return {
    title: guia ? `${guia.title} — WYD` : "Guia não encontrado — WYD",
  };
}

export default async function GuiaPage({ params }: Params) {
  const { slug } = await params;
  const guia = getGuia(slug);
  if (!guia) notFound();

  // Markdown vem do repositório (docs/), não de entrada de usuário: o HTML é
  // gerado no servidor e servido como conteúdo estático. A navegação dentro da
  // seção evita que "combate" fique sem saída para as outras 11 páginas.
  const { html, siblings } = renderGuia(guia);

  return (
    <main className="guias">
      <Link className="guias__back" href="/guias">
        ← Todos os guias
      </Link>
      <p className="guias__eyebrow">{guia.sectionLabel}</p>
      <h1>{guia.title}</h1>
      <article className="guias__doc" dangerouslySetInnerHTML={{ __html: html }} />

      {siblings.length > 0 && (
        <nav className="guias__siblings" aria-label="Outros guias desta seção">
          <h2>Nesta seção</h2>
          <ul>
            {siblings.map((item) => (
              <li key={item.slug}>
                <Link href={`/guias/${item.slug}`}>{item.title}</Link>
              </li>
            ))}
          </ul>
        </nav>
      )}
    </main>
  );
}
