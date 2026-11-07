"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { Combobox, type ComboOption } from "@/components/Combobox";
import { Field } from "@/components/Field";
import { Form } from "@/components/Form";
import { useToast } from "@/components/Toast";
import {
  MERCHANT_LABELS,
  MERCHANT_VALUES,
  ROUTE_LABELS,
  hasShop,
  type MapZone,
  type MerchantTemplate,
  type NpcDefinition,
  type NpcFormValues,
  type NpcShopItem,
} from "@/lib/npc";
import { upsertNpc } from "@/server/actions/npcs";

import { ShopEditor } from "./shop-editor";

/**
 * Formulário de definição + aba de loja.
 *
 * O `slug` é a chave estável do `UpsertNpc`: no modo edição ele fica travado,
 * porque mudar o slug cria **outro** NPC em vez de renomear.
 */
export function NpcForm({
  mode,
  npc,
  templates,
  zones,
  shopNames = {},
}: {
  mode: "create" | "edit";
  npc?: NpcDefinition;
  templates: readonly MerchantTemplate[];
  zones: readonly MapZone[];
  shopNames?: Record<string, string>;
}) {
  const router = useRouter();
  const toast = useToast();
  const [values, setValues] = useState<NpcFormValues>({
    slug: npc?.slug ?? "",
    display_name: npc?.display_name ?? "",
    template_name: npc?.template_name ?? "",
    map_id: npc?.map_id ?? 0,
    pos_x: npc?.pos_x ?? 0,
    pos_y: npc?.pos_y ?? 0,
    route_type: npc?.route_type ?? 0,
    merchant: npc?.merchant ?? 0,
    enabled: npc?.enabled ?? true,
  });
  const [tab, setTab] = useState<"def" | "shop">("def");
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [savedNpcId, setSavedNpcId] = useState<number | null>(npc?.id ?? null);

  const templateByValue = useMemo(
    () => new Map(templates.map((template) => [template.template_name, template])),
    [templates],
  );

  /** Filtro local: o scan do web-api já trouxe os templates no carregamento. */
  const loadTemplates = useCallback(
    async (query: string): Promise<ComboOption[]> => {
      const needle = query.trim().toLowerCase();
      return templates
        .filter(
          (template) =>
            !needle ||
            template.template_name.toLowerCase().includes(needle) ||
            (template.display_name ?? "").toLowerCase().includes(needle),
        )
        .slice(0, 50)
        .map((template) => ({
          value: template.template_name,
          label: template.template_name,
          hint: `${template.display_name || "sem nome"} · merchant ${template.merchant}`,
        }));
    },
    [templates],
  );

  function update<K extends keyof NpcFormValues>(key: K, value: NpcFormValues[K]) {
    setValues((current) => ({ ...current, [key]: value }));
  }

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError(null);
    const result = await upsertNpc(values);
    setPending(false);

    if (!result.ok) {
      setError(result.error);
      toast(result.error, "error");
      return;
    }

    toast("Definição salva — aplica no próximo restart do tmserver", "ok");
    setSavedNpcId(result.npcId ?? savedNpcId);
    if (mode === "create" && result.npcId) router.push(`/npcs/${result.npcId}`);
    else router.refresh();
  }

  const canShop = hasShop(values.merchant);

  return (
    <>
      <div className="ui-filters" role="tablist" aria-label="Seções do NPC">
        <button
          type="button"
          className="ui-chip"
          role="tab"
          aria-pressed={tab === "def"}
          onClick={() => setTab("def")}
        >
          Definição
        </button>
        <button
          type="button"
          className="ui-chip"
          role="tab"
          aria-pressed={tab === "shop"}
          onClick={() => setTab("shop")}
        >
          Loja {canShop ? "" : "(mercador desligado)"}
        </button>
        {npc && (
          <span className="ui-field__hint" style={{ marginLeft: "auto" }}>
            #{npc.id} · origem <Badge tone={npc.origin === "content" ? "warn" : "ok"}>{npc.origin}</Badge>
            {npc.generator_index >= 0 ? ` · NPCGener #${npc.generator_index}` : ""}
          </span>
        )}
      </div>

      {tab === "def" && (
        <>
          <Form onSubmit={submit} pending={pending} submitLabel={mode === "create" ? "Criar NPC" : "Salvar definição"}>
            <Field label="Slug" hint="chave estável: a-z, 0-9 e hífen (3–40) — não muda depois de criado" htmlFor="slug">
              <input
                id="slug"
                className="ui-input"
                value={values.slug}
                onChange={(event) => update("slug", event.target.value)}
                disabled={mode === "edit"}
                autoComplete="off"
              />
            </Field>

            <Field label="Nome exibido" hint="1–40 caracteres, é o que aparece no jogo" htmlFor="display_name">
              <input
                id="display_name"
                className="ui-input"
                value={values.display_name}
                onChange={(event) => update("display_name", event.target.value)}
                maxLength={40}
              />
            </Field>

            <Field
              label="Template"
              hint={
                templates.length > 0
                  ? `${templates.length} templates merchant em tmserver/content/TMsrv/run/npc/`
                  : "picker indisponível (web-api sem -content): digite o template manualmente"
              }
              htmlFor="template_name"
            >
              {templates.length > 0 ? (
                <Combobox
                  label="Template do NPC"
                  load={loadTemplates}
                  placeholder={values.template_name || "buscar template…"}
                  onSelect={(option) => {
                    update("template_name", option.value);
                    const template = templateByValue.get(option.value);
                    if (template && values.merchant === 0 && template.merchant > 0) {
                      update("merchant", template.merchant);
                    }
                  }}
                />
              ) : (
                <input
                  id="template_name"
                  className="ui-input"
                  value={values.template_name}
                  onChange={(event) => update("template_name", event.target.value)}
                  autoComplete="off"
                />
              )}
              {values.template_name && (
                <p className="ui-field__hint" style={{ marginTop: 4 }}>
                  template escolhido: <span className="ui-mono">{values.template_name}</span>
                </p>
              )}
            </Field>

            <div className="ui-detail__grid">
              <Field label="Mapa" hint="hoje é rótulo: o mundo roda num grid único" htmlFor="map_id">
                <select
                  id="map_id"
                  className="ui-select"
                  value={values.map_id}
                  onChange={(event) => update("map_id", Number(event.target.value))}
                >
                  {zones.length === 0 && <option value={values.map_id}>#{values.map_id}</option>}
                  {zones.map((zone) => (
                    <option key={zone.id} value={zone.id}>
                      {zone.name}
                    </option>
                  ))}
                </select>
              </Field>

              <Field label="Posição X" hint="coordenada do grid do mundo" htmlFor="pos_x">
                <input
                  id="pos_x"
                  className="ui-input"
                  type="number"
                  value={values.pos_x}
                  onChange={(event) => update("pos_x", Number(event.target.value))}
                />
              </Field>

              <Field label="Posição Y" hint="coordenada do grid do mundo" htmlFor="pos_y">
                <input
                  id="pos_y"
                  className="ui-input"
                  type="number"
                  value={values.pos_y}
                  onChange={(event) => update("pos_y", Number(event.target.value))}
                />
              </Field>

              <Field label="Rota" hint="0 = parado" htmlFor="route_type">
                <select
                  id="route_type"
                  className="ui-select"
                  value={values.route_type}
                  onChange={(event) => update("route_type", Number(event.target.value))}
                >
                  {[0, 1, 2, 3].map((value) => (
                    <option key={value} value={value}>
                      {value} — {ROUTE_LABELS[value]}
                    </option>
                  ))}
                </select>
              </Field>

              <Field label="Tipo de mercador" hint="define se abre loja, armazém ou nada" htmlFor="merchant">
                <select
                  id="merchant"
                  className="ui-select"
                  value={values.merchant}
                  onChange={(event) => update("merchant", Number(event.target.value))}
                >
                  {MERCHANT_VALUES.map((value) => (
                    <option key={value} value={value}>
                      {value} — {MERCHANT_LABELS[value]}
                    </option>
                  ))}
                </select>
              </Field>

              <Field label="Visibilidade" hint="oculto = não spawna, definição preservada" htmlFor="enabled">
                <label className="ui-field__hint" style={{ display: "block", paddingTop: 8 }}>
                  <input
                    id="enabled"
                    type="checkbox"
                    checked={values.enabled}
                    onChange={(event) => update("enabled", event.target.checked)}
                  />{" "}
                  aparece no mundo
                </label>
              </Field>
            </div>
          </Form>

          <p className="ui-banner" style={{ marginTop: 12 }} role="note">
            A gravação vai para o Postgres na hora; o mundo reflete no <strong>próximo restart</strong>{" "}
            do tmserver (ou no poll de ~15 s). Restart:{" "}
            <code>docker compose restart tmserver</code>.{" "}
            <Link href="/logs">ver logs</Link>
          </p>
        </>
      )}

      {tab === "shop" &&
        (savedNpcId && canShop ? (
          <ShopEditor
            npcId={savedNpcId}
            initial={npc?.shop ?? []}
            itemNames={shopNames}
            onSaved={() => router.refresh()}
          />
        ) : (
          <div className="ui-card ui-empty">
            <strong>Loja indisponível</strong>
            <p>
              {canShop
                ? "Salve a definição primeiro: a loja é gravada por npc_id."
                : "Este NPC não é mercador. Escolha “loja normal” (1) ou “loja tipo 3” (19) na aba Definição e salve para liberar o estoque."}
            </p>
          </div>
        ))}
    </>
  );
}
