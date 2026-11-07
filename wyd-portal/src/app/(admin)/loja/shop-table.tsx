"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { Combobox, type ComboOption } from "@/components/Combobox";
import { DataTable, type Column } from "@/components/DataTable";
import { Dialog } from "@/components/Dialog";
import { Field } from "@/components/Field";
import { ItemIcon } from "@/components/ItemIcon";
import { useToast } from "@/components/Toast";
import {
  CREDIT_MAX,
  CREDIT_MIN,
  DESCRIPTION_MAX,
  EFFECT_MAX,
  EXPIRES_MAX,
  PRICE_MAX,
  PRICE_MIN,
  REASON_MAX,
  REASON_MIN,
  TITLE_MAX,
  TITLE_MIN,
  effectsSummary,
  emptyShopItem,
  expiryLabel,
  fmtDonate,
  priceLabel,
  validateCredit,
  validateShopItem,
  type CreditInput,
  type DonateShopItem,
} from "@/lib/donate";
import { loadAccountOptions } from "@/lib/accountsearch";
import { loadItemOptions } from "@/lib/itemsearch";
import {
  creditDonateBalance,
  deleteShopItem,
  saveShopItem,
  setShopItemEnabled,
} from "@/server/actions/donate";
import type { CatalogItem, ItemEffectType } from "@/server/grpc";

export interface ShopRow extends DonateShopItem {
  /** `null` quando o item não está no catálogo (oferta antiga ou catálogo fora). */
  item: CatalogItem | null;
}

const EFFECT_PAIRS = [
  ["eff1", "effv1"],
  ["eff2", "effv2"],
  ["eff3", "effv3"],
] as const;

function emptyCredit(): CreditInput {
  return { account_id: 0, amount: 0, reason: "", confirmed: false };
}

/**
 * Tabela + editor da loja de donate (issue 29).
 *
 * Preço sempre com o rótulo "donate" porque a confusão com coin é o risco
 * clássico desta tela. Apagar exige digitar o título: a oferta vale dinheiro
 * real e não tem undo.
 */
