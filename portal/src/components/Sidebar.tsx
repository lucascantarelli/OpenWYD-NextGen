"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { Badge } from "./Badge";
import { Icon } from "./Icon";
import { NAV } from "./nav";

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="ui-sidebar" aria-label="Navegação principal">
      <div className="ui-sidebar__brand">
        WYD2 <span>administração</span>
      </div>

      {NAV.map((group) => (
        <nav key={group.group} className="ui-nav-group" aria-label={group.group}>
          <div className="ui-nav-group__title">{group.group}</div>
          {group.items.map((item) => {
            const active =
              pathname === item.href || pathname.startsWith(`${item.href}/`);

            if (item.soon) {
              return (
                <span
                  key={item.href}
                  className="ui-nav-item ui-nav-item--soon"
                  aria-disabled="true"
                  title={`${item.label} — em breve`}
                >
                  <Icon name={item.icon} />
                  <span className="ui-nav-item__label">{item.label}</span>
                  <Badge>em breve</Badge>
                </span>
              );
            }

            return (
              <Link
                key={item.href}
                href={item.href}
                className="ui-nav-item"
                aria-current={active ? "page" : undefined}
                title={item.label}
              >
                <Icon name={item.icon} />
                <span className="ui-nav-item__label">{item.label}</span>
              </Link>
            );
          })}
        </nav>
      ))}
    </aside>
  );
}
