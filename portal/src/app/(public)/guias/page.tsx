import Link from "next/link";

import { groupGuias } from "@/server/guias";

import "./guias.css";

export const metadata = {
  title: "Guias — OpenWYD",
  description: "Guias do jogador, do administrador, sistemas de jogo e história do mundo.",
};

/**
 * Índice público dos guias. O conteúdo é o mesmo markdown que alimenta o site
 * de documentação (mkdocs): uma fonte, dois destinos.
 */
export default function GuiasPage() {
  const sections = groupGuias();

  return (
    <main className="guias">
      <p className="guias__eyebrow">Documentação</p>
      <h1>Guias</h1>
      <p>
        Escritos para quem joga e para quem administra. O mesmo texto é publicado no site de
        documentação; aqui você lê sem sair do portal.
      </p>

      {sections.map((section) => (
        <section key={section.id}>
          <h2>{section.label}</h2>
          <ul className="guias__grid">
            {section.items.map((guia) => (
              <li key={guia.slug}>
                <Link className="guias__card" href={`/guias/${guia.slug}`}>
                  {guia.title}
                  <span>Ler o guia</span>
                </Link>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </main>
  );
}
