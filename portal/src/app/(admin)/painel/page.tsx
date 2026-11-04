import { Stat } from "@/components/Stat";
import { fmtCoin } from "@/lib/format";
import { getItemCatalog } from "@/server/catalog";

export const dynamic = "force-dynamic";

export const metadata = { title: "Visão geral · WYD2" };

export default async function PainelPage() {
  let itemCount: number | null = null;
  let catalogVersion = "";
  try {
    const catalog = await getItemCatalog();
    itemCount = catalog.items.length;
    catalogVersion = catalog.version;
  } catch {
    itemCount = null;
  }

  return (
    <>
      <h1 className="ui-page-title">Visão geral</h1>
      <p className="ui-page-sub">
        Estado do servidor lido do web-api. Números ainda sem RPC aparecem como “—”,
        com a issue que os traz.
      </p>

      <div className="ui-stats">
        <Stat
          label="Itens no catálogo"
          value={fmtCoin(itemCount ?? 0)}
          pending={itemCount === null}
          hint={
            itemCount === null
              ? "web-api indisponível"
              : `ItemCatalogService · versão ${catalogVersion.slice(0, 8)}`
          }
        />
        <Stat
          label="Templates de mob"
          value=""
          pending
          hint="issue 25 · MobTemplateAdminService"
        />
        <Stat label="NPCs definidos" value="" pending hint="issue 23 · NpcAdminService" />
        <Stat label="Contas" value="" pending hint="issue 32 · contas e papéis" />
      </div>

      <div className="ui-card">
        <h2 style={{ fontSize: 14, margin: "0 0 6px" }}>Como este portal grava</h2>
        <p style={{ margin: 0, color: "var(--muted)", fontSize: 13 }}>
          Nada aqui escreve em <code>Release/</code> nem no estado vivo do personagem.
          Conteúdo é overlay no Postgres, aplicado pelo tmserver no boot; concessões a
          jogadores vão por <code>delivery_queue</code>. Toda RPC admin revalida o papel
          da conta no servidor — o papel mostrado no cabeçalho é só a dica da UI.
        </p>
      </div>
    </>
  );
}
