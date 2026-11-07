"use client";

export function Form({
  onSubmit,
  children,
  submitLabel = "Salvar",
  pending = false,
  formError,
  extraActions,
}: {
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
  children: React.ReactNode;
  submitLabel?: string;
  pending?: boolean;
  formError?: string | null;
  extraActions?: React.ReactNode;
}) {
  return (
    <form
      className="ui-form"
      onSubmit={onSubmit}
      aria-busy={pending || undefined}
      noValidate
    >
      {children}
      {formError && (
        <p className="ui-field__error" role="alert">
          {formError}
        </p>
      )}
      <div className="ui-form__actions">
        <button type="submit" className="ui-btn ui-btn--primary" disabled={pending}>
          {pending ? "Salvando…" : submitLabel}
        </button>
        {extraActions}
      </div>
    </form>
  );
}
