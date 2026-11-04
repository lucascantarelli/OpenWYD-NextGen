/**
 * Leitura de templates de mob e dos seus stats (issue 25).
 *
 * `ListMobTemplates` devolve os 620 templates do content tree (monstros
 * inclusos, diferente de `ListMerchantTemplates`). `GetMobTemplateStat` tem
 * read-through: sem override salvo, vêm os valores reais do arquivo
 * (`overridden=false`), então o editor abre com o valor de partida em vez de
 * zeros.
 */
import type { MobStat, MobTemplateFile } from "@/lib/mob";

import { adminCall } from "./admin";
import { mobAdmin, type GetMobTemplateStatResponse, type ListMobTemplatesResponse } from "./grpc";

export async function listMobTemplates(): Promise<MobTemplateFile[]> {
  const res = await adminCall<Record<string, never>, ListMobTemplatesResponse>(
    mobAdmin.listMobTemplates.bind(mobAdmin),
    {},
    { action: "mob.templates" },
  );
  return res.templates ?? [];
}

/** `null` quando o template não existe (ou o web-api subiu sem `-content`). */
export async function getMobStat(templateName: string): Promise<MobStat | null> {
  if (!templateName) return null;
  const res = await adminCall<{ template_name: string }, GetMobTemplateStatResponse>(
    mobAdmin.getMobTemplateStat.bind(mobAdmin),
    { template_name: templateName },
    { action: "mob.get", target: `mob:${templateName}` },
  );
  return res.stat ?? null;
}
