import { Coins, Receipt, ShieldAlert } from "lucide-react";

import { Badge } from "@/components/Badge";
import { Stat } from "@/components/Stat";
import { getEconomySummary, type EconomyView } from "@/server/economy";
import { AdminError } from "@/server/errors";

export const dynamic = "force-dynamic";

export const metadata = { title: "Economia · WYD2" };

const fmtInt = (value: number): string =>
  value.toLocaleString("pt-BR", { maximumFractionDigits: 0 });

const KIND_LABEL: Record<string, string> = {
  item: "item",
  gold: "ouro",
  level: "nível",
};

/**
 * Economia: números reais, cada um com a fonte ao lado.
 *
 * Imposto vem do livro-caixa (`city_tax_ledger`, uma linha por recolhimento) e
 * concessões da fila de entrega (`delivery_queue`). Tabela vazia é "sem
 * movimento registrado" — o painel não estima nem inventa valor.
 */
export default async function EconomiaPage() {
  let summary: EconomyView = { taxByCity: [], grants: [] };
  let error: string | null = null;

  try {
    summary = await getEconomySummary();
  } catch (err) {
    error =
      err instanceof AdminError ? err.message : "economia indisponível no momento";
  }

  const taxTotal = summary.taxByCity.reduce((acc, row) => acc + row.total, 0);
  const goldGranted = summary.grants.reduce((acc, row) => acc + row.gold, 0);
  const grantsTotal = summary.grants.reduce((acc, row) => acc + row.count, 0);

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Governança</p>
          <h1 className="page-head__title">Economia</h1>
          <p className="page-head__sub">
            Imposto arrecadado e concessões de staff, sempre a partir da fonte auditada.
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
              <strong>Economia indisponível</strong>
              <p>{error}</p>
            </div>
          </div>
        </div>
      ) : (
        <>
          <div className="stats">
            <Stat
              label="Imposto arrecadado"
              value={fmtInt(taxTotal)}
              hint="soma de city_tax_ledger"
              icon={Coins}
            />
            <Stat
              label="Cidades com movimento"
              value={fmtInt(summary.taxByCity.length)}
              hint="linhas do livro-caixa"
              icon={Receipt}
            />
            <Stat
              label="Concessões enfileiradas"
              value={fmtInt(grantsTotal)}
              hint="linhas de delivery_queue"
            />
            <Stat
              label="Ouro concedido"
              value={fmtInt(goldGranted)}
              hint="soma de gold_amount"
            />
          </div>

          <div className="monitor-grid">
            <section className="card">
              <div className="card__head">
                <div>
                  <h2 className="card__title">Imposto por cidade</h2>
                  <p className="card__sub">Fonte: `city_tax_ledger` (migration 0010).</p>
                </div>
              </div>
              <div className="card__body card__body--flush">
                {summary.taxByCity.length === 0 ? (
                  <p className="muted small" style={{ padding: 20, margin: 0 }}>
                    Nenhum recolhimento registrado. A tabela existe e está vazia — não há número a
                    mostrar.
                  </p>
                ) : (
                  <div className="ui-table-scroll">
                    <table className="ui-table">
                      <thead>
                        <tr>
                          <th>Cidade</th>
                          <th className="ui-align-right">Imposto</th>
                        </tr>
                      </thead>
                      <tbody>
                        {summary.taxByCity.map((row) => (
                          <tr key={row.city}>
                            <td className="mono">{row.city}</td>
                            <td className="ui-align-right">{fmtInt(row.total)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </section>

            <section className="card">
              <div className="card__head">
                <div>
                  <h2 className="card__title">Concessões por tipo</h2>
                  <p className="card__sub">Fonte: `delivery_queue` (migration 0006).</p>
                </div>
              </div>
              <div className="card__body card__body--flush">
                {summary.grants.length === 0 ? (
                  <p className="muted small" style={{ padding: 20, margin: 0 }}>
                    Nenhuma concessão enfileirada até agora.
                  </p>
                ) : (
                  <div className="ui-table-scroll">
                    <table className="ui-table">
                      <thead>
                        <tr>
                          <th>Tipo</th>
                          <th className="ui-align-right">Linhas</th>
                          <th className="ui-align-right">Ouro</th>
                        </tr>
                      </thead>
                      <tbody>
                        {summary.grants.map((row) => (
                          <tr key={row.kind}>
                            <td>{KIND_LABEL[row.kind] ?? row.kind}</td>
                            <td className="ui-align-right">{fmtInt(row.count)}</td>
                            <td className="ui-align-right">
                              {row.kind === "gold" ? fmtInt(row.gold) : "—"}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
              <div className="card__foot">
                A fila é drenada pelo servidor: linha aqui é intenção registrada, não item já entregue.
              </div>
            </section>
          </div>
        </>
      )}
    </>
  );
}
