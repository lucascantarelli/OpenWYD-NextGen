import { redirect } from "next/navigation";

import { Badge } from "@/components/Badge";
import { LogoutButton } from "@/components/LogoutButton";
import { roleLabel } from "@/lib/role";
import { Sidebar } from "@/components/Sidebar";
import { ToastProvider } from "@/components/Toast";
import { getSession, isStaff } from "@/server/session";

export const dynamic = "force-dynamic";

/**
 * Shell da área administrativa.
 *
 * A guarda aqui é de **experiência**: esconde a casca de quem não é staff. A
 * autorização real é do `wyd-admin`, que relê o papel da conta em toda chamada.
 */
export default async function AdminLayout({ children }: { children: React.ReactNode }) {
  const session = await getSession();
  if (!session) redirect("/login");
  if (!isStaff(session.role)) redirect("/sem-permissao");

  return (
    <div className="shell">
      <Sidebar />
      <div className="shell__main">
        <header className="topbar">
          <div className="crumb">
            <span>Administração</span>
            <span className="crumb__sep" aria-hidden="true">
              /
            </span>
            <strong>WYD2</strong>
          </div>
          <div className="topbar__actions">
            <div className="who">
              <span>Conta #{session.accountId}</span>
              <Badge tone="accent">{roleLabel(session.role)}</Badge>
              <span className="who__avatar" aria-hidden="true">
                {session.role.slice(0, 1)}
              </span>
            </div>
            <LogoutButton />
          </div>
        </header>
        <main className="page">
          <ToastProvider>{children}</ToastProvider>
        </main>
      </div>
    </div>
  );
}
