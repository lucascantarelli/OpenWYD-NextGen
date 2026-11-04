/**
 * Tipo e validação do editor de stats de template de mob (issue 25).
 *
 * Os intervalos espelham **exatamente** a checagem do web-api
 * (`webserver/internal/grpcsrv/mobtemplateadmin.go: mobTemplateStatInRange`),
 * que por sua vez estreita para os tipos do `STRUCT_MOB` (816 B). Validar aqui
 * evita uma ida ao servidor só para receber `INVALID`, mas a decisão final
 * continua sendo do servidor.
 *
 * `exp` é int64: o proto usa `longs: Number`, então o limite seguro em JS é
 * 2^53, bem acima do campo real (int32 no struct).
 */
import { z } from "zod";

export interface MobTemplateFile {
  template_name: string;
  display_name: string;
  merchant: number;
  /** true = já existe override de stats no banco para este template. */
  overridden: boolean;
}

/** Um slot de `Equip[16]`, com os três pares de efeito do `STRUCT_ITEM`. */
export interface MobEquipSlot {
  slot: number;
  item_index: number;
  eff1: number;
  effv1: number;
  eff2: number;
  effv2: number;
  eff3: number;
  effv3: number;
}

/** `AdminMobTemplateStat` (web.proto:559) em camel/snake do proto. */
export interface MobStat {
  template_name: string;
  /** false = read-through do arquivo: o template ainda não tem override. */
  overridden: boolean;
  display_name: string;
  clan: number;
  merchant: number;
  class: number;
  coin: number;
  exp: number;
  spx: number;
  spy: number;
  level: number;
  ac: number;
  damage: number;
  chaos_rate: number;
  attack_run: number;
  direction: number;
  str: number;
  intel: number;
  dex: number;
  con: number;
  special1: number;
  special2: number;
  special3: number;
  special4: number;
  max_hp: number;
  hp: number;
  max_mp: number;
  mp: number;
  learned_skill: number;
  score_bonus: number;
  skill_bar1: number;
  skill_bar2: number;
  skill_bar3: number;
  skill_bar4: number;
  regen_hp: number;
  regen_mp: number;
  resist1: number;
  resist2: number;
  resist3: number;
  resist4: number;
  equip: MobEquipSlot[];
}

const u8 = z.number().int().min(0).max(255);
const i8 = z.number().int().min(-128).max(127);
const i16 = z.number().int().min(-32768).max(32767);
const u16 = z.number().int().min(0).max(65535);
const nonNegative = z.number().int().min(0);

export const mobStatSchema = z.object({
  template_name: z.string().min(1),
  display_name: z.string().max(63),
  clan: u8,
  merchant: u8,
  class: u8,
  coin: nonNegative,
  exp: nonNegative,
  spx: i16,
  spy: i16,
  level: u8,
  ac: nonNegative,
  damage: nonNegative,
  chaos_rate: u8,
  attack_run: u8,
  direction: u8,
  str: i16,
  intel: i16,
  dex: i16,
  con: i16,
  special1: i16,
  special2: i16,
  special3: i16,
  special4: i16,
  max_hp: nonNegative,
  hp: nonNegative,
  max_mp: nonNegative,
  mp: nonNegative,
  learned_skill: nonNegative,
  score_bonus: u16,
  skill_bar1: u8,
  skill_bar2: u8,
  skill_bar3: u8,
  skill_bar4: u8,
  regen_hp: u16,
  regen_mp: u16,
  resist1: i8,
  resist2: i8,
  resist3: i8,
  resist4: i8,
});

export type MobStatValues = z.infer<typeof mobStatSchema>;

export type NumericField = keyof MobStatValues & string;

/** `display_name` é texto e vive fora dos grupos numéricos. */
export interface FieldSpec {
  key: NumericField;
  label: string;
  hint?: string;
}

export interface FieldGroup {
  id: string;
  title: string;
  note?: string;
  fields: FieldSpec[];
}

/**
 * Grupos da tela, na ordem da ficha do `EDITAPPMOB`. `class` e `merchant` do
 * template não são os do NPC (`AdminNpc.merchant`, issue 23) — o hint diz isso
 * para ninguém editar o campo errado.
 */
