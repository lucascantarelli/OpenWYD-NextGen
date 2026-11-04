import { redirect } from "next/navigation";

import { LogoutButton } from "@/components/LogoutButton";
import { Sidebar } from "@/components/Sidebar";
import { ToastProvider } from "@/components/Toast";
import { Badge } from "@/components/Badge";
import { getSession, isStaff } from "@/server/session";

export const dynamic = "force-dynamic";

/**
 * Shell da área administrativa.
 *
 * ⚠️ A guarda aqui é **UX**: esconde a casca de quem não é staff. A autorização
 * real é do web-api, que relê `account.role` em toda RPC admin.
 */
export default async function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const session = await getSession();
  if (!session) redirect("/login");
  if (!isStaff(session.role)) redirect("/sem-permissao");

  return (
    <div className="ui-shell">
      <Sidebar />
      <div className="ui-shell__main">
        <header className="ui-header">
          <div className="ui-header__who">
            <span>conta #{session.accountId}</span>
            <Badge tone="accent">{session.role}</Badge>
          </div>
          <LogoutButton />
        </header>
        <main className="ui-content">
          <ToastProvider>{children}</ToastProvider>
        </main>
      </div>
    </div>
  );
}
