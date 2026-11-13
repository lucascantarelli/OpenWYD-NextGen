/**
 * Trilha de auditoria vista pelo painel (WEB-03).
 *
 * A fonte é `audit_log` no Postgres, lida pelo gateway (`audit_admin/recent`)
 * com `adminCall` — o browser nunca vê o token nem fala com o banco. Leitura
 * exclusiva de admin: a trilha contém ações de todo o staff.
 */
import { adminCall } from "./admin";
import { auditAdmin, type ListAuditResponse } from "./admin-client";

/** Linha pronta para a tela (milissegundos → `Date`). */
export interface AuditEntryView {
  at: Date;
  actorId: number;
  actorRole: string;
  action: string;
  target: string;
  detail: string;
}

/** Teto da rota no gateway (`api::moderation::recent_audit`). */
export const MAX_AUDIT_ROWS = 200;

/** Limite pedido, dentro da faixa que o gateway aceita. */
export function clampAuditLimit(limit: number): number {
  if (!Number.isFinite(limit)) return 50;
  return Math.min(Math.max(1, Math.trunc(limit)), MAX_AUDIT_ROWS);
}

/** Filtros da tela de auditoria (tudo opcional, aplicado sobre a trilha lida). */
export interface AuditFilter {
  actor?: string;
  action?: string;
  target?: string;
  /** Só linhas a partir deste instante (ms desde epoch). */
  sinceMs?: number;
}

function contains(haystack: string, needle: string | undefined): boolean {
  const clean = needle?.trim().toLowerCase();
  if (!clean) return true;
  return haystack.toLowerCase().includes(clean);
}

/**
 * Filtra a trilha na tela (ator, ação, alvo, período).
 *
 * O gateway não tem rota de busca na trilha (`audit_admin/recent` só pagina por
 * limite), então o filtro roda sobre as últimas linhas lidas — a tela avisa o
 * tamanho da janela. Comparação case-insensitive por substring.
 */
export function filterAuditTrail(entries: AuditEntryView[], filter: AuditFilter): AuditEntryView[] {
  return entries.filter(
    (entry) =>
      contains(`#${entry.actorId} ${entry.actorRole}`, filter.actor) &&
      contains(entry.action, filter.action) &&
      contains(entry.target, filter.target) &&
      (filter.sinceMs === undefined || entry.at.getTime() >= filter.sinceMs),
  );
}

/**
 * Histórico de login de uma conta, a partir da mesma trilha de auditoria.
 *
 * O gateway registra `auth.login` com o login tentado como alvo (`panel.rs`);
 * a ficha da conta filtra esse verbo pelo login exato, sem rota nova.
 */
export function loginHistoryFor(entries: AuditEntryView[], login: string): AuditEntryView[] {
  const clean = login.trim().toLowerCase();
  if (clean === "") return [];
  return entries.filter(
    (entry) => entry.action === "auth.login" && entry.target.trim().toLowerCase() === clean,
  );
}

/**
 * Últimas linhas, da mais nova para a mais antiga.
 *
 * Sem trilha montada no gateway a chamada falha com `AdminUnavailableError`:
 * a tela mostra o erro, nunca uma tabela vazia fingindo "nada aconteceu".
 */
export async function getAuditTrail(limit = 50): Promise<AuditEntryView[]> {
  const bounded = clampAuditLimit(limit);
  const res = await adminCall<{ limit: number }, ListAuditResponse>(
    auditAdmin.recent.bind(auditAdmin),
    { limit: bounded },
    { action: "audit.list", target: "audit_log" },
  );
  return (res.entries ?? []).map((row) => ({
    at: new Date(row.at_ms),
    actorId: row.actor_id,
    actorRole: row.actor_role,
    action: row.action,
    target: row.target,
    detail: row.detail,
  }));
}
