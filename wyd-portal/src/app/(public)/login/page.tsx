import Link from "next/link";
import { redirect } from "next/navigation";

import { SiteHeader } from "@/components/SiteHeader";
import { getSession } from "@/server/session";

import LoginForm from "./login-form";

export const dynamic = "force-dynamic";

export const metadata = {
  title: "Entrar · WYD2",
  description: "Entre com a conta do jogo para ver seus personagens e o ranking.",
  robots: { index: false, follow: false },
};

/**
 * Entrada do jogador.
 *
 * A mesma credencial do jogo autentica o portal; o papel da conta decide o
 * destino depois do login. A área administrativa **não** é anunciada aqui nem em
 * nenhuma página pública: quem é staff chega por URL direta e continua barrado
 * pelo `proxy.ts`, pelo layout de `(admin)` e pelo gateway.
 */
export default async function LoginPage() {
  const session = await getSession();
  if (session) redirect("/conta");

  return (
    <>
      <SiteHeader />

      <main className="login-shell login-shell--site">
        <section className="login-brand" aria-hidden="true">
          <div className="login-wordmark">
            <span className="brand__mark">W</span>
            <span>WYD2</span>
          </div>

          <div className="login-hero">
            <p className="login-hero__eyebrow">Área do jogador</p>
            <h2>Seus personagens, sua conta, seu reino.</h2>
            <p>
              Use o mesmo login e senha do jogo. Aqui você vê os personagens vinculados à conta,
              o ranking público e troca a senha numérica usada na entrada.
            </p>
            <div className="login-rules">
              <span>Mesma conta do jogo</span>
              <span>Personagens e ranking</span>
              <span>Senha numérica</span>
            </div>
          </div>

          <p className="login-hint">With Your Destiny · servidor privado</p>
        </section>

        <section className="login-panel">
          <div className="login-box">
            <h1 className="login-title">Entrar</h1>
            <p className="login-sub">Conta do jogo — usuário e senha cadastrados no cliente.</p>
            <LoginForm />
            <p className="login-hint">
              Ainda não tem conta? Ela é criada no cliente, na tela de login.{" "}
              <Link href="/downloads">Baixe o launcher</Link>.
            </p>
            <p className="login-hint">
              Esqueceu a senha? Peça a recuperação na <Link href="/conta">área do jogador</Link>{" "}
              depois de entrar, ou fale com a equipe.
            </p>
          </div>
        </section>
      </main>
    </>
  );
}
