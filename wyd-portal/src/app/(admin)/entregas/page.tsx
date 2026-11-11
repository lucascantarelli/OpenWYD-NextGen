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
  // Chave explícita na URL: reenviar depois de um refresh usa a MESMA chave.
  const rawKey = params.key;
  const keyParam = (Array.isArray(rawKey) ? rawKey[0] : rawKey ?? "").trim();
  const initialKey =
    keyParam.length >= 8 && keyParam.length <= 64 ? keyParam : undefined;

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

  const selectedName =
    characters.find((character) => character.id === selected)?.name ?? `#${selected}`;

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Operação</p>
          <h1 className="page-head__title">Entregas</h1>
          <p className="page-head__sub">
            Ouro concedido ao personagem entra na fila do servidor e cai no próximo ciclo.
            Reenviar o mesmo pedido não credita duas vezes.
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

          <section className="card">
            <div className="card__head">
              <div>
                <h2 className="card__title">Personagens</h2>
                <p className="card__sub">
                  {term
                    ? `${characters.length} resultado(s) para “${term}”.`
                    : `${characters.length} personagem(ns). Digite um nome para achar um específico.`}
                </p>
              </div>
            </div>
            <div className="card__body card__body--flush">
              <div className="ui-table-scroll">
                <table className="ui-table" aria-label="Personagens">
                  <thead>
                    <tr>
                      <th>Personagem</th>
                      <th className="ui-align-right">Conta</th>
                      <th className="ui-align-right">Nível</th>
                      <th className="ui-align-right">Ouro</th>
                      <th className="ui-align-right">Conceder ouro</th>
                      <th />
                    </tr>
                  </thead>
                  <tbody>
                    {characters.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="ui-table__empty">
                          Nenhum personagem encontrado.
                        </td>
                      </tr>
                    ) : (
                      characters.map((character) => (
                        <tr key={character.id} data-testid={`char-${character.id}`}>
                          <td>
                            <strong>{character.name}</strong>
                            <span className="muted small"> #{character.id}</span>
                          </td>
                          <td className="ui-align-right mono small">#{character.account_id}</td>
                          <td className="ui-align-right">{character.level}</td>
                          <td className="ui-align-right" data-testid={`gold-${character.id}`}>
                            {fmtInt(character.gold)}
                          </td>
                          <td className="ui-align-right">
                            <GrantForm characterId={character.id} initialKey={initialKey} />
                          </td>
                          <td className="ui-align-right">
                            <Link
                              className="small"
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
          </section>

          {selected > 0 && (
            <section className="card">
              <div className="card__head">
                <div>
                  <h2 className="card__title">
                    <Gift size={15} strokeWidth={1.8} aria-hidden="true" /> Fila de {selectedName}
                  </h2>
                  <p className="card__sub">
                    Pedidos registrados para este personagem. “Na fila” ainda não foi aplicado;
                    “entregue” já caiu no personagem.
                  </p>
                </div>
              </div>
              <div className="card__body card__body--flush">
                <div className="ui-table-scroll">
                  <table className="ui-table" aria-label="Fila de entregas">
                    <thead>
                      <tr>
                        <th>Pedido</th>
                        <th>Tipo</th>
                        <th className="ui-align-right">Ouro</th>
                        <th>Estado</th>
                        <th>Identificador</th>
                      </tr>
                    </thead>
                    <tbody>
                      {grants.length === 0 ? (
                        <tr>
                          <td colSpan={5} className="ui-table__empty">
                            Nenhuma entrega registrada para este personagem.
                          </td>
                        </tr>
                      ) : (
                        grants.map((grant) => (
                          <tr key={grant.id} data-testid={`grant-${grant.id}`}>
                            <td className="mono small">#{grant.id}</td>
                            <td>{grant.kind}</td>
                            <td className="ui-align-right">{fmtInt(grant.gold)}</td>
                            <td>
                              <Badge tone={grant.status === "claimed" ? "ok" : "accent"}>
                                {STATUS_LABEL[grant.status] ?? grant.status}
                              </Badge>
                            </td>
                            <td className="mono small muted">
                              {grant.idempotency_key ?? "—"}
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
            </section>
          )}
        </>
      )}
    </>
  );
}
