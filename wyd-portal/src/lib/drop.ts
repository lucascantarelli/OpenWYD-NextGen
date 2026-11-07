/**
 * Tipos e vocabulário do relatório de drops (issue 26).
 *
 * Puro e isomórfico: a página (server), as actions e a tabela (client) usam o
 * mesmo módulo. Nada aqui recalcula drop — `rate_divisor` vem do web-api
 * (g_pDropRate[slot] de `webserver/internal/droptool`, cópia de `loot.go`).
 *
 * A chance por slot **não** existe como porcentagem: o runtime divide por
 * `DropRate[slot]` e compara com o RNG. Por isso a tela mostra **faixa de
 * raridade**, nunca "x%".
 */
import type { Tone } from "@/lib/item";

/** Um mob/slot que pode dropar o item (visão item → mobs). */
export interface DropMobRef {
  template_name: string;
  mob_name: string;
  mob_level: number;
  slot: number;
  rate_divisor: number;
}

/** Um item como a visão item → mobs o devolve, com todos os seus slots. */
export interface ItemDropEntry {
  item_index: number;
  item_name: string;
  mobs: DropMobRef[];
}

/** Um slot `Carry[]` do template (visão mob → itens). */
export interface MobDropSlot {
  slot: number;
  item_index: number;
  item_name: string;
  rate_divisor: number;
}

/** Um template de mob com todos os slots de drop decodificados. */
export interface MobDropEntry {
  template_name: string;
  mob_name: string;
  mob_level: number;
  items: MobDropSlot[];
}

export type RarityKey = "sempre" | "muito-comum" | "comum" | "incomum" | "raro";

export interface RarityBand {
  key: RarityKey;
  label: string;
  tone: Tone | "neutral";
  /** Ordenação da faixa (0 = mais comum). */
  order: number;
}

/**
 * Faixas derivadas dos divisores que realmente existem em `g_pDropRate`:
 * 1 (slot 56), 4, 35, 500, 900, 2000, 2500, 3000, 5000, 10000, 20000.
 * Os cortes são do divisor, não do slot — assim uma mudança futura no vetor
 * não deixa a tabela mentindo sobre o slot.
 */
const BANDS: readonly (RarityBand & { max: number })[] = [
  { key: "sempre", label: "sempre cai", tone: "ok", order: 0, max: 1 },
  { key: "muito-comum", label: "muito comum", tone: "ok", order: 1, max: 100 },
  { key: "comum", label: "comum", tone: "accent", order: 2, max: 1000 },
  { key: "incomum", label: "incomum", tone: "neutral", order: 3, max: 3000 },
  { key: "raro", label: "raro", tone: "warn", order: 4, max: Number.MAX_SAFE_INTEGER },
];

export function rarityOf(divisor: number): RarityBand {
  const band = BANDS.find((entry) => divisor <= entry.max) ?? BANDS[BANDS.length - 1];
  const { max: _max, ...rest } = band;
  return rest;
}

/** Explicação fixa da tela: o número é divisor e mora no código. */
export const DROP_NOTE =
  "a chance por slot é o divisor de g_pDropRate (loot.go): quanto maior, mais raro, e o slot 56 é sempre. " +
  "Editar a chance é a issue 34.";

/**
 * Linha achatada do relatório item → mobs. A tela agrupa por item, mas a
 * tabela trabalha com pares (item, mob/slot) — é o que o CSV exporta.
 */
export interface ItemDropRow {
  item_index: number;
  item_name: string;
  template_name: string;
  mob_name: string;
  mob_level: number;
  slot: number;
  rate_divisor: number;
}

export function flattenItemDrops(entries: readonly ItemDropEntry[]): ItemDropRow[] {
  const rows: ItemDropRow[] = [];
  for (const entry of entries) {
    for (const mob of entry.mobs) {
      rows.push({
        item_index: entry.item_index,
        item_name: entry.item_name,
        template_name: mob.template_name,
        mob_name: mob.mob_name,
        mob_level: mob.mob_level,
        slot: mob.slot,
        rate_divisor: mob.rate_divisor,
      });
    }
  }
  return rows;
}

/** Linha achatada do relatório mob → itens (um slot por linha). */
export interface MobDropRow {
  template_name: string;
  mob_name: string;
  mob_level: number;
  slot: number;
  item_index: number;
  item_name: string;
  rate_divisor: number;
}

export function flattenMobDrops(entries: readonly MobDropEntry[]): MobDropRow[] {
  const rows: MobDropRow[] = [];
  for (const entry of entries) {
    for (const item of entry.items) {
      rows.push({
        template_name: entry.template_name,
        mob_name: entry.mob_name,
        mob_level: entry.mob_level,
        slot: item.slot,
        item_index: item.item_index,
        item_name: item.item_name,
        rate_divisor: item.rate_divisor,
      });
    }
  }
  return rows;
}

/** Contadores do topo: itens distintos, mobs distintos e pares. */
export function dropCounters(
  rows: readonly { item_index: number; template_name: string }[],
): { items: number; mobs: number; pairs: number } {
  const items = new Set<number>();
  const mobs = new Set<string>();
  for (const row of rows) {
    items.add(row.item_index);
    mobs.add(row.template_name);
  }
  return { items: items.size, mobs: mobs.size, pairs: rows.length };
}

/**
 * CSV das duas visões. Só cabeçalho + linhas, sem HTML — o arquivo é aberto em
 * planilha. A ordem das colunas é a mesma da tabela.
 */
export function dropsToCsv(rows: readonly Record<string, string | number>[]): string {
  if (rows.length === 0) return "";
  const headers = Object.keys(rows[0]);
  const escape = (value: string | number) => {
    const text = String(value ?? "");
    return /[",;\n]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text;
  };
  return [
    headers.join(";"),
    ...rows.map((row) => headers.map((header) => escape(row[header])).join(";")),
  ].join("\n");
}
