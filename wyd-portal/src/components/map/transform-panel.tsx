"use client";

import { useMemo, useState } from "react";

import { Histogram } from "@/components/map/histogram";
import { RectPicker } from "@/components/map/rect-picker";
import { useToast } from "@/components/Toast";
import {
  ATTRIBUTE_OPERATIONS,
  operationOf,
  shortSha,
  transformError,
  transformKey,
  transformQuery,
  type AttributeMapFilter,
  type AttributeMapMeaning,
  type AttributeMapRect,
  type TransformParams,
  type TransformSummary,
} from "@/lib/attributemap";
import { fmtCoin } from "@/lib/format";
import { previewAttributeMap } from "@/server/actions/attributemap";

const DEFAULT_RECT: AttributeMapRect = { min_x: 0, min_y: 0, max_x: 4095, max_y: 4095 };
const DEFAULT_FILTER: AttributeMapFilter = {
  enabled: false,
  exact_value: 0,
  mask: 0,
  match_value: 0,
};

/**
 * Transformação em massa do `AttributeMap.dat` (issue 27).
 *
 * Fluxo obrigatório: pré-visualizar → conferir contagem/histograma/hash →
 * baixar. O botão de download só habilita para **os mesmos parâmetros** que
 * foram pré-visualizados, então não dá para baixar um mapa que ninguém viu.
 *
 * O retângulo é digitado em coordenadas de mundo; a tela mostra a célula ao
 * lado porque `x/4` é a conversão que o servidor aplica.
 */
