import { LogIn, UserRound } from "lucide-react";
import Link from "next/link";

import { getSession } from "@/server/session";

/**
 * Cabeçalho do site público.
 *
 * Regra de segurança do portal: **nenhum link para a administração** aparece
 * aqui. O staff chega à área administrativa pela URL direta, que segue
 * protegida pelo `proxy.ts`, pelo layout de `(admin)` e pelo gateway.
 */
const LINKS: readonly { href: string; label: string }[] = [
  { href: "/", label: "Início" },
  { href: "/noticias", label: "Notícias" },
  { href: "/ranking", label: "Ranking" },
  { href: "/downloads", label: "Downloads" },
  { href: "/guias", label: "Guias" },
  { href: "/enciclopedia", label: "Enciclopédia" },
];

export async function SiteHeader({ current }: { current?: string }) {
  const session = await getSession();

  return (
    <header className="sitebar">
      <Link className="brand" href="/" aria-label="WYD2 — início">
        <span className="brand__mark" aria-hidden="true">
          W
        </span>
        <span className="brand__text">
          <span className="brand__name">WYD2</span>
          <span className="brand__tag">With Your Destiny</span>
        </span>
      </Link>

      <nav className="sitebar__links" aria-label="Site">
        {LINKS.map((link) => (
          <Link
            key={link.href}
            href={link.href}
            aria-current={current === link.href ? "page" : undefined}
          >
            {link.label}
          </Link>
        ))}
      </nav>

      <div className="sitebar__account">
        {session ? (
          <Link className="ui-btn ui-btn--sm" href="/conta">
            <UserRound size={15} strokeWidth={1.8} aria-hidden="true" />
            Minha conta
          </Link>
        ) : (
          <Link className="ui-btn ui-btn--primary ui-btn--sm" href="/login">
            <LogIn size={15} strokeWidth={1.8} aria-hidden="true" />
            Entrar
          </Link>
        )}
      </div>
    </header>
  );
}
