import { KeyRound, LifeBuoy, Swords } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";

import { Badge } from "@/components/Badge";
import { LogoutButton } from "@/components/LogoutButton";
import { TEXTO } from "@/lib/i18n";
import { roleLabel, roleTone } from "@/lib/role";
import { getPanelAccount, getPanelCharacters, getPanelRankings } from "@/server/panel";
import { getSession } from "@/server/session";

import PinForm from "./pin-form";
import RecoveryForm from "./recovery-form";

export const dynamic = "force-dynamic";

export const metadata = { title: "Minha conta · WYD2" };

const CLASSES: Record<number, string> = {
  0: "Guerreiro",
  1: "Transknight",
  2: "Foema",
  3: "BeastMaster",
  4: "Mortal",
  5: "Arch",
};

function fmtClass(code: number): string {
  return CLASSES[code] ?? `classe ${code}`;
}

/**
 * Área do jogador: conta, personagens, ranking e credenciais.
 *
 * Toda leitura passa por `server/panel`, que injeta o `account_id` do cookie de
 * sessão — a página nunca aceita id vindo da URL. Cada bloco falha sozinho.
 */
export default async function ContaPage() {
  const session = await getSession();
  if (!session) redirect("/login");

  const [conta, personagens, ranking] = await Promise.all([
    getPanelAccount().catch(() => null),
    getPanelCharacters().catch(() => null),
    getPanelRankings(10).catch(() => null),
  ]);

  return (
    <>
      <header className="sitebar">
        <Link className="brand" href="/conta" aria-label="WYD2">
          <span className="brand__mark" aria-hidden="true">
            W
          </span>
          <span className="brand__text">
            <span className="brand__name">WYD2</span>
            <span className="brand__tag">Área do jogador</span>
          </span>
        </Link>
        <nav className="sitebar__links" aria-label="Área do jogador">
          <Link href="/conta" aria-current="page">
            Minha conta
          </Link>
          <Link href="/guias">Guias</Link>
          <Link href="/enciclopedia">Enciclopédia</Link>
          {session.role === "admin" || session.role === "moderator" ? (
            <Link href="/painel">Administração</Link>
          ) : null}
        </nav>
        <LogoutButton />
      </header>

      <main className="page page--narrow">
        <div className="player-hero">
          <div>
            <h1>{conta ? `Olá, ${conta.login}.` : "Minha conta"}</h1>
            <p>
              Seus personagens, o ranking do servidor e as credenciais que o jogo pede na entrada.
            </p>
          </div>
          <div className="chips">
            <span className="status">sessão #{session.accountId}</span>
            <Badge tone={roleTone(session.role)}>{roleLabel(session.role)}</Badge>
          </div>
        </div>

        <section className="card" style={{ marginBottom: 16 }}>
          <div className="card__head">
            <div>
              <h2 className="card__title">
                <Swords size={15} strokeWidth={1.8} aria-hidden="true" /> Personagens
              </h2>
              <p className="card__sub">Vínculo de conta, classe e nível.</p>
            </div>
            <Badge>{personagens === null ? "indisponível" : `${personagens.length} na conta`}</Badge>
          </div>
          <div className="card__body">
            {personagens === null ? (
              <div className="empty">
                <strong>Não foi possível carregar seus personagens</strong>
                <p>Não foi possível ler a lista agora. Tente recarregar em instantes.</p>
              </div>
            ) : personagens.length === 0 ? (
              <div className="empty">
                <strong>Nenhum personagem nesta conta</strong>
                <p>Crie o primeiro personagem no cliente, na tela de seleção.</p>
              </div>
            ) : (
              <div className="character-grid" style={{ padding: 0 }}>
                {personagens.map((character) => (
                  <article className="character" key={character.id}>
                    <span className="character__slot">Slot {character.slot}</span>
                    <span className="character__name">{character.name}</span>
                    <span className="character__meta">
                      {fmtClass(character.class)} · nível {character.level}
                    </span>
                  </article>
                ))}
              </div>
            )}
          </div>
        </section>

        <div className="grid-2">
          <section className="card">
            <div className="card__head">
              <div>
                <h2 className="card__title">Ranking</h2>
                <p className="card__sub">Top 10 por nível, sem dado de conta.</p>
              </div>
            </div>
            <div className="card__body card__body--flush">
              {ranking === null ? (
                <p className="muted small" style={{ padding: 20, margin: 0 }}>
                  Ranking indisponível.
                </p>
              ) : ranking.length === 0 ? (
                <p className="muted small" style={{ padding: 20, margin: 0 }}>
                  Nenhum personagem no ranking ainda.
                </p>
              ) : (
                <ol className="rank-list">
                  {ranking.map((entry) => (
                    <li className="rank" key={`${entry.rank}-${entry.name}`}>
                      <span className="rank__pos">#{entry.rank}</span>
                      <span className="rank__name">{entry.name}</span>
                      <span className="rank__meta">
                        {fmtClass(entry.class)} · {entry.level}
                      </span>
                    </li>
                  ))}
                </ol>
              )}
            </div>
          </section>

          <section className="card">
            <div className="card__head">
              <div>
                <h2 className="card__title">Conta</h2>
                <p className="card__sub">Identidade usada no login do jogo.</p>
              </div>
            </div>
            <div className="card__body">
              {conta ? (
                <dl className="kv">
                  <dt>Login</dt>
                  <dd className="mono">{conta.login}</dd>
                  <dt>Papel</dt>
                  <dd>{roleLabel(conta.role)}</dd>
                  <dt>Sessão</dt>
                  <dd className="mono">#{conta.id}</dd>
                </dl>
              ) : (
                <p className="muted small">Não foi possível carregar os dados da conta agora.</p>
              )}
            </div>
          </section>
        </div>

        <div className="grid-2">
          <section className="card">
            <div className="card__head">
              <div>
                <h2 className="card__title">
                  <KeyRound size={15} strokeWidth={1.8} aria-hidden="true" /> Senha numérica
                </h2>
                <p className="card__sub">Criada no cliente, no primeiro acesso; trocada aqui ou no jogo.</p>
              </div>
              <Badge tone={!conta ? "neutral" : conta.pin_defined ? "ok" : "warn"}>
                {!conta
                  ? TEXTO.pin.estadoIndisponivel
                  : conta.pin_defined
                    ? TEXTO.pin.estadoDefinida
                    : TEXTO.pin.estadoNaoDefinida}
              </Badge>
            </div>
            <div className="card__body">
              {conta ? (
                <PinForm login={conta.login} defined={conta.pin_defined} />
              ) : (
                <p className="muted small">{TEXTO.pin.consultaIndisponivel}</p>
              )}
            </div>
          </section>

          <section className="card">
            <div className="card__head">
              <div>
                <h2 className="card__title">
                  <LifeBuoy size={15} strokeWidth={1.8} aria-hidden="true" /> Recuperar acesso
                </h2>
                <p className="card__sub">Pedido registrado para a equipe.</p>
              </div>
            </div>
            <div className="card__body">
              <RecoveryForm />
            </div>
          </section>
        </div>
      </main>
    </>
  );
}
