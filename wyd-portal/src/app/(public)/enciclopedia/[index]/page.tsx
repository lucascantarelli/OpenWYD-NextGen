import Link from "next/link";
import { notFound } from "next/navigation";

import { gradeLabel, gradeTone } from "@/lib/item";
import { Badge } from "@/components/Badge";
import { getItem } from "@/server/enciclopedia";

import "../../guias/guias.css";

interface Props {
  params: Promise<{ index: string }>;
}

export async function generateMetadata({ params }: Props) {
  const { index } = await params;
  const item = getItem(Number(index));
  return { title: item ? `${item.name} — WYD` : "Item não encontrado — WYD" };
}

/**
 * Página do item. Renderizada sob demanda (3.220 páginas estáticas não pagam o
 * custo de build por um catálogo que muda por migração, não por request).
 */
export default async function ItemPage({ params }: Props) {
  const { index } = await params;
  const itemIndex = Number(index);
  if (!Number.isInteger(itemIndex)) notFound();

  const item = getItem(itemIndex);
  if (!item) notFound();

  const rows: { label: string; value: string; mono?: boolean }[] = [
    { label: "Índice", value: String(item.index), mono: true },
    { label: "Tipo", value: item.kind ?? "não classificado" },
    { label: "Sistema", value: item.system ?? "não classificado" },
    { label: "Grau", value: item.grade > 0 ? gradeLabel(item.grade) : "sem grau" },
    { label: "Nível exigido", value: item.reqLevel ? String(item.reqLevel) : "—" },
    { label: "Preço base", value: item.price ? item.price.toLocaleString("pt-BR") : "—" },
    { label: "Negociável", value: item.tradeable ? "sim" : "não" },
    { label: "Só evento", value: item.eventOnly ? "sim" : "não" },
  ];

  return (
    <main className="guias">
      <Link className="guias__back" href="/enciclopedia">
        ← Enciclopédia
      </Link>
      <p className="guias__eyebrow">Item #{item.index}</p>
      <h1>{item.name}</h1>
      <p>
        {item.grade > 0 ? (
          <Badge tone={gradeTone(item.grade)}>{gradeLabel(item.grade)}</Badge>
        ) : (
          <span className="guias__hint">sem grau no catálogo</span>
        )}
      </p>

      <table>
        <tbody>
          {rows.map((row) => (
            <tr key={row.label}>
              <th style={{ width: "32%" }}>{row.label}</th>
              <td className={row.mono ? "ui-mono" : undefined}>{row.value}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2>Efeitos base</h2>
      {item.effects.length > 0 ? (
        <ul>
          {item.effects.map((effect) => (
            <li key={`${effect.code}-${effect.value}`}>
              <code>EF #{effect.code}</code> → {effect.value}
            </li>
          ))}
        </ul>
      ) : (
        <p className="guias__hint">
          Sem par <code>EF_*</code> no catálogo: item sem adicional base.
        </p>
      )}

      <h2>Onde aparece</h2>
      <p>
        O catálogo guarda <strong>o item</strong>, não a origem dele. Drop por monstro está em{" "}
        <Link href="/guias/manual-sistemas-drops">Drops</Link>; recompensa por nível em{" "}
        <Link href="/guias/manual-progressao">Progressão</Link>; venda por NPC em{" "}
        <Link href="/guias/manual-itens">Itens</Link>.
      </p>
      <p className="guias__hint">
        A taxa de drop por item não existe em fonte pública nem no cliente: vem da configuração do
        servidor e ainda não foi importada. Preferimos dizer isso a inventar número.
      </p>
    </main>
  );
}
