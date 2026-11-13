import {
  ArrowRight,
  BookOpen,
  Download,
  ScrollText,
  ShieldCheck,
  Swords,
  Trophy,
} from "lucide-react";
import Link from "next/link";

import { SiteFooter } from "@/components/SiteFooter";
import { SiteHeader } from "@/components/SiteHeader";
import { formatarData, listNoticias } from "@/server/noticias";
import { getPublicRankings } from "@/server/panel";
import { CLIENT_BUILD, PROTOCOL_VERSION, getStatusPublico } from "@/server/public-site";

export const dynamic = "force-dynamic";

export const metadata = {
  title: "WYD2 · With Your Destiny",
  description:
    "Servidor privado de With Your Destiny: notícias, ranking, downloads, guias e a área do jogador.",
};

const CLASSES = [
  { nome: "Guerreiro", papel: "Linha de frente, golpe pesado e resistência." },
  { nome: "TransKnight", papel: "Lâmina e magia, equilíbrio entre ataque e suporte." },
  { nome: "Foema", papel: "Conjuradora: dano em área e controle." },
  { nome: "BeastMaster", papel: "Invocações e combate à distância." },
] as const;

export default async function Home() {
  const [status, noticias, ranking] = await Promise.all([
    getStatusPublico(),
    Promise.resolve(listNoticias()),
    getPublicRankings(5).catch(() => null),
  ]);

  const [capa, ...resto] = noticias;
  const destaques = resto.slice(0, 2);

  return (
    <>
      <SiteHeader current="/" />

      <main className="site">
        <section className="site-hero">
          <div className="site-hero__copy">
            <p className="site-hero__eyebrow">Servidor privado · cliente {CLIENT_BUILD}</p>
            <h1>O mundo continua. Agora com um portal de verdade.</h1>
            <p className="site-hero__sub">
              Entre no reino, crie seu personagem e jogue com a comunidade. Notícias do servidor,
              ranking público, guias e downloads — tudo sem depender do cliente para saber o que
              está acontecendo.
            </p>
            <div className="site-hero__actions">
              <Link className="ui-btn ui-btn--primary" href="/downloads">
                <Download size={16} strokeWidth={1.8} aria-hidden="true" />
                Jogar agora
              </Link>
              <Link className="ui-btn" href="/login">
                Entrar na conta
              </Link>
            </div>
            <div className="site-hero__chips">
              <span className="status">
                <span className={`dot ${status.gateway ? "dot--ok" : "dot--danger"}`} />
                Servidor <strong>{status.gateway ? "no ar" : "indisponível"}</strong>
              </span>
              <span className="status">
                <span className={`dot ${status.database ? "dot--ok" : "dot--warn"}`} />
                Banco <strong>{status.database ? "conectado" : "sem leitura"}</strong>
              </span>
              <span className="status">
                Conteúdo <strong>rev. {status.configRevision ?? "—"}</strong>
              </span>
              <span className="status">
                Protocolo <strong>{PROTOCOL_VERSION}</strong>
              </span>
            </div>
          </div>

          <div className="site-hero__side">
            <div className="site-hero__card">
              <h2>
                <Trophy size={16} strokeWidth={1.8} aria-hidden="true" /> Ranking do servidor
              </h2>
              {ranking === null ? (
                <p className="muted small" style={{ margin: 0 }}>
                  Ranking indisponível agora. Tente novamente em instantes.
                </p>
              ) : ranking.length === 0 ? (
                <p className="muted small" style={{ margin: 0 }}>
                  Nenhum personagem no ranking ainda — o primeiro pode ser o seu.
                </p>
              ) : (
                <ol className="site-rank">
                  {ranking.map((entry) => (
                    <li key={`${entry.rank}-${entry.name}`}>
                      <span className="site-rank__pos">#{entry.rank}</span>
                      <span className="site-rank__name">{entry.name}</span>
                      <span className="site-rank__meta">
                        nv. {entry.level} · classe {entry.class}
                      </span>
                    </li>
                  ))}
                </ol>
              )}
              <Link className="site-hero__more" href="/ranking">
                Ver ranking completo <ArrowRight size={14} aria-hidden="true" />
              </Link>
            </div>
          </div>
        </section>

        <section className="site-section">
          <header className="site-section__head">
            <div>
              <p className="site-section__eyebrow">Notícias</p>
              <h2>O que mudou no servidor</h2>
            </div>
            <Link className="ui-btn ui-btn--ghost ui-btn--sm" href="/noticias">
              Todas as notícias <ArrowRight size={14} aria-hidden="true" />
            </Link>
          </header>

          <div className="news-grid">
            {capa && (
              <article className="news-card news-card--lead">
                <Link className="news-card__link" href={`/noticias/${capa.slug}`}>
                  <span className="news-card__tag">{capa.tag}</span>
                  <h3>{capa.title}</h3>
                  <p>{capa.summary}</p>
                  <span className="news-card__meta">
                    {formatarData(capa.date)} · {capa.author}
                  </span>
                </Link>
              </article>
            )}

            {destaques.map((noticia) => (
              <article className="news-card" key={noticia.slug}>
                <Link className="news-card__link" href={`/noticias/${noticia.slug}`}>
                  <span className="news-card__tag">{noticia.tag}</span>
                  <h3>{noticia.title}</h3>
                  <p>{noticia.summary}</p>
                  <span className="news-card__meta">{formatarData(noticia.date)}</span>
                </Link>
              </article>
            ))}
          </div>
        </section>

        <section className="site-section">
          <header className="site-section__head">
            <div>
              <p className="site-section__eyebrow">Primeiros passos</p>
              <h2>Do download ao primeiro passo no reino</h2>
            </div>
          </header>

          <div className="site-steps">
            <div className="site-step">
              <span className="site-step__num">1</span>
              <h3>Baixe o cliente</h3>
              <p>
                Use o launcher do projeto para conferir a versão {CLIENT_BUILD} e receber os
                arquivos de jogo pelo manifesto assinado.
              </p>
            </div>
            <div className="site-step">
              <span className="site-step__num">2</span>
              <h3>Crie sua conta</h3>
              <p>
                A conta é criada e autenticada pelo próprio jogo. No portal você consulta a
                conta, os personagens e troca a senha numérica.
              </p>
            </div>
            <div className="site-step">
              <span className="site-step__num">3</span>
              <h3>Entre no mundo</h3>
              <p>
                Escolha a classe, entre no Campo de Treinamento e siga a quest inicial. Os
                mercadores e o chefe já estão no mapa.
              </p>
            </div>
          </div>
        </section>

        <section className="site-section">
          <header className="site-section__head">
            <div>
              <p className="site-section__eyebrow">O jogo</p>
              <h2>Quatro classes, um reino inteiro</h2>
            </div>
            <Link className="ui-btn ui-btn--ghost ui-btn--sm" href="/enciclopedia">
              Enciclopédia <ArrowRight size={14} aria-hidden="true" />
            </Link>
          </header>

          <div className="site-cards">
            {CLASSES.map((classe) => (
              <article className="site-tile" key={classe.nome}>
                <Swords size={18} strokeWidth={1.7} aria-hidden="true" />
                <h3>{classe.nome}</h3>
                <p>{classe.papel}</p>
              </article>
            ))}
          </div>

          <div className="site-cards site-cards--links">
            <Link className="site-link-card" href="/guias">
              <BookOpen size={18} strokeWidth={1.7} aria-hidden="true" />
              <div>
                <h3>Guias do jogador</h3>
                <p>Combate, refino, quests, comércio e as regras do reino.</p>
              </div>
            </Link>
            <Link className="site-link-card" href="/downloads">
              <Download size={18} strokeWidth={1.7} aria-hidden="true" />
              <div>
                <h3>Downloads e launcher</h3>
                <p>Versão do cliente, protocolo e revisão de conteúdo vigentes.</p>
              </div>
            </Link>
            <Link className="site-link-card" href="/noticias">
              <ScrollText size={18} strokeWidth={1.7} aria-hidden="true" />
              <div>
                <h3>Notícias e atualizações</h3>
                <p>Cada entrega do servidor explicada em linguagem de jogador.</p>
              </div>
            </Link>
            <Link className="site-link-card" href="/login">
              <ShieldCheck size={18} strokeWidth={1.7} aria-hidden="true" />
              <div>
                <h3>Área do jogador</h3>
                <p>Seus personagens, o ranking e a senha numérica da entrada.</p>
              </div>
            </Link>
          </div>
        </section>
      </main>

      <SiteFooter />
    </>
  );
}
