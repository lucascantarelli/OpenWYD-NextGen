import Link from "next/link";

import { MobForm } from "@/components/mob/mob-form";
import { RetryButton } from "@/components/RetryButton";
import type { MobStat } from "@/lib/mob";
import { getCatalog } from "@/server/catalog";
import { AdminError } from "@/server/errors";
import { getMobStat } from "@/server/mobs";

export const dynamic = "force-dynamic";

export const metadata = { title: "Template de mob · WYD2" };

/**
 * Detalhe/edição de um template (issue 25). O nome vem da URL codificado
 * (`encodeURIComponent`) porque os templates usam `@`, `[` e afins.
 */
export default async function MobDetailPage({
  params,
}: {
  params: Promise<{ template: string }>;
}) {
  const { template } = await params;
  const templateName = decodeURIComponent(template);

  let stat: MobStat | null = null;
  let error: string | null = null;
  try {
    stat = await getMobStat(templateName);
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível carregar o template do web-api";
  }

  // Nomes dos itens do equip: resolvidos aqui (catálogo é cacheado por
  // catalog_version). Catálogo fora do ar não impede editar stats — o slot cai
  // no fallback "item #N".
  const itemNames: Record<string, string> = {};
  if (stat?.equip?.length) {
    try {
      const catalog = await getCatalog();
      for (const slot of stat.equip) {
        const item = catalog.byIndex.get(slot.item_index);
        if (item) itemNames[String(slot.item_index)] = item.display_name || item.name;
      }
    } catch {
      // silencioso de propósito: é rótulo, não dado
    }
  }

  return (
    <>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            <span className="ui-mono">{templateName}</span>
          </h1>
          <p className="ui-page-sub" style={{ margin: "var(--space-1) 0 0" }}>
            Stats de combate do template. Salvar cria/atualiza o override no banco; reverter volta ao
            arquivo.{" "}
            <Link href="/mobs">voltar à lista</Link>
          </p>
        </div>
      </div>

      {error && (
        <div className="ui-card ui-empty" style={{ marginBottom: "var(--space-3)" }}>
          <strong>Não deu para carregar o template</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      )}

      {!error && !stat && (
        <div className="ui-card ui-empty">
          <strong>Template não encontrado</strong>
          <p>
            O web-api não conhece <span className="ui-mono">{templateName}</span>. Confira o nome na{" "}
            <Link href="/mobs">lista de templates</Link>.
          </p>
        </div>
      )}

      {!error && stat && <MobForm stat={stat} itemNames={itemNames} />}
    </>
  );
}
