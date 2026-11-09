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
  type AccountSummary,
  type CreateAccountResponse,
  type FindAccountResponse,
} from "./admin-client";
import { AdminError } from "./errors";
import type { Role } from "@/lib/role";

export type { AccountSummary };

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
