import { Badge } from "@/components/Badge";
import { RetryButton } from "@/components/RetryButton";
import { AdminError } from "@/server/errors";
import { getRates, getServerFlags, type FlagsSnapshot, type RatesSnapshot } from "@/server/rates";

import { RatesPanel } from "./rates-panel";

export const dynamic = "force-dynamic";

export const metadata = { title: "Rates e flags · WYD2" };

/**
 * Rates de conteúdo e flags do processo (issue 36).
 *
 * SancRate/CompRate/QuestsRate são arquivos `:ro` em `tmserver/content/Common/Settings/`;
 * a edição é um overlay no Postgres e o tmserver aplica no boot com
 * `-rates-editing`. As flags de processo são só leitura: elas vivem no
 * `command:` do compose e no ambiente, e o tmserver grava o snapshot no boot.
 */
export default async function ConfigPage() {
  let snapshot: RatesSnapshot | null = null;
  let flags: FlagsSnapshot | null = null;
  let error: string | null = null;

  try {
    [snapshot, flags] = await Promise.all([getRates(), getServerFlags()]);
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível carregar rates e flags do web-api";
  }

  return (
    <>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            Rates e flags
          </h1>
          <p className="ui-page-sub" style={{ margin: "4px 0 0" }}>
            Refino, combine e recompensas de quest são editáveis por overlay; as flags de processo são
            só leitura — elas vêm do <code>docker-compose.yaml</code> e do ambiente do tmserver.
          </p>
        </div>
      </div>

      <div className="ui-card" style={{ marginBottom: 12 }}>
        <strong>Aplicação</strong>
        <p style={{ margin: "4px 0 0" }}>
          O tmserver aplica o overlay de taxas com <code>-rates-editing</code> /{" "}
          <code>W2PP_RATES_EDITING=true</code>, no boot. Sem a flag, o mundo usa só os arquivos.
          Nenhuma flag de processo é recarregada em runtime: mudar uma exige editar o compose/ambiente e{" "}
          <code>docker compose up -d tmserver</code>.
        </p>
      </div>

      {error && (
        <div className="ui-card ui-empty" style={{ marginBottom: 12 }}>
          <strong>Não deu para carregar rates e flags</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      )}

      {!error && flags && (
        <section className="ui-card" style={{ marginBottom: 14 }}>
          <div
            style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}
          >
            <strong>Flags do tmserver</strong>
            {flags.available ? (
              <span className="ui-field__hint" style={{ margin: 0 }}>
                snapshot do último boot · {flags.capturedAt || "sem data"}
              </span>
            ) : (
              <Badge tone="warn">sem snapshot</Badge>
            )}
          </div>

          {!flags.available ? (
            <p className="ui-field__hint" style={{ margin: "6px 0 0" }}>
              {flags.message ||
                "o tmserver ainda não gravou um snapshot: ele precisa subir com -dbserver."}
            </p>
          ) : (
            <table className="ui-table" style={{ marginTop: 8, width: "100%" }}>
              <thead>
                <tr>
                  <th>flag</th>
                  <th>valor atual</th>
                  <th>efeito</th>
                  <th>restart?</th>
                </tr>
              </thead>
              <tbody>
                {flags.flags.map((flag) => (
                  <tr key={flag.name}>
                    <td className="ui-mono">{flag.name}</td>
                    <td className="ui-mono">{flag.value === "" ? "—" : flag.value}</td>
                    <td>{flag.hint}</td>
                    <td>
                      <Badge tone={flag.restart_required ? "warn" : "neutral"}>
                        {flag.restart_required ? "sim" : "não"}
                      </Badge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      )}

      {!error && snapshot && <RatesPanel rates={snapshot.rates} warning={snapshot.warning} />}
    </>
  );
}
