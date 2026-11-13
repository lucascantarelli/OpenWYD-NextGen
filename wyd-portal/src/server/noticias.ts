/**
 * Notícias publicadas no portal.
 *
 * A fonte é `docs/noticias/*.md`; o JSON versionado sai de `npm run noticias`
 * (`scripts/build-noticias.mjs`) pelo mesmo motivo dos guias: a imagem do portal
 * copia só `wyd-portal/`, então `docs/` não existe no container.
 */
import { marked } from "marked";

import raw from "@/content/noticias.json";

export interface Noticia {
  slug: string;
  title: string;
  /** `AAAA-MM-DD`, como escrito no frontmatter. */
  date: string;
  author: string;
  tag: string;
  summary: string;
  body: string;
}

const NOTICIAS = raw as Noticia[];

if (!Array.isArray(NOTICIAS) || NOTICIAS.length === 0) {
  throw new Error("noticias.json vazio: rode `npm run noticias` na pasta portal");
}

/** Todas as notícias, da mais recente para a mais antiga. */
export function listNoticias(): readonly Noticia[] {
  return NOTICIAS;
}

export function getNoticia(slug: string): Noticia | undefined {
  return NOTICIAS.find((noticia) => noticia.slug === slug);
}

export function noticiaStaticParams(): { slug: string }[] {
  return NOTICIAS.map((noticia) => ({ slug: noticia.slug }));
}

/** Corpo markdown → HTML. Fica fora da página para o teste poder verificar. */
export function renderNoticia(noticia: Noticia): string {
  return marked.parse(noticia.body, { async: false }) as string;
}

/** `2026-11-11` → `11 de novembro de 2026`. */
export function formatarData(iso: string): string {
  const date = new Date(`${iso}T12:00:00Z`);
  if (Number.isNaN(date.getTime())) return iso;
  return new Intl.DateTimeFormat("pt-BR", {
    day: "2-digit",
    month: "long",
    year: "numeric",
    timeZone: "UTC",
  }).format(date);
}

/** A notícia mais recente — usada como destaque na home. */
export function destaque(): Noticia {
  return NOTICIAS[0]!;
}
