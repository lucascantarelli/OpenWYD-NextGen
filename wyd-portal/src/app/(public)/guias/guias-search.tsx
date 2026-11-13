"use client";

import Link from "next/link";
import { useMemo, useState } from "react";

import { foldMatch } from "@/lib/fold";

export interface SearchGuia {
  slug: string;
  title: string;
}

export interface SearchSection {
  id: string;
  label: string;
  items: SearchGuia[];
}

/**
 * Busca dos guias no navegador: a lista é pequena (dezenas de páginas) e já vem
 * no HTML, então filtrar no cliente evita uma ida ao servidor por tecla. A dobra
 * é a mesma do banco (`fold`), então o resultado não depende de acento.
 */
export function GuiasSearch({ sections }: { sections: SearchSection[] }) {
  const [query, setQuery] = useState("");

  const filtered = useMemo(() => {
    const term = query.trim();
    if (!term) return sections;
    return sections
      .map((section) => ({
        ...section,
        items: section.items.filter((item) => foldMatch(item.title, term)),
      }))
      .filter((section) => section.items.length > 0);
  }, [sections, query]);

  const total = filtered.reduce((count, section) => count + section.items.length, 0);

  return (
    <>
      <div className="guias__search">
        <input
          type="search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Buscar guia por título (sem acento também funciona)"
          aria-label="Buscar guia"
        />
        <Link className="guias__cta" href="/enciclopedia">
          Enciclopédia de itens
        </Link>
      </div>

      {query.trim() ? (
        <p className="guias__hint">
          {total} guia(s) para “{query.trim()}”.
        </p>
      ) : null}

      <nav className="guias__nav" aria-label="Seções do manual">
        {sections.map((section) => (
          <a key={section.id} href={`#${section.id}`}>
            {section.label}
          </a>
        ))}
      </nav>

      {filtered.length === 0 ? (
        <p className="guias__hint">Nenhum guia com esse título.</p>
      ) : (
        filtered.map((section) => (
          <section key={section.id} id={section.id}>
            <h2>{section.label}</h2>
            <ul className="guias__grid">
              {section.items.map((item) => (
                <li key={item.slug}>
                  <Link className="guias__card" href={`/guias/${item.slug}`}>
                    {item.title}
                    <span>Ler o guia</span>
                  </Link>
                </li>
              ))}
            </ul>
          </section>
        ))
      )}
    </>
  );
}
