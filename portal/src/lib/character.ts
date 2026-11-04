/**
 * Tipos da tela de personagens (issue 43).
 *
 * Por enquanto **leitura**: o web-api expõe busca e detalhe a partir do estado
 * frio (Postgres). Concessão/edição exigem `delivery_queue` e editor de campo
 * offline, que ainda não existem — ver a issue.
 */

export interface CharacterSummary {
  id: number;
  account_id: number;
  account_name: string;
  name: string;
  slot: number;
  class: number;
  level: number;
  exp: number;
  coin: number;
}

export interface CharacterItem {
  slot: number;
  item_index: number;
  /** Nome do catálogo (`ItemList.csv`); vazio quando o índice não está lá. */
  name: string;
  /** URL do ícone gerado; vazio quando o item não tem mapping ou o pack não subiu. */
  icon_url: string;
  eff1: number;
  effv1: number;
  eff2: number;
  effv2: number;
  eff3: number;
  effv3: number;
  /** Unix seconds; 0 = permanente. */
  expires_at: number;
}

export interface CharacterDetail {
  summary: CharacterSummary;
  max_hp: number;
  hp: number;
  max_mp: number;
  mp: number;
  str: number;
  intel: number;
  dex: number;
  con: number;
  last_city: number;
  save_x: number;
  save_y: number;
  learned_skill: number;
  skill_bar: number[];
  special: number[];
  equip: CharacterItem[];
  carry: CharacterItem[];
  /**
   * índice da skill → nome do `SkillData.csv`, só para as skills aprendidas e a
   * barra. Vazio quando o web-api subiu sem `-content`.
   */
  skill_names?: Record<string, string>;
  /** código de efeito → rótulo legível ("Dano", "Refino"…). */
  add_labels?: Record<string, string>;
}

/** Índices de classe do cliente (Basedef.h): 0 TK, 1 FM, 2 BM, 3 HT. */
export const CLASS_LABELS = ["TK", "FM", "BM", "HT"] as const;

export function classLabel(classIndex: number): string {
  return CLASS_LABELS[classIndex] ?? `#${classIndex}`;
}

/** As quatro cidades do servidor, na ordem de `world.cities`. */
export const CITY_LABELS = ["Armia", "Azran", "Erion", "Nippleheim", "Noatum"] as const;

export function cityLabel(lastCity: number): string {
  return CITY_LABELS[lastCity] ?? `#${lastCity}`;
}

/**
 * Rótulo de um item: o nome do catálogo é o que a pessoa reconhece; o índice
 * fica como fallback (`#1234`) só quando o catálogo não tem a linha.
 */
export function itemLabel(item: { item_index: number; name?: string }): string {
  return item.name && item.name.length > 0 ? item.name : `#${item.item_index}`;
}

/**
 * Rótulo de um add: o nome do vocabulário (ItemEffect.h) quando o web-api
 * conhece o código; senão `efeito N`, em vez de mentir sobre o significado.
 */
export function addLabel(labels: Record<string, string> | undefined, code: number): string {
  return labels?.[String(code)] ?? `efeito ${code}`;
}

/** `STRUCT_ITEM` mostra 3 pares efeito/valor; zero significa par vazio. */
export function itemEffects(item: CharacterItem): { eff: number; value: number }[] {
  const pairs = [
    { eff: item.eff1, value: item.effv1 },
    { eff: item.eff2, value: item.effv2 },
    { eff: item.eff3, value: item.effv3 },
  ];
  return pairs.filter((pair) => pair.eff !== 0);
}

/**
 * Rótulo de uma skill: nome do `SkillData.csv` quando existe; senão o índice.
 * `255` é o sentinela de slot vazio na barra.
 */
export function skillLabel(names: Record<string, string> | undefined, index: number): string {
  if (index === 255) return "vazio";
  return names?.[String(index)] ?? `skill ${index}`;
}

/** Skills aprendidas são um bitmask de 32 bits (`learned_skill`). */
export function learnedSkills(mask: number): number[] {
  const out: number[] = [];
  for (let bit = 0; bit < 32; bit += 1) {
    if ((mask & (1 << bit)) !== 0) out.push(bit);
  }
  return out;
}