export function TransformPanel({ meanings }: { meanings: readonly AttributeMapMeaning[] }) {
  const toast = useToast();
  const [operation, setOperation] = useState(1);
  const [operand, setOperand] = useState(64);
  const [rectEnabled, setRectEnabled] = useState(false);
  const [rect, setRect] = useState<AttributeMapRect>(DEFAULT_RECT);
  const [filter, setFilter] = useState<AttributeMapFilter>(DEFAULT_FILTER);
  const [previewed, setPreviewed] = useState<{ key: string; summary: TransformSummary } | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const op = operationOf(operation);
  const params: TransformParams = useMemo(
    () => ({
      operation,
      operand,
      rect: rectEnabled ? rect : null,
      filter: filter.enabled ? filter : null,
    }),
    [operation, operand, rectEnabled, rect, filter],
  );
  const key = transformKey(params);
  const invalid = transformError(params);
  const canDownload = !invalid && previewed?.key === key;

  async function preview() {
    setPending(true);
    const result = await previewAttributeMap(params);
    setPending(false);
    if (!result.ok) {
      setError(result.error);
      toast(result.error, "error");
      return;
    }
    setError(null);
    setPreviewed({ key, summary: result.summary });
  }

  const downloadHref = `/api/mapa/download?${transformQuery(params)}`;

  return (
    <section className="ui-card" style={{ marginTop: "var(--space-3)" }}>
      <strong>Transformação em massa</strong>
      <p className="ui-field__hint" style={{ margin: "var(--space-1) 0 var(--space-2)" }}>
        A transformação é aplicada em memória e devolvida como arquivo novo. O
        <code> AttributeMap.dat </code> do servidor só muda quando você substituir o arquivo e
        reiniciar o tmserver.
      </p>

      <label className="ui-field" style={{ maxWidth: 520 }}>
        <span>Operação</span>
        <select value={operation} onChange={(event) => setOperation(Number(event.target.value))}>
          {ATTRIBUTE_OPERATIONS.map((candidate) => (
            <option key={candidate.value} value={candidate.value}>
              {candidate.label}
            </option>
          ))}
        </select>
        <span className="ui-field__hint">{op.hint}</span>
        <span className="ui-field__hint">{op.effect}</span>
      </label>

      <div style={{ display: "flex", gap: "var(--space-3)", flexWrap: "wrap", marginTop: "var(--space-2)" }}>
        <label className="ui-field" style={{ width: 160 }}>
          <span>Operando {op.usesOperand ? "" : "(ignorado)"}</span>
          <input
            type="number"
            min={0}
            max={255}
            value={operand}
            disabled={!op.usesOperand}
            onChange={(event) => setOperand(Number(event.target.value))}
          />
        </label>
        <div className="ui-field" style={{ justifyContent: "end" }}>
          <label style={{ display: "inline-flex", gap: "var(--space-1)", alignItems: "center", minHeight: 32 }}>
            <input
              type="checkbox"
              checked={rectEnabled}
              onChange={(event) => setRectEnabled(event.target.checked)}
            />
            limitar a um retângulo
          </label>
          <span className="ui-field__hint">sem retângulo, a operação percorre o mapa inteiro (1.048.576 células)</span>
        </div>
      </div>

      {rectEnabled && (
        <div style={{ marginTop: "var(--space-2)" }}>
          <RectPicker rect={rect} onChange={setRect} disabled={pending} />
        </div>
      )}

      <div className="ui-card" style={{ marginTop: "var(--space-3)", padding: "var(--space-2)" }}>
        <label style={{ display: "inline-flex", gap: "var(--space-1)", alignItems: "center", minHeight: 32 }}>
          <input
            type="checkbox"
            checked={filter.enabled}
            onChange={(event) =>
              setFilter((current) => ({ ...current, enabled: event.target.checked }))
            }
          />
          filtrar células por valor
        </label>

        {filter.enabled && (
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fit, minmax(150px, 1fr))",
              gap: "var(--space-2)",
              marginTop: "var(--space-2)",
            }}
          >
            <label className="ui-field">
              <span>máscara (0 = valor exato)</span>
              <input
                type="number"
                min={0}
                max={255}
                value={filter.mask}
                onChange={(event) => setFilter((current) => ({ ...current, mask: Number(event.target.value) }))}
              />
            </label>
            <label className="ui-field">
              <span>valor exato</span>
              <input
                type="number"
                min={0}
                max={255}
                value={filter.exact_value}
                disabled={filter.mask !== 0}
                onChange={(event) =>
                  setFilter((current) => ({ ...current, exact_value: Number(event.target.value) }))
                }
              />
            </label>
            <label className="ui-field">
              <span>valor de comparação</span>
              <input
                type="number"
                min={0}
                max={255}
                value={filter.match_value}
                disabled={filter.mask === 0}
                onChange={(event) =>
                  setFilter((current) => ({ ...current, match_value: Number(event.target.value) }))
                }
              />
            </label>
          </div>
        )}

        <p className="ui-field__hint" style={{ marginBottom: 0 }}>
          {filter.enabled
            ? filter.mask === 0
              ? `só células com valor exatamente ${filter.exact_value}`
              : `só células em que (valor & ${filter.mask}) = (${filter.match_value} & ${filter.mask})`
            : "sem filtro: todas as células do retângulo entram na conta"}
        </p>
      </div>

      {invalid && (
        <p className="ui-field__error" role="alert" style={{ marginTop: "var(--space-2)" }}>
          {invalid}
        </p>
      )}
      {error && (
        <p className="ui-field__error" role="alert" style={{ marginTop: "var(--space-2)" }}>
          {error}
        </p>
      )}

      <div className="ui-form__actions">
        <button
          type="button"
          className="ui-btn ui-btn--primary"
          disabled={pending || !!invalid}
          onClick={preview}
        >
          {pending ? "calculando…" : "pré-visualizar"}
        </button>
        <a
          className="ui-btn"
          href={canDownload ? downloadHref : undefined}
          aria-disabled={!canDownload}
          download
          onClick={(event) => {
            if (!canDownload) event.preventDefault();
          }}
          style={canDownload ? undefined : { opacity: 0.5, pointerEvents: "none" }}
        >
          baixar AttributeMap_New.dat
        </a>
        {!previewed && (
          <span className="ui-field__hint" style={{ margin: 0 }}>
            pré-visualize antes de baixar
          </span>
        )}
        {previewed && previewed.key !== key && (
          <span className="ui-field__hint" style={{ margin: 0 }}>
            os parâmetros mudaram: pré-visualize de novo para liberar o download
          </span>
        )}
      </div>

      {previewed && (
        <div style={{ marginTop: "var(--space-3)" }}>
          <div style={{ display: "flex", gap: "var(--space-2)", flexWrap: "wrap", alignItems: "baseline" }}>
            <strong>
              {previewed.summary.changed_count === 0
                ? "nenhuma célula mudaria"
                : `${fmtCoin(previewed.summary.changed_count)} células mudariam`}
            </strong>
            <span className="ui-field__hint" style={{ margin: 0 }} title={previewed.summary.new_sha256}>
              novo arquivo {shortSha(previewed.summary.new_sha256)} · original{" "}
              {shortSha(previewed.summary.original_sha256)}
            </span>
            <span className="ui-field__hint" style={{ margin: 0 }}>
              {previewed.summary.filename}
            </span>
          </div>

          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))",
              gap: "var(--space-2)",
              marginTop: "var(--space-2)",
            }}
          >
            <div>
              <span className="ui-field__hint" style={{ margin: 0 }}>antes</span>
              <Histogram buckets={previewed.summary.before_histogram} meanings={meanings} compact />
            </div>
            <div>
              <span className="ui-field__hint" style={{ margin: 0 }}>depois</span>
              <Histogram buckets={previewed.summary.after_histogram} meanings={meanings} compact />
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
