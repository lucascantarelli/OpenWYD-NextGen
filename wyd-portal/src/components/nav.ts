/**
 * Fonte única do menu administrativo.
 *
 * Só entram aqui telas que existem no contrato atual do `wyd-admin`
 * (`crates/wyd-admin/src/transport/mod.rs`). Menu não anuncia o que ainda não é
 * servido pelo backend: nada de item "em breve" nem rota morta.
 */

export type NavIcon =
  | "overview"
  | "accounts"
  | "grants"
  | "monitoring"
  | "catalog"
  | "config"
  | "audit"
  | "economy";

export interface NavItem {
  href: string;
  label: string;
  icon: NavIcon;
  /** Texto curto do `title`/tooltip; a sidebar não repete o rótulo. */
  hint?: string;
}

export interface NavGroup {
  group: string;
  items: readonly NavItem[];
}

export const NAV: readonly NavGroup[] = [
  {
    group: "Operação",
    items: [
      { href: "/painel", label: "Visão geral", icon: "overview", hint: "Estado do servidor" },
      { href: "/contas", label: "Contas", icon: "accounts", hint: "Busca e criação de conta" },
      {
        href: "/entregas",
        label: "Entregas",
        icon: "grants",
        hint: "Conceder ouro e conferir a fila",
      },
      {
        href: "/monitoramento",
        label: "Monitoramento",
        icon: "monitoring",
        hint: "Métricas, logs e presença",
      },
    ],
  },
  {
    group: "Conteúdo",
    items: [
      { href: "/itens", label: "Catálogo de itens", icon: "catalog", hint: "Itens do servidor" },
      {
        href: "/configuracoes",
        label: "Configurações",
        icon: "config",
        hint: "Taxas, drops, NPC e teleportes",
      },
    ],
  },
  {
    group: "Governança",
    items: [
      {
        href: "/auditoria",
        label: "Auditoria",
        icon: "audit",
        hint: "Trilha de ações do staff (admin)",
      },
      {
        href: "/economia",
        label: "Economia",
        icon: "economy",
        hint: "Imposto e concessões com fonte auditada",
      },
    ],
  },
] as const;

/**
 * Subpáginas do monitoramento: não entram na sidebar (o item é único), mas precisam
 * da mesma sessão de staff do shell. Manter aqui evita rota filha fora do proxy.
 */
export const MONITOR_ROUTES: readonly string[] = [
  "/monitoramento/servidor",
  "/monitoramento/grafana",
  "/monitoramento/prometheus",
  "/monitoramento/logs",
];

/** Rotas do shell que exigem sessão de staff (usado pelo proxy e pela sidebar). */
export const SHELL_ROUTES: readonly string[] = [
  ...NAV.flatMap((group) => group.items.map((item) => item.href)),
  ...MONITOR_ROUTES,
];

/**
 * Área do jogador e documentação: públicas por desenho, por isso ficam fora de
 * `NAV`/`SHELL_ROUTES` (o `matcher` do proxy não deve barrar leitura pública).
 */
export const PUBLIC_LINKS: readonly { href: string; label: string }[] = [
  { href: "/conta", label: "Minha conta" },
  { href: "/guias", label: "Guias" },
  { href: "/enciclopedia", label: "Enciclopédia" },
] as const;
