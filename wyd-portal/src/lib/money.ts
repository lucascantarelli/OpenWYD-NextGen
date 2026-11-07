/**
 * Dinheiro em **centavos inteiros** (issue 31).
 *
 * Todo valor monetário do faturamento chega do web-api como `*_cents` int64
 * (BRL). Nada aqui divide por 100 para depois formatar: a parte inteira e os
 * centavos são separados por aritmética inteira, então `R$ 1.234.567,89` nunca
 * vira `1234567.8899999999`. Donate é outra unidade e não passa por este módulo.
 */

const INT = new Intl.NumberFormat("pt-BR", { maximumFractionDigits: 0 });

/** `R$ 1.234,56` — sinal antes do símbolo, centavos sempre com dois dígitos. */
export function fmtBRL(cents: number): string {
  const safe = Number.isFinite(cents) ? Math.round(cents) : 0;
  const sign = safe < 0 ? "-" : "";
  return `${sign}R$ ${INT.format(Math.trunc(Math.abs(safe) / 100))},${centsToDecimal(safe).slice(-2)}`;
}

/**
 * `1234,56` — valor decimal **sem** `R$`, para CSV e planilha em pt-BR.
 * Negativo preserva o sinal.
 */
export function centsToDecimal(cents: number): string {
  const safe = Number.isFinite(cents) ? Math.round(cents) : 0;
  const abs = Math.abs(safe);
  const reais = Math.trunc(abs / 100);
  const rest = abs % 100;
  return `${safe < 0 ? "-" : ""}${reais},${String(rest).padStart(2, "0")}`;
}

/** Inteiro com separador de milhar (contagens, créditos, pedidos). */
export function fmtCount(value: number): string {
  const safe = Number.isFinite(value) ? Math.trunc(value) : 0;
  return INT.format(safe);
}

/** Ticket médio em centavos; zero pedidos devolve zero, nunca NaN. */
export function averageCents(totalCents: number, count: number): number {
  if (!count) return 0;
  return Math.round(totalCents / count);
}

/** Participação em %, uma casa decimal: `24,7`. */
export function fmtPercent(part: number, total: number): string {
  if (!total) return "0%";
  const value = Math.round((part / total) * 1000) / 10;
  return `${value.toFixed(1).replace(".", ",").replace(",0", "")}%`;
}
