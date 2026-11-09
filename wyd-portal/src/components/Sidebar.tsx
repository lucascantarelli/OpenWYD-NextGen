"use client";

import { Activity, LayoutDashboard, Package, Users } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { NAV, PUBLIC_LINKS, type NavIcon } from "./nav";

const ICONS: Record<NavIcon, typeof Activity> = {
  overview: LayoutDashboard,
  accounts: Users,
  monitoring: Activity,
  catalog: Package,
};

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="rail" aria-label="Navegação principal">
      <Link className="brand" href="/painel" aria-label="WYD2 — visão geral">
        <span className="brand__mark" aria-hidden="true">
          W
        </span>
        <span className="brand__text">
          <span className="brand__name">WYD2</span>
          <span className="brand__tag">Administração</span>
        </span>
      </Link>

      <nav className="nav" aria-label="Administração">
        {NAV.map((group) => (
          <div key={group.group} className="nav__group">
            <div className="nav__label">{group.group}</div>
            {group.items.map((item) => {
              const Icon = ICONS[item.icon];
              const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className="nav__item"
                  aria-current={active ? "page" : undefined}
                  title={item.hint ?? item.label}
                >
                  <Icon size={17} strokeWidth={1.7} aria-hidden="true" />
                  <span className="nav__text">{item.label}</span>
                </Link>
              );
            })}
          </div>
        ))}
      </nav>

      <div className="rail__foot">
        {PUBLIC_LINKS.map((link) => (
          <Link key={link.href} href={link.href}>
            {link.label}
          </Link>
        ))}
      </div>
    </aside>
  );
}