export function ShopTable({
  rows,
  effectTypes,
  iconPackVersion,
}: {
  rows: readonly ShopRow[];
  effectTypes: readonly ItemEffectType[];
  iconPackVersion: string;
}) {
  const router = useRouter();
  const toast = useToast();
  const [draft, setDraft] = useState<DonateShopItem | null>(null);
  const [picked, setPicked] = useState<{ index: number; name: string } | null>(null);
  const [pending, setPending] = useState(false);
  const [toDelete, setToDelete] = useState<ShopRow | null>(null);
  const [deleteTitle, setDeleteTitle] = useState("");
  const [credit, setCredit] = useState<CreditInput | null>(null);
  const [creditAccount, setCreditAccount] = useState<ComboOption | null>(null);

  const errors = draft ? validateShopItem(draft) : {};
  const invalid = Object.keys(errors).length > 0;
  const creditErrors = credit ? validateCredit(credit) : {};
  const creditInvalid = Object.keys(creditErrors).length > 0;

  function openNew() {
    setDraft(emptyShopItem());
    setPicked(null);
  }

  function openEdit(row: ShopRow) {
    setDraft({ ...row });
    setPicked({
      index: row.item_index,
      name: row.item ? row.item.display_name || row.item.name : `item #${row.item_index}`,
    });
  }

  function set<K extends keyof DonateShopItem>(key: K, value: DonateShopItem[K]) {
    setDraft((current) => (current ? { ...current, [key]: value } : current));
  }

  function setCreditField<K extends keyof CreditInput>(key: K, value: CreditInput[K]) {
    setCredit((current) => (current ? { ...current, [key]: value } : current));
  }

  async function run(
    action: () => Promise<{ ok: true; message: string } | { ok: false; error: string }>,
  ): Promise<boolean> {
    setPending(true);
    try {
      const result = await action();
      if (!result.ok) {
        toast(result.error, "error");
        return false;
      }
      toast(result.message, "ok");
      router.refresh();
      return true;
    } catch {
      // Uma action que rejeita (500 do Next, rede) não pode deixar o botão preso.
      toast("falha ao falar com o servidor: tente de novo", "error");
      return false;
    } finally {
      setPending(false);
    }
  }

  async function submit() {
    if (!draft) return;
    if (invalid) {
      toast(Object.values(errors)[0] ?? "oferta inválida", "error");
      return;
    }
    if (await run(() => saveShopItem(draft))) {
      setDraft(null);
      setPicked(null);
    }
  }

  async function submitCredit() {
    if (!credit) return;
    if (creditInvalid) {
      toast(Object.values(creditErrors)[0] ?? "crédito inválido", "error");
      return;
    }
    const result = await creditDonateBalance(credit);
    if (!result.ok) {
      toast(result.error, "error");
      return;
    }
    toast(
      result.new_balance === undefined
        ? result.message
        : `${result.message} (saldo: ${fmtDonate(result.new_balance)} donate)`,
      "ok",
    );
    setCredit(null);
    router.refresh();
  }

  const columns = useMemo<Column<ShopRow>[]>(
    () => [
      {
        key: "title",
        header: "oferta",
        sortValue: (row) => row.title,
        render: (row) => (
          <div>
            <span>{row.title}</span>
            {row.description && (
              <span className="ui-field__hint" style={{ display: "block" }}>
                {row.description}
              </span>
            )}
            <span className="ui-field__hint" style={{ display: "block" }}>
              entrega no próximo login
            </span>
          </div>
        ),
      },
      {
        key: "item",
        header: "item vendido",
        sortValue: (row) => row.item?.display_name || row.item?.name || row.item_index,
        render: (row) => (
          <div style={{ display: "flex", alignItems: "center", gap: "var(--space-2)" }}>
            {row.item && <ItemIcon item={row.item} iconPackVersion={iconPackVersion} />}
            <span>{row.item ? row.item.display_name || row.item.name : `item #${row.item_index}`}</span>
          </div>
        ),
      },
      {
        key: "price",
        header: "preço",
        align: "right",
        sortValue: (row) => row.price,
        render: (row) => <span className="ui-mono">{priceLabel(row.price)}</span>,
      },
      {
        key: "expires_days",
        header: "validade",
        align: "right",
        sortValue: (row) => row.expires_days,
        render: (row) => expiryLabel(row.expires_days),
      },
      {
        key: "effects",
        header: "adicionais",
        render: (row) => effectsSummary(row),
      },
      {
        key: "enabled",
        header: "estado",
        render: (row) => (
          <Badge tone={row.enabled ? "ok" : "neutral"}>
            {row.enabled ? "na vitrine" : "fora da vitrine"}
          </Badge>
        ),
      },
      {
        key: "actions",
        header: "ações",
        render: (row) => (
          <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
            <button className="ui-btn ui-btn--ghost" onClick={() => openEdit(row)}>
              editar
            </button>
            <button
              className="ui-btn ui-btn--ghost"
              disabled={pending}
              onClick={() => void run(() => setShopItemEnabled({ id: row.id, enabled: !row.enabled }))}
            >
              {row.enabled ? "desabilitar" : "habilitar"}
            </button>
            <button
              className="ui-btn ui-btn--danger"
              disabled={pending}
              onClick={() => {
                setToDelete(row);
                setDeleteTitle("");
              }}
            >
              apagar
            </button>
          </div>
        ),
      },
    ],
    // `run`/`openEdit` fecham sobre estado local e são recriados a cada render.
    [iconPackVersion, pending],
  );

  return (
    <>
      <div className="ui-filters">
        <div>
          <p className="ui-page-sub" style={{ margin: 0 }}>
            Preço em <strong>donate</strong>, nunca em coin. A compra debita a carteira da conta e o
            item chega pelo <code>delivery_queue</code> no <strong>próximo login</strong> do
            personagem.
          </p>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", gap: "var(--space-2)" }}>
          <button
            className="ui-btn"
            onClick={() => {
              setCredit(emptyCredit());
              setCreditAccount(null);
            }}
          >
            creditar saldo
          </button>
          <button className="ui-btn ui-btn--primary" onClick={openNew}>
            nova oferta
          </button>
        </div>
      </div>

      <DataTable
        rows={rows}
        columns={columns}
        getRowKey={(row) => row.id}
        searchText={(row) =>
          `${row.title} ${row.description} ${row.item?.display_name ?? ""} ${row.item?.name ?? ""} ${
            row.item_index
          } ${row.price}`
        }
        searchPlaceholder="Buscar por título, item ou preço…"
        emptyMessage="Nenhuma oferta na loja de donate ainda."
        caption={`${rows.length} oferta${rows.length === 1 ? "" : "s"} · preços em donate · pedidos e compradores são a issue 31`}
      />

      <Dialog
        open={draft !== null}
        onClose={() => setDraft(null)}
        title={draft && draft.id > 0 ? "Editar oferta" : "Nova oferta de donate"}
        description="Entrega no próximo login do personagem, via delivery_queue. Preço em donate."
        footer={
          <>
            <button className="ui-btn ui-btn--ghost" onClick={() => setDraft(null)}>
              cancelar
            </button>
            <button
              className="ui-btn ui-btn--primary"
              disabled={pending || invalid}
              onClick={() => void submit()}
            >
              {pending ? "salvando…" : draft && draft.id > 0 ? "salvar oferta" : "criar oferta"}
            </button>
          </>
        }
      >
        {draft && (
          <div className="ui-form">
            <Field
              label="título"
              error={errors.title}
              hint={`${draft.title.trim().length}/${TITLE_MAX} · mínimo ${TITLE_MIN}`}
            >
              <input
                className="ui-input"
                value={draft.title}
                maxLength={TITLE_MAX}
                onChange={(event) => set("title", event.target.value)}
                placeholder="ex.: Kit de refino"
              />
            </Field>

            <Field
              label="descrição"
              error={errors.description}
              hint={`${draft.description.length}/${DESCRIPTION_MAX} · aparece na vitrine do jogador`}
            >
              <textarea
                className="ui-input"
                rows={2}
                value={draft.description}
                maxLength={DESCRIPTION_MAX}
                onChange={(event) => set("description", event.target.value)}
              />
            </Field>

            <Field label="item vendido" error={errors.item_index} hint="vai para o cargo no próximo login">
              <Combobox
                label="item da loja"
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
              label="preço em donate"
              error={errors.price}
              hint={`${PRICE_MIN} a ${fmtDonate(PRICE_MAX)} donate · não é coin`}
            >
              <input
                className="ui-input"
                type="number"
                min={PRICE_MIN}
                max={PRICE_MAX}
                value={draft.price}
                onChange={(event) => set("price", Number(event.target.value))}
              />
            </Field>

            <Field
              label="validade do item"
              error={errors.expires_days}
              hint="0 = permanente; o cliente mostra os dias no nome do item"
            >
              <input
                className="ui-input"
                type="number"
                min={0}
                max={EXPIRES_MAX}
                value={draft.expires_days}
                onChange={(event) => set("expires_days", Number(event.target.value))}
              />
            </Field>

            <div>
              <span className="ui-field__label">adicionais do item</span>
              <div style={{ display: "grid", gap: "var(--space-2)", marginTop: "var(--space-1)" }}>
                {EFFECT_PAIRS.map(([effectKey, valueKey], index) => (
                  <div
                    key={effectKey}
                    style={{
                      display: "grid",
                      gridTemplateColumns: "1fr 120px",
                      gap: "var(--space-2)",
                      alignItems: "center",
                    }}
                  >
                    <select
                      className="ui-select"
                      aria-label={`efeito ${index + 1}`}
                      value={String(draft[effectKey])}
                      onChange={(event) => set(effectKey, Number(event.target.value))}
                    >
                      <option value="0">nenhum</option>
                      {effectTypes.map((type) => (
                        <option key={type.code} value={type.code}>
                          {type.label || type.name}
                        </option>
                      ))}
                      {draft[effectKey] !== 0 &&
                        !effectTypes.some((type) => type.code === draft[effectKey]) && (
                          <option value={draft[effectKey]}>efeito {draft[effectKey]}</option>
                        )}
                    </select>
                    <input
                      className="ui-input"
                      type="number"
                      min={0}
                      max={EFFECT_MAX}
                      aria-label={`valor do efeito ${index + 1}`}
                      value={draft[valueKey]}
                      onChange={(event) => set(valueKey, Number(event.target.value))}
                    />
                    {(errors[effectKey] || errors[valueKey]) && (
                      <span className="ui-field__error" style={{ gridColumn: "1 / -1" }}>
                        {errors[effectKey] ?? errors[valueKey]}
                      </span>
                    )}
                  </div>
                ))}
              </div>
            </div>

            <label
              style={{
                display: "flex",
                alignItems: "flex-start",
                gap: "var(--space-2)",
                minHeight: 32,
                cursor: "pointer",
              }}
            >
              <input
                type="checkbox"
                checked={draft.enabled}
                onChange={(event) => set("enabled", event.target.checked)}
                style={{ accentColor: "var(--accent)", width: 16, height: 16, marginTop: 8 }}
              />
              <span>
                <span style={{ fontSize: 13 }}>na vitrine</span>
                <span className="ui-field__hint" style={{ display: "block" }}>
                  fora da vitrine a compra é recusada, mesmo com a oferta existindo
                </span>
              </span>
            </label>
          </div>
        )}
      </Dialog>

      <Dialog
        open={toDelete !== null}
        onClose={() => setToDelete(null)}
        title="Apagar oferta da loja?"
        description="Não tem undo. Se a intenção é só tirar da vitrine, use desabilitar."
        footer={
          <>
            <button className="ui-btn ui-btn--ghost" onClick={() => setToDelete(null)}>
              cancelar
            </button>
            <button
              className="ui-btn ui-btn--danger"
              disabled={pending || deleteTitle.trim().length === 0}
              onClick={async () => {
                const target = toDelete;
                if (!target) return;
                const done = await run(() => deleteShopItem({ id: target.id, title: deleteTitle }));
                if (done) setToDelete(null);
              }}
            >
              {pending ? "apagando…" : "apagar oferta"}
            </button>
          </>
        }
      >
        <div className="ui-form">
          <p style={{ margin: 0 }}>
            Digite o título <strong>{toDelete?.title}</strong> para confirmar.
          </p>
          <Field label="título da oferta" hint="o servidor confere o título antes de apagar">
            <input
              className="ui-input"
              value={deleteTitle}
              onChange={(event) => setDeleteTitle(event.target.value)}
              placeholder={toDelete?.title ?? ""}
            />
          </Field>
        </div>
      </Dialog>

      <Dialog
        open={credit !== null}
        onClose={() => setCredit(null)}
        title="Crédito manual de donate"
        description="Cria saldo do nada: exige motivo auditado e confirmação. Não tem rollback automático."
        footer={
          <>
            <button className="ui-btn ui-btn--ghost" onClick={() => setCredit(null)}>
              cancelar
            </button>
            <button
              className="ui-btn ui-btn--primary"
              disabled={pending || creditInvalid}
              onClick={() => void submitCredit()}
            >
              {pending ? "creditando…" : "creditar saldo"}
            </button>
          </>
        }
      >
        {credit && (
          <div className="ui-form">
            <Field
              label="conta"
              error={creditErrors.account_id}
              hint="busque pelo login: cada opção mostra id, saldo e papel"
            >
              <Combobox
                label="conta que recebe o crédito"
                load={loadAccountOptions}
                placeholder="Login da conta (mín. 2 letras)…"
                value={creditAccount}
                onSelect={(option) => {
                  setCreditAccount(option);
                  setCreditField("account_id", Number(option.value));
                }}
              />
            </Field>

            <Field
              label="valor em donate"
              error={creditErrors.amount}
              hint={`${CREDIT_MIN} a ${fmtDonate(CREDIT_MAX)} donate`}
            >
              <input
                className="ui-input"
                type="number"
                min={CREDIT_MIN}
                max={CREDIT_MAX}
                value={credit.amount || ""}
                onChange={(event) => setCreditField("amount", Number(event.target.value))}
              />
            </Field>

            <Field
              label="motivo"
              error={creditErrors.reason}
              hint={`${credit.reason.trim().length}/${REASON_MAX} · mínimo ${REASON_MIN} · vai para a auditoria`}
            >
              <textarea
                className="ui-input"
                rows={2}
                value={credit.reason}
                maxLength={REASON_MAX}
                placeholder="ex.: compensação de compra duplicada — pedido #123"
                onChange={(event) => setCreditField("reason", event.target.value)}
              />
            </Field>

            <label
              style={{
                display: "flex",
                alignItems: "flex-start",
                gap: "var(--space-2)",
                cursor: "pointer",
              }}
            >
              <input
                type="checkbox"
                checked={credit.confirmed}
                onChange={(event) => setCreditField("confirmed", event.target.checked)}
                style={{ accentColor: "var(--accent)", width: 16, height: 16, marginTop: 8 }}
              />
              <span>
                <span style={{ fontSize: 13 }}>confirmo o crédito manual</span>
                <span className="ui-field__hint" style={{ display: "block" }}>
                  a operação fica registrada em nome da minha conta, com o motivo acima
                </span>
              </span>
            </label>
            {creditErrors.confirmed && (
              <span className="ui-field__error" role="alert">
                {creditErrors.confirmed}
              </span>
            )}
          </div>
        )}
      </Dialog>
    </>
  );
}
