/**
 * Regras de apresentação do catálogo de itens.
 *
 * Puro e isomórfico: usado pela página (server) e pela tabela (client). Nada
 * aqui inventa dado — `slots` vem decodificado do `slot_mask` pelo web-api e
 * `icon_key` é **opaco**, só serve para saber se existe ícone.
 */
import type { CatalogItem } from "@/server/admin-client";

/** `grade` do ItemList.csv: 1=Normal … 4=Lendário (proto:ItemCatalogEntry). */
const GRADE: Record<number, string> = {
  1: "Normal",
  2: "Místico",
  3: "Arcano",
  4: "Lendário",
};

export type Tone = "ok" | "warn" | "danger" | "accent";

export function gradeLabel(grade: number): string {
  return GRADE[grade] ?? (grade > 0 ? `Grau ${grade}` : "—");
}

export function gradeTone(grade: number): Tone | undefined {
  const map: Record<number, Tone> = { 1: "ok", 2: "accent", 3: "warn", 4: "danger" };
  return map[grade];
}

/** Sigla do fallback de ícone: até 2 letras do nome de exibição. */
export function itemInitials(item: Pick<CatalogItem, "display_name" | "name">): string {
  const source = (item.display_name || item.name || "?").replace(/[_()[\]]/g, " ").trim();
  const words = source.split(/\s+/).filter(Boolean);
  if (words.length === 0) return "?";
  if (words.length === 1) return words[0].slice(0, 2).toUpperCase();
  return (words[0][0] + words[1][0]).toUpperCase();
}

/** Slots decodificados, como vieram (`boots`, `weapon`…), sem tradução inventada. */
export function slotsText(item: Pick<CatalogItem, "slots" | "slot_mask">): string {
  if (item.slots?.length) return item.slots.join(", ");
  return item.slot_mask ? `mask ${item.slot_mask}` : "—";
}

/** Texto de busca: nome cru, nome de exibição e índice. */
export function itemSearchText(item: CatalogItem): string {
  return `${item.item_index} ${item.name} ${item.display_name ?? ""}`.toLowerCase();
}

/** Todos os slots distintos presentes no catálogo, ordenados. */
export function distinctSlots(items: readonly CatalogItem[]): string[] {
  const set = new Set<string>();
  for (const item of items) for (const slot of item.slots ?? []) set.add(slot);
  return [...set].sort((a, b) => a.localeCompare(b, "pt-BR"));
}
