"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";

import { Badge } from "@/components/Badge";
import { Combobox } from "@/components/Combobox";
import { Dialog } from "@/components/Dialog";
import { Field } from "@/components/Field";
import { useToast } from "@/components/Toast";
import { fmtCoin } from "@/lib/format";
import {
  ITEM_INDEX_MAX,
  RATE_MAX,
  configChanged,
  rateLabel,
  remainingDrops,
  validateWorldEvent,
  worldEventStatus,
  type WorldEventConfig,
} from "@/lib/worldevent";
import { loadItemOptions } from "@/lib/itemsearch";
import { saveWorldEventConfig } from "@/server/actions/events";

export interface EventItem {
  index: number;
  name: string;
}

function Toggle({
  label,
  hint,
  checked,
  onChange,
  disabled,
}: {
  label: string;
  hint?: string;
  checked: boolean;
  onChange: (value: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <label
      style={{
        display: "flex",
        alignItems: "flex-start",
        gap: "var(--space-2)",
        minHeight: 32,
        cursor: disabled ? "not-allowed" : "pointer",
      }}
    >
      <input
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={(event) => onChange(event.target.checked)}
        style={{ accentColor: "var(--accent)", width: 16, height: 16, marginTop: 8 }}
      />
      <span>
        <span style={{ fontSize: 13 }}>{label}</span>
        {hint && (
          <span className="ui-field__hint" style={{ display: "block" }}>
            {hint}
          </span>
        )}
      </span>
    </label>
  );
}

/**
 * Editor do evento global (issue 28).
 *
 * `SetWorldEventConfig` substitui o config inteiro, então o rascunho sempre
 * carrega os 10 campos e o "salvar" manda tudo. O estado mostrado no topo é o
 * que o servidor confirmou (`saved`), não o rascunho: um evento ligado com
 * número inválido apareceria como "ativo" se a tela confiasse no formulário.
 */
export function EventPanel({
  config,
  version,
  item,
}: {
  config: WorldEventConfig;
  version: number;
  item: EventItem | null;
}) {
  const router = useRouter();
  const toast = useToast();
  const [saved, setSaved] = useState<{ config: WorldEventConfig; version: number }>({
    config,
    version,
  });
  const [draft, setDraft] = useState<WorldEventConfig>(config);
  const [picked, setPicked] = useState<EventItem | null>(item);
  const [pending, setPending] = useState(false);
  const [doubleGate, setDoubleGate] = useState(false);
  const [offGate, setOffGate] = useState(false);

  // O progresso (`current_index`) muda sem a versão mudar, quando o tmserver
  // persiste um drop. Comparar a assinatura por valor mantém o rascunho do
  // moderador enquanto o servidor não mudou nada de fato.
  const signature = `${version}:${JSON.stringify(config)}`;
  const lastSignature = useRef(signature);
  useEffect(() => {
    if (lastSignature.current === signature) return;
    lastSignature.current = signature;
    setSaved({ config, version });
    setDraft(config);
    setPicked(item);
  }, [signature, config, version, item]);

  const errors = validateWorldEvent(draft);
  const invalid = Object.keys(errors).length > 0;
  const dirty = configChanged(draft, saved.config);
  const status = worldEventStatus(saved.config);
  const draftStatus = worldEventStatus(draft);

  const span = Math.max(1, saved.config.end_index - saved.config.start_index);
  const done = Math.min(span, Math.max(0, saved.config.current_index - saved.config.start_index));
  const pct = Math.round((done / span) * 100);

  function set<K extends keyof WorldEventConfig>(key: K, value: WorldEventConfig[K]) {
    setDraft((current) => ({ ...current, [key]: value }));
  }

  async function run(next: WorldEventConfig) {
    setPending(true);
    const result = await saveWorldEventConfig(next);
    setPending(false);
    if (!result.ok) {
      toast(result.error, "error");
      return;
    }
    setSaved({ config: result.config, version: result.version });
    setDraft(result.config);
    toast(result.message, "ok");
    router.refresh();
  }

  function save() {
    if (invalid) {
      toast(Object.values(errors)[0] ?? "configuração inválida", "error");
      return;
    }
    if (draft.double_exp_enabled && !saved.config.double_exp_enabled) {
      setDoubleGate(true);
      return;
    }
    void run(draft);
  }

  function shutdown() {
    void run({
      ...saved.config,
      enabled: false,
      notice_enabled: false,
      double_exp_enabled: false,
      newbie_event_enabled: false,
    });
    setOffGate(false);
  }

  return (
    <>
      <section className="ui-card" style={{ marginBottom: "var(--space-3)" }}>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "baseline",
            gap: "var(--space-2)",
            flexWrap: "wrap",
          }}
        >
          <strong>Estado no servidor</strong>
          <span className="ui-field__hint" style={{ margin: 0 }}>
            versão {fmtCoin(saved.version)} · relido pelo tmserver a cada 15 s
          </span>
        </div>

        <div
          style={{ display: "flex", gap: "var(--space-2)", flexWrap: "wrap", marginTop: "var(--space-2)" }}
        >
          <Badge tone={status.tone}>{status.label}</Badge>
          <Badge tone={saved.config.double_exp_enabled ? "warn" : "neutral"}>
            EXP dobrada {saved.config.double_exp_enabled ? "ligada" : "desligada"}
          </Badge>
          <Badge tone={saved.config.newbie_event_enabled ? "warn" : "neutral"}>
            evento newbie {saved.config.newbie_event_enabled ? "ligado" : "desligado"}
          </Badge>
          <Badge tone={saved.config.notice_enabled ? "accent" : "neutral"}>
            anúncio {saved.config.notice_enabled ? "ligado" : "desligado"}
          </Badge>
        </div>

        <p className="ui-field__hint" style={{ marginTop: "var(--space-2)" }}>
          {status.hint}
        </p>

        <div className="ui-stats" style={{ marginTop: "var(--space-3)", marginBottom: 0 }}>
          <div className="ui-stat">
            <div className="ui-stat__label">item do evento</div>
            <div className="ui-stat__value" style={{ fontSize: 18 }}>
              {item
                ? item.name
                : saved.config.item_index > 0
                  ? `item #${fmtCoin(saved.config.item_index)}`
                  : "nenhum"}
            </div>
            <div className="ui-stat__hint">
              {saved.config.indexed ? "sai com serial (efeitos 62/63)" : "sai sem serial"}
            </div>
          </div>
          <div className="ui-stat">
            <div className="ui-stat__label">chance por kill</div>
            <div className="ui-stat__value" style={{ fontSize: 18 }}>
              {rateLabel(saved.config.rate)}
            </div>
            <div className="ui-stat__hint">divisor do roll, não porcentagem final</div>
          </div>
          <div className="ui-stat">
            <div className="ui-stat__label">progresso</div>
            <div className="ui-stat__value" style={{ fontSize: 18 }}>
              {fmtCoin(saved.config.current_index)} / {fmtCoin(saved.config.end_index)}
            </div>
            <div className="ui-stat__hint">
              restam {fmtCoin(remainingDrops(saved.config))} drops · início{" "}
              {fmtCoin(saved.config.start_index)}
            </div>
          </div>
        </div>

        {saved.config.enabled && (
          <div style={{ marginTop: "var(--space-3)" }}>
            <div
              style={{ height: 8, borderRadius: 999, background: "var(--surface-2)", overflow: "hidden" }}
              role="progressbar"
              aria-valuenow={pct}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-label="Progresso do evento de drop"
            >
              <div style={{ width: `${pct}%`, height: "100%", background: "var(--accent)" }} />
            </div>
            <p className="ui-field__hint" style={{ marginTop: "var(--space-1)" }}>
              {pct}% da faixa percorrida
            </p>
          </div>
        )}
      </section>

      <section className="ui-card" style={{ marginBottom: "var(--space-3)" }}>
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            gap: "var(--space-2)",
            flexWrap: "wrap",
          }}
        >
          <strong>Evento de drop</strong>
          <Toggle
            label="ligado"
            hint="desligado, o item de evento não é entregue"
            checked={draft.enabled}
            onChange={(value) => set("enabled", value)}
          />
        </div>

        <div
          style={{
            display: "grid",
            gap: "var(--space-3)",
            gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))",
            marginTop: "var(--space-3)",
          }}
        >
          <Field
            label="item entregue"
            error={errors.item_index}
            hint={`índice até ${fmtCoin(ITEM_INDEX_MAX)}`}
          >
            <Combobox
              label="item do evento"
              load={loadItemOptions}
              placeholder="Buscar item por nome ou índice…"
              value={picked ? { value: String(picked.index), label: picked.name } : null}
              onSelect={(option) => {
                const index = Number(option.value);
                set("item_index", index);
                setPicked({ index, name: option.label });
              }}
            />
          </Field>

          <Field
            label="divisor de chance"
            error={errors.rate}
            hint={`${rateLabel(draft.rate)} · maior = mais raro`}
          >
            <input
              className="ui-input"
              type="number"
              // Com o evento desligado os zeros são config válida (o web-api
              // aceita): um `min=1` fixo marcaria o campo como inválido à toa.
              min={draft.enabled ? 1 : 0}
              max={RATE_MAX}
              value={draft.rate}
              onChange={(event) => set("rate", Number(event.target.value))}
            />
          </Field>
        </div>

        <div
          style={{
            display: "grid",
            gap: "var(--space-3)",
            gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
            marginTop: "var(--space-3)",
          }}
        >
          <Field
            label="índice inicial"
            error={errors.start_index}
            hint="primeiro serial/progresso válido"
          >
            <input
              className="ui-input"
              type="number"
              min={draft.enabled ? 1 : 0}
              max={ITEM_INDEX_MAX}
              value={draft.start_index}
              onChange={(event) => set("start_index", Number(event.target.value))}
            />
          </Field>
          <Field
            label="progresso atual"
            error={errors.current_index}
            hint="o tmserver avança sozinho a cada drop entregue"
          >
            <input
              className="ui-input"
              type="number"
              min={0}
              max={ITEM_INDEX_MAX}
              value={draft.current_index}
              onChange={(event) => set("current_index", Number(event.target.value))}
            />
          </Field>
          <Field
            label="índice final"
            error={errors.end_index}
            hint="o evento para quando o progresso chega aqui"
          >
            <input
              className="ui-input"
              type="number"
              min={draft.enabled ? 1 : 0}
              max={ITEM_INDEX_MAX}
              value={draft.end_index}
              onChange={(event) => set("end_index", Number(event.target.value))}
            />
          </Field>
        </div>

        <div style={{ display: "grid", gap: "var(--space-2)", marginTop: "var(--space-3)" }}>
          <Toggle
            label="item com serial"
            hint="usa e grava os efeitos 62/63 e o valor aleatório 59 — permite rastrear o drop"
            checked={draft.indexed}
            onChange={(value) => set("indexed", value)}
          />
          <Toggle
            label="anunciar no jogo"
            hint="manda um aviso no chat de todos os jogadores online depois da entrega"
            checked={draft.notice_enabled}
            onChange={(value) => set("notice_enabled", value)}
          />
        </div>
      </section>

      <section className="ui-card" style={{ marginBottom: "var(--space-3)" }}>
        <strong>Multiplicadores</strong>
        <div
          style={{
            display: "grid",
            gap: "var(--space-3)",
            gridTemplateColumns: "repeat(auto-fit, minmax(260px, 1fr))",
            marginTop: "var(--space-2)",
          }}
        >
          <Toggle
            label="EXP dobrada"
            hint="afeta toda a economia de EXP em PvE. É o efeito do double mode, sem restart — ao contrário da flag -double-exp."
            checked={draft.double_exp_enabled}
            onChange={(value) => set("double_exp_enabled", value)}
          />
          <Toggle
            label="evento newbie"
            hint="+15% de EXP e bônus para personagens abaixo do nível 100 — é o NewbieEventServer."
            checked={draft.newbie_event_enabled}
            onChange={(value) => set("newbie_event_enabled", value)}
          />
        </div>
      </section>

      {dirty && draftStatus.status !== status.status && (
        <p className="ui-banner" role="note" style={{ marginBottom: "var(--space-2)" }}>
          O rascunho ficaria <strong>{draftStatus.label}</strong> — o servidor ainda está{" "}
          <strong>{status.label}</strong> até salvar.
        </p>
      )}

      <div
        style={{
          display: "flex",
          gap: "var(--space-2)",
          flexWrap: "wrap",
          alignItems: "center",
          marginBottom: "var(--space-4)",
        }}
      >
        <button
          className="ui-btn ui-btn--primary"
          onClick={save}
          disabled={pending || invalid || !dirty}
        >
          {pending ? "salvando…" : "salvar evento"}
        </button>
        <button className="ui-btn ui-btn--ghost" onClick={() => router.refresh()} disabled={pending}>
          recarregar do servidor
        </button>
        <button className="ui-btn ui-btn--danger" onClick={() => setOffGate(true)} disabled={pending}>
          desligar tudo
        </button>
        <span className="ui-field__hint" style={{ margin: 0 }}>
          {invalid
            ? "corrija os campos destacados"
            : dirty
              ? "alterações não salvas"
              : "tudo salvo"}
        </span>
      </div>

      <Dialog
        open={doubleGate}
        onClose={() => setDoubleGate(false)}
        title="Ligar EXP dobrada para o servidor inteiro?"
        description="Vale para todo PvE, para todos os jogadores, até alguém desligar."
        footer={
          <>
            <button className="ui-btn ui-btn--ghost" onClick={() => setDoubleGate(false)}>
              cancelar
            </button>
            <button
              className="ui-btn ui-btn--primary"
              disabled={pending}
              onClick={() => {
                setDoubleGate(false);
                void run(draft);
              }}
            >
              confirmar e salvar
            </button>
          </>
        }
      >
        <p style={{ margin: 0 }}>
          A EXP de cada kill dobra assim que o tmserver reler o config (até 15 s). A curva por nível
          não muda — só o valor entregue. Se a intenção é mexer na curva, isso é outra tela.
        </p>
      </Dialog>

      <Dialog
        open={offGate}
        onClose={() => setOffGate(false)}
        title="Desligar todos os eventos?"
        description="Drop global, anúncio, EXP dobrada e evento newbie — numa chamada."
        footer={
          <>
            <button className="ui-btn ui-btn--ghost" onClick={() => setOffGate(false)}>
              cancelar
            </button>
            <button className="ui-btn ui-btn--danger" disabled={pending} onClick={shutdown}>
              desligar tudo
            </button>
          </>
        }
      >
        <p style={{ margin: 0 }}>
          O item, a faixa e o divisor continuam guardados para religar depois. Jogadores online
          perdem o bônus de EXP no próximo ciclo do tmserver.
        </p>
      </Dialog>
    </>
  );
}
