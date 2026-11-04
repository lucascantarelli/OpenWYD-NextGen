/**
 * Tipos e agrupamentos das taxas de conteúdo (issue 36).
 *
 * A identidade de uma célula é `(file, family, name)` — `name` é o índice do
 * refino, a chave do `CompRate.txt` (`Item_+7`) ou `"<tier>.<coluna>"` nas quests.
 * Usar posição numérica faria todas as chaves de uma família colidirem.
 */

export interface Rate {
  file: string;
  family: string;
  name: string;
  /** Valor efetivo: arquivo + overlay. */
  value: number;
  /** Valor que o arquivo declara. */
  original: number;
  min_value: number;
  max_value: number;
  label: string;
  has_override: boolean;
}

/**
 * Flag de processo como o tmServer a registrou no boot (issue 36, aba Flags).
 * Read-only: nenhuma delas é recarregada em runtime.
 */
export interface ServerFlag {
  name: string;
  value: string;
  restart_required: boolean;
  hint: string;
}

/** Tabelas editáveis, na ordem em que aparecem na tela. */
export const RATE_FILES = [
  {
    file: "SancRate",
    title: "Refino",
    hint: "Sucesso por âmbar (PO/PL/Âmago) e índice do refino. Índices que o arquivo não lista mantêm o padrão compilado.",
  },
  {
    file: "CompRate",
    title: "Combine",
    hint: "Taxas por família e chave das receitas (Ehre, Compositor, Odin, mercadores).",
  },
  {
    file: "QuestsRate",
    title: "Quests",
    hint: "EXP e moedas por tier e as janelas de nível mortal/arch.",
  },
] as const;

export type RateFile = (typeof RATE_FILES)[number]["file"];

export function fileTitle(file: string): string {
  return RATE_FILES.find((entry) => entry.file === file)?.title ?? file;
}

/** Agrupa por família preservando a ordem que veio do web-api. */
export function groupByFamily(rates: readonly Rate[]): Map<string, Rate[]> {
  const map = new Map<string, Rate[]>();
  for (const rate of rates) {
    const list = map.get(rate.family) ?? [];
    list.push(rate);
    map.set(rate.family, list);
  }
  return map;
}

export function overriddenCount(rates: readonly Rate[]): number {
  return rates.filter((rate) => rate.has_override).length;
}
