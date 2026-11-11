import { ShieldAlert, Wallet } from "lucide-react";

import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import { listShopItems, type ShopItemWire } from "@/server/donate";
import { AdminError } from "@/server/errors";

import { creditDonateAction, saveShopItemAction, setShopItemAction } from "./actions";

export const dynamic = "force-dynamic";

export const metadata = { title: "Financeiro · WYD2" };

const fmtInt = (value: number): string =>
  value.toLocaleString("pt-BR", { maximumFractionDigits: 0 });

/**
 * Financeiro (PORTAL-03, lado staff): vitrine de donate e crédito de saldo.
 *
 * A loja vende itens por créditos já conciliados — dinheiro real nunca passa
 * aqui. O crédito cai em `account_wallet` com idempotência por chave
 * (`donate_credit`): mesma chave devolve o mesmo razão sem creditar duas vezes.
 * Extrato e receita por comprador não existem nesta entrega: `donate_revenue_*`
 * está declarado no contrato mas ainda sem handler no gateway (responde 501),
 * então a tela mostra só o que é servido de verdade.
 */
export default async function FinanceiroPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const params = await searchParams;
  const one = (value: string | string[] | undefined): string =>
    Array.isArray(value) ? (value[0] ?? "") : (value ?? "");
  const ok = one(params.ok).trim();
  const erro = one(params.erro).trim();

  let shop: ShopItemWire[] = [];
  let error: string | null = null;
  try {
    shop = await listShopItems();
  } catch (err) {
    error = err instanceof AdminError ? err.message : "não foi possível ler a loja agora";
  }

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Governança</p>
          <h1 className="page-head__title">Financeiro</h1>
          <p className="page-head__sub">
            Vitrine de donate e crédito de saldo em conta. Toda escrita é auditada com autor;
            replay com a mesma chave nunca credita duas vezes.
          </p>
        </div>
        <div className="page-head__actions">
          <Badge tone={error ? "warn" : "ok"}>{error ? "indisponível" : "no ar"}</Badge>
        </div>
      </div>

      {erro && (
        <div className="card">
          <div className="card__body">
            <p role="alert" className="login-error">
              {erro}
            </p>
          </div>
        </div>
      )}
      {ok && (
        <div className="card">
          <div className="card__body">
            <p role="status">Registrado: {ok}.</p>
          </div>
        </div>
      )}

      {error ? (
        <div className="card">
          <div className="card__body">
            <div className="empty">
              <ShieldAlert size={22} strokeWidth={1.6} aria-hidden="true" />
              <strong>Financeiro indisponível</strong>
              <p>{error}</p>
              <RetryButton />
            </div>
          </div>
        </div>
      ) : (
        <>
          <div className="card">
            <div className="card__head">
              <div>
                <h2 className="card__title">
                  <Wallet size={15} strokeWidth={1.8} aria-hidden="true" /> Creditar saldo
                </h2>
                <p className="card__sub">
                  Conta por id, créditos inteiros positivos e chave da conciliação externa.
                </p>
              </div>
            </div>
            <div className="card__body">
              <form action={creditDonateAction} className="field-grid">
                <label className="login-field">
                  <span>Conta (id)</span>
                  <input className="ui-input" name="account_id" type="number" min={1} required />
                </label>
                <label className="login-field">
                  <span>Créditos</span>
                  <input className="ui-input" name="credits" type="number" min={1} required />
                </label>
                <label className="login-field">
                  <span>Chave de idempotência</span>
                  <input
                    className="ui-input"
                    name="key"
                    type="text"
                    minLength={1}
                    maxLength={120}
                    required
                    placeholder="conciliacao-2026-001"
                  />
                </label>
                <button className="ui-btn ui-btn--primary" type="submit">
                  Creditar
                </button>
              </form>
            </div>
          </div>

          <div className="card">
            <div className="card__head">
              <div>
                <h2 className="card__title">Vitrine ({shop.length})</h2>
                <p className="card__sub">
                  Preço em créditos; duração vazia = permanente. Desligar esconde sem apagar
                  compra já feita.
                </p>
              </div>
            </div>
            <div className="card__body">
              <form action={saveShopItemAction} className="field-grid">
                <input type="hidden" name="id" value="" />
                <label className="login-field">
                  <span>Índice do item</span>
                  <input className="ui-input" name="item_index" type="number" min={1} required />
                </label>
                <label className="login-field">
                  <span>Preço (créditos)</span>
                  <input className="ui-input" name="price_credits" type="number" min={1} required />
                </label>
                <label className="login-field">
                  <span>Duração (dias, vazio = permanente)</span>
                  <input className="ui-input" name="duration_days" type="number" min={1} />
                </label>
                <label className="login-field">
                  <span>Visível</span>
                  <select className="ui-select" name="enabled" defaultValue="true">
                    <option value="true">sim</option>
                    <option value="false">não</option>
                  </select>
                </label>
                <button className="ui-btn ui-btn--primary" type="submit">
                  Adicionar item
                </button>
              </form>
            </div>
            {shop.length === 0 ? (
              <div className="card__body">
                <p className="muted small" style={{ margin: 0 }}>
                  Vitrine vazia — o primeiro item cadastrado aparece aqui.
                </p>
              </div>
            ) : (
              <div className="card__body card__body--flush">
                <div className="ui-table-scroll">
                  <table className="ui-table">
                    <thead>
                      <tr>
                        <th>#</th>
                        <th>Item</th>
                        <th>Preço</th>
                        <th>Duração</th>
                        <th>Visível</th>
                        <th />
                      </tr>
                    </thead>
                    <tbody>
                      {shop.map((item) => (
                        <tr key={item.id}>
                          <td>#{item.id}</td>
                          <td className="mono">{item.item_index}</td>
                          <td>{fmtInt(item.price_credits)} créditos</td>
                          <td>
                            {item.duration_days === null
                              ? "permanente"
                              : `${item.duration_days} dias`}
                          </td>
                          <td>
                            <Badge tone={item.enabled ? "ok" : "neutral"}>
                              {item.enabled ? "sim" : "não"}
                            </Badge>
                          </td>
                          <td>
                            <form action={setShopItemAction} className="ui-actions" style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
                              <input type="hidden" name="id" value={item.id} />
                              <button
                                className="ui-btn"
                                type="submit"
                                name="intent"
                                value={item.enabled ? "disable" : "enable"}
                              >
                                {item.enabled ? "Desligar" : "Ligar"}
                              </button>
                              <button
                                className="ui-btn"
                                type="submit"
                                name="intent"
                                value="delete"
                              >
                                Remover
                              </button>
                            </form>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        </>
      )}
    </>
  );
}
