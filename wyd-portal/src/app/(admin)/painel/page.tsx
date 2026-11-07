import Link from "next/link";

import { Stat } from "@/components/Stat";
import { fmtBRL, fmtCount } from "@/lib/money";
import { getItemCatalog } from "@/server/catalog";
import { getDailyRewards } from "@/server/daily";
import { getShopItems } from "@/server/donate";
import { listMobTemplates } from "@/server/mobs";
import { listNpcs } from "@/server/npcs";
import { getRevenueSummary } from "@/server/revenue";
import { getSession } from "@/server/session";

export const dynamic = "force-dynamic";

export const metadata = { title: "Visão geral · WYD2" };

/** Conta itens de uma leitura, devolvendo `null` quando o serviço está fora. */
async function countOf<T>(load: () => Promise<T[]>): Promise<number | null> {
  try {
    return (await load()).length;
  } catch {
    return null;
  }
}

/**
 * Visão geral (issue 18/20, atualizada com as telas entregues).
 *
 * Só leitura e cada bloco falha sozinho: um serviço fora do ar deixa o cartão em
 * "—" com o motivo, sem derrubar a página inteira. Receita aparece apenas para
 * `admin`, porque `/faturamento` também é.
 */
export default async function PainelPage() {
  const session = await getSession();

  let items: number | null = null;
  let catalogVersion = "";
  try {
    const catalog = await getItemCatalog();
    items = catalog.items.length;
    catalogVersion = catalog.version;
  } catch {
    items = null;
  }

  const mobs = await countOf(listMobTemplates);
  const npcs = await countOf(listNpcs);
  const daily = await countOf(getDailyRewards);

  let shop: { total: number; enabled: number } | null = null;
  try {
    const offers = await getShopItems();
    shop = { total: offers.length, enabled: offers.filter((offer) => offer.enabled).length };
  } catch {
    shop = null;
  }

  let revenue: { gross: number; paid: number; buyers: number } | null = null;
  if (session?.role === "admin") {
    try {
      const to = Math.floor(Date.now() / 1000);
      const summary = await getRevenueSummary({
        from: to - 30 * 86_400,
        to,
        bucket: 0,
        accountId: 0,
      });
      revenue = {
        gross: summary.totals.gross_cents,
        paid: summary.totals.paid_orders,
        buyers: summary.totals.distinct_buyers,
      };
    } catch {
      revenue = null;
    }
  }

  return (
    <>
      <h1 className="ui-page-title">Visão geral</h1>
      <p className="ui-page-sub">
        Estado do servidor lido do web-api. Cada número vem de uma RPC de leitura; o que ainda não
        tem contrato aparece como “—”, com a issue que o traz.
      </p>

      <div className="ui-stats">
        <Stat
          label="Itens no catálogo"
          value={fmtCount(items ?? 0)}
          pending={items === null}
          hint={
            items === null
              ? "web-api indisponível"
              : `ItemCatalogService · versão ${catalogVersion.slice(0, 8)}`
          }
        />
        <Stat
          label="Templates de mob"
          value={fmtCount(mobs ?? 0)}
          pending={mobs === null}
          hint={mobs === null ? "MobTemplateAdminService indisponível" : "arquivos de tmserver/content/TMsrv/run/npc"}
        />
        <Stat
          label="NPCs e mercadores"
          value={fmtCount(npcs ?? 0)}
          pending={npcs === null}
          hint={npcs === null ? "NpcAdminService indisponível" : "definidos no banco"}
        />
        <Stat
          label="Ofertas de donate"
          value={fmtCount(shop?.total ?? 0)}
          pending={shop === null}
          hint={
            shop === null
              ? "DonateAdminService indisponível"
              : `${fmtCount(shop.enabled)} na vitrine do jogador`
          }
        />
      </div>

      <div className="ui-stats">
        <Stat
          label="Recompensas diárias"
          value={fmtCount(daily ?? 0)}
          pending={daily === null}
          hint={daily === null ? "DailyRewardAdminService indisponível" : "ofertas no catálogo"}
        />
        {session?.role === "admin" && (
          <>
            <Stat
              label="Receita paga (30 dias)"
              value={fmtBRL(revenue?.gross ?? 0)}
              pending={revenue === null}
              hint={
                revenue === null
                  ? "DonateRevenueAdminService indisponível"
                  : `${fmtCount(revenue.paid)} pedidos pagos · ver /faturamento`
              }
            />
            <Stat
              label="Compradores (30 dias)"
              value={fmtCount(revenue?.buyers ?? 0)}
              pending={revenue === null}
              hint="contas distintas que pagaram"
            />
          </>
        )}
        <Stat
          label="Contas e papéis"
          value=""
          pending
          hint="issue 32 · sem RPC de contas"
        />
      </div>

      <div className="ui-card" style={{ marginBottom: 20 }}>
        <h2 style={{ fontSize: 14, margin: "0 0 8px" }}>Atalhos</h2>
        <div style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-2)" }}>
          <Link className="ui-btn" href="/loja">
            Loja de donate
          </Link>
          {session?.role === "admin" && (
            <Link className="ui-btn" href="/faturamento">
              Faturamento
            </Link>
          )}
          <Link className="ui-btn" href="/eventos">
            Eventos globais
          </Link>
          <Link className="ui-btn" href="/logs">
            Logs dos containers
          </Link>
          <Link className="ui-btn" href="/config">
            Rates e flags
          </Link>
        </div>
      </div>

      <div className="ui-card">
        <h2 style={{ fontSize: 14, margin: "0 0 6px" }}>Como este portal grava</h2>
        <p style={{ margin: 0, color: "var(--muted)", fontSize: 13 }}>
          Nada aqui escreve em <code>tmserver/content/</code> nem no estado vivo do personagem. Conteúdo é
          overlay no Postgres, aplicado pelo tmserver no boot; concessões a jogadores vão por{" "}
          <code>delivery_queue</code>. Toda RPC admin revalida o papel da conta no servidor — o papel
          mostrado no cabeçalho é só a dica da UI. Pedidos e receita são leitura pura: o
          faturamento não altera saldo, pedido ou auditoria.
        </p>
      </div>
    </>
  );
}
