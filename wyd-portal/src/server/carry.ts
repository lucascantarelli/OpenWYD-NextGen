/**
 * Leitura do `Carry[]` de um template (issue 34, fatia 3).
 *
 * Separado de `server/drops.ts` porque é por template e não tem cache: abrir o
 * editor precisa do valor real daquele momento, e o resultado é pequeno (no
 * máximo 64 posições).
 */
import type { MobCarry } from "@/lib/mobcarry";

import { adminCall } from "./admin";
import { dropAdmin, type ListMobCarryResponse } from "./admin-client";

export async function listMobCarry(templateName: string): Promise<MobCarry> {
  const res = await adminCall<{ template_name: string }, ListMobCarryResponse>(
    dropAdmin.listMobCarry.bind(dropAdmin),
    { template_name: templateName },
    { action: "drop.carry", target: `drop:carry:${templateName}` },
  );
  return {
    template_name: res.template_name ?? templateName,
    display_name: res.display_name ?? "",
    level: res.level ?? 0,
    overridden: !!res.overridden,
    slots: res.slots ?? [],
  };
}
