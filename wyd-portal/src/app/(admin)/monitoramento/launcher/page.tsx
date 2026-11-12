import { MonitorNav } from "@/components/MonitorNav";
import { resumoTabela, resumo } from "@/server/launcher-telemetria";

export const dynamic = "force-dynamic";

export const metadata = { title: "Launcher · Monitoramento · WYD2" };

const fmtInt = (value: number): string =>
  value.toLocaleString("pt-BR", { maximumFractionDigits: 0 });

const fmtPct = (value: number): string =>
  `${(value * 100).toLocaleString("pt-BR", { maximumFractionDigits: 1 })}%`;

const fmtQuando = (iso: string): string => {
  const data = new Date(iso);
  return Number.isNaN(data.getTime()) ? "—" : data.toLocaleString("pt-BR");
};

/**
 * Telemetria do launcher, agregada por `versao + operacao`.
 *
 * A fonte é o mesmo módulo que recebe o `POST` público: nada aqui lê banco nem
 * chama o gateway. A agregação é em processo, então a tela deixa explícito que a
 * janela é “desde o último boot” e que o log estruturado é a série histórica.
 */
export default function LauncherTelemetriaPage() {
  const linhas = resumoTabela();
  const total = resumo();
  const falhas = linhas.reduce((acc, linha) => acc + linha.falhas, 0);
  const eventos = linhas.reduce((acc, linha) => acc + linha.eventos, 0);

  return (
    <>
      <div className="page-head">
        <div>
          <p className="page-head__eyebrow">Operação</p>
          <h1 className="page-head__title">Launcher</h1>
          <p className="page-head__sub">
            Eventos anônimos enviados pelo launcher após check, config, patch e launch —
            agregados por versão e operação. Nada aqui identifica jogador.
          </p>
        </div>
      </div>

      <MonitorNav current="/monitoramento/launcher" />

      <div className="monitor-strip" role="list" aria-label="Resumo da telemetria">
        <span className="status" role="listitem">
          <span className={`dot ${eventos > 0 ? "dot--ok" : "dot--neutral"}`} aria-hidden="true" />
          Eventos <strong>{fmtInt(eventos)}</strong>
        </span>
        <span className="status" role="listitem">
          <span className={`dot ${falhas > 0 ? "dot--warn" : "dot--ok"}`} aria-hidden="true" />
          Falhas <strong>{fmtInt(falhas)}</strong>
        </span>
        <span className="status" role="listitem">
          Chaves <strong>{fmtInt(linhas.length)}</strong>
        </span>
        <span className="status" role="listitem">
          Descartadas <strong>{fmtInt(total.descartados)}</strong>
        </span>
      </div>

      <section className="card">
        <div className="card__head">
          <div>
            <h2 className="card__title">Por versão e operação</h2>
            <p className="card__sub">
              Janela desde o último boot do portal. O log estruturado
              (<code>[launcher-telemetria]</code>) continua sendo a série histórica.
            </p>
          </div>
        </div>
        {linhas.length === 0 ? (
          <div className="card__body">
            <div className="empty">
              <strong>Nenhum evento ainda</strong>
              <p>
                Assim que um launcher publicar telemetria nesta instância, os agregados
                aparecem aqui.
              </p>
            </div>
          </div>
        ) : (
          <div className="card__body card__body--flush">
            <div className="ui-table-scroll">
              <table className="ui-table">
                <thead>
                  <tr>
                    <th>Versão</th>
                    <th>Operação</th>
                    <th className="ui-align-right">Eventos</th>
                    <th className="ui-align-right">Falhas</th>
                    <th className="ui-align-right">Taxa de falha</th>
                    <th className="ui-align-right">Duração média</th>
                    <th className="ui-align-right">Pior duração</th>
                    <th>Último evento</th>
                  </tr>
                </thead>
                <tbody>
                  {linhas.map((linha) => (
                    <tr key={`${linha.versao}-${linha.operacao}`}>
                      <td className="mono">{linha.versao}</td>
                      <td className="mono">{linha.operacao}</td>
                      <td className="ui-align-right">{fmtInt(linha.eventos)}</td>
                      <td className="ui-align-right">{fmtInt(linha.falhas)}</td>
                      <td className="ui-align-right">{fmtPct(linha.taxaFalha)}</td>
                      <td className="ui-align-right">{`${fmtInt(linha.mediaMs)} ms`}</td>
                      <td className="ui-align-right">{`${fmtInt(linha.duracaoMaxMs)} ms`}</td>
                      <td className="small">{fmtQuando(linha.ultimaEm)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </section>
    </>
  );
}
