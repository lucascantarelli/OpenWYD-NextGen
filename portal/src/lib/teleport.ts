/**
 * Tipos e validação da tela de teleportes (issue 42).
 *
 * A lista é **mesclada** pelo web-api: rota compilada (`internal/teleportdata`) +
 * linha do overlay (`teleport_route`). Por isso toda rota tem `from_content` e
 * `has_override`: é o que permite dizer na tela “padrão do servidor” vs “editada
 * na web” e oferecer “voltar ao padrão” só quando há linha.
 */
import { z } from "zod";

export const REQUIREMENTS = ["none", "fairy", "kefra"] as const;

export type TeleportRequirement = (typeof REQUIREMENTS)[number];

export const requirementLabel: Record<TeleportRequirement, string> = {
  none: "nenhum",
  fairy: "Fada do Vale (item 3916)",
  kefra: "Kefra derrotado",
};

export interface TeleportRoute {
  src_x: number;
  src_y: number;
  dst_x: number;
  dst_y: number;
  cost: number;
  requirement: string;
  enabled: boolean;
  note: string;
  updated_by: number;
  updated_at: string;
  /** Origem existe na tabela compilada do servidor. */
  from_content: boolean;
  /** Existe linha em `teleport_route` (o resto vem do código). */
  has_override: boolean;
  /** Nome do lugar da origem, resolvido pelo atlas (issue 47). */
  src_zone?: string;
  /** Nome do lugar do destino, resolvido pelo atlas. */
  dst_zone?: string;
}

export interface TeleportSimulation {
  ok: boolean;
  src_x: number;
  src_y: number;
  dst_x: number;
  dst_y: number;
  cost: number;
  requirement: string;
  warnings: string[];
}

/** O servidor resolve por tile: `x &^ 3` (a mesma conta de `teleportDest`). */
export function roundTile(value: number): number {
  return value - (value % 4 + 4) % 4;
}

export function isRequirement(value: string): value is TeleportRequirement {
  return (REQUIREMENTS as readonly string[]).includes(value);
}

/** “Armia → Noatum” não existe no banco; a tela mostra tile e origem. */
export function routeOrigin(route: TeleportRoute): string {
  if (route.has_override && !route.from_content) return "criada na web";
  if (route.has_override) return "editada na web";
  return "padrão do servidor";
}

export const routeFormSchema = z.object({
  src_x: z.number().int().min(0).max(4095),
  src_y: z.number().int().min(0).max(4095),
  dst_x: z.number().int().min(0).max(4095),
  dst_y: z.number().int().min(0).max(4095),
  cost: z.number().int().min(0).max(1_000_000),
  requirement: z.enum(REQUIREMENTS),
  enabled: z.boolean(),
  note: z.string().max(120),
});

export type RouteFormValues = z.infer<typeof routeFormSchema>;

export function toFormValues(route: TeleportRoute): RouteFormValues {
  return {
    src_x: route.src_x,
    src_y: route.src_y,
    dst_x: route.dst_x,
    dst_y: route.dst_y,
    cost: route.cost,
    requirement: isRequirement(route.requirement) ? route.requirement : "none",
    enabled: route.enabled,
    note: route.note ?? "",
  };
}
