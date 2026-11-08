import { redirect } from "next/navigation";

import { getPanelAccount, getPanelCharacters, getPanelRankings } from "@/server/panel";
import { getSession } from "@/server/session";

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
 * Painel do jogador (FASE-07-2): conta, personagens e ranking.
 *
 * Toda leitura passa por `server/panel`, que injeta o `account_id` do cookie de
 * sessão — a página nunca aceita id vindo da URL. Cada bloco falha sozinho para
 * o painel não cair inteiro com o backend fora.
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
    <main style={{ maxWidth: 880, margin: "0 auto", padding: "64px 24px", display: "grid", gap: 32 }}>
      <header>
        <h1 style={{ fontSize: 24, marginBottom: 4 }}>Minha conta</h1>
        <p style={{ color: "var(--muted)", margin: 0, fontSize: 13 }}>
          Sessão #{session.accountId} · papel {session.role}
        </p>
      </header>

      <section>
        <h2 style={{ fontSize: 16, marginBottom: 8 }}>Conta</h2>
        {conta ? (
          <dl style={{ display: "grid", gridTemplateColumns: "auto 1fr", gap: "6px 16px", fontSize: 13 }}>
            <dt style={{ color: "var(--muted)" }}>Login</dt>
            <dd style={{ margin: 0 }}>{conta.login}</dd>
            <dt style={{ color: "var(--muted)" }}>Papel</dt>
            <dd style={{ margin: 0 }}>{conta.role}</dd>
          </dl>
        ) : (
          <p style={{ color: "var(--muted)", fontSize: 13 }}>Serviço de contas indisponível.</p>
        )}
      </section>

      <section>
        <h2 style={{ fontSize: 16, marginBottom: 8 }}>Personagens</h2>
        {personagens === null ? (
          <p style={{ color: "var(--muted)", fontSize: 13 }}>Serviço de personagens indisponível.</p>
        ) : personagens.length === 0 ? (
          <p style={{ color: "var(--muted)", fontSize: 13 }}>Nenhum personagem nesta conta.</p>
        ) : (
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 13 }}>
            <thead>
              <tr style={{ textAlign: "left", color: "var(--muted)" }}>
                <th style={{ padding: "6px 4px" }}>Slot</th>
                <th style={{ padding: "6px 4px" }}>Nome</th>
                <th style={{ padding: "6px 4px" }}>Classe</th>
                <th style={{ padding: "6px 4px" }}>Nível</th>
              </tr>
            </thead>
            <tbody>
              {personagens.map((character) => (
                <tr key={character.id} style={{ borderTop: "1px solid var(--border)" }}>
                  <td style={{ padding: "6px 4px" }}>{character.slot}</td>
                  <td style={{ padding: "6px 4px" }}>{character.name}</td>
                  <td style={{ padding: "6px 4px" }}>{fmtClass(character.class)}</td>
                  <td style={{ padding: "6px 4px" }}>{character.level}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section>
        <h2 style={{ fontSize: 16, marginBottom: 8 }}>Ranking</h2>
        {ranking === null ? (
          <p style={{ color: "var(--muted)", fontSize: 13 }}>Ranking indisponível.</p>
        ) : (
          <ol style={{ margin: 0, paddingLeft: 20, fontSize: 13, display: "grid", gap: 4 }}>
            {ranking.map((entry) => (
              <li key={`${entry.rank}-${entry.name}`}>
                {entry.name} · {fmtClass(entry.class)} · nível {entry.level}
              </li>
            ))}
          </ol>
        )}
      </section>

      <section>
        <h2 style={{ fontSize: 16, marginBottom: 8 }}>Recuperar acesso</h2>
        <p style={{ color: "var(--muted)", fontSize: 13, marginTop: 0 }}>
          A solicitação é registrada para a equipe; a resposta não confirma se a conta existe.
        </p>
        <RecoveryForm />
      </section>
    </main>
  );
}
