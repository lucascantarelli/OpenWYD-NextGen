import Link from "next/link";

import { SiteFooter } from "@/components/SiteFooter";
import { SiteHeader } from "@/components/SiteHeader";
import { getPublicRankings } from "@/server/panel";

export const dynamic = "force-dynamic";

export const metadata = {
  title: "Ranking · WYD2",
  description: "Os personagens de maior nível do servidor, com classe e posição.",
};

const CLASSES: Record<number, string> = {
  0: "Guerreiro",
  1: "TransKnight",
  2: "Foema",
  3: "BeastMaster",
  4: "Mortal",
  5: "Arch",
};

export default async function RankingPage() {
  const ranking = await getPublicRankings(50).catch(() => null);

  return (
    <>
      <SiteHeader current="/ranking" />

      <main className="site site--narrow">
        <header className="site-page-head">
          <p className="site-section__eyebrow">Ranking</p>
          <h1>Quem está no topo</h1>
          <p>
            Ordem por nível do personagem. Só nome, classe e nível são públicos — nenhum dado de
            conta aparece aqui.
          </p>
        </header>

        {ranking === null ? (
          <div className="banner">
            O ranking não respondeu agora. O servidor pode estar reiniciando; tente novamente em
            instantes.
          </div>
        ) : ranking.length === 0 ? (
          <div className="empty">
            <strong>O ranking ainda está vazio</strong>
            <p>Crie o primeiro personagem e entre no mundo para inaugurar a lista.</p>
            <Link className="ui-btn ui-btn--primary" href="/downloads">
              Como começar
            </Link>
          </div>
        ) : (
          <div className="ui-table-wrap">
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th scope="col" style={{ width: 80 }}>
                      <span>Posição</span>
                    </th>
                    <th scope="col">
                      <span>Personagem</span>
                    </th>
                    <th scope="col">
                      <span>Classe</span>
                    </th>
                    <th scope="col" style={{ width: 110 }}>
                      <span>Nível</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {ranking.map((entry) => (
                    <tr key={`${entry.rank}-${entry.name}`}>
                      <td className="mono" style={{ color: "var(--accent)" }}>
                        #{entry.rank}
                      </td>
                      <td>
                        <strong>{entry.name}</strong>
                      </td>
                      <td className="muted">{CLASSES[entry.class] ?? `classe ${entry.class}`}</td>
                      <td className="ui-align-right">{entry.level}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className="ui-table-footer">
              {ranking.length} personagem(ns) · leitura pública, sem identidade de conta
            </p>
          </div>
        )}
      </main>

      <SiteFooter />
    </>
  );
}
