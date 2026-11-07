import Link from "next/link";

import { NpcForm } from "@/components/npc/npc-form";
import type { MapZone, MerchantTemplate } from "@/lib/npc";
import { AdminError } from "@/server/errors";
import { loadNpcFormOptions } from "@/server/npcs";

export const dynamic = "force-dynamic";

export const metadata = { title: "Novo NPC · WYD2" };

export default async function NovoNpcPage() {
  let templates: MerchantTemplate[] = [];
  let zones: MapZone[] = [];
  let error: string | null = null;

  try {
    ({ templates, zones } = await loadNpcFormOptions());
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível carregar templates e zonas do web-api";
  }

  return (
    <>
      <p className="ui-field__hint" style={{ marginBottom: 8 }}>
        <Link href="/npcs">← NPCs</Link>
      </p>
      <h1 className="ui-page-title">Novo NPC</h1>
      <p className="ui-page-sub">
        A definição é criada no Postgres com <code>origin=custom</code>; o tmserver a materializa no
        próximo restart (ou no poll de ~15 s).
      </p>

      {error ? (
        <div className="ui-card ui-empty">
          <strong>Formulário bloqueado</strong>
          <p>{error}</p>
        </div>
      ) : (
        <NpcForm mode="create" templates={templates} zones={zones} />
      )}
    </>
  );
}
