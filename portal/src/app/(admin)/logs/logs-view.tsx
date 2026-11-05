"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { Badge } from "@/components/Badge";
import { Tabs } from "@/components/Tabs";

const TABS = [
  { id: "all", label: "Tudo" },
  { id: "tmserver", label: "tmserver" },
  { id: "tmserver2", label: "tmserver2" },
  { id: "dbserver", label: "dbserver" },
  { id: "binserver", label: "binserver" },
  { id: "webserver", label: "webserver" },
  { id: "db", label: "db" },
  { id: "portal", label: "portal" },
] as const;

/** Buffer da tela: o tmserver passa de 30 linhas/s em briga de mob. */
const MAX_LINES = 2000;
/** Re-render a cada 100ms, nunca por linha (uma lista ingênua trava a aba). */
const FLUSH_MS = 100;

type Connection = "conectando" | "ao vivo" | "reconectando" | "erro";

interface ServiceState {
  service: string;
  state: string;
  health: string;
}

function lineTone(text: string): string | undefined {
  if (/\b(error|panic|fatal|failed)\b/i.test(text)) return "var(--danger)";
  if (/\bwarn(ing)?\b/i.test(text)) return "var(--warn)";
  return undefined;
}

/**
 * Visualizador dos logs (issue 40).
 *
 * O `EventSource` reconecta sozinho; quando ele reabre pedimos `tail` de novo e o
 * buffer descarta as linhas antigas, então não cresce indefinidamente. Pausar
 * **não** fecha o stream: continua acumulando no buffer e retoma de onde parou.
 */
