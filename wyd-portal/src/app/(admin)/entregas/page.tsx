import { Gift, Search, ShieldAlert } from "lucide-react";
import Link from "next/link";

import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import {
  listCharacterGrants,
  searchCharacters,
  type CharacterWire,
  type GrantRowWire,
} from "@/server/deliveries";
import { AdminError } from "@/server/errors";

import { GrantForm } from "./grant-form";

export const dynamic = "force-dynamic";

export const metadata = { title: "Entregas · WYD2" };

const fmtInt = (value: number): string =>
  value.toLocaleString("pt-BR", { maximumFractionDigits: 0 });

const STATUS_LABEL: Record<string, string> = {
  pending: "na fila",
  claimed: "entregue",
};

/**
 * Entregas: escolhe o personagem, enfileira ouro e confere a fila.
 *
 * A leitura vem do gateway (`character_admin/*`); a escrita é o grant durável
 * (`grants/request`). O ouro exibido é o do banco **antes** da entrega — depois
 * do dreno do servidor, recarregar a tela mostra o valor creditado. Nada aqui
 * aplica ouro: o portal só enfileira.
 */
export default async function EntregasPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const params = await searchParams;
  const rawTerm = params.term;
  const term = (Array.isArray(rawTerm) ? rawTerm[0] ?? "" : rawTerm ?? "").trim();
  const rawChar = params.char;
  const selected = Number(Array.isArray(rawChar) ? rawChar[0] : rawChar ?? "");

  let characters: CharacterWire[] = [];
  let grants: GrantRowWire[] = [];
  let error: string | null = null;
  try {
    characters = await searchCharacters(term);
    if (Number.isInteger(selected) && selected > 0) {
      grants = await listCharacterGrants(selected);
    }
  } catch (err) {
    error = err instanceof AdminError ? err.message : "não foi possível ler as entregas agora";
  }

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Operação</p>
          <h1 className="page-head__title">Entregas</h1>
          <p className="page-head__sub">
            Concede ouro a um personagem pela fila durável: a chave de idempotência impede entrega
            dupla e o servidor de jogo aplica o valor.
          </p>
        </div>
        <div className="page-head__actions">
          <Badge tone={error ? "warn" : "ok"}>{error ? "indisponível" : "no ar"}</Badge>
        </div>
      </div>

      {error ? (
        <div className="card">
          <div className="card__body">
            <div className="empty">
              <ShieldAlert size={22} strokeWidth={1.6} aria-hidden="true" />
              <strong>Entregas indisponíveis</strong>
              <p>{error}</p>
              <RetryButton />
            </div>
          </div>
        </div>
      ) : (
        <>
          <form className="ui-filters" action="/entregas" method="get" role="search">
            <div className="search">
              <Search size={16} strokeWidth={1.8} aria-hidden="true" />
              <input
                className="ui-input"
                type="search"
                name="term"
                defaultValue={term}
                placeholder="Nome do personagem (vazio = todos)"
                aria-label="Nome do personagem"
              />
            </div>
            <button className="ui-btn ui-btn--primary" type="submit">
              Buscar
            </button>
          </form>

          <div className="card">
            <div className="card__body">
              <table className="table" aria-label="Personagens">
                <thead>
                  <tr>
                    <th>Personagem</th>
                    <th>Conta</th>
                    <th>Nível</th>
                    <th>Ouro</th>
                    <th>Concessão</th>
                  </tr>
                </thead>
                <tbody>
                  {characters.length === 0 ? (
                    <tr>
                      <td colSpan={5}>nenhum personagem encontrado</td>
                    </tr>
                  ) : (
                    characters.map((character) => (
                      <tr key={character.id} data-testid={`char-${character.id}`}>
                        <td>
                          <strong>{character.name}</strong>
                          <span className="muted"> #{character.id}</span>
                        </td>
                        <td>#{character.account_id}</td>
                        <td>{character.level}</td>
                        <td data-testid={`gold-${character.id}`}>{fmtInt(character.gold)}</td>
                        <td>
                          <GrantForm characterId={character.id} />
                          <Link
                            className="muted"
                            href={`/entregas?term=${encodeURIComponent(term)}&char=${character.id}`}
                          >
                            ver fila
                          </Link>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {selected > 0 && (
            <div className="card">
              <div className="card__body">
                <h2>
                  <Gift size={18} strokeWidth={1.7} aria-hidden="true" /> Fila do personagem #
                  {selected}
                </h2>
                <table className="table" aria-label="Fila de entregas">
                  <thead>
                    <tr>
                      <th>#</th>
                      <th>Tipo</th>
                      <th>Ouro</th>
                      <th>Estado</th>
                      <th>Chave</th>
                    </tr>
                  </thead>
                  <tbody>
                    {grants.length === 0 ? (
                      <tr>
                        <td colSpan={5}>sem entregas registradas</td>
                      </tr>
                    ) : (
                      grants.map((grant) => (
                        <tr key={grant.id} data-testid={`grant-${grant.id}`}>
                          <td>#{grant.id}</td>
                          <td>{grant.kind}</td>
                          <td>{fmtInt(grant.gold)}</td>
                          <td>
                            <Badge tone={grant.status === "claimed" ? "ok" : "accent"}>
                              {STATUS_LABEL[grant.status] ?? grant.status}
                            </Badge>
                          </td>
                          <td>
                            <code>{grant.idempotency_key ?? "—"}</code>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}
    </>
  );
}
