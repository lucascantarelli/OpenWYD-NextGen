/**
 * Vocabulário do editor de `Carry[]` por template (issue 34, fatia 3).
 *
 * O `Carry[64]` é a tabela de drop do mob: o **slot** decide a raridade (o
 * divisor vem de `DropRate[slot]`) e o item decide o que cai. Por isso a tela
 * agrupa as posições pelas mesmas faixas de raridade do editor de taxa e nomeia
 * a faixa antes do número — "Sempre cai · posição 56", nunca "slot 56" sozinho.
 *
 * Os efeitos do `STRUCT_ITEM` não são editados aqui: o web-api preserva os do
 * item atual quando o índice não muda, e zera ao trocar o item. Mandar efeitos
 * do browser seria confiar em dado que a tela não edita.
 */
import { SLOT_GROUPS, type SlotGroup } from "@/lib/droprate";

export const CARRY_SLOTS = 64;

export interface MobCarrySlot {
  slot: number;
  item_index: number;
  item_name: string;
  eff1: number;
  effv1: number;
  eff2: number;
  effv2: number;
  eff3: number;
  effv3: number;
  from_override: boolean;
}

export interface MobCarry {
  template_name: string;
  display_name: string;
  level: number;
  overridden: boolean;
  slots: MobCarrySlot[];
}

/** O backend manda só as posições ocupadas; vazio é ausência de linha. */
export function slotMap(slots: readonly MobCarrySlot[]): Map<number, MobCarrySlot> {
  return new Map(slots.map((slot) => [slot.slot, slot]));
}

export interface CarryGroup extends SlotGroup {
  occupied: MobCarrySlot[];
  free: number[];
}

/** Distribui as posições ocupadas pelas faixas e lista as livres de cada uma. */
export function groupCarry(slots: readonly MobCarrySlot[]): CarryGroup[] {
  const bySlot = slotMap(slots);
  return SLOT_GROUPS.map((group) => {
    const occupied: MobCarrySlot[] = [];
    const free: number[] = [];
    for (let slot = group.from; slot <= group.to; slot++) {
      const item = bySlot.get(slot);
      if (item) occupied.push(item);
      else free.push(slot);
    }
    return { ...group, occupied, free };
  });
}

/** Efeitos no formato curto "código:valor", só para leitura. */
export function effectLabel(slot: MobCarrySlot): string {
  return [
    [slot.eff1, slot.effv1],
    [slot.eff2, slot.effv2],
    [slot.eff3, slot.effv3],
  ]
    .filter(([code]) => code !== 0)
    .map(([code, value]) => `${code}:${value}`)
    .join(" ");
}

export const CARRY_EDITOR_NOTE =
  "salvar substitui a tabela inteira do template: posição vazia deixa de dropar. Trocar o item zera os efeitos daquele slot; manter o mesmo item preserva os efeitos atuais. O tmserver aplica no próximo boot com -mob-carry-editing.";
