import type { LucideIcon } from "lucide-react";

export function Stat({
  label,
  value,
  hint,
  pending,
  icon: Icon,
}: {
  label: string;
  value: string;
  hint?: string;
  /** Sem leitura: mostra "—" e o motivo em `hint`. */
  pending?: boolean;
  icon?: LucideIcon;
}) {
  return (
    <div className="stat">
      <div className="stat__top">
        <div className="stat__label">{label}</div>
        {Icon ? (
          <span className="stat__icon" aria-hidden="true">
            <Icon size={16} strokeWidth={1.7} />
          </span>
        ) : null}
      </div>
      <div className="stat__value" aria-busy={pending || undefined}>
        {pending ? "—" : value}
      </div>
      {hint ? <div className="stat__hint">{hint}</div> : null}
    </div>
  );
}
