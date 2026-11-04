"use client";

import { createContext, useCallback, useContext, useMemo, useState } from "react";

export type ToastTone = "info" | "ok" | "error";

interface ToastItem {
  id: number;
  message: string;
  tone: ToastTone;
}

type PushToast = (message: string, tone?: ToastTone) => void;

const ToastContext = createContext<PushToast>(() => {});

/** `useToast()` devolve `push(message, tone?)`; cada toast some em 4s. */
export function useToast(): PushToast {
  return useContext(ToastContext);
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([]);

  const push = useCallback<PushToast>((message, tone = "info") => {
    const id = Date.now() + Math.random();
    setItems((current) => [...current, { id, message, tone }]);
    setTimeout(() => {
      setItems((current) => current.filter((item) => item.id !== id));
    }, 4000);
  }, []);

  // `push` é estável; o value só muda se o provider remontar.
  const value = useMemo(() => push, [push]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="ui-toasts" role="status" aria-live="polite">
        {items.map((item) => (
          <div
            key={item.id}
            className={item.tone === "info" ? "ui-toast" : `ui-toast ui-toast--${item.tone}`}
          >
            {item.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
