/**
 * Tipos e rótulos do estado competitivo do mundo (issue 45): guildas, relações,
 * zonas do reino com a taxa da cidade, torre (GTorre), quest do castelo e Kefra.
 *
 * Tudo aqui é estado persistido que o tmserver carrega **no boot**: uma escrita
 * chega ao mundo só depois de `docker compose restart tmserver`. Os valores de
 * `guild_level`/`kind` vêm do CHECK das tabelas, não de invenção da UI.
 */

export interface GuildMember {
  character_id: number;
  account_id: number;
  slot: number;
  name: string;
  guild_level: number;
}

export interface Guild {
  id: number;
  name: string;
  clan: number;
  fame: number;
  citizen: number;
  /** Membro de nível 9; vazio quando a guilda não tem líder gravado. */
  master_name: string;
  members: GuildMember[];
  /** Zonas do reino em que esta guilda é a `charge_guild`. */
  city_zones: number[];
  tower_owner: boolean;
}

export interface GuildRelation {
  guild_id: number;
  guild_name: string;
  target_guild_id: number;
  target_name: string;
  /** 1 = aliada, 2 = inimiga (`guild_relation` CHECK). */
  kind: number;
}

export interface KingdomZone {
  zone: number;
  charge_guild: number;
  charge_guild_name: string;
  challenge_guild: number;
  challenge_guild_name: string;
  clan: number;
  victory: number;
  city_tax: number;
  challenge_money: number;
  tax_vault: number;
}

export interface CastleQuest {
  level: number;
  time_left: number;
  clear: boolean;
  leader_name: string;
}

export interface Kefra {
  /** false enquanto o tmserver nunca gravou um ciclo — a escrita é recusada. */
  initialized: boolean;
  defeated: boolean;
  next_spawn_unix: number;
  last_spawn_unix: number;
  revision: number;
}

export interface WorldState {
  guilds: Guild[];
  relations: GuildRelation[];
  zones: KingdomZone[];
  tower_owner_guild: number;
  tower_owner_name: string;
  tower_updated_unix: number;
  castle: CastleQuest;
  kefra: Kefra;
}

/** CHECK `guild_member.guild_level BETWEEN 0 AND 9`; 6..8 são os sub-líderes. */
export const GUILD_LEVEL_LABEL: Record<number, string> = {
  0: "membro",
  6: "sub-líder",
  7: "sub-líder",
  8: "sub-líder",
  9: "mestre",
};

export function guildLevelLabel(level: number): string {
  return GUILD_LEVEL_LABEL[level] ?? `nível ${level}`;
}

export const RELATION_LABEL: Record<number, string> = {
  1: "aliada",
  2: "inimiga",
};

export function relationLabel(kind: number): string {
  return RELATION_LABEL[kind] ?? `tipo ${kind}`;
}

/** Faixa aceita pelo web-api (`worldstateadmin.MaxCityTax`). */
export const MAX_CITY_TAX = 30;

/** Cidades do reino por zona, como o cliente as conhece. */
export const ZONE_NAMES: Record<number, string> = {
  0: "Armia",
  1: "Noatum",
  2: "Nippleheim",
  3: "Erion",
  4: "Dolphin",
};

export function zoneName(zone: number): string {
  return ZONE_NAMES[zone] ?? `zona ${zone}`;
}

/** Data curta em pt-BR a partir de um unix (0 = nunca). */
export function formatUnix(unix: number): string {
  if (!unix || unix <= 0) return "—";
  return new Date(unix * 1000).toLocaleString("pt-BR", {
    dateStyle: "short",
    timeStyle: "short",
  });
}
