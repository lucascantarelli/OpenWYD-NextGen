/** Formatação dos números do jogo (pt-BR) e datas no fuso do servidor. */

const integer = new Intl.NumberFormat("pt-BR", { maximumFractionDigits: 0 });
const decimal = new Intl.NumberFormat("pt-BR", {
  minimumFractionDigits: 1,
  maximumFractionDigits: 1,
});

/** Coin e valores inteiros grandes: `1234567` → `1.234.567`. */
export function fmtCoin(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return "—";
  return integer.format(Math.trunc(value));
}

/**
 * EXP em tabela: sufixo `k`/`M` a partir de 10 mil para caber na coluna.
 * Abaixo disso mantém o número cheio, que é como o jogador lê no cliente.
 */
export function fmtExp(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return "—";
  const n = Math.trunc(value);
  const abs = Math.abs(n);
  if (abs >= 1_000_000) return `${decimal.format(n / 1_000_000)}M`;
  if (abs >= 10_000) return `${decimal.format(n / 1_000)}k`;
  return integer.format(n);
}

/** Percentual já em pontos percentuais (ex.: 12.5 → "12,5%"). */
export function fmtPercent(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return "—";
  return `${decimal.format(value)}%`;
}

const dateTime = new Intl.DateTimeFormat("pt-BR", {
  timeZone: "America/Sao_Paulo",
  dateStyle: "short",
  timeStyle: "short",
});

/** ISO → `03/11/2026 09:12` em America/Sao_Paulo. */
export function fmtDate(value: string | Date | null | undefined): string {
  if (!value) return "—";
  const d = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(d.getTime())) return "—";
  return dateTime.format(d);
}
