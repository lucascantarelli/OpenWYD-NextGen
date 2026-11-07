/**
 * Resolução de coordenada → nome de lugar (issue 47).
 *
 * Puro e isomórfico: recebe as caixas que o atlas já publica (cidades com
 * `limit_*` e regiões do `Regions.txt`) e devolve o nome mais específico que
 * contém o ponto. É o que permite `/teleportes` mostrar "Armia → Montanha
 * Gelada" em vez de só `100, 200 → 1400, 900`.
 *
 * Nada aqui lê arquivo nem chama RPC: quem monta as caixas é a página, a partir
 * de `server/atlas.ts`.
 */

/** Caixa de mundo (mesma ordem dos campos do atlas: canto 1 e canto 2). */
export interface ZoneBox {
  name: string;
  x1: number;
  y1: number;
  x2: number;
  y2: number;
}

/** Usado quando nenhuma caixa contém o ponto. */
export const UNKNOWN_ZONE = "fora de região nomeada";

function contains(box: ZoneBox, x: number, y: number): boolean {
  const left = Math.min(box.x1, box.x2);
  const right = Math.max(box.x1, box.x2);
  const top = Math.min(box.y1, box.y2);
  const bottom = Math.max(box.y1, box.y2);
  return x >= left && x <= right && y >= top && y <= bottom;
}

function area(box: ZoneBox): number {
  return Math.abs(box.x2 - box.x1) * Math.abs(box.y2 - box.y1);
}

/**
 * Nome do lugar que contém `(x, y)`, preferindo a **menor** caixa: uma cidade
 * fica dentro de uma região, e o operador quer o nome da cidade. Empate de área
 * é resolvido pelo nome, para a saída não depender da ordem do array.
 */
export function zoneAt(boxes: readonly ZoneBox[], x: number, y: number): string {
  let best: ZoneBox | null = null;
  for (const box of boxes) {
    if (!contains(box, x, y)) continue;
    if (best === null) {
      best = box;
      continue;
    }
    const diff = area(box) - area(best);
    if (diff < 0 || (diff === 0 && box.name.localeCompare(best.name, "pt-BR") < 0)) best = box;
  }
  return best?.name ?? UNKNOWN_ZONE;
}

/** Rótulo curto para tabela: nome do lugar, ou o fallback compartilhado. */
export function zoneLabel(boxes: readonly ZoneBox[], x: number, y: number): string {
  return zoneAt(boxes, x, y);
}
