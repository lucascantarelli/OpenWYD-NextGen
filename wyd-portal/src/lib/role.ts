/**
 * Papel da conta — vocabulário único da UI.
 *
 * O valor vem do backend (`player` | `moderator` | `admin`); desconhecido cai em
 * `player` para não promover ninguém por engano na tela.
 */
export type Role = "player" | "moderator" | "admin";

export const ROLES: readonly Role[] = ["player", "moderator", "admin"];

export function normalizeRole(raw: string): Role {
  const value = raw.trim().toLowerCase();
  return (ROLES as readonly string[]).includes(value) ? (value as Role) : "player";
}

const LABEL: Record<Role, string> = {
  player: "jogador",
  moderator: "moderador",
  admin: "administrador",
};

const TONE: Record<Role, "neutral" | "accent" | "warn"> = {
  player: "neutral",
  moderator: "accent",
  admin: "warn",
};

export function roleLabel(role: string): string {
  return LABEL[normalizeRole(role)];
}

export function roleTone(role: string): "neutral" | "accent" | "warn" {
  return TONE[normalizeRole(role)];
}
