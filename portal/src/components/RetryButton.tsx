"use client";

import { useRouter } from "next/navigation";
import { useTransition } from "react";

/**
 * Ação de recuperação dos estados de erro (issue 47).
 *
 * Toda tela que falha ao carregar precisa de um caminho de volta: a mensagem
 * sozinha deixa o operador sem saída. `router.refresh()` reexecuta o server
 * component — para as leituras cacheadas por processo no BFF, é também o que
 * traz dados novos depois de um restart de serviço.
 */
export function RetryButton({ label = "tentar de novo" }: { label?: string }) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();

  return (
    <button
      className="ui-btn"
      type="button"
      disabled={pending}
      onClick={() => startTransition(() => router.refresh())}
    >
      {pending ? "recarregando…" : label}
    </button>
  );
}
