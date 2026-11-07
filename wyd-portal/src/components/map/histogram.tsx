import { barHeight, topValues, totalCells, valueLabel, type AttributeMapMeaning, type AttributeMapValueCount } from "@/lib/attributemap";
import { fmtCoin } from "@/lib/format";

/**
 * Histograma do `AttributeMap.dat` (issue 27).
 *
 * As 256 barras são o arquivo inteiro; a tabela ao lado mostra os 16 valores mais
 * frequentes com o significado conhecido. Não é um gráfico decorativo: é a
 * conferência de que a transformação atingiu o que se esperava.
 */
export function Histogram({
  buckets,
  meanings = [],
  compact = false,
}: {
  buckets: readonly AttributeMapValueCount[];
  meanings?: readonly AttributeMapMeaning[];
  compact?: boolean;
}) {
  if (buckets.length === 0) {
    return <p className="ui-field__hint">sem histograma para mostrar</p>;
  }
  const max = Math.max(...buckets.map((bucket) => bucket.count), 1);
  const total = totalCells(buckets);
  const top = topValues(buckets, compact ? 6 : 16);

  return (
    <div style={{ display: "grid", gap: "var(--space-2)" }}>
      <div
        role="img"
        aria-label={`Histograma de ${buckets.length} valores do AttributeMap`}
        style={{
          display: "flex",
          alignItems: "flex-end",
          gap: 1,
          height: compact ? 56 : 96,
          padding: "var(--space-1)",
          border: "1px solid var(--border)",
          borderRadius: "var(--radius-2)",
          background: "var(--surface-2)",
        }}
      >
        {buckets.map((bucket) => (
          <span
            key={bucket.value}
            title={`valor ${bucket.value}: ${fmtCoin(bucket.count)} células`}
            style={{
              flex: 1,
              height: `${barHeight(bucket.count, max)}%`,
              background: bucket.count > 0 ? "var(--accent)" : "var(--border)",
              opacity: bucket.count > 0 ? 0.75 : 0.3,
              borderRadius: 1,
              alignSelf: "flex-end",
            }}
          />
        ))}
      </div>

      <div style={{ display: "flex", justifyContent: "space-between", gap: "var(--space-2)", flexWrap: "wrap" }}>
        <span className="ui-field__hint" style={{ margin: 0 }}>
          {buckets.length} valores · {fmtCoin(total)} células
        </span>
        <span className="ui-field__hint" style={{ margin: 0 }}>
          barras na escala do valor mais frequente
        </span>
      </div>

      <table className="ui-table">
        <thead>
          <tr>
            <th scope="col">valor</th>
            <th scope="col">células</th>
            <th scope="col">fatia</th>
          </tr>
        </thead>
        <tbody>
          {top.map((bucket) => (
            <tr key={bucket.value}>
              <td title={meanings.find((m) => !m.bit && m.value === bucket.value)?.description ?? undefined}>
                {valueLabel(bucket.value, meanings)}
              </td>
              <td>{fmtCoin(bucket.count)}</td>
              <td>{total > 0 ? `${((bucket.count / total) * 100).toFixed(2)}%` : "—"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
