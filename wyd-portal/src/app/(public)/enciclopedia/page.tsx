import Link from "next/link";

import { ENCYCLOPEDIA_COUNT, itemKinds, searchItems } from "@/server/enciclopedia";

import "../guias/guias.css";

export const metadata = {
  title: "Enciclopédia de itens — WYD",
  description: "Todos os itens do catálogo: tipo, grau, nível, preço e efeitos base.",
};

interface Props {
  searchParams: Promise<{ q?: string; kind?: string }>;
}

const KIND_LABEL: Record<string, string> = {
  weapon: "arma",
  shield: "escudo",
  armor: "armadura",
  helmet: "elmo",
  boots: "botas",
  gloves: "luvas",
  cloak: "capa",
  accessory: "acessório",
  consumable: "consumível",
  material: "material",
  quest: "quest",
  cash: "cash",
  ammo: "munição",
  container: "container",
  cosmetic: "cosmético",
};

/**
 * Lista pública. A busca é feita no servidor com a mesma dobra do Postgres
 * (`fold`), então "pocao", "Poção" e "POCAO" devolvem o mesmo conjunto.
 */
export default async function EnciclopediaPage({ searchParams }: Props) {
  const { q = "", kind = "" } = await searchParams;
  const { items, total, folded } = searchItems({ q, kind });
  const kinds = itemKinds();

  return (
    <main className="guias">
      <Link className="guias__back" href="/guias">
        ← Guias
      </Link>
      <p className="guias__eyebrow">Referência</p>
      <h1>Enciclopédia de itens</h1>
      <p>
        {ENCYCLOPEDIA_COUNT.toLocaleString("pt-BR")} itens do catálogo. O grau e os efeitos vêm do
        cadastro do servidor; o que ainda não tem classificação aparece assim, sem rótulo
        inventado.
      </p>

      <form className="guias__search" method="get" role="search">
        <input
          type="search"
          name="q"
          defaultValue={q}
          placeholder="Nome do item (sem acento também funciona)"
          aria-label="Buscar item por nome"
        />
        <select name="kind" defaultValue={kind} aria-label="Filtrar por tipo">
          <option value="">Todos os tipos</option>
          {kinds.map((entry) => (
            <option key={entry.kind} value={entry.kind}>
              {KIND_LABEL[entry.kind] ?? entry.kind} ({entry.count})
            </option>
          ))}
        </select>
        <button type="submit">Buscar</button>
      </form>

      <p className="guias__hint">
        {total.toLocaleString("pt-BR")} resultado(s)
        {folded ? ` para “${folded}”` : ""}
        {total > items.length ? ` — mostrando os ${items.length} primeiros` : ""}.
      </p>

      <ul className="guias__grid">
        {items.map((item) => (
          <li key={item.index}>
            <Link className="guias__card" href={`/enciclopedia/${item.index}`}>
              {item.name}
              <span>
                #{item.index}
                {item.kind ? ` · ${KIND_LABEL[item.kind] ?? item.kind}` : ""}
                {item.reqLevel ? ` · nível ${item.reqLevel}` : ""}
              </span>
            </Link>
          </li>
        ))}
      </ul>
    </main>
  );
}
