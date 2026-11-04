import { LogsView } from "./logs-view";

export const dynamic = "force-dynamic";

export const metadata = { title: "Logs dos containers · WYD2" };

/**
 * Logs da stack na UI (issue 40).
 *
 * O stream vem do sidecar `logpipe` no compose (socket do Docker read-only) ou
 * do `docker compose logs -f` quando o portal roda no host; a tela não sabe
 * qual dos dois — o BFF escolhe por `LOG_PIPE_URL`. Sem fonte, mostra o motivo
 * em vez de fingir um stream vazio.
 */
export default function LogsPage() {
  return (
    <>
      <h1 className="ui-page-title">Logs dos containers</h1>
      <p className="ui-page-sub">
        Stream do stdout/stderr dos containers, por serviço. Segredos e DSN são mascarados no
        servidor antes de chegar aqui; fechar a aba encerra o stream.
      </p>
      <LogsView />
    </>
  );
}
