import { redirect } from "next/navigation";

import { getSession, isStaff } from "@/server/session";

import LoginForm from "./login-form";

export const dynamic = "force-dynamic";

export const metadata = { title: "Entrar · WYD2" };

export default async function LoginPage() {
  const session = await getSession();
  if (session && isStaff(session.role)) redirect("/painel");

  return (
    <main className="login-shell">
      <section className="login-brand" aria-hidden="true">
        <div className="login-wordmark">
          <span className="brand__mark">W</span>
          <span>WYD2</span>
        </div>

        <div className="login-hero">
          <p className="login-hero__eyebrow">Administração</p>
          <h2>O servidor inteiro, numa tela que responde.</h2>
          <p>
            Contas, conteúdo e operação do mundo em um só lugar — sem tocar no estado vivo do
            personagem e sem expor credencial ao browser.
          </p>
          <div className="login-rules">
            <span>Auditoria em toda ação</span>
            <span>Sessão assinada</span>
            <span>Acesso por papel</span>
          </div>
        </div>

        <p className="login-hint">With Your Destiny · servidor privado</p>
      </section>

      <section className="login-panel">
        <div className="login-box">
          <h1 className="login-title">Entrar na administração</h1>
          <p className="login-sub">
            Use a conta do jogo com papel de moderador ou administrador.
          </p>
          <LoginForm />
          <p className="login-hint">
            O papel é lido a cada login; mudanças de permissão valem na próxima sessão.
          </p>
        </div>
      </section>
    </main>
  );
}
