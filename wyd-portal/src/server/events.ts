/**
 * Eventos de mundo (WEB-11, WEB-06).
 *
 * Foto (`world_event_admin/get_world_event_config`) e liga/desliga auditado
 * (`set_world_event_config`), via `adminCall`. Ligar/desligar vale para o
 * próximo leitor do dado (boot/tick que lê a tabela): nenhum push para o Hot Path.
 */
import { adminCall } from "./admin";
import {
  worldEventAdmin,
  type RevisionResponse,
  type WorldEventResponse,
  type WorldEventWire,
} from "./admin-client";

export type { WorldEventWire };

const WEEKDAYS = ["dom", "seg", "ter", "qua", "qui", "sex", "sáb"] as const;

/** Dia da semana 0..6 em rótulo curto; fora da faixa volta o número. */
export function weekdayLabel(weekday: number): string {
  return WEEKDAYS[weekday] ?? `dia ${weekday}`;
}

/** `HH:MM` + duração em minutos em rótulo de grade; dado ausente vira "—". */
export function scheduleLabel(start: string, durationMinutes: number): string {
  const time = start.trim();
  if (time === "" || !Number.isFinite(durationMinutes) || durationMinutes <= 0) return "—";
  return `${time} · ${durationMinutes} min`;
}

/** Foto dos eventos com horários, na ordem do banco. */
export async function getWorldEvents(): Promise<WorldEventWire[]> {
  const res = await adminCall<Record<string, never>, WorldEventResponse>(
    worldEventAdmin.getWorldEventConfig.bind(worldEventAdmin),
    {},
    { action: "config.events", target: "events" },
  );
  return res.events ?? [];
}

/** Liga/desliga um evento; devolve a revisão. Código vazio nem chega ao gateway. */
export async function setWorldEventEnabled(code: string, enabled: boolean): Promise<number> {
  const clean = code.trim().slice(0, 60);
  if (clean === "") throw new Error("código de evento inválido");
  const res = await adminCall<{ code: string; enabled: boolean }, RevisionResponse>(
    worldEventAdmin.setWorldEventConfig.bind(worldEventAdmin),
    { code: clean, enabled },
    { action: "config.event.set", target: `event:${clean}` },
  );
  return res.revision ?? 0;
}
