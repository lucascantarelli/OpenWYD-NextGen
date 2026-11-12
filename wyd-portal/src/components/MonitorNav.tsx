import Link from "next/link";

/**
 * Navegação das telas de monitoramento.
 *
 * O índice (`/monitoramento`) é o resumo; cada subpágina aprofunda uma fonte.
 * A navegação é a mesma em todas para o staff não se perder entre elas.
 */
const ROUTES = [
  { href: "/monitoramento", label: "Resumo" },
  { href: "/monitoramento/servidor", label: "Servidor de jogo" },
  { href: "/monitoramento/grafana", label: "Painéis" },
  { href: "/monitoramento/prometheus", label: "Métricas e alertas" },
  { href: "/monitoramento/logs", label: "Logs" },
  { href: "/monitoramento/launcher", label: "Launcher" },
] as const;

export function MonitorNav({ current }: { current: string }) {
  return (
    <nav className="embed-tabs" aria-label="Seções do monitoramento" style={{ marginBottom: 16 }}>
      {ROUTES.map((route) => (
        <Link
          key={route.href}
          className="chip"
          href={route.href}
          aria-current={route.href === current ? "page" : undefined}
        >
          {route.label}
        </Link>
      ))}
    </nav>
  );
}
