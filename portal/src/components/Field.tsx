export function Field({
  label,
  hint,
  error,
  htmlFor,
  children,
}: {
  label: string;
  hint?: string;
  error?: string;
  htmlFor?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="ui-field">
      <label className="ui-field__label" htmlFor={htmlFor}>
        {label}
      </label>
      {children}
      {error ? (
        <span className="ui-field__error" role="alert">
          {error}
        </span>
      ) : (
        hint && <span className="ui-field__hint">{hint}</span>
      )}
    </div>
  );
}
