/**
 * Modelo do editor de NPC/mercador — compartilhado entre server e client.
 *
 * Espelha `NpcAdminService` (`api/web/v1/web.proto:198`) e a semântica de
 * domínio de `docs/integrations/npc-admin-nextjs.md §5`. Nada aqui é
 * autorização: a decisão é sempre do `AdminResult` que o web-api devolve.
 */
import { z } from "zod";

export interface NpcShopItem {
  slot: number;
  item_index: number;
  eff1: number;
  effv1: number;
  eff2: number;
  effv2: number;
  eff3: number;
  effv3: number;
  quantity: number;
}

export interface NpcDefinition {
  id: number;
  slug: string;
  template_name: string;
  display_name: string;
  enabled: boolean;
  map_id: number;
  pos_x: number;
  pos_y: number;
  route_type: number;
  merchant: number;
  shop: NpcShopItem[];
  /** `content` = veio do NPCGener.txt (não pode ser apagado); `custom` = criado aqui. */
  origin: string;
  generator_index: number;
}

export interface MerchantTemplate {
  template_name: string;
  display_name: string;
  merchant: number;
}

export interface MapZone {
  id: number;
  name: string;
}

/** Tipos de mercador aceitos pelo web-api (qualquer outro vira `INVALID`). */
export const MERCHANT_LABELS: Record<number, string> = {
  0: "não-mercador (sem loja)",
  1: "loja normal",
  2: "guarda-carga (armazém)",
  19: "loja tipo 3",
  100: "NPC de quest",
};

export const MERCHANT_VALUES: readonly number[] = [0, 1, 2, 19, 100];

export function merchantLabel(value: number): string {
  return MERCHANT_LABELS[value] ?? `tipo ${value}`;
}

export function hasShop(merchant: number): boolean {
  return merchant === 1 || merchant === 19;
}

export function originLabel(origin: string): string {
  return origin === "content" ? "conteúdo" : "painel";
}

/** `MSG_ShopList` 0..26 = 3 abas de 9 (`§5.2`). */
export const SHOP_SLOTS = 27;
export const SHOP_TABS: readonly { label: string; from: number; to: number }[] = [
  { label: "Aba 1", from: 0, to: 8 },
  { label: "Aba 2", from: 9, to: 17 },
  { label: "Aba 3", from: 18, to: 26 },
];

export const ROUTE_LABELS: Record<number, string> = {
  0: "parado",
  1: "rota 1",
  2: "rota 2",
  3: "rota 3",
};

export const npcFormSchema = z.object({
  slug: z
    .string()
    .trim()
    .regex(/^[a-z0-9-]{3,40}$/, "use de 3 a 40 caracteres: a-z, 0-9 e hífen"),
  display_name: z.string().trim().min(1, "informe o nome exibido").max(40, "máximo de 40"),
  template_name: z.string().trim().min(1, "escolha um template").max(64),
  map_id: z.number().int().min(0).max(4),
  pos_x: z.number().int().min(-32768).max(32767),
  pos_y: z.number().int().min(-32768).max(32767),
  route_type: z.number().int().min(0).max(3),
  merchant: z
    .number()
    .int()
    .refine((value) => MERCHANT_VALUES.includes(value), "tipo de mercador inválido"),
  enabled: z.boolean(),
});

export type NpcFormValues = z.infer<typeof npcFormSchema>;

export const shopItemSchema = z.object({
  slot: z.number().int().min(0).max(SHOP_SLOTS - 1),
  item_index: z.number().int().min(1),
  quantity: z.number().int().min(1).max(255),
  eff1: z.number().int().default(0),
  effv1: z.number().int().default(0),
  eff2: z.number().int().default(0),
  effv2: z.number().int().default(0),
  eff3: z.number().int().default(0),
  effv3: z.number().int().default(0),
});

export const shopSchema = z.object({
  npcId: z.number().int().positive(),
  items: z
    .array(shopItemSchema)
    .max(SHOP_SLOTS)
    .refine(
      (items) => new Set(items.map((item) => item.slot)).size === items.length,
      "há dois itens no mesmo slot",
    ),
});

export function emptyShopItem(slot: number): NpcShopItem {
  return {
    slot,
    item_index: 0,
    quantity: 1,
    eff1: 0,
    effv1: 0,
    eff2: 0,
    effv2: 0,
    eff3: 0,
    effv3: 0,
  };
}

export function sortShop(items: readonly NpcShopItem[]): NpcShopItem[] {
  return [...items].sort((a, b) => a.slot - b.slot);
}

/**
 * `SetNpcShop` **substitui** a loja inteira (`§5.2`): este diff é o que a tela
 * mostra antes de salvar, justamente para o moderador ver o que vai sumir.
 */
export function shopDiff(
  initial: readonly NpcShopItem[],
  current: readonly NpcShopItem[],
): { added: number; removed: number; changed: number } {
  const before = new Map(initial.map((item) => [item.slot, item]));
  const after = new Map(current.map((item) => [item.slot, item]));
  let added = 0;
  let removed = 0;
  let changed = 0;
  for (const [slot, item] of after) {
    const previous = before.get(slot);
    if (!previous) added += 1;
    else if (
      previous.item_index !== item.item_index ||
      previous.quantity !== item.quantity ||
      previous.eff1 !== item.eff1 ||
      previous.effv1 !== item.effv1 ||
      previous.eff2 !== item.eff2 ||
      previous.effv2 !== item.effv2 ||
      previous.eff3 !== item.eff3 ||
      previous.effv3 !== item.effv3
    ) {
      changed += 1;
    }
  }
  for (const slot of before.keys()) if (!after.has(slot)) removed += 1;
  return { added, removed, changed };
}
