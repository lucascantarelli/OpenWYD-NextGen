"use client";

import { useEffect, useId, useRef, useState } from "react";

export interface ComboOption {
  value: string;
  label: string;
  hint?: string;
}

/**
 * Busca assíncrona com debounce de 200ms.
 *
 * `load` precisa ser estável (definido no módulo ou em `useCallback`), senão o
 * efeito re-dispara a cada render e a lista pisca.
 */
export function Combobox({
  load,
  value,
  onSelect,
  placeholder = "Buscar…",
  label,
  emptyLabel = "Nenhum resultado",
}: {
  load: (query: string) => Promise<ComboOption[]>;
  value?: ComboOption | null;
  onSelect: (option: ComboOption) => void;
  placeholder?: string;
  label: string;
  emptyLabel?: string;
}) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [options, setOptions] = useState<ComboOption[]>([]);
  const [highlight, setHighlight] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const listId = useId();
  const inputRef = useRef<HTMLInputElement | null>(null);

  useEffect(() => {
    if (!open) return;
    let alive = true;
    const timer = setTimeout(async () => {
      setBusy(true);
      try {
        const found = await load(query);
        if (!alive) return;
        setOptions(found);
        setHighlight(0);
        setError(null);
      } catch {
        if (alive) setError("Falha na busca.");
      } finally {
        if (alive) setBusy(false);
      }
    }, 200);

    return () => {
      alive = false;
      clearTimeout(timer);
    };
  }, [query, open, load]);

  function choose(option: ComboOption) {
    onSelect(option);
    setQuery(option.label);
    setOpen(false);
  }

  return (
    <div className="ui-combo">
      <input
        ref={inputRef}
        className="ui-input"
        role="combobox"
        aria-label={label}
        aria-expanded={open}
        aria-controls={listId}
        aria-autocomplete="list"
        aria-activedescendant={open && options[highlight] ? `${listId}-${highlight}` : undefined}
        placeholder={placeholder}
        value={query}
        onChange={(event) => {
          setQuery(event.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
        onBlur={() => {
          // Deixa o clique na opção acontecer antes de fechar.
          setTimeout(() => setOpen(false), 120);
        }}
        onKeyDown={(event) => {
          if (event.key === "ArrowDown") {
            event.preventDefault();
            setOpen(true);
            setHighlight((i) => Math.min(i + 1, Math.max(options.length - 1, 0)));
          } else if (event.key === "ArrowUp") {
            event.preventDefault();
            setHighlight((i) => Math.max(i - 1, 0));
          } else if (event.key === "Enter") {
            if (open && options[highlight]) {
              event.preventDefault();
              choose(options[highlight]);
            }
          } else if (event.key === "Escape") {
            setOpen(false);
          }
        }}
      />

      {open && (
        <ul className="ui-combo__list" id={listId} role="listbox">
          {busy && <li className="ui-combo__hint" style={{ padding: "6px 8px" }}>Buscando…</li>}
          {error && (
            <li className="ui-combo__hint" style={{ padding: "6px 8px" }}>
              {error}
            </li>
          )}
          {!busy && !error && options.length === 0 && (
            <li className="ui-combo__hint" style={{ padding: "6px 8px" }}>
              {emptyLabel}
            </li>
          )}
          {options.map((option, index) => (
            <li
              key={option.value}
              id={`${listId}-${index}`}
              role="option"
              aria-selected={index === highlight}
              className="ui-combo__option"
              onMouseEnter={() => setHighlight(index)}
              onMouseDown={(event) => {
                event.preventDefault();
                choose(option);
              }}
            >
              <span>{option.label}</span>
              {option.hint && <span className="ui-combo__hint">{option.hint}</span>}
            </li>
          ))}
        </ul>
      )}

      {value && (
        <p className="ui-field__hint" style={{ marginTop: 4 }}>
          Selecionado: <span className="ui-mono">{value.value}</span> — {value.label}
        </p>
      )}
    </div>
  );
}
