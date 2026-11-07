import Link from "next/link";

/**
 * Navegação da seção de drops: relatório (issue 26) e taxa por slot (issue 34).
 *
 * Dois `Link` com `aria-current`, não um tab client: são rotas de verdade, com
 * URL própria e o mesmo estado do lado do servidor.
 */
export function DropsNav({ active }: { active: "relatorio" | "taxas" | "carry" }) {
  const items = [
    { id: "relatorio", href: "/drops", label: "Relatório item ↔ mob" },
    { id: "taxas", href: "/drops/editar", label: "Taxa por slot" },
    { id: "carry", href: "/drops/carry", label: "Itens que caem" },
  ] as const;

  return (
    <nav className="ui-filters" aria-label="Seções de drops">
      {items.map((item) => (
        <Link
          key={item.id}
          href={item.href}
          className="ui-chip"
          aria-current={item.id === active ? "page" : undefined}
          style={
            item.id === active
              ? { borderColor: "var(--accent)", color: "var(--accent)" }
              : undefined
          }
        >
          {item.label}
        </Link>
      ))}
    </nav>
  );
}
