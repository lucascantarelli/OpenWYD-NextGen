/**
 * Conteúdo publicado em `/guias`, gerado a partir de `docs/` por
 * `portal/scripts/build-guias.mjs` (`npm run guias`).
 *
 * O JSON é versionado porque a imagem do portal copia só `portal/` — `docs/`
 * não existe dentro do container. O CI regenera e reprova se divergir.
 */
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
