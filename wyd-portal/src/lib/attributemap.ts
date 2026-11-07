/**
 * Domínio do editor de `AttributeMap.dat` (issue 27).
 *
 * O arquivo tem 1024² bytes e cada célula cobre um bloco 4×4 do mundo 4096². A
 * confusão entre **coordenada de mundo** e **célula** é o erro caro desta tela:
 * o retângulo é sempre digitado em mundo e a UI mostra a célula resultante ao
 * lado, sem esconder a conversão.
 *
 * As regras de operação e de filtro espelham `webserver/internal/attributemap`
 * (`apply`, `matches`, `transform`). O cálculo de verdade é do servidor; aqui só
 * validamos cedo o que o servidor recusaria com `INVALID`.
 */

export const MAP_DIM = 1024;
export const WORLD_SCALE = 4;
export const WORLD_MAX = MAP_DIM * WORLD_SCALE - 1; // 4095
export const MAP_BYTES = MAP_DIM * MAP_DIM;

export interface AttributeMapValueCount {
  value: number;
  count: number;
}

export interface AttributeMapMeaning {
  value: number;
  name: string;
  description: string;
  bit: boolean;
}

export interface AttributeMapInfo {
  dim: number;
  world_scale: number;
  sha256: string;
  histogram: AttributeMapValueCount[];
  meanings: AttributeMapMeaning[];
}

export interface AttributeMapRect {
  min_x: number;
  min_y: number;
  max_x: number;
  max_y: number;
}

export interface AttributeMapFilter {
  enabled: boolean;
  exact_value: number;
  mask: number;
  match_value: number;
}

export interface TransformParams {
  operation: number;
  operand: number;
  rect: AttributeMapRect | null;
  filter: AttributeMapFilter | null;
}

export interface TransformSummary {
  changed_count: number;
  before_histogram: AttributeMapValueCount[];
  after_histogram: AttributeMapValueCount[];
  original_sha256: string;
  new_sha256: string;
  filename: string;
}

export interface AttributeOperation {
  value: number;
  label: string;
  /** Só as operações de bit/valor usam `operand`; a legada ignora. */
  usesOperand: boolean;
  hint: string;
  /** Dica do que muda, em texto de operador — nunca o nome do enum. */
  effect: string;
}

/** Espelho das operações do proto, na ordem em que o editor legado as oferecia. */
export const ATTRIBUTE_OPERATIONS: readonly AttributeOperation[] = [
  {
    value: 1,
    label: "Marcar PvP e perda de EXP (regra legada)",
    usesOperand: false,
    hint: "aplica `value | 64` em toda célula com valor < 64, exceto valor 1 (cidade)",
    effect: "áreas PvE viram área com flag de PvP/perda de EXP; cidades ficam intactas",
  },
  {
    value: 2,
    label: "Definir valor",
    usesOperand: true,
    hint: "substitui o byte da célula pelo valor informado (0–255)",
    effect: "apaga os outros bits da célula",
  },
  {
    value: 3,
    label: "Ligar bits",
    usesOperand: true,
    hint: "aplica `value | operando` (operando 1–255)",
    effect: "soma a flag sem mexer nas que já existem",
  },
  {
    value: 4,
    label: "Desligar bits",
    usesOperand: true,
    hint: "aplica `value & ~operando` (operando 1–255)",
    effect: "remove a flag e preserva as demais",
  },
  {
    value: 5,
    label: "Alternar bits",
    usesOperand: true,
    hint: "aplica `value ^ operando` (operando 1–255)",
    effect: "inverte a flag: quem tinha perde, quem não tinha ganha",
  },
];

export function operationOf(value: number): AttributeOperation {
  return ATTRIBUTE_OPERATIONS.find((op) => op.value === value) ?? ATTRIBUTE_OPERATIONS[0];
}

/** Mundo → célula, arredondando para baixo (o servidor faz `x/4` inteiro). */
export function worldToCell(world: number): number {
  return Math.trunc(world / WORLD_SCALE);
}

/** "mundo 100,100–199,199 → células 25,25–49,49" */
export function describeRect(rect: AttributeMapRect | null): string {
  if (!rect) return "mapa inteiro (1024 × 1024 células)";
  const x1 = worldToCell(rect.min_x);
  const y1 = worldToCell(rect.min_y);
  const x2 = worldToCell(rect.max_x);
  const y2 = worldToCell(rect.max_y);
  return `mundo ${rect.min_x},${rect.min_y}–${rect.max_x},${rect.max_y} → células ${x1},${y1}–${x2},${y2}`;
}

export function rectError(rect: AttributeMapRect | null): string | null {
  if (!rect) return null;
  for (const [label, value] of [
    ["x mínimo", rect.min_x],
    ["y mínimo", rect.min_y],
    ["x máximo", rect.max_x],
    ["y máximo", rect.max_y],
  ] as const) {
    if (!Number.isInteger(value) || value < 0 || value > WORLD_MAX) {
      return `${label} precisa ser inteiro entre 0 e ${WORLD_MAX}`;
    }
  }
  if (rect.min_x > rect.max_x) return "x mínimo não pode ser maior que o x máximo";
  if (rect.min_y > rect.max_y) return "y mínimo não pode ser maior que o y máximo";
  return null;
}

