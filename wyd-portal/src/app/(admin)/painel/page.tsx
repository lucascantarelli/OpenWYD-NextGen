import { AlertTriangle, Package, Server, Users } from "lucide-react";
import Link from "next/link";

import { Badge } from "@/components/Badge";
import { Stat } from "@/components/Stat";
import { fmtCoin } from "@/lib/format";
import { roleLabel } from "@/lib/role";
import { getCatalog } from "@/server/catalog";
import { getMonitoringSnapshot } from "@/server/monitoring";
import { getSession } from "@/server/session";

export const dynamic = "force-dynamic";

export const metadata = { title: "Visão geral · WYD2" };

const fmtInt = (value: number | null): string =>
  value === null ? "—" : value.toLocaleString("pt-BR", { maximumFractionDigits: 0 });

/**
 * Visão geral: o estado operacional em uma tela.
 *
 * Cada leitura falha sozinha — gateway fora deixa o cartão correspondente em
 * "—", sem derrubar a página. Nenhum número é estimado: o que não tem contrato
 * não aparece.
 */
export default async function PainelPage() {
  const session = await getSession();

  let itemCount: number | null = null;
  try {
    itemCount = (await getCatalog()).items.length;
  } catch {
    itemCount = null;
  }

  const snapshot = await getMonitoringSnapshot();
  const online = snapshot.presence.reachable ? snapshot.presence.total : null;
  const channelsUp = snapshot.channels.filter((channel) => channel.up).length;
  const firing = snapshot.alerts.filter((alert) => alert.state === "firing");
  const errors15m = snapshot.logs.services.reduce((total, row) => total + row.errors, 0);

  const services = [
    {
      name: "Servidor e dados",
      detail: snapshot.presence.reachable
        ? `${snapshot.presence.channels.length} canais lidos`
        : snapshot.presence.error,
      up: snapshot.presence.reachable,
    },
    {
      name: "Métricas",
      detail: snapshot.reachable ? "coleta respondendo" : snapshot.error,
      up: snapshot.reachable,
    },
    {
      name: "Logs",
      detail: snapshot.logs.reachable
        ? `${fmtInt(snapshot.logs.totalLines15m)} linhas em 15 min`
        : snapshot.logs.error,
      up: snapshot.logs.reachable,
    },
    {
      name: "Dashboards",
      detail: snapshot.grafana.health.reachable ? "painéis no ar" : snapshot.grafana.health.error,
      up: snapshot.grafana.health.reachable,
    },
  ];

  return (
    <>
      <div className="hero">
        <div>
          <p className="hero__eyebrow">Operação</p>
          <h1 className="hero__title">O mundo em tempo real.</h1>
          <p className="hero__sub">
            Presença, saúde da stack e conteúdo carregado. O portal só lê o estado vivo — quem
            escreve é o servidor, no tick.
          </p>
          <div className="hero__chips">
            <Badge tone={!snapshot.reachable ? "neutral" : firing.length === 0 ? "ok" : "danger"}>
              {!snapshot.reachable
                ? "alertas indisponíveis"
                : firing.length === 0
                  ? "nenhum alerta ativo"
                  : `${firing.length} alertas ativos`}
            </Badge>
            <Badge tone={snapshot.presence.reachable ? "ok" : "warn"}>
              {snapshot.presence.reachable ? "servidor conectado" : "servidor indisponível"}
            </Badge>
            <Badge tone="accent">sessão {roleLabel(session?.role ?? "player")}</Badge>
          </div>
        </div>

        <div className="hero__numbers">
          <div className="hero__number">
            <span>Jogadores online</span>
            <strong>{fmtInt(online)}</strong>
          </div>
          <div className="hero__number">
            <span>Canais no ar</span>
            <strong>
              {snapshot.reachable ? (
                <>
                  {fmtInt(channelsUp)}
                  <span className="muted"> / {fmtInt(snapshot.channels.length)}</span>
                </>
              ) : (
                "—"
              )}
            </strong>
          </div>
        </div>
      </div>

      <div className="stats">
        <Stat
          label="Jogadores online"
          value={fmtInt(online)}
          pending={online === null}
          hint={online === null ? "presença indisponível" : "sessões ativas agora"}
          icon={Users}
        />
        <Stat
          label="Canais no ar"
          value={`${channelsUp}/${snapshot.channels.length}`}
          pending={!snapshot.reachable}
          hint={snapshot.reachable ? "canais monitorados" : "métricas indisponíveis"}
          icon={Server}
        />
        <Stat
          label="Alertas ativos"
          value={fmtInt(firing.length)}
          pending={!snapshot.reachable}
          hint={
            !snapshot.reachable
              ? "métricas indisponíveis"
              : firing.length === 0
                ? "nada disparando"
                : "ver monitoramento"
          }
          icon={AlertTriangle}
        />
        <Stat
          label="Itens no catálogo"
          value={fmtCoin(itemCount ?? 0)}
          pending={itemCount === null}
          hint={itemCount === null ? "servidor indisponível" : "conteúdo carregado no servidor"}
          icon={Package}
        />
      </div>

      <div className="grid-2">
        <section className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">Serviços</h2>
              <p className="card__sub">Cada fonte responde por si.</p>
            </div>
            <Link className="ui-btn ui-btn--sm ui-btn--ghost" href="/monitoramento">
              Abrir monitoramento
            </Link>
          </div>
          <div className="card__body card__body--flush">
            {services.map((service) => (
              <div className="log-row" key={service.name}>
                <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
                  <span className={`dot ${service.up ? "dot--ok" : "dot--danger"}`} aria-hidden="true" />
                  <strong style={{ fontWeight: 550 }}>{service.name}</strong>
                </span>
                <span className="muted small">{service.detail || "—"}</span>
              </div>
            ))}
          </div>
        </section>

        <section className="card">
          <div className="card__head">
            <div>
              <h2 className="card__title">Ações rápidas</h2>
              <p className="card__sub">O dia a dia do operador.</p>
            </div>
          </div>
          <div className="card__body">
            <div className="chips">
              <Link className="chip" href="/contas">
                Buscar conta
              </Link>
              <Link className="chip" href="/itens">
                Catálogo de itens
              </Link>
              <Link className="chip" href="/monitoramento">
                Saúde da stack
              </Link>
              <Link className="chip" href="/guias">
                Guias
              </Link>
            </div>
            <div className="card__sub" style={{ marginTop: 16 }}>
              <span className="status">
                <span
                  className={`dot ${
                    !snapshot.logs.reachable ? "dot--danger" : errors15m === 0 ? "dot--ok" : "dot--warn"
                  }`}
                  aria-hidden="true"
                />
                {snapshot.logs.reachable ? (
                  <>
                    <strong>{fmtInt(errors15m)}</strong> erros de log em 15 min
                  </>
                ) : (
                  "logs indisponíveis"
                )}
              </span>
            </div>
          </div>
        </section>
      </div>

      <section className="card">
        <div className="card__head">
          <div>
            <h2 className="card__title">Como o portal opera</h2>
            <p className="card__sub">As regras que valem em toda tela.</p>
          </div>
        </div>
        <div className="card__body">
          <ul className="small muted" style={{ margin: 0, paddingLeft: 18, lineHeight: 1.7 }}>
            <li>Toda ação administrativa passa por sessão de staff e deixa registro de auditoria.</li>
            <li>O portal não escreve no estado vivo do personagem; intenções vão pela fila do servidor.</li>
            <li>Credencial, hash e PIN nunca chegam ao browser nem ao log do portal.</li>
          </ul>
        </div>
      </section>
    </>
  );
}
