import Link from "next/link";
import { notFound } from "next/navigation";

import { NpcForm } from "@/components/npc/npc-form";
import { Badge } from "@/components/Badge";
import type { MapZone, MerchantTemplate, NpcDefinition } from "@/lib/npc";
import { getCatalog } from "@/server/catalog";
import { AdminError } from "@/server/errors";
import { getNpc, loadNpcFormOptions } from "@/server/npcs";

export const dynamic = "force-dynamic";

export default async function NpcPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const npcId = Number(id);
  if (!Number.isInteger(npcId) || npcId <= 0) notFound();

  let npc: NpcDefinition | null = null;
  try {
    npc = await getNpc(npcId);
  } catch (err) {
    if (err instanceof AdminError && err.status === 404) notFound();
    throw err;
  }
  if (!npc) notFound();

  let templates: MerchantTemplate[] = [];
  let zones: MapZone[] = [];
  try {
    ({ templates, zones } = await loadNpcFormOptions());
  } catch {
    // Picker indisponível não bloqueia a edição da definição (`§5.5`).
  }

  // Nome dos itens já na loja: vem do catálogo em cache, não de 27 round-trips.
  const shopNames: Record<string, string> = {};
  try {
    const catalog = await getCatalog();
    for (const item of npc.shop ?? []) {
      const entry = catalog.byIndex.get(item.item_index);
      if (entry) shopNames[String(item.item_index)] = entry.display_name || entry.name;
    }
  } catch {
    // Sem catálogo a loja mostra só o índice.
  }

  return (
    <>
      <p className="ui-field__hint" style={{ marginBottom: 8 }}>
        <Link href="/npcs">← NPCs</Link>
      </p>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            {npc.display_name || npc.slug}
          </h1>
          <p className="ui-page-sub" style={{ margin: "4px 0 0" }}>
            <span className="ui-mono">{npc.slug}</span> · template{" "}
            <span className="ui-mono">{npc.template_name}</span>{" "}
            <Badge tone={npc.enabled ? "ok" : "danger"}>{npc.enabled ? "visível" : "oculto"}</Badge>
          </p>
        </div>
      </div>

      <NpcForm
        mode="edit"
        npc={npc}
        templates={templates}
        zones={zones}
        shopNames={shopNames}
      />
    </>
  );
}