export const FIELD_GROUPS: readonly FieldGroup[] = [
  {
    id: "identity",
    title: "Identidade",
    fields: [
      { key: "level", label: "nível", hint: "0–255" },
      { key: "class", label: "classe" },
      { key: "clan", label: "clã" },
      { key: "merchant", label: "merchant do template", hint: "não é o merchant do NPC" },
      { key: "direction", label: "direção" },
    ],
  },
  {
    id: "combat",
    title: "Combate",
    fields: [
      { key: "damage", label: "dano" },
      { key: "ac", label: "AC" },
      { key: "attack_run", label: "attack run" },
      { key: "chaos_rate", label: "chaos rate" },
    ],
  },
  {
    id: "attributes",
    title: "Atributos",
    note: "int16 (−32768…32767), como no STRUCT_MOB.",
    fields: [
      { key: "str", label: "força" },
      { key: "intel", label: "inteligência" },
      { key: "dex", label: "destreza" },
      { key: "con", label: "constituição" },
      { key: "special1", label: "special 1" },
      { key: "special2", label: "special 2" },
      { key: "special3", label: "special 3" },
      { key: "special4", label: "special 4" },
    ],
  },
  {
    id: "vitality",
    title: "Vitalidade",
    fields: [
      { key: "max_hp", label: "HP máximo" },
      { key: "hp", label: "HP atual" },
      { key: "max_mp", label: "MP máximo" },
      { key: "mp", label: "MP atual" },
      { key: "regen_hp", label: "regen HP", hint: "0–65535" },
      { key: "regen_mp", label: "regen MP", hint: "0–65535" },
    ],
  },
  {
    id: "progression",
    title: "Progressão",
    note: "EXP é int64 e é o valor que decide quanto o mob entrega ao morrer.",
    fields: [
      { key: "exp", label: "EXP" },
      { key: "coin", label: "coin" },
      { key: "spx", label: "SP X", hint: "int16 (pode ser negativo)" },
      { key: "spy", label: "SP Y", hint: "int16" },
      { key: "score_bonus", label: "score bonus", hint: "0–65535" },
      { key: "learned_skill", label: "skill aprendida" },
      { key: "skill_bar1", label: "barra 1" },
      { key: "skill_bar2", label: "barra 2" },
      { key: "skill_bar3", label: "barra 3" },
      { key: "skill_bar4", label: "barra 4" },
    ],
  },
  {
    id: "resist",
    title: "Resistências",
    note: "int8 (−128…127).",
    fields: [
      { key: "resist1", label: "resist 1" },
      { key: "resist2", label: "resist 2" },
      { key: "resist3", label: "resist 3" },
      { key: "resist4", label: "resist 4" },
    ],
  },
];

/** Campos alterados em relação ao carregado — o diff da sessão. */
export function changedFields(initial: MobStatValues, current: MobStatValues): NumericField[] {
  const out: NumericField[] = [];
  for (const group of FIELD_GROUPS) {
    for (const field of group.fields) {
      const key = field.key as keyof MobStatValues;
      if (key === "display_name") continue;
      const before = initial[key] as number;
      const after = current[key] as number;
      if (before !== after) out.push(field.key);
    }
  }
  return out;
}

/** `Equip[16]` do `STRUCT_MOB`: slots 0–15, índice de item > 0. */
export const MOB_EQUIP_SLOTS = 16;

export const mobEquipSchema = z.object({
  slot: z.number().int().min(0).max(MOB_EQUIP_SLOTS - 1),
  item_index: z.number().int().positive(),
  eff1: u8,
  effv1: u8,
  eff2: u8,
  effv2: u8,
  eff3: u8,
  effv3: u8,
});

export const mobEquipListSchema = z
  .array(mobEquipSchema)
  .max(MOB_EQUIP_SLOTS)
  .refine((items) => new Set(items.map((it) => it.slot)).size === items.length, {
    message: "slot duplicado",
  });

/** Slot vazio para o picker: sem efeitos, item escolhido depois. */
export function emptyEquipSlot(slot: number): MobEquipSlot {
  return { slot, item_index: 0, eff1: 0, effv1: 0, eff2: 0, effv2: 0, eff3: 0, effv3: 0 };
}

export function sortEquip(items: readonly MobEquipSlot[]): MobEquipSlot[] {
  return [...items].sort((a, b) => a.slot - b.slot);
}

export interface EquipDiff {
  added: number;
  removed: number;
  changed: number;
}

/** Diff do equip por slot (o slot é a chave, como no `Equip[16]`). */
export function equipDiff(
  initial: readonly MobEquipSlot[],
  current: readonly MobEquipSlot[],
): EquipDiff {
  const before = new Map(initial.map((it) => [it.slot, it]));
  const after = new Map(current.map((it) => [it.slot, it]));
  let added = 0;
  let removed = 0;
  let changed = 0;
  for (const slot of after.keys()) if (!before.has(slot)) added++;
  for (const slot of before.keys()) if (!after.has(slot)) removed++;
  for (const [slot, item] of after) {
    const previous = before.get(slot);
    if (!previous) continue;
    if (previous.item_index !== item.item_index) changed++;
  }
  return { added, removed, changed };
}

/** Projeção do stat completo (sem `overridden`) para o proto. */
export function toStatValues(stat: MobStat): MobStatValues {
  const { overridden: _ignored, ...rest } = stat;
  return rest;
}