export function filterError(filter: AttributeMapFilter | null): string | null {
  if (!filter?.enabled) return null;
  const fields: [string, number][] = [
    ["valor exato", filter.exact_value],
    ["máscara", filter.mask],
    ["valor de comparação", filter.match_value],
  ];
  for (const [label, value] of fields) {
    if (!Number.isInteger(value) || value < 0 || value > 255) {
      return `${label} precisa ser inteiro entre 0 e 255`;
    }
  }
  return null;
}

export function transformError(params: TransformParams): string | null {
  const op = ATTRIBUTE_OPERATIONS.find((candidate) => candidate.value === params.operation);
  if (!op) return "escolha uma operação";
  if (op.usesOperand) {
    if (!Number.isInteger(params.operand) || params.operand < 0 || params.operand > 255) {
      return "o operando precisa ser inteiro entre 0 e 255";
    }
    // Definir valor 0 é legítimo (apagar a célula); nas operações de bit, 0 é no-op.
    if (params.operand === 0 && params.operation !== 2) {
      return "operando 0 não altera nada: escolha um bit entre 1 e 255";
    }
  }
  return rectError(params.rect) ?? filterError(params.filter);
}

/** Chave estável dos parâmetros: o download só habilita para o que foi pré-visualizado. */
export function transformKey(params: TransformParams): string {
  return JSON.stringify([
    params.operation,
    params.operand,
    params.rect ? [params.rect.min_x, params.rect.min_y, params.rect.max_x, params.rect.max_y] : null,
    params.filter?.enabled
      ? [params.filter.exact_value, params.filter.mask, params.filter.match_value]
      : null,
  ]);
}

export function transformQuery(params: TransformParams): string {
  const query = new URLSearchParams({
    operation: String(params.operation),
    operand: String(params.operand),
  });
  if (params.rect) {
    query.set("min_x", String(params.rect.min_x));
    query.set("min_y", String(params.rect.min_y));
    query.set("max_x", String(params.rect.max_x));
    query.set("max_y", String(params.rect.max_y));
  }
  if (params.filter?.enabled) {
    query.set("filter", "1");
    query.set("exact_value", String(params.filter.exact_value));
    query.set("mask", String(params.filter.mask));
    query.set("match_value", String(params.filter.match_value));
  }
  return query.toString();
}

function intParam(query: URLSearchParams, name: string): number {
  const raw = query.get(name);
  if (raw === null || raw.trim() === "") return 0;
  const value = Number(raw);
  return Number.isFinite(value) ? Math.trunc(value) : Number.NaN;
}

/** Lê os parâmetros do download; ausência de retângulo/filtro significa "não usar". */
export function parseTransformQuery(query: URLSearchParams): TransformParams {
  const rect =
    query.has("min_x") && query.has("min_y") && query.has("max_x") && query.has("max_y")
      ? {
          min_x: intParam(query, "min_x"),
          min_y: intParam(query, "min_y"),
          max_x: intParam(query, "max_x"),
          max_y: intParam(query, "max_y"),
        }
      : null;
  const filter =
    query.get("filter") === "1"
      ? {
          enabled: true,
          exact_value: intParam(query, "exact_value"),
          mask: intParam(query, "mask"),
          match_value: intParam(query, "match_value"),
        }
      : null;
  return {
    operation: intParam(query, "operation"),
    operand: intParam(query, "operand"),
    rect,
    filter,
  };
}

/** Os 16 valores mais frequentes; o histograma completo continua disponível. */
export function topValues(histogram: readonly AttributeMapValueCount[], limit = 16): AttributeMapValueCount[] {
  return [...histogram].sort((a, b) => b.count - a.count).slice(0, limit);
}

export function totalCells(histogram: readonly AttributeMapValueCount[]): number {
  return histogram.reduce((sum, bucket) => sum + bucket.count, 0);
}

/** Rótulo de uma célula: "64 · PvP Exp Loss" quando o valor é conhecido. */
export function valueLabel(value: number, meanings: readonly AttributeMapMeaning[]): string {
  const meaning = meanings.find((candidate) => !candidate.bit && candidate.value === value);
  return meaning ? `${value} · ${meaning.name}` : String(value);
}

/** Altura de uma barra em % do maior bucket (mínimo visível para bucket vazio). */
export function barHeight(count: number, max: number): number {
  if (max <= 0) return 0;
  return Math.max(count > 0 ? 2 : 0, Math.round((count / max) * 100));
}

/** SHA curto para caber na tela sem perder a possibilidade de conferir. */
export function shortSha(sha: string): string {
  return sha ? sha.slice(0, 12) : "—";
}
