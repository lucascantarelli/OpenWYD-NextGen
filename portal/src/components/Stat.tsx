export function Stat({
  label,
  value,
  hint,
  pending,
}: {
  label: string;
  value: string;
  hint?: string;
  /** Sem dados ainda: mostra "—" e o motivo em `hint`. */
  pending?: boolean;
}) {
  return (
    <div className="ui-stat">
      <div className="ui-stat__label">{label}</div>
      <div className="ui-stat__value" aria-busy={pending || undefined}>
        {pending ? "—" : value}
      </div>
      {hint && <div className="ui-stat__hint">{hint}</div>}
    </div>
  );
}
