import Link from "next/link";

import { Badge } from "@/components/Badge";
import { getSession, isStaff } from "@/server/session";

export const dynamic = "force-dynamic";

export const metadata = { title: "WYD2 · With Your Destiny" };

/**
 * Página pública do portal (PORTAL-02).
 *
 * O `/` não exige sessão: jogador encontra jogo, conta, guias e enciclopédia, e
 * quem tem sessão de staff descobre a administração aqui (link, não botão
 * obrigatório — a guarda real continua no proxy + `(admin)/layout` + gateway).
 * Nada de admin vaza: só links públicos e, para staff, o atalho de entrada.
 */
export default async function Home() {
  const session = await getSession();
  const staff = session && isStaff(session.role);

  return (
    <main className="login-shell">
      <section className="login-brand" aria-hidden="true">
        <div className="login-wordmark">
          <span className="brand__mark">W</span>
          <span>WYD2</span>
        </div>

        <div className="login-hero">
          <p className="login-hero__eyebrow">With Your Destiny</p>
          <h2>O mundo continua — agora com portal.</h2>
          <p>
            Gerencie sua conta, troque a senha numérica, consulte o ranking e leia os guias. Sem
            tocar no cliente do jogo.
          </p>
          <div className="login-rules">
            <span>Conta e PIN</span>
            <span>Ranking</span>
            <span>Guias e enciclopédia</span>
          </div>
        </div>

        <p className="login-hint">With Your Destiny · servidor privado</p>
      </section>

      <section className="login-panel">
        <div className="login-box">
          <h1 className="login-title">Portal do jogador</h1>
          <p className="login-sub">Escolha por onde começar.</p>
          <nav aria-label="Portal" style={{ display: "grid", gap: 8 }}>
            <Link className="ui-btn ui-btn--primary" href="/conta">
              Minha conta
            </Link>
            <Link className="ui-btn" href="/guias">
              Guias
            </Link>
            <Link className="ui-btn" href="/enciclopedia">
              Enciclopédia
            </Link>
            {!session && (
              <Link className="ui-btn" href="/login">
                Entrar (staff)
              </Link>
            )}
          </nav>
          {staff && (
            <p style={{ marginTop: 16, marginBottom: 0 }}>
              <Badge tone="accent">staff</Badge>{" "}
              <Link href="/painel">Abrir administração</Link>
            </p>
          )}
          <p className="login-hint">
            A área administrativa exige papel de moderador ou administrador.
          </p>
        </div>
      </section>
    </main>
  );
}
