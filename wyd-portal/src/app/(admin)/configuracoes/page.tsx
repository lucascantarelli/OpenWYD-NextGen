import { SlidersHorizontal, ShieldAlert } from "lucide-react";

import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import {
  getNpc,
  getRates,
  listDrops,
  listRoutes,
  type AccessoryRuleWire,
  type DropWire,
  type NpcWire,
  type RateWire,
  type RatesView,
  type RouteWire,
} from "@/server/config";
import { AdminError } from "@/server/errors";

import { scheduleLabel, weekdayLabel } from "@/server/events";
import type { WorldEventWire } from "@/server/events";
import { getWorldEvents } from "@/server/events";

import {
  resetRateAction,
  saveDropAction,
  saveNpcAction,
  saveRateBatchAction,
  saveRouteAction,
  setEventAction,
} from "./actions";

export const dynamic = "force-dynamic";

export const metadata = { title: "Configurações · WYD2" };

const pct = (bps: number): string =>
  (bps / 100).toLocaleString("pt-BR", { maximumFractionDigits: 2 });

const FAIL_MODE: Record<string, string> = {
  keep: "mantém o refino",
  reset: "volta a +0",
  destroy: "destrói o item",
};

const KIND: Record<string, string> = {
  earring: "brinco",
  necklace: "colar",
  belt: "cinto",
  amulet: "amuleto",
  ankh: "ankh",
  spirit_stone: "pedra espiritual",
};

