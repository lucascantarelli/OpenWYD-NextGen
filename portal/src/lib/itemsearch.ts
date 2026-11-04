/**
 * Busca de itens para os `Combobox` do portal.
 *
 * Módulo próprio porque `load` precisa ser **estável** entre renders: definido
 * dentro do componente, ele reinicia o efeito a cada tecla e a lista pisca. Um
 * único lugar também mantém a rota `/api/search/items` (com sessão) como a
 * única porta para o catálogo.
 */
import type { ComboOption } from "@/components/Combobox";

export async function loadItemOptions(query: string): Promise<ComboOption[]> {
  const res = await fetch(`/api/search/items?q=${encodeURIComponent(query)}`, {
    cache: "no-store",
  });
  if (!res.ok) throw new Error("catálogo indisponível");
  const payload = (await res.json()) as { options?: ComboOption[] };
  return payload.options ?? [];
}
