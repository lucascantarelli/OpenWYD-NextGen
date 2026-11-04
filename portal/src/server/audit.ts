/**
 * Auditoria das ações admin do portal.
 *
 * Uma linha JSON por ação: conta, papel, ação, alvo, resultado e status. **Sem
 * PII além do id** — nunca senha, hash, PIN ou conteúdo inteiro de item.
 *
 * Sempre vai para o log estruturado do processo (`console.info`), que o Docker
 * coleta. Com `AUDIT_LOG_DIR` definido, também grava
 * `<dir>/audit-YYYY-MM-DD.ndjson` (append, fire-and-forget: auditoria não pode
 * derrubar a ação do moderador).
 */
import { appendFile, mkdir } from "node:fs/promises";
import path from "node:path";

import type { Role } from "./session";

export interface AuditEntry {
  accountId: number;
  role: Role;
  action: string;
  target?: string;
  result: number;
  status: number;
}

export function logAudit(entry: AuditEntry): void {
  const now = new Date();
  const line = JSON.stringify({ ts: now.toISOString(), ...entry });
  console.info(`[audit] ${line}`);

  const dir = process.env.AUDIT_LOG_DIR;
  if (!dir) return;

  const file = path.join(dir, `audit-${now.toISOString().slice(0, 10)}.ndjson`);
  void mkdir(dir, { recursive: true })
    .then(() => appendFile(file, `${line}\n`, "utf8"))
    .catch((err: unknown) => {
      console.error("[audit] falha ao gravar arquivo:", err instanceof Error ? err.message : err);
    });
}
