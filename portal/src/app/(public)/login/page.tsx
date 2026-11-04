import { redirect } from "next/navigation";

import { getSession, isStaff } from "@/server/session";
import LoginForm from "./login-form";

export const dynamic = "force-dynamic";

export const metadata = { title: "Entrar · WYD2" };

export default async function LoginPage() {
  const session = await getSession();
  // Já logado como staff? Não faz sentido pedir credencial de novo.
  if (session && isStaff(session.role)) redirect("/painel");

  return (
    <main style={{ maxWidth: 380, margin: "0 auto", padding: "96px 24px" }}>
      <h1 style={{ fontSize: 22, marginBottom: 4 }}>WYD2 · Administração</h1>
      <p style={{ color: "var(--muted)", marginTop: 0, marginBottom: 28, fontSize: 13 }}>
        Entre com a conta do jogo que tem papel de moderador ou administrador.
      </p>
      <LoginForm />
      <p style={{ color: "var(--muted)", fontSize: 12, marginTop: 24 }}>
        O papel é lido no momento do login. Promover alguém no banco só vale na próxima sessão.
      </p>
    </main>
  );
}
