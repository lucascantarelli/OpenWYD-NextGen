import { DropExplorer } from "@/components/drop/drop-explorer";
import { DropsNav } from "@/components/drop/drops-nav";

export const dynamic = "force-dynamic";

export const metadata = { title: "Drops · WYD2" };

/**
 * Relatório de drops (issue 26).
 *
 * Duas visões do mesmo dado (`Carry[]` de cada template): quem dropa o item e o
 * que o mob dropa. A página é só o cabeçalho — a busca vive no componente, que
 * consulta sob demanda para não trafegar o catálogo inteiro.
 */
export default function DropsPage() {
  return (
    <>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            Drops
          </h1>
          <p className="ui-page-sub" style={{ margin: "var(--space-1) 0 0" }}>
            O que cada template <em>permite</em> dropar, lido de{" "}
            <code>tmserver/content/TMsrv/run/npc/</code> no boot do web-api. Não é a chance real: o divisor
            por slot está em código, em <code>loot.go</code>.
          </p>
        </div>
      </div>

      <DropsNav active="relatorio" />

      <DropExplorer />
    </>
  );
}
