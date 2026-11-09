/**
 * Ações de moderação do painel de contas (FASE-07-3).
 *
 * Toda escrita passa por `adminCall` (sessão de staff + auditoria) e pelo
 * gateway, que relê o papel da sessão e aplica as travas (auto-lockout, staff
 * só por admin). Aqui só há validação de forma e tradução para a tela.
 */
"use server";

import { revalidatePath } from "next/cache";

import { ROLES, roleLabel, type Role } from "@/lib/role";

import { setAccountBlocked, setAccountRole } from "@/server/accounts";
import { AdminError } from "@/server/errors";

export type ModerationActionResult = { ok: true; message: string } | { ok: false; error: string };

const MAX_REASON = 200;

export async function moderateAccountAction(input: {
  login: string;
  action: "role" | "block" | "unblock";
  role?: Role;
  reason?: string;
}): Promise<ModerationActionResult> {
  const login = input.login.trim();
  if (login === "") return { ok: false, error: "login obrigatório" };

  try {
    if (input.action === "role") {
      if (!input.role || !ROLES.includes(input.role)) {
        return { ok: false, error: "papel inválido" };
      }
      const estado = await setAccountRole(login, input.role);
      revalidatePath("/contas");
      return {
        ok: true,
        message: `${estado.account.login} agora é ${roleLabel(estado.account.role as Role)}`,
      };
    }

    const blocked = input.action === "block";
    const reason = (input.reason ?? "").trim();
    if (blocked && reason === "") return { ok: false, error: "informe o motivo do bloqueio" };
    if (reason.length > MAX_REASON) {
      return { ok: false, error: `o motivo aceita no máximo ${MAX_REASON} caracteres` };
    }
    const estado = await setAccountBlocked({ login, blocked, reason });
    revalidatePath("/contas");
    return {
      ok: true,
      message: blocked
        ? `${estado.account.login} bloqueada — o acesso ao jogo e ao portal é recusado`
        : `${estado.account.login} liberada`,
    };
  } catch (error) {
    if (error instanceof AdminError) return { ok: false, error: error.message };
    return { ok: false, error: "serviço de administração indisponível" };
  }
}
