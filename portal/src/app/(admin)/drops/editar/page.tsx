import { DropRateEditor } from "@/components/drop/drop-rate-editor";
import { DropsNav } from "@/components/drop/drops-nav";
import { RetryButton } from "@/components/RetryButton";
import type { DropRateSlot } from "@/lib/droprate";
import { AdminError } from "@/server/errors";
import { listDropRates } from "@/server/drops";

export const dynamic = "force-dynamic";

export const metadata = { title: "Taxa de drop por slot · WYD2" };

/**
 * Taxa de drop por slot (issue 34).
 *
 * O divisor vive no código do tmserver (`loot.go`, espelho de Basedef.cpp:222) e
 * a web grava um overlay no Postgres. O tmserver lê esse overlay **uma vez no
 * boot**, com `-drop-config-editing`, e a tela diz isso em vez de prometer efeito
 * imediato.
 */
export default async function DropRatesPage() {
  let slots: DropRateSlot[] = [];
  let error: string | null = null;

  try {
    slots = await listDropRates();
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível carregar a tabela de drop do web-api";
  }

  return (
    <>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            Taxa de drop por slot
          </h1>
          <p className="ui-page-sub" style={{ margin: "var(--space-1) 0 0" }}>
            Cada posição do <code>Carry[]</code> tem um divisor próprio: quanto maior, mais raro. Quem
            define a raridade de um item é a posição que ele ocupa no template, não o item.
          </p>
        </div>
      </div>

      <DropsNav active="taxas" />

      <div className="ui-card" style={{ marginBottom: "var(--space-3)" }}>
        <strong>Aplicação</strong>
        <p style={{ margin: "var(--space-1) 0 0" }}>
          O tmserver aplica este overlay com <code>-drop-config-editing</code> /{" "}
          <code>W2PP_DROP_CONFIG_EDITING=true</code>, no boot. Sem a flag, o mundo usa exatamente os
          divisores do código — nenhuma edição daqui vale em runtime.
        </p>
      </div>

      {error && (
        <div className="ui-card ui-empty" style={{ marginBottom: "var(--space-3)" }}>
          <strong>Não deu para carregar a tabela</strong>
          <p>{error}</p>
          <RetryButton />
        </div>
      )}

      {!error && slots.length === 0 && (
        <div className="ui-card ui-empty">
          <strong>Nenhuma posição retornada</strong>
          <p>
            O web-api deveria devolver as 64 posições do <code>DropRate</code>. Se a lista veio vazia,
            o serviço de drop subiu sem o catálogo — confira os logs do <code>webserver</code>.
          </p>
          <RetryButton />
        </div>
      )}

      {!error && slots.length > 0 && <DropRateEditor slots={slots} />}
    </>
  );
}
