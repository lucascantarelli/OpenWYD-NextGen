import { Mail, Search, ShieldAlert } from "lucide-react";
import Link from "next/link";

import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import { getCatalog } from "@/server/catalog";
import {
  listCharacterGrants,
  searchCharacters,
  type CharacterWire,
  type GrantRowWire,
} from "@/server/deliveries";
import { AdminError } from "@/server/errors";

import { GrantItemForm } from "./grant-item-form";

export const dynamic = "force-dynamic";

export const metadata = { title: "Correio · WYD2" };

const fmtInt = (value: number): string =>
  value.toLocaleString("pt-BR", { maximumFractionDigits: 0 });

const STATUS_LABEL: Record<string, string> = {
  pending: "na fila",
  claimed: "entregue",
};

/**
 * Correio (PORTAL-03, lado staff): concede itens pela fila durável e confere o estado.
 *
 * A escrita é `character_admin/grant_item` (idempotente por chave); a entrega
 * acontece no dreno do servidor, como o ouro em `/entregas` — esta tela não
 * duplica aquela: ouro continua em `/entregas`, item entra aqui. A caixa de
 * entrada do jogador (área pública) não existe nesta entrega: a fila só é
 * legível por staff (`character_admin/*` exige staff), então não há inbox
 * honesto a mostrar sem rota nova no gateway.
 */
export default async function CorreioPage({
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
  let itemNames: string[] = [];
  let error: string | null = null;
  try {
    characters = await searchCharacters(term);
    if (Number.isInteger(selected) && selected > 0) {
      grants = await listCharacterGrants(selected);
    }
    try {
      const catalog = await getCatalog();
      itemNames = catalog.items.slice(0, 500).map((item) => String(item.item_index));
    } catch {
      itemNames = [];
    }
  } catch (err) {
    error = err instanceof AdminError ? err.message : "não foi possível ler o correio agora";
  }

  const pending = grants.filter((grant) => grant.status === "pending").length;
  const claimed = grants.filter((grant) => grant.status === "claimed").length;

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Operação</p>
          <h1 className="page-head__title">Correio</h1>
          <p className="page-head__sub">
            Concede itens a um personagem pela fila durável: a chave de idempotência impede
            entrega dupla e o servidor de jogo aplica no dreno. Ouro continua em{" "}
            <Link href="/entregas">Entregas</Link>.
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
              <strong>Correio indisponível</strong>
              <p>{error}</p>
              <RetryButton />
            </div>
          </div>
        </div>
      ) : (
        <>
          <form className="ui-filters" action="/correio" method="get" role="search">
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
                    <th>Item</th>
                  </tr>
                </thead>
                <tbody>
                  {characters.length === 0 ? (
                    <tr>
                      <td colSpan={4}>nenhum personagem encontrado</td>
                    </tr>
                  ) : (
                    characters.map((character) => (
                      <tr key={character.id} data-testid={`correio-${character.id}`}>
                        <td>
                          <strong>{character.name}</strong>
                          <span className="muted"> #{character.id}</span>
                        </td>
                        <td>#{character.account_id}</td>
                        <td>{character.level}</td>
                        <td>
                          <GrantItemForm characterId={character.id} itemNames={itemNames} />
                          <Link
                            className="muted"
                            href={`/correio?term=${encodeURIComponent(term)}&char=${character.id}`}
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
              <div className="card__head">
                <div>
                  <h2 className="card__title">
                    <Mail size={16} strokeWidth={1.8} aria-hidden="true" /> Fila do personagem #
                    {selected}
                  </h2>
                  <p className="card__sub">
                    {pending} na fila · {claimed} entregue(s) · {fmtInt(grants.length)} registro(s).
                    Origem rastreável pela chave de idempotência.
                  </p>
                </div>
              </div>
              <div className="card__body card__body--flush">
                {grants.length === 0 ? (
                  <p className="muted small" style={{ padding: 20, margin: 0 }}>
                    sem entregas registradas
                  </p>
                ) : (
                  <div className="ui-table-scroll">
                    <table className="ui-table">
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
                        {grants.map((grant) => (
                          <tr key={grant.id}>
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
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>
          )}
        </>
      )}
    </>
  );
}
