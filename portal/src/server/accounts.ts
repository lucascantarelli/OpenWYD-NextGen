/**
 * Leituras do `AccountAdminService` (issue 32).
 *
 * A página `/contas` e o detalhe só leem por aqui; toda escrita vai pelas
 * actions em `server/actions/accounts.ts`, que também passam pelo `adminCall`.
 * O browser nunca monta `moderator_id` — ele vem da sessão.
 */
import type { AccountAudit, AccountFilters, AdminAccount } from "@/lib/account";

import { adminCall } from "./admin";
import {
  accountAdmin,
  type GetAccountAdminResponse,
  type ListAccountsResponse,
} from "./grpc";

export interface AccountPage {
  accounts: AdminAccount[];
  total: number;
  /** Cursor da próxima página; 0 = acabou. */
  nextId: number;
}

export async function listAccounts(filters: AccountFilters): Promise<AccountPage> {
  const res = await adminCall<
    {
      query: string;
      role: string;
      blocked: number;
      limit: number;
      after_id: number;
    },
    ListAccountsResponse
  >(
    accountAdmin.listAccounts.bind(accountAdmin),
    {
      query: filters.query,
      role: filters.role,
      blocked: filters.blocked,
      limit: filters.limit,
      after_id: filters.afterId,
    },
    {
      action: "account.list",
      target: `query:${filters.query.slice(0, 32) || "*"}`,
    },
  );

  const accounts = res.accounts ?? [];
  return {
    accounts,
    total: res.total_count ?? 0,
    // O web-api devolve o último id da página; uma página incompleta é o fim.
    nextId: accounts.length > 0 && accounts.length === filters.limit ? res.next_id ?? 0 : 0,
  };
}

export async function getAccount(
  accountId: number,
): Promise<{ account: AdminAccount; audit: AccountAudit[] } | null> {
  if (!Number.isInteger(accountId) || accountId <= 0) return null;
  const res = await adminCall<{ account_id: number }, GetAccountAdminResponse>(
    accountAdmin.getAccount.bind(accountAdmin),
    { account_id: accountId },
    { action: "account.get", target: `account:${accountId}` },
  );
  if (!res.account) return null;
  return { account: res.account, audit: res.audit ?? [] };
}
