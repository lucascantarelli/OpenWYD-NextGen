"use client";

import { describeRect, rectError, WORLD_MAX, type AttributeMapRect } from "@/lib/attributemap";

const FIELDS: { key: keyof AttributeMapRect; label: string }[] = [
  { key: "min_x", label: "x mínimo" },
  { key: "min_y", label: "y mínimo" },
  { key: "max_x", label: "x máximo" },
  { key: "max_y", label: "y máximo" },
];

/**
 * Retângulo de mundo (0..4095) do editor de `AttributeMap` (issue 27).
 *
 * O esquema ao lado é só coordenada — não é o terreno. Ele existe para o
 * operador ver a extensão do retângulo e a conversão para célula (`x/4`), que é
 * onde esse tipo de ferramenta costuma errar por um fator de 4.
 */
export function RectPicker({
  rect,
  onChange,
  disabled,
}: {
  rect: AttributeMapRect;
  onChange: (rect: AttributeMapRect) => void;
  disabled?: boolean;
}) {
  const error = rectError(rect);
  const scale = 200 / (WORLD_MAX + 1);
  const width = Math.max(2, (rect.max_x - rect.min_x + 1) * scale);
  const height = Math.max(2, (rect.max_y - rect.min_y + 1) * scale);

  return (
    <div style={{ display: "grid", gap: "var(--space-2)", gridTemplateColumns: "minmax(220px, 1fr) 200px", alignItems: "start" }}>
      <div>
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(110px, 1fr))",
            gap: "var(--space-2)",
          }}
        >
          {FIELDS.map((field) => (
            <label key={field.key} className="ui-field">
              <span>{field.label} (mundo)</span>
              <input
                type="number"
                min={0}
                max={WORLD_MAX}
                value={rect[field.key]}
                disabled={disabled}
                aria-invalid={error ? true : undefined}
                onChange={(event) =>
                  onChange({ ...rect, [field.key]: Number(event.target.value) })
                }
              />
            </label>
          ))}
        </div>
        <p className="ui-field__hint" style={{ marginTop: "var(--space-1)" }}>
          {describeRect(rect)}
        </p>
        {error && (
          <p className="ui-field__error" role="alert">
            {error}
          </p>
        )}
      </div>

      <svg
        viewBox="0 0 200 200"
        width={200}
        height={200}
        role="img"
        aria-label={`Retângulo do mundo: ${describeRect(rect)}`}
        style={{ border: "1px solid var(--border)", borderRadius: "var(--radius-2)", background: "var(--surface-2)" }}
      >
        {[0, 50, 100, 150, 200].map((tick) => (
          <g key={tick} stroke="var(--border)" strokeWidth={tick === 0 || tick === 200 ? 1 : 0.5} opacity={0.6}>
            <line x1={tick} y1={0} x2={tick} y2={200} />
            <line x1={0} y1={tick} x2={200} y2={tick} />
          </g>
        ))}
        <rect
          x={rect.min_x * scale}
          y={rect.min_y * scale}
          width={width}
          height={height}
          fill="var(--accent)"
          fillOpacity={0.25}
          stroke="var(--accent)"
          strokeWidth={1.5}
        />
      </svg>
    </div>
  );
}
