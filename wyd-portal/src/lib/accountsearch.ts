/**
 * Busca de contas para os `Combobox` do portal (issues 29 e 31).
 *
 * Substitui o campo "id numérico da conta" do crédito manual: o operador digita
 * o login e escolhe a conta, sem precisar consultar o banco. Estável por módulo,
 * como `itemsearch` — senão o efeito do `Combobox` reinicia a cada render.
 */
import type { ComboOption } from "@/components/Combobox";
import { fmtCount } from "@/lib/money";
import { roleLabel } from "@/lib/revenue";

export async function loadAccountOptions(query: string): Promise<ComboOption[]> {
  const res = await fetch(`/api/search/accounts?q=${encodeURIComponent(query)}`, {
    cache: "no-store",
  });
  if (res.status === 401 || res.status === 403) throw new Error("sessão sem permissão");
  if (!res.ok) throw new Error("contas indisponíveis");

  const payload = (await res.json()) as { options?: ComboOption[] };
  return payload.options ?? [];
}

/** Rótulo + resumo de uma conta, usado nas opções e no filtro aplicado. */
export function accountOptionHint(account: {
  id: number;
  donate_balance: number;
  role: string;
  is_blocked: boolean;
}): string {
  const parts = [
    `#${account.id}`,
    `${fmtCount(account.donate_balance)} donate`,
    roleLabel(account.role),
  ];
  if (account.is_blocked) parts.push("bloqueada");
  return parts.join(" · ");
}

export function accountOptionLabel(account: { id: number; name: string; email: string }): string {
  const name = account.name || `conta #${account.id}`;
  return account.email ? `${name} · ${account.email}` : name;
}
