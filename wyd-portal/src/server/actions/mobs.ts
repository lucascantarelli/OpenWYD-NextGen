"use server";

/**
 * Escritas do editor de stats de mob (issue 25).
 *
 * Duas regras que vêm do servidor e não podem ser contrariadas aqui:
 *
 * 1. o upsert **substitui o override inteiro**, `Equip[]` incluso — por isso o
 *    `saveMobStat` relê o equip do servidor antes de gravar, em vez de confiar
 *    no que o browser mandou. Se o moderador acabou de editar o equip em outra
 *    seção, um save de stats com cópia velha apagaria a alteração;
 * 2. `SetMobTemplateEquip` exige override existente: sem linha em
 *    `mob_template_stat` o web-api responde NOT_FOUND e a tela orienta a salvar
 *    os stats primeiro.
 */
import type { MobEquipSlot, MobStatValues } from "@/lib/mob";
import { mobEquipListSchema, mobStatSchema } from "@/lib/mob";

import { adminCall } from "@/server/admin";
import { AdminError, isInfraFailure } from "@/server/errors";
import {
  mobAdmin,
  type AdminAck,
  type GetMobTemplateStatResponse,
  type UpsertMobTemplateStatResponse,
} from "@/server/admin-client";

export type MobActionResult = { ok: true } | { ok: false; error: string; status: number };

function failure(error: unknown): MobActionResult {
  if (error instanceof AdminError) return { ok: false, error: error.message, status: error.status };
  if (isInfraFailure(error)) {
    return { ok: false, error: "serviço de administração indisponível", status: 502 };
  }
  console.error("[mob] falha inesperada:", error instanceof Error ? error.message : String(error));
  return { ok: false, error: "erro interno do portal", status: 500 };
}

export async function saveMobStat(input: { stat: MobStatValues }): Promise<MobActionResult> {
  const parsed = mobStatSchema.safeParse(input.stat);
  if (!parsed.success) {
    const issue = parsed.error.issues[0];
    const field = issue?.path.join(".");
    return {
      ok: false,
      status: 422,
      error: field ? `${field}: ${issue?.message}` : "dados inválidos",
    };
  }

  try {
    // O equip atual é do servidor, não do formulário: só os stats são editáveis aqui.
    const current = await adminCall<{ template_name: string }, GetMobTemplateStatResponse>(
      mobAdmin.getMobTemplateStat.bind(mobAdmin),
      { template_name: parsed.data.template_name },
      { action: "mob.get", target: `mob:${parsed.data.template_name}` },
    );
    const equip = current.stat?.equip ?? [];

    await adminCall<{ stat: MobStatValues & { equip: MobEquipSlot[] } }, UpsertMobTemplateStatResponse>(
      mobAdmin.upsertMobTemplateStat.bind(mobAdmin),
      { stat: { ...parsed.data, equip } },
      { action: "mob.upsert", target: `mob:${parsed.data.template_name}` },
    );
    return { ok: true };
  } catch (error) {
    return failure(error);
  }
}

/** Substitui `Equip[16]` — exige override de stats já salvo para o template. */
export async function saveMobEquip(input: {
  templateName: string;
  items: MobEquipSlot[];
}): Promise<MobActionResult> {
  const parsed = mobEquipListSchema.safeParse(input.items);
  if (!parsed.success) {
    const issue = parsed.error.issues[0];
    const slot = issue?.path[0];
    return {
      ok: false,
      status: 422,
      error:
        issue?.message === "slot duplicado"
          ? "slot duplicado no equip"
          : slot !== undefined
            ? `slot ${String(slot)}: ${issue?.message}`
            : "equip inválido",
    };
  }

  try {
    await adminCall<{ template_name: string; items: MobEquipSlot[] }, AdminAck>(
      mobAdmin.setMobTemplateEquip.bind(mobAdmin),
      { template_name: input.templateName, items: parsed.data },
      { action: "mob.equip", target: `mob:${input.templateName}` },
    );
    return { ok: true };
  } catch (error) {
    if (error instanceof AdminError && error.status === 404) {
      return {
        ok: false,
        status: 409,
        error: "salve os stats do template antes de editar o equip",
      };
    }
    return failure(error);
  }
}

/** Remove o override: o template volta a valer pelo arquivo (que não é tocado). */
export async function revertMobStat(templateName: string): Promise<MobActionResult> {
  try {
    await adminCall<{ template_name: string }, AdminAck>(
      mobAdmin.deleteMobTemplateStat.bind(mobAdmin),
      { template_name: templateName },
      { action: "mob.revert", target: `mob:${templateName}` },
    );
    return { ok: true };
  } catch (error) {
    return failure(error);
  }
}
