import Link from "next/link";
import { notFound } from "next/navigation";

import { Badge } from "@/components/Badge";
import { GrantDialog } from "@/components/character/grant-dialog";
import {
  addLabel,
  cityLabel,
  classLabel,
  itemEffects,
  itemLabel,
  learnedSkills,
  skillLabel,
  type CharacterDetail,
  type CharacterItem,
} from "@/lib/character";
import { getCatalog } from "@/server/catalog";
import { AdminError } from "@/server/errors";
import { getCharacter } from "@/server/characters";

export const dynamic = "force-dynamic";

export const metadata = { title: "Personagem · WYD2" };

/**
 * Detalhe do personagem (issue 43): status, skills e inventário reais.
 *
 * Leitura pura — cada item aponta para o catálogo (`/itens/<index>`), que já
 * resolve nome e ícone. Edição/concessão entram na fase de escrita da issue.
 */
export default async function CharacterPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const characterId = Number(id);
  if (!Number.isInteger(characterId) || characterId <= 0) notFound();

  let detail: CharacterDetail | null = null;
  let error: string | null = null;
  try {
    detail = await getCharacter(characterId);
  } catch (err) {
    error =
      err instanceof AdminError ? err.message : "não foi possível carregar o personagem";
  }

  if (error) {
    return (
      <div className="ui-card ui-empty">
        <strong>Não deu para carregar o personagem</strong>
        <p>{error}</p>
        <Link href="/personagens">voltar</Link>
      </div>
    );
  }
  if (!detail) notFound();

  // Vocabulário de efeitos para o seletor de adds da concessão (issue 44).
  let effectTypes: { code: number; label: string }[] = [];
  try {
    effectTypes = (await getCatalog()).effectTypes.map(({ code, label }) => ({ code, label }));
  } catch {
    // sem vocabulário o seletor fica só com “nenhum”; a concessão continua válida
  }

  const summary = detail.summary;
  const skills = learnedSkills(detail.learned_skill);

  return (
    <>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            {summary.name}
          </h1>
          <p className="ui-page-sub" style={{ margin: "4px 0 0" }}>
            conta <span className="ui-mono">{summary.account_name}</span> ·{" "}
            {classLabel(summary.class)} · nível {summary.level} · slot {summary.slot}
          </p>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: 8 }}>
          <GrantDialog
            characterId={summary.id}
            characterName={summary.name}
            effectTypes={effectTypes}
          />
          <Link href="/personagens" className="ui-btn ui-btn--ghost">
            voltar à busca
          </Link>
        </div>
      </div>

      <div className="ui-card" style={{ marginBottom: 12 }}>
        <strong>Leitura + concessão</strong>
        <p style={{ margin: "4px 0 0" }}>
          O painel lê o save em Postgres. Concessão de <strong>item</strong> vai por{" "}
          <code>delivery_queue</code> (chega no próximo dreno, seguro com o personagem online).
          Concessão de gold e edição de campo continuam pendentes: não há caminho de fila para
          gold e não existe presença online para bloquear a edição.
        </p>
      </div>

      <h2 className="ui-page-title" style={{ fontSize: 15 }}>
        Status
      </h2>
      <div className="ui-card">
        <dl
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))",
            gap: 10,
            margin: 0,
          }}
        >
          <Stat label="HP" value={`${detail.hp} / ${detail.max_hp}`} />
          <Stat label="MP" value={`${detail.mp} / ${detail.max_mp}`} />
          <Stat label="Gold" value={summary.coin.toLocaleString("pt-BR")} />
          <Stat label="EXP" value={summary.exp.toLocaleString("pt-BR")} />
          <Stat label="Força" value={detail.str} />
          <Stat label="Inteligência" value={detail.intel} />
          <Stat label="Destreza" value={detail.dex} />
          <Stat label="Constituição" value={detail.con} />
          <Stat label="Cidade" value={cityLabel(detail.last_city)} />
          <Stat label="Save point" value={`${detail.save_x}, ${detail.save_y}`} />
        </dl>
      </div>

      <h2 className="ui-page-title" style={{ fontSize: 15, marginTop: 20 }}>
        Skills
      </h2>
      <div className="ui-card">
        {skills.length === 0 ? (
          <p className="ui-field__hint" style={{ margin: 0 }}>
            nenhuma skill aprendida no save
          </p>
        ) : (
          <p style={{ margin: 0, display: "flex", gap: 6, flexWrap: "wrap" }}>
            {skills.map((skill) => (
              <Badge key={skill} tone="neutral">
                {skillLabel(detail.skill_names, skill)}
              </Badge>
            ))}
          </p>
        )}
        <p className="ui-field__hint" style={{ marginBottom: 0 }}>
          barra:{" "}
          {detail.skill_bar
            .map((slot) => skillLabel(detail.skill_names, slot))
            .join(" · ")}
          {"   "}pontos de maestria: {detail.special.join(" · ")}
        </p>
      </div>

      <h2 className="ui-page-title" style={{ fontSize: 15, marginTop: 20 }}>
        Equipamento ({detail.equip.length}/16)
      </h2>
      <ItemTable
        items={detail.equip}
        addLabels={detail.add_labels}
        emptyMessage="Nenhum item equipado."
      />

      <h2 className="ui-page-title" style={{ fontSize: 15, marginTop: 20 }}>
        Inventário ({detail.carry.length}/64)
      </h2>
      <ItemTable items={detail.carry} addLabels={detail.add_labels} emptyMessage="Cargo vazio." />
    </>
  );
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <dt className="ui-field__label" style={{ margin: 0 }}>
        {label}
      </dt>
      <dd style={{ margin: 0 }} className="ui-mono">
        {value}
      </dd>
    </div>
  );
}

function ItemTable({
  items,
  emptyMessage,
  addLabels,
}: {
  items: CharacterItem[];
  emptyMessage: string;
  addLabels?: Record<string, string>;
}) {
  if (items.length === 0) {
    return (
      <div className="ui-card ui-empty">
        <p style={{ margin: 0 }}>{emptyMessage}</p>
      </div>
    );
  }
  return (
    <div className="ui-table-wrap">
      <div className="ui-table-scroll">
        <table className="ui-table">
          <thead>
            <tr>
              <th>Slot</th>
              <th>Item</th>
              <th>Adds</th>
              <th>Expira</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={`${item.slot}-${item.item_index}`}>
                <td>{item.slot}</td>
                <td>
                  <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
                    {item.icon_url ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        src={item.icon_url}
                        alt=""
                        width={20}
                        height={20}
                        style={{ imageRendering: "pixelated", borderRadius: 4 }}
                      />
                    ) : null}
                    <Link href={`/itens/${item.item_index}`} title={`índice ${item.item_index}`}>
                      {itemLabel(item)}
                    </Link>
                  </span>
                </td>
                <td>
                  {itemEffects(item).length === 0 ? (
                    <span className="ui-field__hint">—</span>
                  ) : (
                    <span>
                      {itemEffects(item)
                        .map((pair) => `${addLabel(addLabels, pair.eff)} ${pair.value}`)
                        .join(" · ")}
                    </span>
                  )}
                </td>
                <td>
                  {item.expires_at > 0
                    ? new Date(item.expires_at * 1000).toLocaleDateString("pt-BR")
                    : "permanente"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
