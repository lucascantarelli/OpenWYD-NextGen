import { PackageSearch } from "lucide-react";
import Link from "next/link";
import { Fragment } from "react";

import { Badge } from "@/components/Badge";
import { findCatalogItem, getCatalog, type Catalog } from "@/server/catalog";
import { listItemPrices, type ItemPriceWire } from "@/server/config";

import { atributosDaFicha, requisitosDaFicha } from "./ficha";
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

  const atributos = ficha ? atributosDaFicha(ficha) : [];
  const requisitos = ficha ? requisitosDaFicha(ficha) : [];
  const efeitos = ficha?.effect_names ?? [];

  // Preço de NPC é configuração do dono do servidor (`item_price`), lida do gateway.
  // Ausência de linha = "sem preço definido", nunca zero.
  let prices: ItemPriceWire[] | null = null;
  if (ficha) {
    try {
      prices = await listItemPrices(ficha.item_index);
    } catch {
      prices = null;
    }
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
                O catálogo é carregado no boot do servidor de jogo. Assim que ele subir com o
                catálogo, os itens aparecem aqui.
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
                Item como está no catálogo do servidor. Para mandar este item a um jogador,
                use o <Link href="/correio">Correio</Link>.
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
          </div>
          <div className="card__body">
            <h3 className="card__subtitle">Atributos do template</h3>
            {atributos.length === 0 ? (
              <p className="muted small" style={{ margin: 0 }}>
                Este item não traz atributo no template: só índice e nome.
              </p>
            ) : (
              <dl className="kv">
                {atributos.map((linha) => (
                  <Fragment key={linha.rotulo}>
                    <dt>{linha.rotulo}</dt>
                    <dd>{linha.valor}</dd>
                  </Fragment>
                ))}
              </dl>
            )}
          </div>
          <div className="card__body">
            <h3 className="card__subtitle">Requisitos para usar</h3>
            {requisitos.length === 0 ? (
              <p className="muted small" style={{ margin: 0 }}>
                Nenhum requisito declarado — qualquer personagem pode usar.
              </p>
            ) : (
              <dl className="kv">
                {requisitos.map((linha) => (
                  <Fragment key={linha.rotulo}>
                    <dt>{linha.rotulo}</dt>
                    <dd>{linha.valor}</dd>
                  </Fragment>
                ))}
              </dl>
            )}
          </div>
          <div className="card__body">
            <h3 className="card__subtitle">Efeitos do template</h3>
            {efeitos.length === 0 ? (
              <p className="muted small" style={{ margin: 0 }}>
                Nenhum efeito nomeado no template deste item.
              </p>
            ) : (
              <div className="chips">
                {efeitos.map((efeito) => (
                  <span key={efeito} className="chip mono">
                    {efeito}
                  </span>
                ))}
              </div>
            )}
          </div>
          <div className="card__body card__body--flush">
            {prices === null ? (
              <p className="muted small" style={{ padding: 20, margin: 0 }}>
                Não foi possível ler os preços de loja agora.
              </p>
            ) : prices.length === 0 ? (
              <p className="muted small" style={{ padding: 20, margin: 0 }}>
                Nenhum mercador vende este item ainda. Defina o preço em{" "}
                <Link href="/configuracoes?aba=precos">Configurações › Preços de loja</Link>.
              </p>
            ) : (
              <div className="ui-table-scroll">
                <table className="ui-table" aria-label="Preços de NPC">
                  <thead>
                    <tr>
                      <th>Mercador</th>
                      <th className="ui-align-right">Compra</th>
                      <th className="ui-align-right">Venda</th>
                    </tr>
                  </thead>
                  <tbody>
                    {prices.map((price) => (
                      <tr key={`${price.item_index}-${price.npc_id}`}>
                        <td>
                          <span className="mono">#{price.npc_id}</span>{" "}
                          {price.npc_slug || <span className="muted">sem slug</span>}
                        </td>
                        <td className="ui-align-right">{price.buy_price.toLocaleString("pt-BR")}</td>
                        <td className="ui-align-right">
                          {price.sell_price.toLocaleString("pt-BR")}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
          <div className="card__foot">
            Valores lidos do template do item no catálogo do servidor. Preço em ouro, alcance em
            tiles; “—” aparece quando o template não declara o campo.
          </div>
        </div>
      )}

      {catalog !== null && catalog.items.length > 0 && (
        <ItemBrowser items={catalog.items} catalogVersion={catalog.catalogVersion} />
      )}
    </>
  );
}