/** Tabela de taxas com um único formulário em lote (só o que mudou é gravado). */
function RateTable({
  title,
  hint,
  rows,
  domain,
  scale,
  source,
}: {
  title: string;
  hint: string;
  rows: RateWire[];
  domain: string;
  scale: "bps" | "percent" | "int";
  source: string;
}) {
  if (rows.length === 0) {
    return (
      <div className="card">
        <div className="card__head">
          <div>
            <h2 className="card__title">{title}</h2>
            <p className="card__sub">{hint}</p>
          </div>
        </div>
        <div className="card__body">
          <div className="empty">
            <SlidersHorizontal size={20} strokeWidth={1.6} aria-hidden="true" />
            <strong>Sem linhas</strong>
            <p>O seed não trouxe nenhuma linha para esta família.</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <form action={saveRateBatchAction} className="card">
      <div className="card__head">
        <div>
          <h2 className="card__title">{title}</h2>
          <p className="card__sub">{hint}</p>
        </div>
        <label className="login-field">
          <span>Fonte desta edição</span>
          <input className="ui-input" name="source" defaultValue={source} maxLength={160} required />
        </label>
      </div>
      <div className="card__body card__body--flush">
        <div className="ui-table-wrap">
          <div className="ui-table-scroll">
            <table className="ui-table">
              <thead>
                <tr>
                  <th>Chave</th>
                  <th>Valor</th>
                  <th>{scale === "bps" ? "Percentual" : scale === "percent" ? "Unidade" : "Tipo"}</th>
                  <th>Fonte</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => {
                  const name = `rate:${domain}:${row.key}`;
                  return (
                    <tr key={name}>
                      <td>
                        <code>{row.key}</code>
                      </td>
                      <td>
                        <input type="hidden" name={`orig:${domain}:${row.key}`} value={row.value} />
                        <input
                          className="ui-input"
                          type="number"
                          name={name}
                          defaultValue={row.value}
                          min={0}
                          max={scale === "percent" ? 100 : undefined}
                          step={1}
                        />
                      </td>
                      <td>{scale === "bps" ? `${pct(row.value)}%` : row.unit}</td>
                      <td>
                        {row.source}
                        {row.overridden ? " · override" : " · seed"}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      </div>
      <div className="card__foot">
        <button className="ui-btn ui-btn--primary" type="submit">
          Gravar alterações
        </button>
      </div>
    </form>
  );
}

/** Tabela de refino de acessório: taxa por passo e receita. */
function AccessoryTable({ rules }: { rules: AccessoryRuleWire[] }) {
  const kinds = Array.from(new Set(rules.map((rule) => rule.kind)));
  return (
    <>
      <div className="card__head">
        <div>
          <h2 className="card__title">Refino de acessório</h2>
          <p className="card__sub">
            Sistema posterior ao 7662. Regras do WYD Global/Bahamut; amuleto, ankh e pedra
            espiritual ficam sem linha até existir fonte — o portal não inventa número. O
            snapshot carrega as regras na recarga, mas o handler de refino de acessório
            ainda não existe no servidor de jogo.
          </p>
        </div>
      </div>
      {kinds.length === 0 ? (
        <div className="card__body">
          <div className="empty">
            <SlidersHorizontal size={20} strokeWidth={1.6} aria-hidden="true" />
            <strong>Nenhuma regra carregada</strong>
            <p>A migration de seed 0038 não foi aplicada neste banco.</p>
          </div>
        </div>
      ) : (
        kinds.map((kind) => {
          const rows = rules.filter((rule) => rule.kind === kind);
          return (
            <div className="card__body" key={kind}>
              <h3 className="card__title">{KIND[kind] ?? kind}</h3>
              <form action={saveRateBatchAction}>
                <input type="hidden" name="source" value="wydglobal/3390" />
                <div className="ui-table-wrap">
                  <div className="ui-table-scroll">
                    <table className="ui-table">
                      <thead>
                        <tr>
                          <th>Receita</th>
                          <th>Passo</th>
                          <th>Taxa</th>
                          <th>Falha</th>
                          <th>Insumo</th>
                        </tr>
                      </thead>
                      <tbody>
                        {rows.map((rule) => {
                          const key = `accessory:${rule.kind}:${rule.recipe}:${rule.step_from}`;
                          return (
                            <tr key={key}>
                              <td>
                                <code>{rule.recipe}</code>
                              </td>
                              <td>
                                +{rule.step_from} → +{rule.step_to}
                              </td>
                              <td>
                                <input
                                  type="hidden"
                                  name={`orig:${key}`}
                                  value={rule.rate_bps}
                                />
                                <input
                                  className="ui-input"
                                  type="number"
                                  name={`rate:${key}`}
                                  defaultValue={rule.rate_bps}
                                  min={0}
                                  max={10000}
                                />
                                <span> {pct(rule.rate_bps)}%</span>
                              </td>
                              <td>
                                {FAIL_MODE[rule.fail_mode] ?? rule.fail_mode}
                                {rule.fail_bonus_bps > 0
                                  ? ` · consolação ${pct(rule.fail_bonus_bps)}%`
                                  : ""}
                              </td>
                              <td>
                                {rule.material_qty}× {rule.material_name}
                                {rule.gold_cost > 0
                                  ? ` · ${rule.gold_cost.toLocaleString("pt-BR")} ouro`
                                  : ""}
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                </div>
                <button className="ui-btn ui-btn--primary" type="submit">
                  Gravar taxas
                </button>
              </form>
            </div>
          );
        })
      )}
    </>
  );
}

/** Editores de drops, NPC e teleporte. */
function DropEditor({ drops }: { drops: DropWire[] }) {
  return (
    <div className="card">
      <div className="card__head">
        <div>
          <h2 className="card__title">Drops</h2>
          <p className="card__sub">
            Peso comparado ao sorteio de 1..1.000.000 por item. Resolvedor pronto:
            `Gameplay::loot` faz um sorteio por item da tabela. Ainda NÃO consumido em jogo:
            nenhum caminho de produção o chama. Inspecionados sem resolução de
            morte/caixa/pickup: `wyd_sim::resolve_attack`/`AttackOutcome`,
            `wyd_sim::TICK_PHASES`, `wyd_sim::formula::drop::drops`/`effective_rate`/
            `gem_bonus`, `wyd_sim::formula::accessory::attempt`,
            `wyd_sim::formula::progression::grant_exp`/`exp_to_next`,
            `wyd_core::systems::combat::attack`, `wyd_core::systems::npc::decide`,
            `wyd_server::world::Command`/`Engine::step` e `cpsock::dispatch`
            (sem Attack/NPC/pickup/teleporte).
          </p>
        </div>
      </div>
      <div className="card__body">
        <form action={saveDropAction} className="field-grid">
          <label className="login-field">
            <span>Template do monstro</span>
            <input className="ui-input" name="template_name" maxLength={120} required />
          </label>
          <label className="login-field">
            <span>Item</span>
            <input className="ui-input" name="item_index" type="number" min={1} required />
          </label>
          <label className="login-field">
            <span>Peso</span>
            <input className="ui-input" name="weight" type="number" min={1} required />
          </label>
          <label className="login-field">
            <span>Qtd. mín.</span>
            <input className="ui-input" name="min_qty" type="number" min={1} defaultValue={1} />
          </label>
          <label className="login-field">
            <span>Qtd. máx.</span>
            <input className="ui-input" name="max_qty" type="number" min={1} defaultValue={1} />
          </label>
          <label className="login-field">
            <span>Fonte</span>
            <input className="ui-input" name="source" maxLength={160} required />
          </label>
          <input type="hidden" name="intent" value="save" />
          <button className="ui-btn ui-btn--primary" type="submit">
            Gravar peso
          </button>
        </form>
      </div>
      <div className="card__body card__body--flush">
        {drops.length === 0 ? (
          <div className="empty">
            <strong>Nenhum peso editado</strong>
            <p>O servidor usa o seed; o primeiro peso gravado aparece aqui.</p>
          </div>
        ) : (
          <div className="ui-table-wrap">
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Monstro</th>
                    <th>Item</th>
                    <th>Peso</th>
                    <th>Qtd.</th>
                    <th>Fonte</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {drops.map((drop) => (
                    <tr key={`${drop.template_name}#${drop.item_index}`}>
                      <td>{drop.template_name}</td>
                      <td>
                        {drop.item_index} · {drop.item_name}
                      </td>
                      <td colSpan={4}>
                        <form action={saveDropAction} className="field-grid">
                          <input type="hidden" name="template_name" value={drop.template_name} />
                          <input type="hidden" name="item_index" value={drop.item_index} />
                          <input
                            className="ui-input"
                            name="weight"
                            type="number"
                            min={1}
                            defaultValue={drop.weight}
                          />
                          <input
                            className="ui-input"
                            name="min_qty"
                            type="number"
                            min={1}
                            defaultValue={drop.min_qty}
                          />
                          <input
                            className="ui-input"
                            name="max_qty"
                            type="number"
                            min={1}
                            defaultValue={drop.max_qty}
                          />
                          <input
                            className="ui-input"
                            name="source"
                            maxLength={160}
                            defaultValue={drop.source}
                          />
                          <button className="ui-btn" type="submit" name="intent" value="save">
                            Salvar
                          </button>
                          <button className="ui-btn" type="submit" name="intent" value="reset">
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
    </div>
  );
}

function NpcEditor({ npc }: { npc: NpcWire | null }) {
  return (
    <div className="card">
      <div className="card__head">
        <div>
          <h2 className="card__title">NPC</h2>
          <p className="card__sub">
            Posição é definição de mundo: mapa 0..255, x/y 0..4095. O snapshot carrega a
            linha na revisão, mas o spawn/diálogo de NPC ainda não existe no servidor de
            jogo: editar aqui não move NPC em jogo ainda.
          </p>
        </div>
      </div>
      <div className="card__body">
        <form method="get" action="/configuracoes" className="field-grid">
          <label className="login-field">
            <span>ID do NPC</span>
            <input className="ui-input" name="npc" type="number" min={1} required />
          </label>
          <button className="ui-btn" type="submit">
            Carregar
          </button>
        </form>
      </div>
      {npc && (
        <div className="card__body">
          <p className="card__sub">
            {npc.slug} · {npc.display_name || "sem nome"} · comerciante {npc.merchant}
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
              <span>Visível</span>
              <select className="ui-select" name="enabled" defaultValue={String(npc.enabled)}>
                <option value="true">sim</option>
                <option value="false">não</option>
              </select>
            </label>
            <button className="ui-btn ui-btn--primary" type="submit">
              Gravar NPC
            </button>
          </form>
        </div>
      )}
    </div>
  );
}

function EventEditor({ events }: { events: WorldEventWire[] }) {
  return (
    <div className="card">
      <div className="card__head">
        <div>
          <h2 className="card__title">Eventos de mundo</h2>
          <p className="card__sub">
            Ligar/desligar vale para o próximo leitor do dado (boot/tick que lê a tabela) —
            nenhum push para o Hot Path. Horários de `event_schedule` com timezone explícita.
          </p>
        </div>
      </div>
      <div className="card__body card__body--flush">
        {events.length === 0 ? (
          <p className="muted small" style={{ padding: 20, margin: 0 }}>
            Nenhum evento cadastrado. Eventos entram por seed/captura; não são inventados aqui.
          </p>
        ) : (
          <div className="ui-table-wrap">
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Evento</th>
                    <th>Grade</th>
                    <th>Ativo</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {events.map((event) => (
                    <tr key={event.code}>
                      <td>
                        <code>{event.code}</code> · {event.name || "sem nome"}
                        {event.description ? (
                          <span className="muted small"> — {event.description}</span>
                        ) : null}
                      </td>
                      <td className="small">
                        {event.schedules.length === 0 ? (
                          <span className="muted">sem horários</span>
                        ) : (
                          <ul style={{ margin: 0, paddingLeft: 18 }}>
                            {event.schedules.map((slot) => (
                              <li key={slot.id}>
                                {weekdayLabel(slot.weekday)} ·{" "}
                                {scheduleLabel(slot.start_time, slot.duration_minutes)} ·{" "}
                                {slot.timezone}
                                {slot.enabled ? "" : " (pausado)"}
                              </li>
                            ))}
                          </ul>
                        )}
                      </td>
                      <td>
                        <Badge tone={event.enabled ? "ok" : "neutral"}>
                          {event.enabled ? "sim" : "não"}
                        </Badge>
                      </td>
                      <td>
                        <form action={setEventAction}>
                          <input type="hidden" name="code" value={event.code} />
                          <button
                            className="ui-btn"
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
      </div>
    </div>
  );
}

function RouteEditor({ routes }: { routes: RouteWire[] }) {
  return (
    <div className="card">
      <div className="card__head">
        <div>
          <h2 className="card__title">Teleportes</h2>
          <p className="card__sub">
            A rota (origem/destino) vem do conteúdo capturado; aqui se edita a política:
            custo, nível mínimo e disponibilidade. O snapshot carrega a política, mas o
            handler de teleporte ainda não existe no servidor de jogo.
          </p>
        </div>
      </div>
      <div className="card__body card__body--flush">
        {routes.length === 0 ? (
          <div className="empty">
            <strong>Nenhuma rota cadastrada</strong>
            <p>Rotas de teleporte entram com a captura do cliente; não são inventadas aqui.</p>
          </div>
        ) : (
          <div className="ui-table-wrap">
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Rota</th>
                    <th>Custo</th>
                    <th>Nível</th>
                    <th>Ativa</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {routes.map((route) => (
                    <tr key={route.id}>
                      <td>
                        #{route.id} · {route.from_map_id} ({route.from_x},{route.from_y}) →{" "}
                        {route.to_map_id} ({route.to_x},{route.to_y})
                      </td>
                      <td colSpan={4}>
                        <form action={saveRouteAction} className="field-grid">
                          <input type="hidden" name="id" value={route.id} />
                          <input
                            className="ui-input"
                            name="cost_gold"
                            type="number"
                            min={0}
                            defaultValue={route.cost_gold}
                          />
                          <input
                            className="ui-input"
                            name="req_level"
                            type="number"
                            min={1}
                            defaultValue={route.req_level}
                          />
                          <select
                            className="ui-select"
                            name="enabled"
                            defaultValue={String(route.enabled)}
                          >
                            <option value="true">sim</option>
                            <option value="false">não</option>
                          </select>
                          <button className="ui-btn" type="submit">
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
      </div>
    </div>
  );
}

/**
 * Configurações do servidor: taxas de refino/composição, refino de acessório, drops,
 * posição de NPC e política de teleporte.
 *
 * O portal não escreve estado vivo nem reinicia o jogo: cada gravação incrementa
 * `config_revision`, e o servidor relê a configuração fora do tick.
 */
export default async function ConfiguracoesPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const params = await searchParams;
  const one = (value: string | string[] | undefined): string =>
    Array.isArray(value) ? (value[0] ?? "") : (value ?? "");
  const ok = one(params.ok).trim();
  const erro = one(params.erro).trim();
  const npcId = Number(one(params.npc));

  let data: RatesView | null = null;
  let drops: DropWire[] = [];
  let routes: RouteWire[] = [];
  let events: WorldEventWire[] = [];
  let npc: NpcWire | null = null;
  let error: string | null = null;
  try {
    data = await getRates();
    [drops, routes, events] = await Promise.all([listDrops(), listRoutes(), getWorldEvents()]);
    if (Number.isInteger(npcId) && npcId > 0) npc = await getNpc(npcId);
  } catch (err) {
    error = err instanceof AdminError ? err.message : "não foi possível ler a configuração agora";
  }

  const refine = data?.rates.filter((row) => row.domain === "refine") ?? [];
  const sanc = data?.rates.filter((row) => row.domain === "sanc") ?? [];
  const combine = data?.rates.filter((row) => row.domain === "combine") ?? [];
  const multipliers = data?.rates.filter((row) => row.domain === "multiplier") ?? [];
  const settings = (data?.rates.filter((row) => row.domain === "setting") ?? []).slice(0, 40);

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Conteúdo</p>
          <h1 className="page-head__title">Configurações</h1>
          <p className="page-head__sub">
            Taxas, drops, NPC e teleportes do servidor, com autor e valor anterior na trilha.
            A gravação não reinicia o jogo: o servidor relê quando a revisão muda. Cada seção
            diz se o valor já tem consumidor no servidor e onde ele é lido.
          </p>
        </div>
        <div className="page-head__actions">
          <Badge tone={error ? "warn" : "ok"}>
            {error ? "indisponível" : `revisão ${data?.revision ?? 0}`}
          </Badge>
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
            <p role="status">
              {ok === "1" ? "Configuração gravada." : `${ok} taxa(s) gravada(s).`}
            </p>
          </div>
        </div>
      )}

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
          <RateTable
            title="Refino +0..+15"
            hint="Basis points (10000 = 100%). O +12 segue o default documentado; mudar aqui não reescreve o seed. Consumido em jogo: `_MSG_UseItem` (`Opcode::UseItem`) → `cpsock::dispatch` → `SelectionStore::use_item` → `Gameplay::use_item` → `Gameplay::refine` lê a taxa a cada tentativa; `tests/config_gameplay.rs` prova recarga e reversão."
            rows={refine}
            domain="refine"
            scale="bps"
            source="WYD International"
          />
          <RateTable
            title="SancRate (pó × refino atual)"
            hint="Tabela do legado preservada: PO cobre 0..5, PL 0..9, Âmago até 11. Consumido em jogo pelo mesmo caminho do refino: `SelectionStore::use_item` → `Gameplay::use_item` → `Gameplay::refine` lê `powder_rate_bps` a cada tentativa; `tests/config_gameplay.rs` prova recarga, reversão e snapshot inválido."
            rows={sanc}
            domain="sanc"
            scale="percent"
            source="SancRate.txt"
          />
          <RateTable
            title="Composição"
            hint="Taxas e custos de Aylin, Compositor e Odin. ‘percent’ é ponto percentual; ‘coin’ é ouro. O snapshot recarrega os valores, mas o handler de composição ainda não existe no servidor de jogo."
            rows={combine}
            domain="combine"
            scale="int"
            source="CompRate.txt"
          />
          <RateTable
            title="Multiplicadores de conteúdo"
            hint="EXP, drop, ouro e ponto de skill (10000 = 100%). Sem linha = sem multiplicador. EXP tem gancho em `gain_exp`, que ainda não é chamado por pacote; drop tem resolvedor em `Gameplay::loot`, mas só teste o chama — não há morte/caixa/pickup nem handler de `_MSG_Attack`; ouro e ponto de skill não têm consumidor."
            rows={multipliers}
            domain="multiplier"
            scale="bps"
            source="game_setting"
          />
          <RateTable
            title="Configurações do servidor"
            hint="Chaves inteiras do gameconfig ligadas a taxa, drop, evento e guerra. As 40 primeiras. O snapshot carrega os valores, mas nenhum sistema de gameplay lê estas chaves ainda."
            rows={settings}
            domain="setting"
            scale="int"
            source="game_setting"
          />

          <div className="card">
            <AccessoryTable rules={data?.accessories ?? []} />
          </div>

          <DropEditor drops={drops} />
          <NpcEditor npc={npc} />
          <RouteEditor routes={routes} />
          <EventEditor events={events} />

          <div className="card">
            <div className="card__head">
              <div>
                <h2 className="card__title">Restaurar default</h2>
                <p className="card__sub">
                  Refino, SancRate, composição e acessório aceitam restauração do valor
                  documentado. Informe o domínio e a chave exatos.
                </p>
              </div>
            </div>
            <div className="card__body">
              <form action={resetRateAction} className="field-grid">
                <label className="login-field">
                  <span>Domínio</span>
                  <select className="ui-select" name="domain" defaultValue="refine">
                    <option value="refine">refine</option>
                    <option value="sanc">sanc</option>
                    <option value="combine">combine</option>
                    <option value="multiplier">multiplier</option>
                  </select>
                </label>
                <label className="login-field">
                  <span>Chave</span>
                  <input className="ui-input" name="key" placeholder="12 · PO:5 · Odin.BASE" required />
                </label>
                <button className="ui-btn" type="submit">
                  Restaurar
                </button>
              </form>
            </div>
          </div>
        </>
      )}
    </>
  );
}
