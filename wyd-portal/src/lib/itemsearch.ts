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

/**
 * Normaliza texto para comparação na busca.
 *
 * O banco guarda os nomes em **NFC** (`Poção` = `c3 a7 c3 a3`), mas a comparação
 * usava string crua: o mesmo termo em **NFD** (macOS, alguns IMEs) devolvia zero
 * resultado para um item que existe. Medido no portal em 05/11/2026:
 * `?q=po%C3%A7%C3%A3o` (NFC) → 20 itens; `?q=po%63%CC%A7a%CC%83o` (NFD) → 0.
 */
export function normalizeSearch(value: string): string {
  return value.normalize("NFC").toLowerCase();
}
