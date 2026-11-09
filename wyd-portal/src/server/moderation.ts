/**
 * Trilha de auditoria vista pelo painel (FASE-07-3).
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
