import { PackageSearch } from "lucide-react";
import Link from "next/link";

import { Badge } from "@/components/Badge";
import { findCatalogItem, getCatalog, type Catalog } from "@/server/catalog";

import { ItemBrowser } from "./item-browser";

export const dynamic = "force-dynamic";

export const metadata = { title: "Catálogo de itens · WYD2" };

/**
 * Catálogo somente leitura dos itens carregados do banco pelo gateway.
 *
 * Dois estados de exceção são distintos de propósito: gateway fora (não há como
 * ler) e catálogo vazio (gateway no ar sem conteúdo carregado).
 */
export default async function ItemsPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const params = await searchParams;
  const rawItem = params.item;
  const itemIndex = Number(Array.isArray(rawItem) ? rawItem[0] ?? "" : rawItem ?? "");

  let catalog: Catalog | null = null;
  try {
    catalog = await getCatalog();
  } catch {
    catalog = null;
  }

  const ficha =
    catalog && Number.isInteger(itemIndex) ? findCatalogItem(catalog.items, itemIndex) : null;

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Conteúdo</p>
          <h1 className="page-head__title">Catálogo de itens</h1>
          <p className="page-head__sub">
            Índice, nome de catálogo e nome de exibição, direto do conteúdo carregado no servidor.
            Leitura pura: nada aqui altera item, preço ou inventário.
          </p>
        </div>
      </div>

      {catalog === null && (
        <div className="card">
          <div className="card__body">
            <div className="empty">
              <PackageSearch size={22} strokeWidth={1.6} aria-hidden="true" />
              <strong>Catálogo indisponível</strong>
              <p>O servidor não respondeu à leitura do catálogo. Tente outra vez em instantes.</p>
            </div>
          </div>
        </div>
      )}

      {catalog !== null && catalog.items.length === 0 && (
        <div className="card">
          <div className="card__body">
            <div className="empty">
              <PackageSearch size={22} strokeWidth={1.6} aria-hidden="true" />
              <strong>Nenhum catálogo carregado no servidor</strong>
              <p>
                O conteúdo é cargado do Postgres no boot do servidor. Sem `item_catalog`
                preenchido, a lista fica vazia.
              </p>
            </div>
          </div>
        </div>
      )}

      {ficha && (
        <div className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">Ficha #{ficha.item_index}</h2>
              <p className="card__sub">
                Dados do catálogo carregado no servidor. Origem rastreável: concessões deste
                item entram na fila com chave de idempotência (ver{" "}
                <Link href="/correio">Correio</Link>).
              </p>
            </div>
            <div className="page-head__actions">
              <Badge tone="ok">no catálogo</Badge>
            </div>
          </div>
          <div className="card__body">
            <dl className="kv">
              <dt>Índice</dt>
              <dd className="mono">{ficha.item_index}</dd>
              <dt>Nome no catálogo</dt>
              <dd className="mono">{ficha.name}</dd>
              <dt>Nome de exibição</dt>
              <dd>{ficha.display_name || "—"}</dd>
            </dl>
            <p className="muted small" style={{ marginBottom: 0 }}>
              Atributos, requisitos e preço de loja só entram aqui quando existirem no payload do
              gateway — a tabela não mostra coluna que o contrato não fornece.
            </p>
          </div>
        </div>
      )}

      {catalog !== null && catalog.items.length > 0 && (
        <ItemBrowser items={catalog.items} catalogVersion={catalog.catalogVersion} />
      )}
    </>
  );
}
