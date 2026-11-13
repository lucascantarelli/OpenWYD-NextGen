import { ShieldAlert } from "lucide-react";
import Link from "next/link";

import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import {
  getNpc,
  getRates,
  listDrops,
  listItemPrices,
  listRoutes,
  type AccessoryRuleWire,
  type DropWire,
  type ItemPriceWire,
  type NpcWire,
  type RateWire,
  type RatesView,
  type RouteWire,
} from "@/server/config";
import { AdminError } from "@/server/errors";
import { getWorldEvents, scheduleLabel, weekdayLabel, type WorldEventWire } from "@/server/events";

import {
  resetRateAction,
  saveDropAction,
  saveItemPriceAction,
  saveNpcAction,
  saveRateBatchAction,
  saveRouteAction,
  setEventAction,
} from "./actions";

export const dynamic = "force-dynamic";

export const metadata = { title: "Configurações · WYD2" };

/** Seções da tela: uma por vez, para a página não virar um rolo de campos. */
const TABS = [
  { id: "refino", label: "Refino" },
  { id: "progressao", label: "EXP e multiplicadores" },
  { id: "drops", label: "Drops" },
  { id: "npc", label: "NPCs" },
  { id: "teleportes", label: "Teleportes" },
  { id: "precos", label: "Preços de loja" },
  { id: "eventos", label: "Eventos" },
] as const;

type TabId = (typeof TABS)[number]["id"];

const isTab = (value: string): value is TabId => TABS.some((tab) => tab.id === value);

/** 10.000 = 100% nas tabelas de chance. */
const pct = (bps: number): string =>
  `${(bps / 100).toLocaleString("pt-BR", { maximumFractionDigits: 2 })}%`;

const FAIL_MODE: Record<string, string> = {
  keep: "mantém o refino",
  reset: "volta a +0",
  destroy: "destrói o item",
};

const KIND: Record<string, string> = {
  earring: "Brinco",
  necklace: "Colar",
  belt: "Cinto",
  amulet: "Amuleto",
  ankh: "Ankh",
  spirit_stone: "Pedra espiritual",
};

/**
 * Tabela de chances com gravação em lote (só o que mudou é enviado).
 *
 * O valor é inteiro na base 10.000; a coluna "Equivale" mostra o percentual para
 * quem pensa em porcentagem. Linha editada pode voltar ao padrão pela própria linha.
 */
