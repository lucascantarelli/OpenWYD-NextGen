import { Coins, Receipt, ShieldAlert, TriangleAlert, Wallet } from "lucide-react";

import { Badge } from "@/components/Badge";
import { Stat } from "@/components/Stat";
import {
  computeEconomyAlerts,
  getDonateRevenue,
  getEconomySummary,
  type DonateRevenueView,
  type EconomyView,
} from "@/server/economy";
import { AdminError } from "@/server/errors";

import { DebitForm } from "./debit-form";

const fmtDateTime = (iso: string): string => {
  const data = new Date(iso);
  if (Number.isNaN(data.getTime())) return iso;
  return data.toLocaleString("pt-BR", { timeZone: "UTC" });
};

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
  let donate: DonateRevenueView | null = null;
  let error: string | null = null;

  try {
    summary = await getEconomySummary();
  } catch (err) {
    error =
      err instanceof AdminError ? err.message : "economia indisponível no momento";
  }

  // Receita de donate é leitura separada: se ela falhar, o resto da tela continua.
  try {
    donate = await getDonateRevenue();
  } catch {
    donate = null;
  }

  const taxTotal = summary.taxByCity.reduce((acc, row) => acc + row.total, 0);
  const goldGranted = summary.grants.reduce((acc, row) => acc + row.gold, 0);
  const grantsTotal = summary.grants.reduce((acc, row) => acc + row.count, 0);
  const alerts = error ? [] : computeEconomyAlerts(summary);
  const maxTax = Math.max(0, ...summary.taxByCity.map((row) => row.total));

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
          {!error && <a className="ui-btn" href="/api/economia/export" download>Exportar CSV</a>}
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
              hint="desde o início do registro"
              icon={Coins}
            />
            <Stat
              label="Cidades com movimento"
              value={fmtInt(summary.taxByCity.length)}
              hint="com imposto registrado"
              icon={Receipt}
            />
            <Stat
              label="Concessões enfileiradas"
              value={fmtInt(grantsTotal)}
              hint="pedidos registrados"
            />
            <Stat
              label="Ouro concedido"
              value={fmtInt(goldGranted)}
              hint="pedidos de ouro"
            />
            <Stat
              label="Créditos de donate vendidos"
              value={donate ? fmtInt(donate.creditsTotal) : "—"}
              hint={donate ? `${fmtInt(donate.topups)} recarga(s)` : "não foi possível ler"}
              icon={Wallet}
            />
            <Stat
              label="Créditos debitados"
              value={donate ? fmtInt(donate.debitsTotal) : "—"}
              hint={donate ? `${fmtInt(donate.debits)} débito(s)` : "não foi possível ler"}
            />
            <Stat
              label="Saldo de donate"
              value={donate ? fmtInt(donate.balance) : "—"}
              hint="vendido menos debitado"
            />
          </div>

          {alerts.length > 0 && (
            <div className="card">
              <div className="card__head">
                <div>
                  <h2 className="card__title">
                    <TriangleAlert size={15} strokeWidth={1.8} aria-hidden="true" /> Alertas
                  </h2>
                  <p className="card__sub">
                    Regras fixas sobre o total: ouro reunido na fila e cidade com imposto zerado
                    enquanto outras têm movimento.
                  </p>
                </div>
              </div>
              <div className="card__body">
                <ul style={{ margin: 0, paddingLeft: 20 }}>
                  {alerts.map((alert) => (
                    <li key={alert.title}>
                      <Badge tone={alert.tone === "warn" ? "warn" : "accent"}>
                        {alert.title}
                      </Badge>{" "}
                      <span className="muted small">{alert.detail}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          )}

          <section className="card">
            <div className="card__head">
              <div>
                <h2 className="card__title">
                  <Wallet size={15} strokeWidth={1.8} aria-hidden="true" /> Receita de donate
                </h2>
                <p className="card__sub">
                  Cada linha é um recibo gravado: recarga de créditos ou débito feito pelo staff.
                </p>
              </div>
              <div className="page-head__actions">
                <Badge tone={donate ? "ok" : "warn"}>
                  {donate ? "razão no ar" : "indisponível"}
                </Badge>
              </div>
            </div>

            {donate === null ? (
              <div className="card__body">
                <p className="muted small" style={{ margin: 0 }}>
                  Não foi possível ler a receita de donate agora. Tente recarregar a página.
                </p>
              </div>
            ) : (
              <>
                <div className="card__body">
                  <dl className="kv">
                    <dt>Créditos vendidos</dt>
                    <dd>{fmtInt(donate.creditsTotal)} créditos</dd>
                    <dt>Créditos debitados</dt>
                    <dd>{fmtInt(donate.debitsTotal)} créditos</dd>
                    <dt>Saldo do razão</dt>
                    <dd>{fmtInt(donate.balance)} créditos</dd>
                  </dl>
                </div>

                <div className="card__body">
                  <h3 className="card__subtitle">Debitar saldo de uma conta</h3>
                  <p className="card__sub muted">
                    O débito baixa o saldo de donate da conta e fica registrado no extrato abaixo,
                    com recibo próprio. Saldo insuficiente é recusado sem gravar nada.
                  </p>
                  <DebitForm />
                </div>

                <div className="card__body card__body--flush">
                  <h3 className="card__subtitle" style={{ padding: "16px 20px 0" }}>
                    Últimas recargas
                  </h3>
                  {donate.topupOrders.length === 0 ? (
                    <p className="muted small" style={{ padding: 20, margin: 0 }}>
                      Nenhuma recarga registrada até agora.
                    </p>
                  ) : (
                    <div className="ui-table-scroll">
                      <table className="ui-table" aria-label="Recargas de donate">
                        <thead>
                          <tr>
                            <th>Recibo</th>
                            <th>Conta</th>
                            <th className="ui-align-right">Créditos</th>
                            <th>Quando (UTC)</th>
                          </tr>
                        </thead>
                        <tbody>
                          {donate.topupOrders.map((row) => (
                            <tr key={`topup-${row.id}`}>
                              <td className="mono">#{row.id}</td>
                              <td>
                                {row.login} <span className="muted small">#{row.accountId}</span>
                              </td>
                              <td className="ui-align-right">{fmtInt(row.credits)}</td>
                              <td className="mono small">{fmtDateTime(row.createdAt)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>

                <div className="card__body card__body--flush">
                  <h3 className="card__subtitle" style={{ padding: "16px 20px 0" }}>
                    Últimos débitos
                  </h3>
                  {donate.spendOrders.length === 0 ? (
                    <p className="muted small" style={{ padding: 20, margin: 0 }}>
                      Nenhum débito registrado até agora.
                    </p>
                  ) : (
                    <div className="ui-table-scroll">
                      <table className="ui-table" aria-label="Débitos de donate">
                        <thead>
                          <tr>
                            <th>Recibo</th>
                            <th>Conta</th>
                            <th className="ui-align-right">Créditos</th>
                            <th>Quando (UTC)</th>
                          </tr>
                        </thead>
                        <tbody>
                          {donate.spendOrders.map((row) => (
                            <tr key={`spend-${row.id}`}>
                              <td className="mono">#{row.id}</td>
                              <td>
                                {row.login} <span className="muted small">#{row.accountId}</span>
                              </td>
                              <td className="ui-align-right">{fmtInt(row.credits)}</td>
                              <td className="mono small">{fmtDateTime(row.createdAt)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>

                {donate.buyers.length > 0 && (
                  <div className="card__body card__body--flush">
                    <h3 className="card__subtitle" style={{ padding: "16px 20px 0" }}>
                      Contas que mais compraram
                    </h3>
                    <div className="ui-table-scroll">
                      <table className="ui-table" aria-label="Compradores de créditos">
                        <thead>
                          <tr>
                            <th>Conta</th>
                            <th className="ui-align-right">Créditos</th>
                            <th className="ui-align-right">Recargas</th>
                          </tr>
                        </thead>
                        <tbody>
                          {donate.buyers.map((row) => (
                            <tr key={`buyer-${row.accountId}`}>
                              <td>
                                {row.login} <span className="muted small">#{row.accountId}</span>
                              </td>
                              <td className="ui-align-right">{fmtInt(row.credits)}</td>
                              <td className="ui-align-right">{fmtInt(row.topups)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                )}
              </>
            )}
            <div className="card__foot">
              O saldo de donate é da conta, nunca do personagem: o servidor de jogo lê o mesmo razão.
            </div>
          </section>

          <div className="monitor-grid">
            <section className="card">
              <div className="card__head">
                <div>
                  <h2 className="card__title">Imposto por cidade</h2>
                  <p className="card__sub">Imposto recolhido por cidade, somado desde o início.</p>
                </div>
              </div>
              <div className="card__body card__body--flush">
                {summary.taxByCity.length === 0 ? (
                  <p className="muted small" style={{ padding: 20, margin: 0 }}>
                    Nenhum recolhimento registrado até agora.
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
                            <td>
                              <div
                                aria-hidden="true"
                                style={{
                                  height: 8,
                                  borderRadius: 4,
                                  background: "var(--chart-bar, #3b82f6)",
                                  width: `${maxTax > 0 ? Math.max(2, Math.round((row.total / maxTax) * 100)) : 0}%`,
                                }}
                              />
                            </td>
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
                  <p className="card__sub">Ouro e itens já pedidos para entrega, por tipo.</p>
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
