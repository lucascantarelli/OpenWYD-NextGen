/**
 * Vocabulário e cálculo da taxa de drop por slot (issue 34).
 *
 * `effectiveDropRate` é espelho de `tmserver/internal/loot/loot.go`
 * (`EffectiveDropRate`): mesmas divisões inteiras, mesmos cortes por nível e os
 * mesmos overrides duros dos slots 8–11. O teste do portal fixa os casos que
 * batem com o Go — se a fórmula mudar de um lado, o outro aparece vermelho.
 *
 * A tabela base entra por parâmetro (e não por estado de módulo) porque o
 * componente client e o server component têm instâncias diferentes do módulo: um
 * `setBaseTable` no servidor não chegaria ao browser.
 *
 * O número é **divisor**, não porcentagem: a tela mostra "≈ 1 em N", nunca "x%".
 */
import type { Tone } from "@/lib/item";

export interface DropRateSlot {
  slot: number;
  base_value: number;
  current_value: number;
  overridden: boolean;
}

export const DROP_RATE_MIN = 1;
export const DROP_RATE_MAX = 1_000_000;

export interface SlotGroup {
  from: number;
  to: number;
  /** Nome do grupo — é o rótulo principal da linha, o slot é secundário. */
  label: string;
  note: string;
  tone: Tone | "neutral";
}

/**
 * As faixas são as do `DropRate` real (Basedef.cpp:222 via loot.go). O agrupamento
 * diz o *papel* do slot — "equipamento", "ouro e poção", "ultra raro" —, porque o
 * número sozinho não significa nada para quem opera.
 */
export const SLOT_GROUPS: readonly SlotGroup[] = [
  { from: 0, to: 7, label: "Equipamento comum", note: "armas e armaduras de nível baixo", tone: "accent" },
  {
    from: 8,
    to: 11,
    label: "Ouro e poção",
    note: "fixo no código: EffectiveDropRate força 4/4/4/1 e ignora a taxa",
    tone: "warn",
  },
  {
    from: 12,
    to: 15,
    label: "Comum",
    note: "mesma faixa do equipamento comum, tabela separada no legado",
    tone: "accent",
  },
  { from: 16, to: 23, label: "Ultra raro", note: "reservado a drops de chefe e itens de evento", tone: "danger" },
  { from: 24, to: 47, label: "Incomum", note: "acessórios e consumíveis", tone: "neutral" },
  { from: 48, to: 55, label: "Raro", note: "equipamento de nível alto", tone: "warn" },
  { from: 56, to: 56, label: "Sempre cai", note: "slot de garantia: todo abate sorteia com divisor 1", tone: "ok" },
  { from: 57, to: 63, label: "Raros progressivos", note: "sete faixas de raridade crescente (35 → 20.000)", tone: "warn" },
];

export function slotGroup(slot: number): SlotGroup {
  return SLOT_GROUPS.find((g) => slot >= g.from && slot <= g.to) ?? SLOT_GROUPS[SLOT_GROUPS.length - 1];
}

/** Slots 8–11 são reescritos por `EffectiveDropRate` depois de qualquer conta. */
export function hardcodedRate(slot: number): number | null {
  switch (slot) {
    case 8:
    case 9:
    case 10:
      return 4;
    case 11:
      return 1;
    default:
      return null;
  }
}

/**
 * Espelho de `loot.EffectiveDropRate`. `base` é `DropRate[64]` do jogo; `slot`
 * indexa a tabela. Divisões truncadas de propósito (Go faz o mesmo com int).
 */
export function effectiveDropRate(
  slot: number,
  killerBonus: number,
  mobLevel: number,
  base: readonly number[],
): number {
  let rate = base[slot] ?? 0;
  const bonus = 100 + killerBonus;
  if (bonus !== 100) {
    rate = Math.trunc((Math.trunc(10000 / (bonus + 1)) * rate) / 100);
  }

  if (slot < 60) {
    const block = Math.trunc(slot / 8);
    if (block === 0 || block === 1 || block === 2) {
      const pct =
        mobLevel < 10 ? 4 : mobLevel < 20 ? 5 : mobLevel < 30 ? 6 : mobLevel < 40 ? 7 : mobLevel < 60 ? 8 : 99;
      rate = Math.trunc((pct * rate) / 100);
    }
  } else {
    const pct =
      mobLevel < 170
        ? 90
        : mobLevel < 200
          ? 60
          : mobLevel < 230
            ? 50
            : mobLevel < 255
              ? 43
              : mobLevel < 320
                ? 38
                : 50;
    rate = Math.trunc((pct * rate) / 100);
  }

  const hard = hardcodedRate(slot);
  if (hard !== null) rate = hard;
  return rate > 32000 ? 32000 : rate;
}

/**
 * Chance como texto de operador. `rate <= 0` cai no caminho "sempre" do runtime
 * (`Drops` devolve true), por isso o rótulo não vira "1 em 0".
 */
export function oddsLabel(rate: number): string {
  if (rate <= 1) return "sempre cai";
  return `≈ 1 em ${rate.toLocaleString("pt-BR")}`;
}

export const DROP_EDITOR_NOTE =
  "o valor é divisor: dobrar o número torna o drop duas vezes mais raro. A chance mostrada já inclui o nível do mob e o bônus do matador, e slots 8–11 são fixos no código.";