function RateCard({
  title,
  description,
  rows,
  domain,
  tab,
  kind,
}: {
  title: string;
  description: string;
  /** `rate` = chance (base 10.000); `int` = número puro; `coin` = ouro. */
  kind: "rate" | "int";
  rows: RateWire[];
  domain: string;
  tab: TabId;
}) {
  return (
    <section className="card">
      <div className="card__head">
        <div>
          <h2 className="card__title">{title}</h2>
          <p className="card__sub">{description}</p>
        </div>
      </div>
      {rows.length === 0 ? (
        <div className="card__body">
          <p className="muted small" style={{ margin: 0 }}>
            Nada configurado nesta tabela.
          </p>
        </div>
      ) : (
        <>
          <div className="card__body card__body--flush">
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>{domain === "refine" ? "Refino" : "Linha"}</th>
                    <th className="ui-align-right">
                      {kind === "rate" ? "Chance (10.000 = 100%)" : "Valor"}
                    </th>
                    <th className="ui-align-right">{kind === "rate" ? "Equivale" : "Unidade"}</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {rows.map((row) => (
                    <tr key={row.key}>
                      <td className="mono">
                        {domain === "refine" ? `+${row.key}` : row.key}
                      </td>
                      <td className="ui-align-right">
                        <input
                          className="ui-input ui-input--compact"
                          type="number"
                          name={`rate:${domain}:${row.key}`}
                          form={`form-${domain}`}
                          defaultValue={row.value}
                          min={0}
                          step={1}
                          aria-label={`Valor de ${row.key}`}
                        />
                        <input
                          type="hidden"
                          name={`orig:${domain}:${row.key}`}
                          value={row.value}
                          form={`form-${domain}`}
                        />
                      </td>
                      <td className="ui-align-right muted small">
                        {kind === "rate" ? pct(row.value) : row.unit || "—"}
                        {row.overridden ? " · editado" : ""}
                      </td>
                      <td className="ui-align-right">
                        {row.overridden ? (
                          <button
                            className="ui-btn ui-btn--ghost ui-btn--sm"
                            type="submit"
                            name="key"
                            value={row.key}
                            form={`reset-${domain}-${row.key}`}
                          >
                            Voltar ao padrão
                          </button>
                        ) : null}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
          <form id={`form-${domain}`} action={saveRateBatchAction} className="card__foot">
            <input type="hidden" name="aba" value={tab} />
            <span className="muted small">Só as linhas alteradas são gravadas.</span>
            <button className="ui-btn ui-btn--primary" type="submit">
              Salvar alterações
            </button>
          </form>
          {rows
            .filter((row) => row.overridden)
            .map((row) => (
              <form
                key={`reset-form-${row.key}`}
                id={`reset-${domain}-${row.key}`}
                action={resetRateAction}
                hidden
              >
                <input type="hidden" name="domain" value={domain} />
                <input type="hidden" name="key" value={row.key} />
              </form>
            ))}
        </>
      )}
    </section>
  );
}

function AccessoryCard({ rules }: { rules: AccessoryRuleWire[] }) {
  const typed = rules;
  const kinds = Array.from(new Set(typed.map((rule) => rule.kind)));
  return (
    <section className="card">
      <div className="card__head">
        <div>
          <h2 className="card__title">Refino de acessório</h2>
          <p className="card__sub">
            Chance por passo e o custo em ouro/insumo. Só o que está na tabela aparece — o
            servidor não inventa regra para acessório sem fonte.
          </p>
        </div>
      </div>
      {kinds.length === 0 ? (
        <div className="card__body">
          <p className="muted small" style={{ margin: 0 }}>
            Nenhuma regra carregada.
          </p>
        </div>
      ) : (
        <>
          {kinds.map((kind) => {
            const rows = typed.filter((rule) => rule.kind === kind);
            return (
              <div className="card__body" key={kind}>
                <h3 className="card__title">{KIND[kind] ?? kind}</h3>
                <div className="ui-table-scroll">
                  <table className="ui-table">
                    <thead>
                      <tr>
                        <th>Receita</th>
                        <th>Passo</th>
                        <th className="ui-align-right">Chance (10.000 = 100%)</th>
                        <th className="ui-align-right">Equivale</th>
                        <th>Se falhar</th>
                        <th>Insumo</th>
                      </tr>
                    </thead>
                    <tbody>
                      {rows.map((rule) => {
                        const key = `accessory:${rule.kind}:${rule.recipe}:${rule.step_from}`;
                        return (
                          <tr key={`${rule.recipe}-${rule.step_from}`}>
                            <td className="mono">{rule.recipe}</td>
                            <td>
                              +{rule.step_from} → +{rule.step_to}
                            </td>
                            <td className="ui-align-right">
                              <input
                                className="ui-input ui-input--compact"
                                type="number"
                                name={`rate:${key}`}
                                form="form-accessory"
                                defaultValue={rule.rate_bps}
                                min={0}
                                step={1}
                                aria-label={`Chance do passo +${rule.step_from}`}
                              />
                              <input
                                type="hidden"
                                name={`orig:${key}`}
                                value={rule.rate_bps}
                                form="form-accessory"
                              />
                            </td>
                            <td className="ui-align-right muted small">{pct(rule.rate_bps)}</td>
                            <td className="small">
                              {FAIL_MODE[rule.fail_mode] ?? rule.fail_mode}
                              {rule.fail_bonus_bps > 0
                                ? ` · consolo de ${pct(rule.fail_bonus_bps)}`
                                : ""}
                            </td>
                            <td className="small">
                              {rule.material_qty}× {rule.material_name}
                              {rule.gold_cost > 0
                                ? ` · ${rule.gold_cost.toLocaleString("pt-BR")} de ouro`
                                : ""}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </div>
            );
          })}
          <form id="form-accessory" action={saveRateBatchAction} className="card__foot">
            <input type="hidden" name="aba" value="refino" />
            <span className="muted small">Só as linhas alteradas são gravadas.</span>
            <button className="ui-btn ui-btn--primary" type="submit">
              Salvar alterações
            </button>
          </form>
        </>
      )}
    </section>
  );
}

function DropEditor({ drops }: { drops: DropWire[] }) {
  return (
    <section className="card">
      <div className="card__head">
        <div>
          <h2 className="card__title">Drops por monstro</h2>
          <p className="card__sub">
            Peso do item no sorteio do monstro. Quanto maior o peso, mais chance de cair.
          </p>
        </div>
      </div>
      <div className="card__body">
        <form action={saveDropAction} className="field-grid">
          <label className="login-field">
            <span>Monstro</span>
            <input className="ui-input" name="template_name" maxLength={120} required />
          </label>
          <label className="login-field">
            <span>Item (índice)</span>
            <input className="ui-input" name="item_index" type="number" min={1} required />
          </label>
          <label className="login-field">
            <span>Peso</span>
            <input className="ui-input" name="weight" type="number" min={1} required />
          </label>
          <label className="login-field">
            <span>Qtd. mínima</span>
            <input className="ui-input" name="min_qty" type="number" min={1} defaultValue={1} />
          </label>
          <label className="login-field">
            <span>Qtd. máxima</span>
            <input className="ui-input" name="max_qty" type="number" min={1} defaultValue={1} />
          </label>
          <input type="hidden" name="intent" value="save" />
          <button className="ui-btn ui-btn--primary" type="submit">
            Adicionar drop
          </button>
        </form>
      </div>
      {drops.length === 0 ? (
        <div className="card__body">
          <p className="muted small" style={{ margin: 0 }}>
            Nenhum drop ajustado ainda. O servidor usa a tabela de fábrica enquanto nada é
            gravado aqui.
          </p>
        </div>
      ) : (
        <div className="card__body card__body--flush">
          <div className="ui-table-scroll">
            <table className="ui-table">
              <thead>
                <tr>
                  <th>Monstro</th>
                  <th>Item</th>
                  <th className="ui-align-right">Peso</th>
                  <th className="ui-align-right">Qtd.</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {drops.map((drop) => (
                  <tr key={`${drop.template_name}#${drop.item_index}`}>
                    <td className="mono">{drop.template_name}</td>
                    <td>
                      {drop.item_name || "sem nome"}
                      <span className="muted small"> #{drop.item_index}</span>
                    </td>
                    <td className="ui-align-right" colSpan={2}>
                      <form action={saveDropAction} className="inline-form">
                        <input type="hidden" name="template_name" value={drop.template_name} />
                        <input type="hidden" name="item_index" value={drop.item_index} />
                        <input
                          className="ui-input ui-input--compact"
                          name="weight"
                          type="number"
                          min={1}
                          defaultValue={drop.weight}
                          aria-label="Peso"
                        />
                        <input
                          className="ui-input ui-input--compact"
                          name="min_qty"
                          type="number"
                          min={1}
                          defaultValue={drop.min_qty}
                          aria-label="Quantidade mínima"
                        />
                        <input
                          className="ui-input ui-input--compact"
                          name="max_qty"
                          type="number"
                          min={1}
                          defaultValue={drop.max_qty}
                          aria-label="Quantidade máxima"
                        />
                        <button className="ui-btn ui-btn--sm" type="submit" name="intent" value="save">
                          Salvar
                        </button>
                        <button
                          className="ui-btn ui-btn--ghost ui-btn--sm"
                          type="submit"
                          name="intent"
                          value="reset"
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
    </section>
  );
}

function NpcEditor({ npc }: { npc: NpcWire | null }) {
  return (
    <section className="card">
      <div className="card__head">
        <div>
          <h2 className="card__title">Posição do NPC</h2>
          <p className="card__sub">
            Onde o NPC fica e se aparece no mapa. Mapa 0–255; X e Y 0–4095.
          </p>
        </div>
      </div>
      <div className="card__body">
        <form method="get" action="/configuracoes" className="field-grid">
          <input type="hidden" name="aba" value="npc" />
          <label className="login-field">
            <span>ID do NPC</span>
            <input
              className="ui-input"
              name="npc"
              type="number"
              min={1}
              defaultValue={npc?.id}
              required
            />
          </label>
          <button className="ui-btn" type="submit">
            Buscar
          </button>
        </form>
      </div>
      {npc ? (
        <div className="card__body">
          <p className="card__sub">
            <strong>{npc.display_name || npc.slug}</strong> · {npc.slug} ·{" "}
            {npc.merchant > 0 ? "comerciante" : "sem loja"} ·{" "}
            {npc.enabled ? "visível" : "oculto"}
          </p>
          <form action={saveNpcAction} className="field-grid">
            <input type="hidden" name="id" value={npc.id} />
            <label className="login-field">
              <span>Mapa</span>
              <input
                className="ui-input"
                name="map_id"
                type="number"
                min={0}
                max={255}
                defaultValue={npc.map_id}
                required
              />
            </label>
            <label className="login-field">
              <span>X</span>
              <input
                className="ui-input"
                name="pos_x"
                type="number"
                min={0}
                max={4095}
                defaultValue={npc.pos_x}
                required
              />
            </label>
            <label className="login-field">
              <span>Y</span>
              <input
                className="ui-input"
                name="pos_y"
                type="number"
                min={0}
                max={4095}
                defaultValue={npc.pos_y}
                required
              />
            </label>
            <label className="login-field">
              <span>Aparece no mapa</span>
              <select className="ui-select" name="enabled" defaultValue={String(npc.enabled)}>
                <option value="true">sim</option>
                <option value="false">não</option>
              </select>
            </label>
            <button className="ui-btn ui-btn--primary" type="submit">
              Salvar NPC
            </button>
          </form>
        </div>
      ) : null}
    </section>
  );
}

function PriceEditor({ itemIndex, prices }: { itemIndex: number; prices: ItemPriceWire[] }) {
  return (
    <section className="card">
      <div className="card__head">
        <div>
          <h2 className="card__title">Preço de loja por NPC</h2>
          <p className="card__sub">
            O mesmo item pode custar diferente em cada mercador. Zero significa que o
            mercador não compra (ou não vende) aquele item.
          </p>
        </div>
      </div>
      <div className="card__body">
        <form method="get" action="/configuracoes" className="field-grid">
          <input type="hidden" name="aba" value="precos" />
          <label className="login-field">
            <span>Ver preços do item</span>
            <input
              className="ui-input"
              name="item"
              type="number"
              min={1}
              defaultValue={itemIndex > 0 ? itemIndex : undefined}
              required
            />
          </label>
          <button className="ui-btn" type="submit">
            Ver preços
          </button>
        </form>
      </div>
      {itemIndex > 0 ? (
        prices.length === 0 ? (
          <div className="card__body">
            <p className="muted small" style={{ margin: 0 }}>
              Nenhum mercador tem preço definido para este item.
            </p>
          </div>
        ) : (
          <div className="card__body card__body--flush">
            <div className="ui-table-scroll">
              <table className="ui-table">
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
                        #{price.npc_id} {price.npc_slug || ""}
                      </td>
                      <td className="ui-align-right">{price.buy_price.toLocaleString("pt-BR")}</td>
                      <td className="ui-align-right">{price.sell_price.toLocaleString("pt-BR")}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )
      ) : null}
      <div className="card__head">
        <div>
          <h3 className="card__title">Definir preço</h3>
          <p className="card__sub">
            Vale para o par item + mercador. Salvar de novo o mesmo par substitui o preço.
          </p>
        </div>
      </div>
      <div className="card__body">
        <form action={saveItemPriceAction} className="field-grid">
          <label className="login-field">
            <span>Item</span>
            <input
              className="ui-input"
              name="item_index"
              type="number"
              min={1}
              defaultValue={itemIndex > 0 ? itemIndex : undefined}
              required
            />
          </label>
          <label className="login-field">
            <span>Mercador (ID do NPC)</span>
            <input className="ui-input" name="npc_id" type="number" min={1} required />
          </label>
          <label className="login-field">
            <span>Preço de compra</span>
            <input className="ui-input" name="buy_price" type="number" min={0} defaultValue={0} required />
          </label>
          <label className="login-field">
            <span>Preço de venda</span>
            <input className="ui-input" name="sell_price" type="number" min={0} defaultValue={0} required />
          </label>
          <button className="ui-btn ui-btn--primary" type="submit">
            Salvar preço
          </button>
        </form>
      </div>
    </section>
  );
}

function EventEditor({ events }: { events: WorldEventWire[] }) {
  return (
    <section className="card">
      <div className="card__head">
        <div>
          <h2 className="card__title">Eventos de mundo</h2>
          <p className="card__sub">
            Liga e desliga eventos já cadastrados. O horário vem do cadastro do evento.
          </p>
        </div>
      </div>
      {events.length === 0 ? (
        <div className="card__body">
          <p className="muted small" style={{ margin: 0 }}>
            Nenhum evento cadastrado.
          </p>
        </div>
      ) : (
        <div className="card__body card__body--flush">
          <div className="ui-table-scroll">
            <table className="ui-table">
              <thead>
                <tr>
                  <th>Evento</th>
                  <th>Quando</th>
                  <th>Ativo</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {events.map((event) => (
                  <tr key={event.code}>
                    <td>
                      <strong>{event.name || event.code}</strong>
                    </td>
                    <td className="small">
                      {event.schedules.length === 0 ? (
                        <span className="muted">sem horário</span>
                      ) : (
                        event.schedules.map((slot) => (
                          <div key={slot.id}>
                            {weekdayLabel(slot.weekday)} ·{" "}
                            {scheduleLabel(slot.start_time, slot.duration_minutes)}
                            {slot.enabled ? "" : " (pausado)"}
                          </div>
                        ))
                      )}
                    </td>
                    <td>
                      <Badge tone={event.enabled ? "ok" : "neutral"}>
                        {event.enabled ? "ligado" : "desligado"}
                      </Badge>
                    </td>
                    <td>
                      <form action={setEventAction}>
                        <input type="hidden" name="code" value={event.code} />
                        <button
                          className="ui-btn ui-btn--sm"
                          type="submit"
                          name="enabled"
                          value={event.enabled ? "false" : "true"}
                        >
                          {event.enabled ? "Desligar" : "Ligar"}
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
    </section>
  );
}

function RouteEditor({ routes }: { routes: RouteWire[] }) {
  return (
    <section className="card">
      <div className="card__head">
        <div>
          <h2 className="card__title">Teleportes</h2>
          <p className="card__sub">
            Custo, nível mínimo e disponibilidade de cada rota. Origem e destino vêm do
            conteúdo do jogo.
          </p>
        </div>
      </div>
      {routes.length === 0 ? (
        <div className="card__body">
          <p className="muted small" style={{ margin: 0 }}>
            Nenhuma rota cadastrada.
          </p>
        </div>
      ) : (
        <div className="card__body card__body--flush">
          <div className="ui-table-scroll">
            <table className="ui-table">
              <thead>
                <tr>
                  <th>Rota</th>
                  <th className="ui-align-right">Ouro</th>
                  <th className="ui-align-right">Nível</th>
                  <th>Ativa</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {routes.map((route) => (
                  <tr key={route.id}>
                    <td>
                      {route.from_map_id} ({route.from_x},{route.from_y}) → {route.to_map_id} (
                      {route.to_x},{route.to_y})
                    </td>
                    <td colSpan={4}>
                      <form action={saveRouteAction} className="inline-form">
                        <input type="hidden" name="id" value={route.id} />
                        <input
                          className="ui-input ui-input--compact"
                          name="cost_gold"
                          type="number"
                          min={0}
                          defaultValue={route.cost_gold}
                          aria-label="Custo em ouro"
                        />
                        <input
                          className="ui-input ui-input--compact"
                          name="req_level"
                          type="number"
                          min={1}
                          defaultValue={route.req_level}
                          aria-label="Nível mínimo"
                        />
                        <select
                          className="ui-select ui-select--compact"
                          name="enabled"
                          defaultValue={String(route.enabled)}
                          aria-label="Disponível"
                        >
                          <option value="true">ativa</option>
                          <option value="false">inativa</option>
                        </select>
                        <button className="ui-btn ui-btn--sm" type="submit">
                          Salvar
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
    </section>
  );
}

/**
 * Configurações do servidor, divididas por assunto.
 *
 * Uma seção por vez (`?aba=`) para a tela não virar um rolo de caixas de texto.
 * Nada aqui reinicia o jogo: o servidor relê a configuração quando a revisão muda,
 * e toda gravação entra na auditoria com quem editou.
 */
export default async function ConfiguracoesPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const params = await searchParams;
  const one = (value: string | string[] | undefined): string =>
    Array.isArray(value) ? (value[0] ?? "") : (value ?? "");

  const abaRaw = one(params.aba);
  const aba: TabId = isTab(abaRaw) ? abaRaw : "refino";
  const ok = one(params.ok).trim();
  const erro = one(params.erro).trim();
  const npcId = Number(one(params.npc));
  const priceItem = Number(one(params.item));
  const priceItemId = Number.isInteger(priceItem) && priceItem > 0 ? priceItem : 0;

  let rates: RatesView | null = null;
  let drops: DropWire[] = [];
  let routes: RouteWire[] = [];
  let events: WorldEventWire[] = [];
  let prices: ItemPriceWire[] = [];
  let npc: NpcWire | null = null;
  let error: string | null = null;

  try {
    if (aba === "refino" || aba === "progressao") rates = await getRates();
    if (aba === "drops") drops = await listDrops();
    if (aba === "teleportes") routes = await listRoutes();
    if (aba === "eventos") events = await getWorldEvents();
    if (aba === "precos" && priceItemId > 0) prices = await listItemPrices(priceItemId);
    if (aba === "npc" && Number.isInteger(npcId) && npcId > 0) npc = await getNpc(npcId);
  } catch (err) {
    error = err instanceof AdminError ? err.message : "não foi possível ler a configuração agora";
  }

  const byDomain = (domain: string): RateWire[] =>
    rates?.rates.filter((row) => row.domain === domain) ?? [];
  const settings = byDomain("setting");
  const multipliers = byDomain("multiplier");

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Conteúdo</p>
          <h1 className="page-head__title">Configurações</h1>
          <p className="page-head__sub">
            Ajustes do servidor, por assunto. O que você salva vale na próxima leitura do
            servidor, sem reiniciar, e fica registrado na auditoria com o seu nome.
          </p>
        </div>
        <div className="page-head__actions">
          <Badge tone={error ? "warn" : "ok"}>
            {error ? "indisponível" : rates ? `revisão ${rates.revision}` : "no ar"}
          </Badge>
        </div>
      </div>

      <nav className="chips" aria-label="Origem da configuração" style={{ marginBottom: 8 }}>
        <Link className="chip" href="/configuracoes" aria-current="page">
          Servidor
        </Link>
        <Link className="chip" href="/configuracoes/launcher">
          Launcher (config assinada)
        </Link>
      </nav>

      <nav className="chips" aria-label="Assuntos da configuração" style={{ marginBottom: 16 }}>
        {TABS.map((tab) => (
          <Link
            key={tab.id}
            className="chip"
            href={`/configuracoes?aba=${tab.id}`}
            aria-current={tab.id === aba ? "page" : undefined}
          >
            {tab.label}
          </Link>
        ))}
      </nav>

      {erro ? (
        <div className="card">
          <div className="card__body">
            <p role="alert" className="login-error">
              {erro}
            </p>
          </div>
        </div>
      ) : null}
      {ok ? (
        <div className="card">
          <div className="card__body">
            <p role="status">Configuração salva.</p>
          </div>
        </div>
      ) : null}

      {error ? (
        <div className="card">
          <div className="card__body">
            <div className="empty">
              <ShieldAlert size={22} strokeWidth={1.6} aria-hidden="true" />
              <strong>Configuração indisponível</strong>
              <p>{error}</p>
              <RetryButton />
            </div>
          </div>
        </div>
      ) : (
        <>
          {aba === "refino" && (
            <>
              <RateCard
                title="Refino de arma e armadura"
                description="Chance de cada passo, de +0 a +15. 10.000 equivale a 100%."
                kind="rate"
                rows={byDomain("refine")}
                domain="refine"
                tab="refino"
              />
              <RateCard
                title="Pó de refino"
                description="Quanto o pó cobre do refino atual. 100 equivale a 100%."
                kind="rate"
                rows={byDomain("sanc")}
                domain="sanc"
                tab="refino"
              />
              <RateCard
                title="Composição"
                description="Chance e custo de composição de itens."
                kind="rate"
                rows={byDomain("combine")}
                domain="combine"
                tab="refino"
              />
              <AccessoryCard rules={rates?.accessories ?? []} />
            </>
          )}

          {aba === "progressao" && (
            <>
              <RateCard
                title="Multiplicadores"
                description="EXP, drop, ouro e ponto de skill. 10.000 equivale a 100%."
                kind="rate"
                rows={multipliers}
                domain="multiplier"
                tab="progressao"
              />
              <RateCard
                title="Parâmetros do servidor"
                description="Valores inteiros usados por eventos, guerra e taxas do jogo."
                kind="int"
                rows={settings}
                domain="setting"
                tab="progressao"
              />
            </>
          )}

          {aba === "drops" && <DropEditor drops={drops} />}
          {aba === "npc" && <NpcEditor npc={npc} />}
          {aba === "teleportes" && <RouteEditor routes={routes} />}
          {aba === "precos" && <PriceEditor itemIndex={priceItemId} prices={prices} />}
          {aba === "eventos" && <EventEditor events={events} />}
        </>
      )}
    </>
  );
}
