/**
 * Faturamento (issue 31 / `DonateRevenueAdminService`).
 *
 * Duas unidades que **nunca** se somam:
 *
 * - `*_cents` = BRL em centavos, dinheiro reconhecido em `confirmed_at`;
 * - `credits` = unidade da carteira de donate (`account.donate_balance`).
 *
 * `pending_cents` é pedido criado e ainda não pago: aparece na tela como
 * histórico, jamais dentro de `gross_cents`.
 *
 * A janela é o intervalo **semiaberto** `[from_unix, to_unix)` que o web-api
 * aplica de fato; 0/0 viraria "últimos 30 dias", mas o portal sempre manda uma
 * janela concreta para conseguir rotular o período com honestidade. Os buckets
 * fecham em `America/Sao_Paulo` (o Brasil não tem mais horário de verão, então o
 * offset fixo −03:00 não erra), e é por isso que as datas digitadas pelo
 * operador são interpretadas nesse fuso, não no do navegador.
 */
import { centsToDecimal } from "./money";

export const SAO_PAULO_TZ = "America/Sao_Paulo";
/** Offset fixo de Brasília desde o fim do horário de verão (2019). */
export const SAO_PAULO_OFFSET = "-03:00";
export const DAY_SECONDS = 86_400;
/** O web-api recusa janela maior que 366 dias (`INVALID`). */
export const MAX_WINDOW_DAYS = 366;

// --- tipos de wire (espelham web.proto; int64 chega como number) ---

export interface RevenueWindowInput {
  from_unix: number;
  to_unix: number;
}

export interface RevenueTotalsWire {
  paid_orders: number;
  gross_cents: number;
  credits_sold: number;
  distinct_buyers: number;
  created_orders: number;
  pending_orders: number;
  pending_cents: number;
  shop_purchases: number;
  credits_spent: number;
  manual_credits: number;
  credits_granted: number;
}

export interface RevenueByMethodWire {
  payment_method: number;
  paid_orders: number;
  gross_cents: number;
}

export interface RevenuePointWire {
  bucket_start_unix: number;
  paid_orders: number;
  gross_cents: number;
  credits_sold: number;
  distinct_buyers: number;
}

export interface TopupOrderRowWire {
  id: number;
  external_reference: string;
  account_id: number;
  account_name: string;
  account_email: string;
  payer_name: string;
  /** Sempre mascarado (`***.456.789-**`); os 11 dígitos nunca saem do web-api. */
  payer_cpf_masked: string;
  credits: number;
  amount_cents: number;
  payment_method: number;
  /** Vazio hoje: `CreateTopupOrder` não grava `provider`. Não construir UI nisso. */
  provider: string;
  status: number;
  created_at_unix: number;
  /** 0 enquanto PENDENTE. */
  confirmed_at_unix: number;
}

export interface TopBuyerRowWire {
  account_id: number;
  account_name: string;
  account_email: string;
  window_paid_orders: number;
  window_gross_cents: number;
  lifetime_paid_orders: number;
  lifetime_gross_cents: number;
  lifetime_credits: number;
  first_paid_at_unix: number;
  last_paid_at_unix: number;
  /** Saldo de donate agora — ponto no tempo, não um valor da janela. */
  donate_balance: number;
}

export interface DonateLedgerRowWire {
  id: number;
  action: number;
  created_at_unix: number;
  subject_account_id: number;
  subject_account_name: string;
  actor_account_id: number;
  actor_account_name: string;
  /** Assinado: negativo na compra, positivo no crédito manual. */
  credits_delta: number;
  balance_after: number;
  shop_item_id: number;
  shop_item_title: string;
  reason: string;
}

export interface AccountSummaryWire {
  id: number;
  name: string;
  email: string;
  donate_balance: number;
  role: string;
  is_blocked: boolean;
}

export const EMPTY_TOTALS: RevenueTotalsWire = {
  paid_orders: 0,
  gross_cents: 0,
  credits_sold: 0,
  distinct_buyers: 0,
  created_orders: 0,
  pending_orders: 0,
  pending_cents: 0,
  shop_purchases: 0,
  credits_spent: 0,
  manual_credits: 0,
  credits_granted: 0,
};

// --- período ---

export type PeriodId = "7d" | "30d" | "90d" | "custom";

export const PERIODS = [
  { id: "7d", label: "7 dias", days: 7 },
  { id: "30d", label: "30 dias", days: 30 },
  { id: "90d", label: "90 dias", days: 90 },
  { id: "custom", label: "Personalizado", days: 0 },
] as const;

export function isPeriodId(value: string | undefined): value is PeriodId {
  return PERIODS.some((period) => period.id === value);
}

export const BUCKET_OPTIONS = [
  { value: 1, label: "por dia" },
  { value: 2, label: "por semana" },
  { value: 3, label: "por mês" },
  { value: 0, label: "sem série" },
] as const;

export type BucketValue = 0 | 1 | 2 | 3;

/** Granularidade padrão pela largura da janela: dia até 45, semana até 200. */
export function bucketForDays(days: number): BucketValue {
  if (days <= 45) return 1;
  if (days <= 200) return 2;
  return 3;
}

export function isBucketValue(value: number): value is BucketValue {
  return value === 0 || value === 1 || value === 2 || value === 3;
}

export interface ResolvedWindow {
  from: number;
  to: number;
  days: number;
}

export type WindowResult = ({ ok: true } & ResolvedWindow) | { ok: false; error: string };

const DAY_PATTERN = /^\d{4}-\d{2}-\d{2}$/;

