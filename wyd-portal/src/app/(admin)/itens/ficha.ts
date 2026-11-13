/**
 * Linhas legíveis da ficha do item.
 *
 * Cada campo vem do template no catálogo (`item_catalog`). `null` significa "o
 * template não declara" — a linha sai da ficha em vez de virar zero inventado.
 */
import type { CatalogItem } from "@/server/admin-client";

export interface FichaLinha {
  rotulo: string;
  valor: string;
}

const fmtInt = (value: number): string => value.toLocaleString("pt-BR");

const simNao = (value: boolean | null | undefined): string | null =>
  value === null || value === undefined ? null : value ? "sim" : "não";

export function atributosDaFicha(item: CatalogItem): FichaLinha[] {
  const linhas: FichaLinha[] = [];
  if (item.price !== undefined) {
    linhas.push({
      rotulo: "Preço de tabela",
      valor: item.price > 0 ? `${fmtInt(item.price)} de ouro` : "sem preço no template",
    });
  }
  if (item.grade !== undefined) {
    linhas.push({ rotulo: "Grade", valor: `grade ${fmtInt(item.grade)}` });
  }
  if (item.attack_range !== undefined) {
    linhas.push({
      rotulo: "Alcance de ataque",
      valor:
        item.attack_range === null
          ? "—"
          : `${fmtInt(item.attack_range)} tile${item.attack_range === 1 ? "" : "s"}`,
    });
  }
  if (item.volatile !== undefined && item.volatile !== null) {
    linhas.push({ rotulo: "Slot volátil", valor: `slot ${fmtInt(item.volatile)}` });
  }
  if (item.kind) linhas.push({ rotulo: "Tipo de item", valor: item.kind });
  if (item.primary_system) linhas.push({ rotulo: "Sistema", valor: item.primary_system });
  const soEvento = simNao(item.event_only);
  if (soEvento) linhas.push({ rotulo: "Só em evento", valor: soEvento });
  const negociavel = simNao(item.tradeable);
  if (negociavel) linhas.push({ rotulo: "Negociável entre jogadores", valor: negociavel });
  const acumulavel = simNao(item.stackable);
  if (acumulavel) linhas.push({ rotulo: "Acumula em pilha", valor: acumulavel });
  if (item.class_mask) linhas.push({ rotulo: "Classes permitidas", valor: item.class_mask });
  return linhas;
}

export function requisitosDaFicha(item: CatalogItem): FichaLinha[] {
  const candidatos: Array<[string, number | undefined]> = [
    ["Nível mínimo", item.req_level],
    ["Força (STR)", item.req_str],
    ["Inteligência (INT)", item.req_int],
    ["Destreza (DEX)", item.req_dex],
    ["Constituição (CON)", item.req_con],
  ];
  return candidatos
    .filter((par): par is [string, number] => typeof par[1] === "number" && par[1] > 0)
    .map(([rotulo, valor]) => ({ rotulo, valor: fmtInt(valor) }));
}
