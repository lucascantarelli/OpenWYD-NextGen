import Link from "next/link";
import { notFound } from "next/navigation";
import { marked } from "marked";

import { getGuia, listGuias } from "@/server/guias";

import "../guias.css";

interface Params {
  params: Promise<{ slug: string }>;
}

/** Todas as páginas são estáticas: o conteúdo só muda quando `npm run guias` roda. */
export function generateStaticParams() {
  return listGuias().map((guia) => ({ slug: guia.slug }));
}

export async function generateMetadata({ params }: Params) {
  const { slug } = await params;
  const guia = getGuia(slug);

  return {
    title: guia ? `${guia.title} — OpenWYD` : "Guia não encontrado — OpenWYD",
  };
}

export default async function GuiaPage({ params }: Params) {
  const { slug } = await params;
  const guia = getGuia(slug);
  if (!guia) notFound();

  // Markdown vem do repositório (docs/), não de entrada de usuário: o HTML é
  // gerado aqui, no servidor, e servido como conteúdo estático.
  const html = marked.parse(guia.body, { async: false }) as string;

  // Navegação dentro da seção: com 12 páginas de sistema, chegar em "combate"
  // e não ver o resto da seção deixaria o guia sem saída.
  const siblings = listGuias().filter(
    (candidate) => candidate.section === guia.section && candidate.slug !== guia.slug,
  );

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