/** `2026-11-01` no fuso de Brasília; `null` quando o formato não confere. */
export function parseDay(value: string | undefined): Date | null {
  if (!value || !DAY_PATTERN.test(value)) return null;
  const date = new Date(`${value}T00:00:00${SAO_PAULO_OFFSET}`);
  return Number.isNaN(date.getTime()) ? null : date;
}

/**
 * Data digitada (`AAAA-MM-DD`) para o `<input type="date">`. Usa o fuso de
 * Brasília para que "hoje" no portal seja o mesmo dia do extrato.
 */
export function todayInput(now = new Date()): string {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: SAO_PAULO_TZ,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(now);
}

/**
 * Resolve a janela que vai no request. Presets contam para trás a partir de
 * agora; o modo personalizado inclui o dia final inteiro (`to` é exclusivo, então
 * soma um dia) — senão o operador escolheria "01 a 31" e perderia o dia 31.
 */
export function resolveWindow(
  input: { period: PeriodId; desde?: string; ate?: string },
  now = new Date(),
): WindowResult {
  if (input.period !== "custom") {
    const preset = PERIODS.find((period) => period.id === input.period) ?? PERIODS[1];
    const to = Math.floor(now.getTime() / 1000);
    return { ok: true, from: to - preset.days * DAY_SECONDS, to, days: preset.days };
  }

  const start = parseDay(input.desde);
  if (!start) return { ok: false, error: "informe a data inicial no formato AAAA-MM-DD" };
  const end = parseDay(input.ate);
  if (!end) return { ok: false, error: "informe a data final no formato AAAA-MM-DD" };

  const from = Math.floor(start.getTime() / 1000);
  const to = Math.floor(end.getTime() / 1000) + DAY_SECONDS;
  if (from >= to) return { ok: false, error: "a data inicial precisa ser anterior à final" };

  const days = Math.ceil((to - from) / DAY_SECONDS);
  if (days > MAX_WINDOW_DAYS) {
    return { ok: false, error: `a janela máxima é de ${MAX_WINDOW_DAYS} dias` };
  }
  return { ok: true, from, to, days };
}

// --- rótulos ---

const dayFmt = new Intl.DateTimeFormat("pt-BR", {
  timeZone: SAO_PAULO_TZ,
  day: "2-digit",
  month: "2-digit",
});

const monthFmt = new Intl.DateTimeFormat("pt-BR", {
  timeZone: SAO_PAULO_TZ,
  month: "short",
  year: "2-digit",
});

const momentFmt = new Intl.DateTimeFormat("pt-BR", {
  timeZone: SAO_PAULO_TZ,
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
});

/** `01/11` — eixo do gráfico diário. */
export function fmtDay(unix: number): string {
  return unix > 0 ? dayFmt.format(new Date(unix * 1000)) : "—";
}

/** `nov/26` — eixo do gráfico mensal. */
export function fmtMonth(unix: number): string {
  if (unix <= 0) return "—";
  return monthFmt.format(new Date(unix * 1000)).replace(/\./g, "").replace(/\sde\s/, "/");
}

/** `01/11/2026 14:30` em Brasília; 0 vira `—` (pedido pendente). */
export function fmtMoment(unix: number): string {
  return unix > 0 ? momentFmt.format(new Date(unix * 1000)) : "—";
}

export function fmtBucketLabel(unix: number, bucket: BucketValue): string {
  if (bucket === 2) return `sem. ${fmtDay(unix)}`;
  if (bucket === 3) return fmtMonth(unix);
  return fmtDay(unix);
}

export function bucketNoun(bucket: BucketValue): string {
  if (bucket === 1) return "dia";
  if (bucket === 2) return "semana";
  return "mês";
}

const PAYMENT_METHODS: Record<number, string> = { 1: "PIX", 2: "Cartão" };
const TOPUP_STATUSES: Record<number, string> = { 1: "Pendente", 2: "Pago" };

export function paymentMethodLabel(method: number): string {
  return PAYMENT_METHODS[method] ?? `método ${method}`;
}

export function topupStatusLabel(status: number): string {
  return TOPUP_STATUSES[status] ?? `status ${status}`;
}

export function ledgerActionLabel(action: number): string {
  if (action === 1) return "compra";
  if (action === 2) return "crédito manual";
  return "—";
}

export function roleLabel(role: string): string {
  if (role === "admin") return "admin";
  if (role === "moderator") return "moderador";
  return "jogador";
}

// --- CSV ---

function csvCell(value: string | number): string {
  const text = String(value ?? "");
  return /[";\r\n]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text;
}

/**
 * CSV do pedido (client-side, sem rota nova). Separador `;` e vírgula decimal
 * porque o destino é o Excel em pt-BR; `valor_reais` acompanha
 * `valor_centavos` para ninguém precisar dividir na planilha.
 */
export function ordersCsv(rows: readonly TopupOrderRowWire[]): string {
  const header = [
    "pedido",
    "referencia",
    "conta_id",
    "login",
    "email",
    "pagador",
    "cpf",
    "creditos",
    "valor_centavos",
    "valor_reais",
    "metodo",
    "status",
    "criado_em",
    "pago_em",
  ];
  const lines = rows.map((row) =>
    [
      row.id,
      row.external_reference,
      row.account_id,
      row.account_name,
      row.account_email,
      row.payer_name,
      row.payer_cpf_masked,
      row.credits,
      row.amount_cents,
      centsToDecimal(row.amount_cents),
      paymentMethodLabel(row.payment_method),
      topupStatusLabel(row.status),
      fmtMoment(row.created_at_unix),
      row.confirmed_at_unix ? fmtMoment(row.confirmed_at_unix) : "",
    ]
      .map(csvCell)
      .join(";"),
  );
  return [`\ufeff${header.join(";")}`, ...lines].join("\r\n") + "\r\n";
}
