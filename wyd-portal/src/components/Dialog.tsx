"use client";

import { useEffect, useId, useRef } from "react";

/**
 * `<dialog>` nativo: `showModal()` já prende o foco no conteúdo, cobre o resto
 * da página e fecha com Esc. Não inventamos focus trap.
 */
export function Dialog({
  open,
  onClose,
  title,
  description,
  children,
  footer,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
}) {
  const ref = useRef<HTMLDialogElement | null>(null);
  const titleId = useId();
  const descId = useId();

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    if (open && !el.open) el.showModal();
    if (!open && el.open) el.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      className="ui-dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      aria-describedby={description ? descId : undefined}
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
      onClose={onClose}
    >
      <div className="ui-dialog__head">
        <h2 className="ui-dialog__title" id={titleId}>
          {title}
        </h2>
        {description && (
          <p className="ui-dialog__desc" id={descId}>
            {description}
          </p>
        )}
      </div>
      <div className="ui-dialog__body">{children}</div>
      {footer && <div className="ui-dialog__foot">{footer}</div>}
    </dialog>
  );
}
