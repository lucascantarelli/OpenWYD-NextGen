import { PackageSearch } from "lucide-react";

import { getCatalog, type Catalog } from "@/server/catalog";

import { ItemBrowser } from "./item-browser";

export const dynamic = "force-dynamic";

export const metadata = { title: "Catálogo de itens · WYD2" };

/**
 * Catálogo somente leitura dos itens carregados do banco pelo gateway.
 *
 * Dois estados de exceção são distintos de propósito: gateway fora (não há como
 * ler) e catálogo vazio (gateway no ar sem conteúdo carregado).
 */
export default async function ItemsPage() {
  let catalog: Catalog | null = null;
  try {
    catalog = await getCatalog();
  } catch {
    catalog = null;
  }

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
              <p>O gateway não respondeu à leitura do catálogo. Tente outra vez em instantes.</p>
            </div>
          </div>
        </div>
      )}

      {catalog !== null && catalog.items.length === 0 && (
        <div className="card">
          <div className="card__body">
            <div className="empty">
              <PackageSearch size={22} strokeWidth={1.6} aria-hidden="true" />
              <strong>O gateway está no ar, mas sem catálogo</strong>
              <p>
                O conteúdo é cargado do Postgres no boot do servidor. Sem `item_catalog`
                preenchido, a lista fica vazia.
              </p>
            </div>
          </div>
        </div>
      )}

      {catalog !== null && catalog.items.length > 0 && (
        <ItemBrowser items={catalog.items} catalogVersion={catalog.catalogVersion} />
      )}
    </>
  );
}
