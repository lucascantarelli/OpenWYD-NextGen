/**
 * Dobra de busca: mesma semântica de `content_fold()` no Postgres (0007).
 *
 * O banco compara com `lower + translate + regexp` porque `unaccent` é STABLE e não
 * entra em índice. Aqui só precisamos da mesma **decisão**: `Poção`, `POCAO` e
 * `pocao` acham o mesmo item. Manter as duas implementações com o mesmo resultado
 * é o que evita "no painel acha, no site não acha".
 */
export function fold(value: string): string {
  return value
    .normalize("NFD")
    // Marcas combinantes: ç vira c + cedilha, ã vira a + til.
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/æ/g, "ae")
    .replace(/œ/g, "oe")
    .replace(/ß/g, "ss")
    .replace(/[^a-z0-9]+/g, " ")
    .trim();
}

/** `true` quando todos os termos digitados aparecem, em qualquer ordem. */
export function foldMatch(text: string, query: string): boolean {
  const target = fold(text);
  const terms = fold(query).split(" ").filter(Boolean);
  return terms.every((term) => target.includes(term));
}