export function LogsView() {
  const [target, setTarget] = useState<string>("tmserver");
  const [lines, setLines] = useState<string[]>([]);
  const [connection, setConnection] = useState<Connection>("conectando");
  const [paused, setPaused] = useState(false);
  const [follow, setFollow] = useState(true);
  const [filter, setFilter] = useState("");
  const [services, setServices] = useState<ServiceState[]>([]);
  const [dockerError, setDockerError] = useState<string | null>(null);

  const buffer = useRef<string[]>([]);
  const dirty = useRef(false);
  const pausedRef = useRef(false);
  const viewRef = useRef<HTMLPreElement | null>(null);

  useEffect(() => {
    pausedRef.current = paused;
  }, [paused]);

  // Troca de serviço: zera o buffer e reassina o stream.
  useEffect(() => {
    buffer.current = [];
    dirty.current = false;
    setLines([]);
    setConnection("conectando");

    const source = new EventSource(`/api/logs/${target}?tail=200`);

    source.onopen = () => setConnection("ao vivo");
    source.onerror = () => {
      // `EventSource` já reconecta sozinho (o `retry:` vem do servidor).
      setConnection(source.readyState === EventSource.CLOSED ? "erro" : "reconectando");
    };
    source.onmessage = (event) => {
      const line = String(event.data ?? "");
      if (!line) return;
      buffer.current.push(line);
      if (buffer.current.length > MAX_LINES) {
        buffer.current.splice(0, buffer.current.length - MAX_LINES);
      }
      dirty.current = true;
      setConnection("ao vivo");
    };

    return () => source.close();
  }, [target]);

  // Flush periódico do buffer; pausado, o buffer acumula sem renderizar.
  useEffect(() => {
    const timer = setInterval(() => {
      if (pausedRef.current || !dirty.current) return;
      dirty.current = false;
      setLines([...buffer.current]);
    }, FLUSH_MS);
    return () => clearInterval(timer);
  }, []);

  // Chips de status: `docker compose ps` a cada 10s.
  useEffect(() => {
    let alive = true;
    const load = async () => {
      try {
        const res = await fetch("/api/logs/status", { cache: "no-store" });
        if (!alive) return;
        if (!res.ok) {
          setDockerError(res.status === 401 ? "sessão expirada" : "sem permissão");
          return;
        }
        const payload = (await res.json()) as { ok: boolean; services: ServiceState[]; error?: string };
        setServices(payload.services ?? []);
        setDockerError(payload.ok ? null : payload.error ?? "docker indisponível");
      } catch {
        if (alive) setDockerError("não foi possível consultar o status");
      }
    };
    void load();
    const timer = setInterval(() => void load(), 10_000);
    return () => {
      alive = false;
      clearInterval(timer);
    };
  }, []);

  // Seguir = autoscroll no rodapé.
  useEffect(() => {
    if (!follow) return;
    const el = viewRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [lines, follow]);

  const visible = useMemo(() => {
    const needle = filter.trim().toLowerCase();
    if (!needle) return lines;
    return lines.filter((line) => line.toLowerCase().includes(needle));
  }, [lines, filter]);

  const download = useCallback(() => {
    const blob = new Blob([`${buffer.current.join("\n")}\n`], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `${target}-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, "-")}.txt`;
    anchor.click();
    URL.revokeObjectURL(url);
  }, [target]);

  const connectionTone =
    connection === "ao vivo" ? "ok" : connection === "erro" ? "danger" : "warn";

  return (
    <>
      <div className="ui-filters" style={{ alignItems: "center", gap: "var(--space-2)" }}>
        <Tabs tabs={TABS} value={target} onChange={setTarget} label="Serviço" />
        <div style={{ marginLeft: "auto", display: "flex", gap: "var(--space-2)", alignItems: "center" }}>
          <Badge tone={connectionTone}>{connection}</Badge>
          <span className="ui-field__hint">
            {lines.length}/{MAX_LINES} linhas
          </span>
        </div>
      </div>

      <div
        className="ui-filters"
        style={{ alignItems: "center", gap: "var(--space-2)", flexWrap: "wrap" }}
      >
        <input
          className="ui-input"
          style={{ maxWidth: 280 }}
          type="search"
          value={filter}
          placeholder="Filtrar linhas…"
          aria-label="Filtrar linhas"
          onChange={(event) => setFilter(event.target.value)}
        />
        <button className="ui-btn" onClick={() => setPaused((value) => !value)}>
          {paused ? "retomar" : "pausar"}
        </button>
        <label className="ui-field__hint" style={{ display: "flex", gap: 6, alignItems: "center" }}>
          <input
            type="checkbox"
            checked={follow}
            onChange={(event) => setFollow(event.target.checked)}
            style={{ accentColor: "var(--accent)", width: 16, height: 16 }}
          />
          seguir
        </label>
        <button
          className="ui-btn"
          onClick={() => {
            buffer.current = [];
            dirty.current = false;
            setLines([]);
          }}
        >
          limpar
        </button>
        <button className="ui-btn" onClick={download}>
          baixar .txt
        </button>
      </div>

      {dockerError && (
        <p className="ui-field__error" role="status">
          {dockerError}
        </p>
      )}

      <div style={{ display: "flex", gap: 6, flexWrap: "wrap", margin: "var(--space-2) 0" }}>
        {services.map((service) => (
          <Badge
            key={service.service}
            tone={service.state === "running" ? "ok" : "neutral"}
            title={service.health ? `health: ${service.health}` : undefined}
          >
            {`${service.service} · ${service.state}${service.health ? ` (${service.health})` : ""}`}
          </Badge>
        ))}
      </div>

      <pre
        ref={viewRef}
        aria-live="off"
        style={{
          margin: 0,
          padding: "var(--space-3)",
          height: "60vh",
          overflow: "auto",
          background: "var(--surface-2, #0f1115)",
          border: "1px solid var(--border, #2a2f3a)",
          borderRadius: 8,
          fontSize: 12,
          lineHeight: 1.45,
          whiteSpace: "pre-wrap",
          wordBreak: "break-word",
        }}
      >
        {visible.length === 0 ? (
          <span className="ui-field__hint">
            {paused ? "pausado — o buffer continua crescendo" : "aguardando linhas…"}
          </span>
        ) : (
          visible.map((line, index) => (
            <div key={`${index}-${line.slice(0, 12)}`} style={{ color: lineTone(line) }}>
              {line}
            </div>
          ))
        )}
      </pre>

      <p className="ui-field__hint" style={{ marginTop: "var(--space-2)" }}>
        {filter
          ? `${visible.length} de ${lines.length} linhas no filtro`
          : `${lines.length} linhas no buffer (limite ${MAX_LINES})`}
      </p>
    </>
  );
}
