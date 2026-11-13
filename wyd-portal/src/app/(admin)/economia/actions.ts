"use server";

/**
 * Débito de saldo de donate pela tela de Economia (WEB-06).
 *
 * A ação aceita login (ou id) da conta, resolve o id no gateway e chama o débito
 * com uma chave nova por envio — repetir o clique com a mesma chave devolve o
 * mesmo razão, então nunca debita duas vezes pela mesma intenção.
 */
import { revalidatePath } from "next/cache";

import { debitDonateBalance, findAccountByLogin } from "@/server/economy";
import { AdminError } from "@/server/errors";

export type DebitDonateState =
  | { status: "idle" }
  | { status: "ok"; message: string; debitId: number; replay: boolean }
  | { status: "error"; message: string };

const MAX_CREDITS = 1_000_000;

export async function debitDonateAction(
  _prev: DebitDonateState,
  formData: FormData,
): Promise<DebitDonateState> {
  const conta = String(formData.get("conta") ?? "").trim();
  const credits = Number(formData.get("credits"));

  if (conta.length === 0) {
    return { status: "error", message: "informe o login ou o id da conta" };
  }
  if (!Number.isInteger(credits) || credits <= 0 || credits > MAX_CREDITS) {
    return { status: "error", message: `informe créditos entre 1 e ${MAX_CREDITS}` };
  }

  try {
    const accountId = /^\d+$/.test(conta)
      ? Number(conta)
      : await findAccountByLogin(conta);
    if (accountId === null) {
      return {
        status: "error",
        message: "conta não encontrada — confira o login ou use o id",
      };
    }
    const receipt = await debitDonateBalance(
      accountId,
      credits,
      `portal-debit-${accountId}-${crypto.randomUUID()}`,
    );
    revalidatePath("/economia");
    return {
      status: "ok",
      debitId: receipt.debitId,
      replay: receipt.replay,
      message: receipt.replay
        ? `recibo #${receipt.debitId} já existia — nada foi debitado de novo`
        : `${credits.toLocaleString("pt-BR")} créditos debitados da conta #${accountId} (recibo #${receipt.debitId})`,
    };
  } catch (error) {
    if (error instanceof AdminError) return { status: "error", message: error.message };
    return { status: "error", message: "serviço de administração indisponível" };
  }
}
