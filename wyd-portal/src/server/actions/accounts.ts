"use server";

/**
 * Escritas de conta (issue 32): papel, bloqueio e criação.
 *
 * Três invariantes:
 *
 * 1. tudo passa por `adminCall` (sessão de staff, `moderator_id` da sessão,
 *    deadline, auditoria do BFF);
 * 2. o motivo é validado aqui **e** no web-api — ele é o registro que explica a
 *    ação depois;
 * 3. a recusa de negócio (`última conta admin`, `não pode a si mesmo`) volta
 *    com a mensagem que o web-api escreveu, não com o genérico "dados inválidos".
 */
import { revalidatePath } from "next/cache";
import { z } from "zod";

import {
  emailError,
  isAccountRole,
  MAX_NAME,
  MIN_NAME,
  MIN_PASSWORD,
  MIN_REASON,
} from "@/lib/account";

import { adminCall } from "@/server/admin";
import { AdminError, isInfraFailure } from "@/server/errors";
import {
  accountAdmin,
  type AccountWriteResponse,
  type AdminCreateAccountResponse,
} from "@/server/admin-client";

export type AccountActionResult =
  | { ok: true; message?: string }
  | { ok: false; error: string; status: number };

const reasonField = z
  .string()
  .trim()
  .min(MIN_REASON, `informe o motivo (mínimo ${MIN_REASON} caracteres)`)
  .max(200, "motivo longo demais (máx. 200)"); // a auditoria é texto curto

const roleSchema = z.enum(["player", "moderator", "admin"]);

const setRoleSchema = z.object({
  accountId: z.number().int().positive(),
  role: roleSchema,
  reason: reasonField,
});

const setBlockedSchema = z.object({
  accountId: z.number().int().positive(),
  blocked: z.boolean(),
  reason: reasonField,
});

const createSchema = z.object({
  name: z
    .string()
    .trim()
    .toLowerCase()
    .min(MIN_NAME)
    .max(MAX_NAME),
  password: z.string().min(MIN_PASSWORD, `senha curta demais (mínimo ${MIN_PASSWORD})`),
  email: z.string().trim(),
  role: roleSchema,
  reason: reasonField,
});

export type SetRoleInput = z.infer<typeof setRoleSchema>;
export type SetBlockedInput = z.infer<typeof setBlockedSchema>;
export type CreateAccountInput = z.infer<typeof createSchema>;

/** Troca o papel de uma conta (admin). */
export async function changeAccountRole(input: SetRoleInput): Promise<AccountActionResult> {
  const parsed = setRoleSchema.safeParse(input);
  if (!parsed.success) {
    return { ok: false, status: 422, error: parsed.error.issues[0]?.message ?? "dados inválidos" };
  }

  try {
    const res = await adminCall<
      { account_id: number; role: string; reason: string },
      AccountWriteResponse
    >(
      accountAdmin.setRole.bind(accountAdmin),
      {
        account_id: parsed.data.accountId,
        role: parsed.data.role,
        reason: parsed.data.reason,
      },
      { action: "account.set_role", target: `account:${parsed.data.accountId}` },
    );
    revalidatePath("/contas");
    revalidatePath(`/contas/${parsed.data.accountId}`);
    return { ok: true, message: res.message || "papel atualizado" };
  } catch (error) {
    return failure("account.set_role", error);
  }
}

/** Bloqueia ou desbloqueia uma conta (admin ou moderador, alvo jogador). */
export async function changeAccountBlocked(
  input: SetBlockedInput,
): Promise<AccountActionResult> {
  const parsed = setBlockedSchema.safeParse(input);
  if (!parsed.success) {
    return { ok: false, status: 422, error: parsed.error.issues[0]?.message ?? "dados inválidos" };
  }

  try {
    await adminCall<
      { account_id: number; blocked: boolean; reason: string },
      AccountWriteResponse
    >(
      accountAdmin.setBlocked.bind(accountAdmin),
      {
        account_id: parsed.data.accountId,
        blocked: parsed.data.blocked,
        reason: parsed.data.reason,
      },
      {
        action: parsed.data.blocked ? "account.block" : "account.unblock",
        target: `account:${parsed.data.accountId}`,
      },
    );
    revalidatePath("/contas");
    revalidatePath(`/contas/${parsed.data.accountId}`);
    return { ok: true, message: parsed.data.blocked ? "conta bloqueada" : "conta desbloqueada" };
  } catch (error) {
    return failure(parsed.data.blocked ? "account.block" : "account.unblock", error);
  }
}

/** Cria uma conta com papel explícito (admin). */
export async function createAccount(input: CreateAccountInput): Promise<AccountActionResult> {
  const parsed = createSchema.safeParse(input);
  if (!parsed.success) {
    return { ok: false, status: 422, error: parsed.error.issues[0]?.message ?? "dados inválidos" };
  }
  if (!isAccountRole(parsed.data.role)) {
    return { ok: false, status: 422, error: "papel inválido" };
  }
  const email = emailError(parsed.data.email);
  if (email) return { ok: false, status: 422, error: email };

  try {
    const res = await adminCall<
      { name: string; password: string; email: string; role: string; reason: string },
      AdminCreateAccountResponse
    >(
      accountAdmin.createAccount.bind(accountAdmin),
      {
        name: parsed.data.name,
        password: parsed.data.password,
        email: parsed.data.email,
        role: parsed.data.role,
        reason: parsed.data.reason,
      },
      { action: "account.create", target: `account:${parsed.data.name}` },
    );
    revalidatePath("/contas");
    return {
      ok: true,
      message: `conta ${parsed.data.name} criada (id ${res.account_id})`,
    };
  } catch (error) {
    return failure("account.create", error);
  }
}

/**
 * Traduz a falha para a action. `AdminError` já traz a mensagem certa — inclusive
 * a recusa de negócio que o `adminCall` repassa do corpo da resposta.
 */
function failure(action: string, error: unknown): AccountActionResult {
  if (error instanceof AdminError) {
    return { ok: false, error: error.message, status: error.status };
  }
  if (isInfraFailure(error)) {
    return { ok: false, error: "serviço de administração indisponível", status: 502 };
  }
  console.error(`[account] ${action} falhou:`, error instanceof Error ? error.message : String(error));
  return { ok: false, error: "erro interno do portal", status: 500 };
}
