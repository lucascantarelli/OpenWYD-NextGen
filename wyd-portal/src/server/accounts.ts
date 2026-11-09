/**
 * Contas no control plane: busca exata e criação administrativa.
 *
 * Toda chamada passa por `adminCall` (sessão de staff + auditoria). Não existe
 * listagem paginada no contrato atual: a tela busca por login, não inventa
 * cursor nem total.
 */
import { adminCall } from "./admin";
import {
  accountsAdmin,
  type AccountModerationWire,
  type AccountSummary,
  type CreateAccountResponse,
  type FindAccountResponse,
  type ModerationResponse,
} from "./admin-client";
import { AdminError } from "./errors";
import type { Role } from "@/lib/role";

export type { AccountSummary };

/** Visão de moderação usada pelas telas (snake_case do fio → camelCase). */
export interface AccountModerationView {
  account: AccountSummary;
  blocked: boolean;
  reason: string;
  blockedAt: number | null;
}

function toModerationView(wire: AccountModerationWire): AccountModerationView {
  return {
    account: wire.account,
    blocked: wire.blocked,
    reason: wire.reason,
    blockedAt: wire.blocked_at_ms,
  };
}

/** Estado de moderação por login (`accounts/get_moderation`, staff). */
export async function getAccountModeration(login: string): Promise<AccountModerationView> {
  const clean = login.trim();
  const res = await adminCall<{ login: string }, ModerationResponse>(
    accountsAdmin.getModeration.bind(accountsAdmin),
    { login: clean },
    { action: "accounts.moderation", target: `login:${clean.slice(0, 16)}` },
  );
  if (!res.moderation) throw new Error("resposta sem o estado de moderação");
  return toModerationView(res.moderation);
}

/** Troca de papel (`accounts/set_role`, exclusivo de admin). */
export async function setAccountRole(login: string, role: Role): Promise<AccountModerationView> {
  const res = await adminCall<{ login: string; role: string }, ModerationResponse>(
    accountsAdmin.setRole.bind(accountsAdmin),
    { login: login.trim(), role },
    { action: "accounts.set_role", target: `login:${login.trim().slice(0, 16)}` },
  );
  if (!res.moderation) throw new Error("resposta sem o estado de moderação");
  return toModerationView(res.moderation);
}

/** Bloqueio/liberação (`accounts/set_blocked`); bloqueio exige motivo. */
export async function setAccountBlocked(input: {
  login: string;
  blocked: boolean;
  reason: string;
}): Promise<AccountModerationView> {
  const res = await adminCall<
    { login: string; blocked: boolean; reason: string },
    ModerationResponse
  >(
    accountsAdmin.setBlocked.bind(accountsAdmin),
    {
      login: input.login.trim(),
      blocked: input.blocked,
      reason: input.reason.trim(),
    },
    {
      action: input.blocked ? "accounts.block" : "accounts.unblock",
      target: `login:${input.login.trim().slice(0, 16)}`,
    },
  );
  if (!res.moderation) throw new Error("resposta sem o estado de moderação");
  return toModerationView(res.moderation);
}

/** Busca por login exato. `null` quando a conta não existe (404 do gateway). */
export async function findAccount(login: string): Promise<AccountSummary | null> {
  const clean = login.trim();
  if (clean === "") return null;
  try {
    const res = await adminCall<{ login: string }, FindAccountResponse>(
      accountsAdmin.find.bind(accountsAdmin),
      { login: clean },
      { action: "accounts.find", target: `login:${clean.slice(0, 16)}` },
    );
    return res.account ?? null;
  } catch (error) {
    // 404 do contrato = conta inexistente, não falha de infraestrutura.
    if (error instanceof AdminError && error.status === 404) return null;
    throw error;
  }
}

/** Cria a conta com papel explícito. O `moderator_id` vem da sessão. */
export async function createAccount(input: {
  login: string;
  role: Role;
}): Promise<AccountSummary> {
  const res = await adminCall<{ login: string; role: string }, CreateAccountResponse>(
    accountsAdmin.create.bind(accountsAdmin),
    { login: input.login, role: input.role },
    { action: "accounts.create", target: `login:${input.login.slice(0, 16)}` },
  );
  if (!res.account) throw new Error("resposta sem a conta criada");
  return res.account;
}
