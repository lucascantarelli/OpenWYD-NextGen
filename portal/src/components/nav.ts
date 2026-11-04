/**
 * Fonte única do menu do portal. Nenhuma tela/rota é hardcoded na sidebar:
 * as issues 22–40 só trocam `soon: true` por `soon: false` quando a tela nasce.
 */

export type NavIcon =
  | "gauge"
  | "package"
  | "swords"
  | "store"
  | "percent"
  | "map"
  | "sparkles"
  | "sliders"
  | "users"
  | "activity"
  | "receipt"
  | "tag"
  | "terminal"
  | "compass"
  | "route"
  | "user"
  | "castle"
  | "flask"
  | "book";

export interface NavItem {
  href: string;
  label: string;
  icon: NavIcon;
  /** Rota ainda não implementada: aparece como "em breve", sem link morto. */
  soon?: boolean;
}

export interface NavGroup {
  group: string;
  items: readonly NavItem[];
}

export const NAV: readonly NavGroup[] = [
  {
    group: "Visão",
    items: [{ href: "/painel", label: "Visão geral", icon: "gauge" }],
  },
  {
    group: "Conteúdo",
    items: [
      { href: "/itens", label: "Catálogo de itens", icon: "package" },
      { href: "/precos", label: "Preços globais", icon: "tag" },
      { href: "/mobs", label: "Mobs e stats", icon: "swords" },
      { href: "/npcs", label: "NPCs e lojas", icon: "store" },
      { href: "/drops", label: "Drops", icon: "percent" },
      { href: "/refino", label: "Refino e combine", icon: "flask", soon: true },
      { href: "/skills", label: "Skills", icon: "book", soon: true },
    ],
  },
  {
    group: "Mundo",
    items: [
      { href: "/atlas", label: "Atlas do mundo", icon: "compass" },
      { href: "/teleportes", label: "Teleportes", icon: "route" },
      { href: "/mapa", label: "Mapa de colisão", icon: "map" },
      { href: "/eventos", label: "Eventos globais", icon: "sparkles", soon: true },
      { href: "/config", label: "Rates e flags", icon: "sliders" },
    ],
  },
  {
    group: "Operação",
    items: [
      { href: "/contas", label: "Contas e papéis", icon: "users", soon: true },
      { href: "/operacao", label: "Ao vivo", icon: "activity", soon: true },
      { href: "/faturamento", label: "Faturamento", icon: "receipt", soon: true },
      { href: "/personagens", label: "Personagens", icon: "user" },
      { href: "/guildas", label: "Guildas", icon: "swords" },
      { href: "/castelo", label: "Castelo e Kefra", icon: "castle" },
      { href: "/logs", label: "Logs dos containers", icon: "terminal", soon: true },
    ],
  },
] as const;

/** Rotas do shell que exigem sessão (usado pelo middleware e pela sidebar). */
export const SHELL_ROUTES: readonly string[] = NAV.flatMap((g) =>
  g.items.map((i) => i.href),
);
