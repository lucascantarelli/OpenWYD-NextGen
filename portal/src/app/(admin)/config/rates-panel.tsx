"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { Field } from "@/components/Field";
import { Tabs } from "@/components/Tabs";
import { useToast } from "@/components/Toast";
import { RATE_FILES, groupByFamily, overriddenCount, type Rate } from "@/lib/rates";
import { resetRate, upsertRate } from "@/server/actions/rates";

/**
 * Editor das taxas (issue 36).
 *
 * Cada linha mostra o valor do arquivo e o efetivo; "restaurar" só aparece quando
 * existe overlay. O motivo é obrigatório porque toda edição vai para
 * `content_rate_audit` com before/after.
 */
export function RatesPanel({
  rates,
  warning,
}: {
  rates: readonly Rate[];
  warning: string;
}) {
  const router = useRouter();
  const toast = useToast();
  const [tab, setTab] = useState<string>(RATE_FILES[0].file);
  const [reason, setReason] = useState("");
  const [draft, setDraft] = useState<Record<string, number>>({});
  const [pending, setPending] = useState(false);

  const visible = useMemo(() => rates.filter((rate) => rate.file === tab), [rates, tab]);
  const grouped = useMemo(() => groupByFamily(visible), [visible]);
  const activeFile = RATE_FILES.find((entry) => entry.file === tab) ?? RATE_FILES[0];

  function draftKey(rate: Rate): string {
    return `${rate.file}/${rate.family}/${rate.name}`;
  }

  async function run(action: () => Promise<{ ok: true; message?: string } | { ok: false; error: string }>) {
    if (reason.trim() === "") {
      toast("Informe o motivo: toda edição de taxa fica na auditoria", "error");
      return;
    }
    setPending(true);
    const result = await action();
    setPending(false);
    if (!result.ok) {
      toast(result.error, "error");
      return;
    }
    toast(result.message ?? "taxa salva", "ok");
    setReason("");
    router.refresh();
  }

  if (rates.length === 0) {
    return (
      <div className="ui-card ui-empty">
        <strong>Nenhuma taxa disponível</strong>
        <p>{warning || "o web-api não conseguiu ler os arquivos de tmserver/content/Common/Settings"}</p>
      </div>
    );
  }

  return (
    <>
      <div className="ui-card" style={{ marginBottom: 12 }}>
        <Field label="motivo" hint="obrigatório — vai para content_rate_audit junto com before/after">
          <input
            className="ui-input"
            value={reason}
            maxLength={200}
            onChange={(event) => setReason(event.target.value)}
            placeholder="ex.: ajuste de curva do refino na temporada 3"
          />
        </Field>
      </div>

      <Tabs
        tabs={RATE_FILES.map((entry) => ({ id: entry.file, label: entry.title }))}
        value={tab}
        onChange={setTab}
        label="Tabelas de taxa"
      />

      <p className="ui-field__hint" style={{ margin: "8px 0 12px" }}>
        {activeFile.hint} · {visible.length} células · {overriddenCount(visible)} com edição
      </p>

      <div style={{ display: "grid", gap: 14 }}>
        {[...grouped.entries()].map(([family, rows]) => (
          <section key={family} className="ui-card">
            <strong>{family}</strong>
            <table className="ui-table" style={{ marginTop: 8, width: "100%" }}>
              <thead>
                <tr>
                  <th>célula</th>
                  <th>arquivo</th>
                  <th>atual</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {rows.map((rate) => {
                  const key = draftKey(rate);
                  const next = draft[key] ?? rate.value;
                  return (
                    <tr key={key}>
                      <td>
                        {rate.label}{" "}
                        {rate.has_override && <Badge tone="accent">editada</Badge>}
                      </td>
                      <td className="ui-mono">{rate.original}</td>
                      <td style={{ width: 130 }}>
                        <input
                          className="ui-input"
                          type="number"
                          min={rate.min_value}
                          max={rate.max_value}
                          value={next}
                          onChange={(event) =>
                            setDraft({ ...draft, [key]: Number(event.target.value) })
                          }
                        />
                      </td>
                      <td>
                        <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                          <button
                            className="ui-btn ui-btn--primary"
                            disabled={pending || next === rate.value}
                            onClick={() =>
                              run(() =>
                                upsertRate({
                                  file: rate.file,
                                  family: rate.family,
                                  name: rate.name,
                                  value: next,
                                  reason,
                                }),
                              )
                            }
                          >
                            salvar
                          </button>
                          {rate.has_override && (
                            <button
                              className="ui-btn ui-btn--ghost"
                              disabled={pending}
                              onClick={() =>
                                run(() =>
                                  resetRate({
                                    file: rate.file,
                                    family: rate.family,
                                    name: rate.name,
                                    reason,
                                  }),
                                )
                              }
                            >
                              restaurar
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </section>
        ))}
      </div>
    </>
  );
}
