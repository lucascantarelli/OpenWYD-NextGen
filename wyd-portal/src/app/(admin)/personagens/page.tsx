import Link from "next/link";

import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import { classLabel, type CharacterSummary } from "@/lib/character";
import { AdminError } from "@/server/errors";
import { searchCharacters } from "@/server/characters";

export const dynamic = "force-dynamic";

export const metadata = { title: "Personagens · WYD2" };

/**
 * Busca de personagens (issue 43).
 *
 * Fase de leitura: a tela mostra o estado **frio** (Postgres) — nível, EXP, gold,
 * classe e conta. Concessão de item/gold e edição de campo entram depois, por
 * `delivery_queue` e por um editor offline; nada aqui escreve no save.
 */
export default async function CharactersPage({
  searchParams,
}: {
  searchParams: Promise<{ q?: string }>;
}) {
  const { q = "" } = await searchParams;
  const query = q.trim();

  let characters: CharacterSummary[] = [];
  let error: string | null = null;

  if (query.length >= 2) {
    try {
      characters = await searchCharacters(query);
    } catch (err) {
      error =
        err instanceof AdminError
          ? err.message
          : "não foi possível buscar personagens no web-api";
    }
  }

  return (
    <>
      <h1 className="ui-page-title">Personagens</h1>
      <p className="ui-page-sub">
        Estado frio em Postgres. A web não escreve o personagem vivo: concessões vão por{" "}
        <code>delivery_queue</code> e edições exigem o personagem offline.
      </p>

      <form className="ui-filters" action="/personagens" method="get" style={{ gap: 8 }}>
        <input
          className="ui-input"
          type="search"
          name="q"
          defaultValue={query}
          placeholder="nome do personagem ou da conta (mín. 2 letras)"
          minLength={2}
          required
          style={{ maxWidth: 420 }}
        />
        <button className="ui-btn ui-btn--primary" type="submit">
          buscar
        </button>
      </form>

      {error && (
        <div className="ui-card ui-empty" style={{ marginBottom: 12 }}>
          <strong>Não deu para buscar</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      )}

      {!error && query.length >= 2 && characters.length === 0 && (
        <div className="ui-card ui-empty">
          <strong>Nenhum personagem encontrado</strong>
          <p>Nada com “{query}” no nome do personagem ou da conta.</p>
        </div>
      )}

      {!error && characters.length > 0 && (
        <>
          <p className="ui-field__hint" style={{ marginBottom: 10 }}>
            {characters.length} personagens ·{" "}
            <Badge tone="neutral">somente leitura</Badge>
          </p>
          <div className="ui-table-wrap">
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Personagem</th>
                    <th>Conta</th>
                    <th>Classe</th>
                    <th>Nível</th>
                    <th>EXP</th>
                    <th>Gold</th>
                    <th>Slot</th>
                  </tr>
                </thead>
                <tbody>
                  {characters.map((character) => (
                    <tr key={character.id}>
                      <td>
                        <Link href={`/personagens/${character.id}`} className="ui-mono">
                          {character.name}
                        </Link>
                      </td>
                      <td className="ui-mono">{character.account_name}</td>
                      <td>{classLabel(character.class)}</td>
                      <td>{character.level}</td>
                      <td>{character.exp}</td>
                      <td>{character.coin}</td>
                      <td>{character.slot}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </>
      )}

      {!error && query.length < 2 && (
        <p className="ui-field__hint">
          Digite pelo menos duas letras. A busca cobre nome do personagem e nome da conta.
        </p>
      )}
    </>
  );
}
