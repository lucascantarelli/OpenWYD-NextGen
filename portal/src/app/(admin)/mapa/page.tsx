import { Histogram } from "@/components/map/histogram";
import { TransformPanel } from "@/components/map/transform-panel";
import { RetryButton } from "@/components/RetryButton";
import {
  MAP_BYTES,
  MAP_DIM,
  shortSha,
  totalCells,
  WORLD_SCALE,
  type AttributeMapInfo,
} from "@/lib/attributemap";
import { fmtCoin } from "@/lib/format";
import { getAttributeMapInfo } from "@/server/attributemap";
import { AdminError } from "@/server/errors";

export const dynamic = "force-dynamic";

export const metadata = { title: "Mapa de colisão · WYD2" };

/**
 * Editor do `AttributeMap.dat` (issue 27).
 *
 * Lê o arquivo do content tree, mostra o histograma e permite transformações em
 * massa com pré-visualização + download. **Não** escreve o arquivo: aplicar é
 * substituir `Release/TMsrv/run/AttributeMap.dat` (com backup) e reiniciar o
 * tmserver.
 */
export default async function AttributeMapPage() {
  let info: AttributeMapInfo | null = null;
  let error: string | null = null;

  try {
    info = await getAttributeMapInfo();
  } catch (err) {
    error =
      err instanceof AdminError
        ? err.message
        : "não foi possível ler o AttributeMap do web-api";
  }

  return (
    <>
      <div className="ui-filters">
        <div>
          <h1 className="ui-page-title" style={{ margin: 0 }}>
            Mapa de colisão
          </h1>
          <p className="ui-page-sub" style={{ margin: "var(--space-1) 0 0" }}>
            <code>AttributeMap.dat</code>: {MAP_DIM} × {MAP_DIM} células, cada uma cobrindo um bloco{" "}
            {WORLD_SCALE} × {WORLD_SCALE} do mundo. Transformações em massa com pré-visualização — o
            arquivo do servidor nunca é reescrito por aqui.
          </p>
        </div>
      </div>

      {error && (
        <div className="ui-card ui-empty" style={{ marginBottom: "var(--space-3)" }}>
          <strong>Não deu para carregar o mapa</strong>
          <p>{error}</p>
          <p className="ui-field__hint">
            O web-api responde <code>INVALID</code> quando subiu sem <code>-content</code>, quando o
            arquivo não existe em <code>Release/TMsrv/run/</code> ou quando o tamanho não é
            exatamente {fmtCoin(MAP_BYTES)} bytes.
          </p>
          <RetryButton />
        </div>
      )}

      {info && (
        <>
          <div className="ui-card" style={{ marginBottom: "var(--space-3)" }}>
            <div style={{ display: "flex", gap: "var(--space-3)", flexWrap: "wrap", alignItems: "baseline" }}>
              <strong>Arquivo atual</strong>
              <span className="ui-field__hint" style={{ margin: 0 }} title={info.sha256}>
                sha256 {shortSha(info.sha256)} · {fmtCoin(MAP_BYTES)} bytes ·{" "}
                {fmtCoin(totalCells(info.histogram))} células
              </span>
            </div>
            <p className="ui-field__hint" style={{ margin: "var(--space-1) 0 var(--space-2)" }}>
              célula = bloco de {info.world_scale} × {info.world_scale} do mundo; para editar em
              coordenadas de mundo, a tela divide por {info.world_scale}.
            </p>
            <Histogram buckets={info.histogram} meanings={info.meanings} />
          </div>

          <div className="ui-card" style={{ marginBottom: "var(--space-3)" }}>
            <strong>Significados</strong>
            <p className="ui-field__hint" style={{ margin: "var(--space-1) 0 var(--space-2)" }}>
              valores fora desta tabela são legado preservado — não invente significado para eles.
            </p>
            <table className="ui-table">
              <thead>
                <tr>
                  <th scope="col">valor</th>
                  <th scope="col">nome</th>
                  <th scope="col">o que faz</th>
                  <th scope="col">tipo</th>
                </tr>
              </thead>
              <tbody>
                {info.meanings.map((meaning) => (
                  <tr key={`${meaning.value}-${meaning.bit ? "bit" : "valor"}`}>
                    <td>{meaning.value}</td>
                    <td>{meaning.name}</td>
                    <td>{meaning.description}</td>
                    <td>{meaning.bit ? "bit" : "valor exato"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <TransformPanel meanings={info.meanings} />

          <div className="ui-card" style={{ marginTop: "var(--space-3)" }}>
            <strong>Como aplicar</strong>
            <ol style={{ margin: "var(--space-1) 0 0", paddingLeft: "1.2em" }}>
              <li>
                baixe <code>AttributeMap_New.dat</code> e confira o <code>sha256</code> mostrado na
                pré-visualização;
              </li>
              <li>
                faça backup do atual e copie o novo para{" "}
                <code>Release/TMsrv/run/AttributeMap.dat</code>;
              </li>
              <li>
                <code>docker compose restart tmserver</code> e confirme{" "}
                <code>walkability grid baked</code> no log.
              </li>
            </ol>
          </div>
        </>
      )}
    </>
  );
}
