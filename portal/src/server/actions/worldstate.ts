"use server";

/**
 * Escrita do estado do mundo (issue 45): taxa da cidade e ciclo do Kefra.
 *
 * As duas são estado de banco que o tmserver lê no boot — a tela avisa que a
 * mudança só entra em vigor no próximo restart. `ADMIN_RESULT.OK` é 1 (ver
 * `admin-result.ts`); qualquer outro valor traz o motivo em `message`.
 */
import { revalidatePath } from "next/cache";

import { adminCall } from "@/server/admin";
import { isOk } from "@/server/admin-result";
import { AdminError, isInfraFailure } from "@/server/errors";
import { worldStateAdmin, type SetWorldStateResponse } from "@/server/grpc";

export type WorldStateActionResult = { ok: true; message?: string } | { ok: false; error: string };

function failure(error: unknown): WorldStateActionResult {
  if (error instanceof AdminError) return { ok: false, error: error.message };
  if (isInfraFailure(error)) return { ok: false, error: "serviço de administração indisponível" };
  console.error(
    "[world_state] falha inesperada:",
    error instanceof Error ? error.message : String(error),
  );
  return { ok: false, error: "erro interno do portal" };
}

function outcome(res: SetWorldStateResponse, okMessage: string): WorldStateActionResult {
  if (isOk(res.result)) return { ok: true, message: okMessage };
  return { ok: false, error: res.message || "o web-api recusou a alteração" };
}

export async function setCityTax(input: {
  zone: number;
  cityTax: number;
}): Promise<WorldStateActionResult> {
  try {
    const res = await adminCall<
      { zone: number; city_tax: number },
      SetWorldStateResponse
    >(
      worldStateAdmin.setCityTax.bind(worldStateAdmin),
      { zone: input.zone, city_tax: input.cityTax },
      { action: "world_state.city_tax", target: `zone:${input.zone}` },
    );
    revalidatePath("/castelo");
    return outcome(res, "taxa salva — vale depois do restart do tmserver");
  } catch (error) {
    return failure(error);
  }
}

export async function setGuildMaster(input: {
  guildId: number;
  characterId: number;
  reason: string;
}): Promise<WorldStateActionResult> {
  try {
    const res = await adminCall<
      { guild_id: number; character_id: number; reason: string },
      SetWorldStateResponse
    >(
      worldStateAdmin.setGuildMaster.bind(worldStateAdmin),
      { guild_id: input.guildId, character_id: input.characterId, reason: input.reason },
      { action: "guild.set_master", target: `guild:${input.guildId}` },
    );
    revalidatePath("/guildas");
    return outcome(res, "mestrado transferido — o vínculo vale no próximo restart");
  } catch (error) {
    return failure(error);
  }
}

export async function removeGuildMember(input: {
  guildId: number;
  characterId: number;
  reason: string;
}): Promise<WorldStateActionResult> {
  try {
    const res = await adminCall<
      { guild_id: number; character_id: number; reason: string },
      SetWorldStateResponse
    >(
      worldStateAdmin.removeGuildMember.bind(worldStateAdmin),
      { guild_id: input.guildId, character_id: input.characterId, reason: input.reason },
      { action: "guild.remove_member", target: `guild:${input.guildId}` },
    );
    revalidatePath("/guildas");
    return outcome(res, "membro removido — o vínculo vale no próximo restart");
  } catch (error) {
    return failure(error);
  }
}

export async function setGuildRelation(input: {
  guildId: number;
  targetGuildId: number;
  kind: number;
  reason: string;
}): Promise<WorldStateActionResult> {
  try {
    const res = await adminCall<
      { guild_id: number; target_guild_id: number; kind: number; reason: string },
      SetWorldStateResponse
    >(
      worldStateAdmin.setGuildRelation.bind(worldStateAdmin),
      {
        guild_id: input.guildId,
        target_guild_id: input.targetGuildId,
        kind: input.kind,
        reason: input.reason,
      },
      { action: "guild.set_relation", target: `guild:${input.guildId}` },
    );
    revalidatePath("/guildas");
    return outcome(
      res,
      input.kind === 0
        ? "relação removida"
        : "relação salva — vale no próximo restart do tmserver",
    );
  } catch (error) {
    return failure(error);
  }
}

export async function setZoneOwner(input: {
  zone: number;
  guildId: number;
  reason: string;
}): Promise<WorldStateActionResult> {
  try {
    const res = await adminCall<
      { zone: number; guild_id: number; reason: string },
      SetWorldStateResponse
    >(
      worldStateAdmin.setZoneOwner.bind(worldStateAdmin),
      { zone: input.zone, guild_id: input.guildId, reason: input.reason },
      { action: "world_state.zone_owner", target: `zone:${input.zone}` },
    );
    revalidatePath("/castelo");
    revalidatePath("/guildas");
    return outcome(
      res,
      input.guildId === 0
        ? "dono removido — a cidade fica sem guilda no poder até o próximo cerco"
        : "dono salvo — a cidade muda no próximo restart do tmserver",
    );
  } catch (error) {
    return failure(error);
  }
}

export async function setTowerOwner(input: {
  guildId: number;
  reason: string;
}): Promise<WorldStateActionResult> {
  try {
    const res = await adminCall<{ guild_id: number; reason: string }, SetWorldStateResponse>(
      worldStateAdmin.setTowerOwner.bind(worldStateAdmin),
      { guild_id: input.guildId, reason: input.reason },
      { action: "world_state.tower_owner", target: "tower" },
    );
    revalidatePath("/castelo");
    revalidatePath("/guildas");
    return outcome(
      res,
      input.guildId === 0 ? "torre sem dono" : "dono da torre salvo — vale no próximo restart",
    );
  } catch (error) {
    return failure(error);
  }
}

export async function setKefraDefeated(input: {
  defeated: boolean;
}): Promise<WorldStateActionResult> {
  try {
    const res = await adminCall<{ defeated: boolean }, SetWorldStateResponse>(
      worldStateAdmin.setKefraDefeated.bind(worldStateAdmin),
      { defeated: input.defeated },
      { action: "world_state.kefra", target: "kefra" },
    );
    revalidatePath("/castelo");
    revalidatePath("/guildas");
    return outcome(
      res,
      input.defeated
        ? "Kefra marcado como morto — o portal do Deserto abre no próximo restart"
        : "Kefra marcado como vivo — o portal do Deserto fecha no próximo restart",
    );
  } catch (error) {
    return failure(error);
  }
}
