/**
 * Enciclopédia pública de itens — leitura do JSON gerado por
 * `tools/content/gen-enciclopedia.mjs` (`node tools/content/gen-enciclopedia.mjs`).
 *
 * Não passa pelo gateway de propósito: é conteúdo estático e a rota é pública. O
 * catálogo administrativo (`/itens`) é outra leitura, atrás de sessão.
 */
import raw from "@/content/itens.json";

import { fold, foldMatch } from "@/lib/fold";

/** Item no formato compacto do gerador. */
type Row = [
  index: number,
  name: string,
  price: number,
  grade: number,
  reqLevel: number,
  kind: string | null,
  system: string | null,
  eventOnly: boolean,
  tradeable: boolean,
  effects: [number, number][],
];

export interface EncyclopediaItem {
  index: number;
  name: string;
  price: number;
  grade: number;
  reqLevel: number;
  kind: string | null;
  system: string | null;
  eventOnly: boolean;
  tradeable: boolean;
  effects: { code: number; value: number }[];
  /** Chave de busca dobrada, calculada uma vez no carregamento. */
  key: string;
}

interface Payload {
  version: number;
  count: number;
  items: Row[];
}

const PAYLOAD = raw as unknown as Payload;

if (!PAYLOAD || !Array.isArray(PAYLOAD.items) || PAYLOAD.items.length === 0) {
  throw new Error("itens.json vazio: rode `node tools/content/gen-enciclopedia.mjs`");
}

/** Ordenado por nome dobrado, para a lista já sair alfabética em português. */
const ITEMS: EncyclopediaItem[] = PAYLOAD.items
  .map(([index, name, price, grade, reqLevel, kind, system, eventOnly, tradeable, effects]) => ({
    index,
    name,
    price,
    grade,
    reqLevel,
    kind,
    system,
    eventOnly,
    tradeable,
    effects: effects.map(([code, value]) => ({ code, value })),
    key: fold(name),
  }))
  .sort((a, b) => a.key.localeCompare(b.key, "pt-BR"));

const BY_INDEX = new Map(ITEMS.map((item) => [item.index, item]));

export const ENCYCLOPEDIA_VERSION = PAYLOAD.version;
export const ENCYCLOPEDIA_COUNT = ITEMS.length;

export interface ItemQuery {
  q?: string;
  kind?: string;
  limit?: number;
}

export interface ItemQueryResult {
  items: EncyclopediaItem[];
  total: number;
  /** Termo usado depois da dobra — mostrado ao usuário para explicar o resultado. */
  folded: string;
}

/** Busca por nome (sem acento/caixa) e filtro por tipo. */
export function searchItems({ q = "", kind = "", limit = 120 }: ItemQuery = {}): ItemQueryResult {
  const folded = fold(q);
  const byKind = kind ? ITEMS.filter((item) => item.kind === kind) : ITEMS;
  const matches = folded ? byKind.filter((item) => foldMatch(item.name, q)) : byKind;

  return {
    items: matches.slice(0, limit),
    total: matches.length,
    folded,
  };
}

export function getItem(index: number): EncyclopediaItem | null {
  return BY_INDEX.get(index) ?? null;
}

/** Tipos presentes no catálogo, com contagem — alimenta os filtros da lista. */
export function itemKinds(): { kind: string; count: number }[] {
  const counts = new Map<string, number>();
  for (const item of ITEMS) {
    const kind = item.kind ?? "sem-tipo";
    counts.set(kind, (counts.get(kind) ?? 0) + 1);
  }
  return [...counts.entries()]
    .map(([kind, count]) => ({ kind, count }))
    .sort((a, b) => b.count - a.count || a.kind.localeCompare(b.kind, "pt-BR"));
}
