"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";

import { Badge } from "@/components/Badge";
import { DataTable, type Column } from "@/components/DataTable";
import { Dialog } from "@/components/Dialog";
import { Field } from "@/components/Field";
import { useToast } from "@/components/Toast";
import {
  guildLevelLabel,
  relationLabel,
  zoneName,
  type Guild,
  type GuildRelation,
} from "@/lib/worldstate";
import { removeGuildMember, setGuildMaster, setGuildRelation } from "@/server/actions/worldstate";

/**
 * Lista de guildas e edição auditada (issue 45, escritas).
 *
 * Toda ação destrutiva exige motivo e o botão diz o que vai acontecer: mestre
 * atual não pode ser removido (transfira antes), e toda escrita avisa que vale no
 * próximo restart — o estado é carregado no boot do tmserver.
 */
export function GuildsTable({
  guilds,
  relations,
}: {
  guilds: readonly Guild[];
  relations: readonly GuildRelation[];
}) {
  const router = useRouter();
  const toast = useToast();
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);
  const [target, setTarget] = useState(0);
  const [kind, setKind] = useState(1);

  const selected = guilds.find((guild) => guild.id === selectedId) ?? null;

  const relationsByGuild = useMemo(() => {
    const map = new Map<number, GuildRelation[]>();
    for (const relation of relations) {
      const list = map.get(relation.guild_id) ?? [];
      list.push(relation);
      map.set(relation.guild_id, list);
    }
    return map;
  }, [relations]);

  function open(guild: Guild) {
    setSelectedId(guild.id);
    setReason("");
    setTarget(0);
    setKind(1);
  }

  async function run(action: () => Promise<{ ok: true; message?: string } | { ok: false; error: string }>) {
    setPending(true);
    const result = await action();
    setPending(false);
    if (!result.ok) {
      toast(result.error, "error");
      return;
    }
    toast(result.message ?? "alteração salva", "ok");
    setReason("");
    router.refresh();
  }

  const columns = useMemo<Column<Guild>[]>(
    () => [
      {
        key: "name",
        header: "Guilda",
        sortValue: (guild) => guild.name,
        render: (guild) => (
          <div style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap" }}>
            <strong>{guild.name}</strong>
            <span className="ui-mono ui-field__hint">#{guild.id}</span>
            {guild.tower_owner && <Badge tone="accent">torre</Badge>}
          </div>
        ),
      },
      {
        key: "master",
        header: "Mestre",
        sortValue: (guild) => guild.master_name,
        render: (guild) => guild.master_name || <span className="ui-field__hint">sem líder</span>,
      },
      {
        key: "members",
        header: "Membros",
        sortValue: (guild) => guild.members.length,
        render: (guild) => guild.members.length,
      },
      {
        key: "cities",
        header: "Cidades",
        sortValue: (guild) => guild.city_zones.length,
        render: (guild) =>
          guild.city_zones.length > 0 ? (
            <div style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
              {guild.city_zones.map((zone) => (
                <Badge key={zone} tone="ok">
                  {zoneName(zone)}
                </Badge>
              ))}
            </div>
          ) : (
            <span className="ui-field__hint">—</span>
          ),
      },
      {
        key: "clan",
        header: "Clã",
        sortValue: (guild) => guild.clan,
        render: (guild) => guild.clan,
      },
      {
        key: "fame",
        header: "Fama",
        sortValue: (guild) => guild.fame,
        render: (guild) => guild.fame.toLocaleString("pt-BR"),
      },
      {
        key: "actions",
        header: "",
        render: (guild) => (
          <button className="ui-btn ui-btn--ghost" onClick={() => open(guild)}>
            gerenciar
          </button>
        ),
      },
    ],
    [],
  );

  const detailRelations = selected ? (relationsByGuild.get(selected.id) ?? []) : [];
  const relationTargets = guilds.filter((guild) => guild.id !== selectedId);

  return (
    <>
      <DataTable<Guild>
        rows={[...guilds]}
        columns={columns}
        getRowKey={(guild) => String(guild.id)}
        searchText={(guild) => `${guild.name} ${guild.master_name} ${guild.id}`}
        emptyMessage="Nenhuma guilda registrada neste servidor."
      />

      <Dialog
        open={selected !== null}
        title={selected ? `${selected.name} · ${selected.members.length} membros` : ""}
        onClose={() => setSelectedId(null)}
        footer={
          <button className="ui-btn ui-btn--ghost" onClick={() => setSelectedId(null)}>
            fechar
          </button>
        }
      >
        {selected && (
          <div style={{ display: "grid", gap: 14 }}>
            <Field
              label="motivo"
              hint="obrigatório: vai para guild_admin_audit junto com o antes/depois"
            >
              <input
                className="ui-input"
                value={reason}
                maxLength={200}
                onChange={(event) => setReason(event.target.value)}
                placeholder="ex.: troca pedida pelo líder anterior"
              />
            </Field>

            <section>
              <strong>Membros</strong>
              {selected.members.length === 0 ? (
                <p className="ui-field__hint" style={{ margin: "4px 0 0" }}>
                  Nenhuma linha em <code>guild_member</code>: a guilda existe sem membros.
                </p>
              ) : (
                <table className="ui-table" style={{ marginTop: 6, width: "100%" }}>
                  <thead>
                    <tr>
                      <th>personagem</th>
                      <th>nível na guilda</th>
                      <th />
                    </tr>
                  </thead>
                  <tbody>
                    {selected.members.map((member) => {
                      const isMaster = member.guild_level === 9;
                      return (
                        <tr key={member.character_id}>
                          <td>
                            {member.name}{" "}
                            <span className="ui-mono ui-field__hint">#{member.character_id}</span>
                          </td>
                          <td>
                            {guildLevelLabel(member.guild_level)}{" "}
                            <span className="ui-mono ui-field__hint">({member.guild_level})</span>
                          </td>
                          <td>
                            <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
                              <button
                                className="ui-btn ui-btn--ghost"
                                disabled={pending || isMaster}
                                onClick={() =>
                                  run(() =>
                                    setGuildMaster({
                                      guildId: selected.id,
                                      characterId: member.character_id,
                                      reason,
                                    }),
                                  )
                                }
                              >
                                tornar mestre
                              </button>
                              <button
                                className="ui-btn ui-btn--ghost"
                                disabled={pending || isMaster}
                                onClick={() =>
                                  run(() =>
                                    removeGuildMember({
                                      guildId: selected.id,
                                      characterId: member.character_id,
                                      reason,
                                    }),
                                  )
                                }
                              >
                                remover
                              </button>
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              )}
              <p className="ui-field__hint" style={{ margin: "6px 0 0" }}>
                O líder não pode ser removido: transfira o mestrado primeiro.
              </p>
            </section>

            <section>
              <strong>Relações</strong>
              {detailRelations.length === 0 ? (
                <p className="ui-field__hint" style={{ margin: "4px 0 6px" }}>
                  Sem relações declaradas.
                </p>
              ) : (
                <ul style={{ margin: "6px 0", paddingLeft: 18 }}>
                  {detailRelations.map((relation) => (
                    <li key={`${relation.guild_id}-${relation.kind}`}>
                      <Badge tone={relation.kind === 2 ? "warn" : "ok"}>
                        {relationLabel(relation.kind)}
                      </Badge>{" "}
                      {relation.target_name || `guilda #${relation.target_guild_id}`}{" "}
                      <button
                        className="ui-btn ui-btn--ghost"
                        disabled={pending}
                        onClick={() =>
                          run(() =>
                            setGuildRelation({
                              guildId: selected.id,
                              targetGuildId: relation.target_guild_id,
                              kind: 0,
                              reason,
                            }),
                          )
                        }
                      >
                        remover
                      </button>
                    </li>
                  ))}
                </ul>
              )}

              {relationTargets.length === 0 ? (
                <p className="ui-field__hint" style={{ margin: 0 }}>
                  Precisa de outra guilda registrada para declarar relação.
                </p>
              ) : (
                <div style={{ display: "grid", gridTemplateColumns: "1fr 140px auto", gap: 8 }}>
                  <select
                    className="ui-input"
                    value={target}
                    onChange={(event) => setTarget(Number(event.target.value))}
                  >
                    <option value={0}>guilda de destino…</option>
                    {relationTargets.map((guild) => (
                      <option key={guild.id} value={guild.id}>
                        {guild.name}
                      </option>
                    ))}
                  </select>
                  <select
                    className="ui-input"
                    value={kind}
                    onChange={(event) => setKind(Number(event.target.value))}
                  >
                    <option value={1}>aliada</option>
                    <option value={2}>inimiga</option>
                  </select>
                  <button
                    className="ui-btn ui-btn--primary"
                    disabled={pending || target === 0}
                    onClick={() =>
                      run(() =>
                        setGuildRelation({
                          guildId: selected.id,
                          targetGuildId: target,
                          kind,
                          reason,
                        }),
                      )
                    }
                  >
                    aplicar
                  </button>
                </div>
              )}
            </section>
          </div>
        )}
      </Dialog>
    </>
  );
}
