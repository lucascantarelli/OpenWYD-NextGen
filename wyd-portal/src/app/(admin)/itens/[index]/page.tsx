import Link from "next/link";
import { notFound } from "next/navigation";

import { Badge } from "@/components/Badge";
import { ItemIcon } from "@/components/ItemIcon";
import { ItemEffectEditor } from "@/components/item/effect-editor";
import { gradeLabel, gradeTone, slotsText } from "@/lib/item";
import { getCatalogItem } from "@/server/catalog";
import { getItemEffects } from "@/server/itemeffects";

export const dynamic = "force-dynamic";

/**
 * Detalhe do item por link direto (`/itens/1234`).
 *
 * Mostra também os campos "crus" (`slot_mask`, `mesh`, `texture`, `icon_key`)
 * para conferência contra o cliente: são eles que explicam por que dois itens
 * parecem iguais mas têm ícones diferentes.
 */
export default async function ItemPage({ params }: { params: Promise<{ index: string }> }) {
  const { index } = await params;
  const itemIndex = Number(index);
  if (!Number.isInteger(itemIndex) || itemIndex < 0) notFound();

  const item = await getCatalogItem(itemIndex);
  if (!item) notFound();

  // O overlay de efeitos é opcional: sem o serviço no web-api a página continua
  // mostrando a leitura do catálogo em vez de derrubar a rota inteira.
  const itemEffects = await getItemEffects(itemIndex).catch(() => null);

  const fields: { label: string; value: string; raw?: boolean }[] = [
    { label: "Índice", value: String(item.item_index), raw: true },
    { label: "Nome no catálogo", value: item.name, raw: true },
    { label: "Nome de exibição", value: item.display_name || "—" },
    { label: "Slots", value: slotsText(item) },
    { label: "slot_mask (nPos)", value: String(item.slot_mask), raw: true },
    { label: "Grau", value: gradeLabel(item.grade) },
    { label: "mesh (IndexMesh)", value: String(item.mesh), raw: true },
    { label: "texture (IndexTexture)", value: String(item.texture), raw: true },
    { label: "icon_key", value: item.icon_key || "— (fallback)", raw: true },
    {
      label: "icon_url",
      value: item.icon_url ? "publicada no pack" : "— (pack de ícones ausente)",
    },
  ];

  return (
    <>
      <p className="ui-field__hint" style={{ marginBottom: 8 }}>
        <Link href="/itens">← catálogo</Link>
      </p>

      <div className="ui-detail__head">
        <span style={{ transform: "scale(1.5)", transformOrigin: "center", display: "inline-block" }}>
          <ItemIcon item={item} iconPackVersion={item.icon_url ? "pack" : ""} />
        </span>
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            {item.display_name || item.name}
          </h1>
          <p className="ui-page-sub" style={{ margin: "4px 0 0" }}>
            <span className="ui-mono">#{item.item_index}</span> ·{" "}
            {item.grade > 0 ? (
              <Badge tone={gradeTone(item.grade)}>{gradeLabel(item.grade)}</Badge>
            ) : (
              "sem grau"
            )}
          </p>
        </div>
      </div>

      <div className="ui-detail__grid">
        {fields.map((field) => (
          <dl key={field.label} className="ui-kv">
            <dt>{field.label}</dt>
            <dd className={field.raw ? "ui-mono" : undefined}>{field.value}</dd>
          </dl>
        ))}
      </div>

      <div className="ui-card" style={{ marginTop: 16 }}>
        <h2 style={{ fontSize: 13, margin: "0 0 8px" }}>
          Efeitos base ({item.effects?.length ?? 0})
        </h2>
        {item.effects && item.effects.length > 0 ? (
          <ul style={{ margin: 0, paddingLeft: 18 }}>
            {item.effects.map((effect) => (
              <li key={`${effect.code}-${effect.value}`}>
                {effect.label} <span className="ui-mono" title={`código ${effect.code}`}>{effect.value}</span>
              </li>
            ))}
          </ul>
        ) : (
          <p className="ui-field__hint" style={{ margin: 0 }}>
            sem pares <code>EF_*</code> no catálogo — item sem add base
          </p>
        )}
        <p className="ui-field__hint">
          Os pares acima vêm do <code>ItemList.csv</code>; abaixo você edita o overlay do Postgres
          (issue 44). O CSV não é reescrito: <code>tmserver/content/</code> é conteúdo.
        </p>
        {itemEffects ? (
          <ItemEffectEditor
            itemIndex={itemIndex}
            effects={itemEffects.effects ?? []}
            effectTypes={itemEffects.effect_types ?? []}
            reason={itemEffects.reason ?? ""}
          />
        ) : (
          <p className="ui-field__hint" style={{ marginBottom: 0 }}>
            editor indisponível — o web-api não respondeu <code>ItemEffectAdminService</code>
          </p>
        )}
      </div>

      <div className="ui-card" style={{ marginTop: 16 }}>
        <h2 style={{ fontSize: 13, margin: "0 0 8px" }}>Onde este item aparece</h2>
        <p style={{ margin: 0, color: "var(--muted)", fontSize: 13 }}>
          Preço global (issue 24), quem dropa (issue 26) e uso em loja de NPC (issue 23) ainda não
          têm RPC ligada no portal — os atalhos entram junto com essas telas.
        </p>
      </div>
    </>
  );
}
