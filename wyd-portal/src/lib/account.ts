/**
 * Contas, papéis e banimento (issue 32 / `AccountAdminService`).
 *
 * Regras que a tela precisa conhecer **antes** de chamar o web-api, para dar
 * mensagem útil sem um round trip:
 *
 * - papel só muda por `admin`; bloqueio, por `admin` ou `moderator`;
 * - `moderator` só bloqueia conta de `player`;
 * - ninguém altera o próprio papel/bloqueio;
 * - a última conta **admin ativa** não pode ser rebaixada nem bloqueada.
 *
 * A regra de verdade continua no banco (dentro da transação, em
 * `internal/store/account_admin.go`); o que está aqui é só o espelho de UX. A
 * validação de formulário é a mesma dos dois lados, por isso mora em `lib/`.
 *
 * Nada nesta tela reescreve personagem: `account.role` e `account.is_blocked` são
 * lidos no **próximo login**, então a mudança vale a partir da próxima sessão.
 */

export const ACCOUNT_ROLES = ["player", "moderator", "admin"] as const;
export type AccountRole = (typeof ACCOUNT_ROLES)[number];

export function isAccountRole(value: string): value is AccountRole {
  return (ACCOUNT_ROLES as readonly string[]).includes(value);
}

const ROLE_LABEL: Record<string, string> = {
  player: "Jogador",
  moderator: "Moderador",
  admin: "Admin",
};

export function roleLabel(role: string): string {
  return ROLE_LABEL[role] ?? role;
}

export function roleTone(role: string): "neutral" | "accent" | "warn" {
  if (role === "admin") return "warn";
  if (role === "moderator") return "accent";
  return "neutral";
}

/** Uma conta como o web-api devolve (sem hash de senha/PIN — o campo nem existe). */
export interface AdminAccount {
  id: number;
  name: string;
  email: string;
  role: string;
  is_blocked: boolean;
  blocked_reason: string;
  donate_balance: number;
  created_at_unix: number;
  /** 0 = nunca alterado pela web. */
  role_changed_at_unix: number;
  /** account.id do autor da última troca de papel; 0 = desconhecido. */
  role_changed_by: number;
  characters: number;
}

export interface AccountAudit {
  id: number;
  actor_id: number;
  action: string;
  old_value: string;
  new_value: string;
  reason: string;
  created_at_unix: number;
}

export function actionLabel(action: string): string {
  switch (action) {
    case "set_role":
      return "papel alterado";
    case "set_blocked":
      return "bloqueio alterado";
    case "create":
      return "conta criada";
    default:
      return action;
  }
}

/** Filtros da lista; os valores vão como estão no wire (`blocked`: -1/0/1). */
export interface AccountFilters {
  query: string;
  role: string;
  /** -1 = todas, 0 = ativas, 1 = bloqueadas. */
  blocked: number;
  afterId: number;
  limit: number;
}

export const BLOCKED_ANY = -1;
export const BLOCKED_NO = 0;
export const BLOCKED_YES = 1;
export const ACCOUNT_PAGE_SIZE = 50;
export const MIN_REASON = 10;
export const MIN_NAME = 4;
export const MAX_NAME = 12;
export const MIN_PASSWORD = 4;

export function blockedLabel(blocked: number): string {
  if (blocked === BLOCKED_YES) return "bloqueadas";
  if (blocked === BLOCKED_NO) return "ativas";
  return "todas";
}

/** Data/hora em America/Sao_Paulo (o operador e o faturamento usam esse fuso). */
export function fmtUnix(seconds: number): string {
  if (!seconds) return "—";
  return new Intl.DateTimeFormat("pt-BR", {
    timeZone: "America/Sao_Paulo",
    dateStyle: "short",
    timeStyle: "short",
  }).format(new Date(seconds * 1000));
}

/** Espelho de `account.admin` em Go: 4–12 letras/números minúsculos. */
export function nameError(name: string): string | null {
  const canonical = name.trim().toLowerCase();
  if (canonical.length < MIN_NAME || canonical.length > MAX_NAME) {
    return `o login precisa de ${MIN_NAME} a ${MAX_NAME} caracteres`;
  }
  if (!/^[a-z0-9]+$/.test(canonical)) {
    return "o login aceita apenas letras sem acento e números";
  }
  return null;
}

export function passwordError(password: string): string | null {
  return password.length < MIN_PASSWORD ? `senha curta demais (mínimo ${MIN_PASSWORD})` : null;
}

export function emailError(email: string): string | null {
  if (email.trim() === "") return null;
  // Deliberadamente frouxo: o web-api valida com net/mail e é a autoridade.
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim()) ? null : "e-mail inválido";
}

/** O motivo é obrigatório em toda escrita e vai inteiro para a auditoria. */
export function reasonError(reason: string): string | null {
  return reason.trim().length < MIN_REASON
    ? `informe o motivo (mínimo ${MIN_REASON} caracteres) — ele vai para a auditoria`
    : null;
}

export function roleError(role: string): string | null {
  return isAccountRole(role) ? null : "papel inválido";
}
