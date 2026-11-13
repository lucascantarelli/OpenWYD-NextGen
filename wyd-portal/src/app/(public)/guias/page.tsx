import { groupGuias } from "@/server/guias";

import { GuiasSearch } from "./guias-search";

import "./guias.css";

export const metadata = {
  title: "Guias — WYD",
  description: "Guias do jogador, do administrador, sistemas de jogo e história do mundo.",
};

/**
 * Índice público dos guias. O conteúdo é o mesmo markdown que alimenta o site
 * de documentação (mkdocs): uma fonte, dois destinos.
 *
 * A navegação é por seção (âncora) e a busca é feita no cliente sobre títulos —
 * ver `guias-search.tsx`. O manual completo de domínio (`docs/wyd/`) entra como
 * uma seção própria, gerada por `npm run guias`.
 */
export default function GuiasPage() {
  const sections = groupGuias().map((section) => ({
    id: section.id,
    label: section.label,
    items: section.items.map((guia) => ({ slug: guia.slug, title: guia.title })),
  }));

  return (
    <main className="guias">
      <p className="guias__eyebrow">Documentação</p>
      <h1>Guias</h1>
      <p>
        Escritos para quem joga e para quem administra. O mesmo texto é publicado no site de
        documentação; aqui você lê sem sair do portal.
      </p>

      <GuiasSearch sections={sections} />
    </main>
  );
}
