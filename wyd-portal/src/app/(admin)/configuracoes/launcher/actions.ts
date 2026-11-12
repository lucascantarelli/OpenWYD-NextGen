"use server";

/**
 * Publicação da configuração assinada do launcher.
 *
 * O staff cola o `launcher.conf` **já assinado** por `tools/launcher/sign-config.mjs`.
 * O portal não tem a chave e não assina nada: valida a forma, recusa revisão
 * repetida ou fora de ordem, grava os bytes exatos e registra o histórico com o
 * autor da sessão. Quem autentica é o launcher, com a chave que só ele/operador tem.
 */
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";

import { AdminError } from "@/server/errors";
import { publicarConfiguracao } from "@/server/launcher-dist";
import { requireStaff } from "@/server/session";

const PATH = "/configuracoes/launcher";

export async function publicarConfiguracaoAction(formData: FormData): Promise<void> {
  const texto = String(formData.get("texto") ?? "");
  try {
    const sessao = await requireStaff();
    publicarConfiguracao(texto, `portal#${sessao.accountId}`);
  } catch (erro) {
    const mensagem =
      erro instanceof AdminError
        ? erro.message
        : erro instanceof Error
          ? erro.message
          : "falha ao publicar a configuração";
    redirect(`${PATH}?erro=${encodeURIComponent(mensagem)}`);
  }
  revalidatePath(PATH);
  redirect(`${PATH}?ok=1`);
}
