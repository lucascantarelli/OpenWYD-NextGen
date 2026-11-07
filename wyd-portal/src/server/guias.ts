/**
 * Conteúdo publicado em `/guias`, gerado a partir de `docs/` por
 * `wyd-portal/scripts/build-guias.mjs` (`npm run guias`).
 *
 * O JSON é versionado porque a imagem do portal copia só `wyd-portal/` — `docs/`
 * não existe dentro do container. O CI regenera e reprova se divergir.
 */
import { marked } from "marked";

import raw from "@/content/guias.json";

export interface Guia {
  slug: string;
  section: string;
  sectionLabel: string;
  sectionOrder: number;
  order: number;
  title: string;
  body: string;
}

export interface GuiaSection {
  id: string;
  label: string;
  order: number;
  items: Guia[];
}

const GUIAS = raw as Guia[];

if (!Array.isArray(GUIAS) || GUIAS.length === 0) {
  throw new Error("guias.json vazio: rode `npm run guias` na pasta portal");
}

/** Todas as páginas publicadas, na ordem de seção e de arquivo. */
export function listGuias(): readonly Guia[] {
  return GUIAS;
}

export function getGuia(slug: string): Guia | undefined {
  return GUIAS.find((guia) => guia.slug === slug);
}

/**
 * Parâmetros das rotas estáticas de `/guias/[slug]`.
 *
 * A página usa **exatamente** esta lista em `generateStaticParams`; o teste de
 * rota compara com `listGuias()` para que uma página do índice não fique sem
 * HTML pré-renderizado sem ninguém notar.
 */
export function guiaStaticParams(): { slug: string }[] {
  return GUIAS.map((guia) => ({ slug: guia.slug }));
}

export interface GuiaRender {
  /** Corpo em HTML (markdown de `docs/` renderizado). */
  html: string;
  /** Guias da mesma seção, para a navegação lateral do portal. */
  siblings: Guia[];
}

/**
 * Renderiza um guia. Fica aqui, e não na página, para que o teste possa provar
 * que o corpo vira HTML de verdade — não só que o título existe.
 */
export function renderGuia(guia: Guia): GuiaRender {
  const html = marked.parse(guia.body, { async: false }) as string;
  const siblings = GUIAS.filter(
    (candidate) => candidate.section === guia.section && candidate.slug !== guia.slug,
  );

  return { html, siblings };
}

/** Agrupa para a página de índice, preservando a ordem declarada no gerador. */
export function groupGuias(): readonly GuiaSection[] {
  const sections: GuiaSection[] = [];

  for (const guia of GUIAS) {
    let section = sections.find((candidate) => candidate.id === guia.section);
    if (!section) {
      section = {
        id: guia.section,
        label: guia.sectionLabel,
        order: guia.sectionOrder,
        items: [],
      };
      sections.push(section);
    }
    section.items.push(guia);
  }

  return sections.sort((a, b) => a.order - b.order);
}
