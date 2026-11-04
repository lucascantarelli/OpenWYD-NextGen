"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { Badge } from "@/components/Badge";
import { Field } from "@/components/Field";
import { useToast } from "@/components/Toast";
import {
  MAX_CITY_TAX,
  formatUnix,
  zoneName,
  type CastleQuest,
  type Guild,
  type Kefra,
  type KingdomZone,
} from "@/lib/worldstate";
import { setCityTax, setKefraDefeated, setTowerOwner, setZoneOwner } from "@/server/actions/worldstate";

/**
 * Dono da cidade, taxa, dono da torre e ciclo do Kefra (issue 45).
 *
 * Dono e taxa são chamadas separadas de propósito: o banco guarda as duas na
 * mesma linha, mas um resultado de cerco não pode mexer no imposto da loja sem
 * alguém decidir isso. Toda escrita pede motivo (vai para a auditoria) e avisa que
 * vale no próximo restart do tmserver.
 */
export function CastlePanel({
  zones,
  guilds,
  kefra,
  castle,
  towerOwnerGuild,
}: {
  zones: readonly KingdomZone[];
  guilds: readonly Guild[];
  kefra: Kefra;
  castle: CastleQuest;
  towerOwnerGuild: number;
}) {
  const router = useRouter();
  const toast = useToast();
  const [tax, setTax] = useState<Record<number, number>>(() =>
    Object.fromEntries(zones.map((zone) => [zone.zone, zone.city_tax])),
  );
  const [owner, setOwner] = useState<Record<number, number>>(() =>
    Object.fromEntries(zones.map((zone) => [zone.zone, zone.charge_guild])),
  );
  const [towerOwner, setTowerOwnerDraft] = useState(towerOwnerGuild);
  const [reason, setReason] = useState("");
  const [pending, setPending] = useState(false);

  async function run(action: () => Promise<{ ok: true; message?: string } | { ok: false; error: string }>) {
    if (reason.trim() === "") {
      toast("Informe o motivo antes de salvar: toda escrita vai para a auditoria", "error");
      return;
    }
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

  // O flip do Kefra não pede motivo: é um booleano com guard de revisão.
  async function setKefraFlip(defeated: boolean) {
    setPending(true);
    const result = await setKefraDefeated({ defeated });
    setPending(false);
    if (!result.ok) {
      toast(result.error, "error");
      return;
    }
    toast(result.message ?? "estado do Kefra atualizado", "ok");
    router.refresh();
  }

  return (
    <div style={{ display: "grid", gap: 14 }}>
      <section className="ui-card">
        <Field label="motivo" hint="obrigatório para dono de cidade, torre e taxa — fica em guild_admin_audit">
          <input
            className="ui-input"
            value={reason}
            maxLength={200}
            onChange={(event) => setReason(event.target.value)}
            placeholder="ex.: resultado do cerco de 03/11"
          />
        </Field>
      </section>

      <section className="ui-card">
        <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
          <strong>Cidades: dono e taxa</strong>
          <span className="ui-field__hint" style={{ margin: 0 }}>
            taxa 0–{MAX_CITY_TAX}% · imposto da loja pessoal
          </span>
        </div>
        {zones.length === 0 ? (
          <p className="ui-field__hint" style={{ margin: "6px 0 0" }}>
            Nenhuma linha em <code>guild_zone</code> — o seed da migration 0012 não rodou.
          </p>
        ) : (
          <table className="ui-table" style={{ marginTop: 8, width: "100%" }}>
            <thead>
              <tr>
                <th>cidade</th>
                <th>no poder</th>
                <th>desafiante</th>
                <th>taxa</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {zones.map((zone) => (
                <tr key={zone.zone}>
                  <td>
                    {zoneName(zone.zone)}{" "}
                    <span className="ui-mono ui-field__hint">zona {zone.zone}</span>
                    {zone.victory > 0 && (
                      <>
                        {" "}
                        <Badge tone="accent">vitória</Badge>
                      </>
                    )}
                  </td>
                  <td>
                    <select
                      className="ui-input"
                      value={owner[zone.zone] ?? zone.charge_guild}
                      onChange={(event) =>
                        setOwner({ ...owner, [zone.zone]: Number(event.target.value) })
                      }
                    >
                      <option value={0}>sem dono</option>
                      {guilds.map((guild) => (
                        <option key={guild.id} value={guild.id}>
                          {guild.name}
                        </option>
                      ))}
                    </select>
                    <button
                      className="ui-btn ui-btn--ghost"
                      style={{ marginTop: 4 }}
                      disabled={pending || (owner[zone.zone] ?? zone.charge_guild) === zone.charge_guild}
                      onClick={() =>
                        run(() =>
                          setZoneOwner({
                            zone: zone.zone,
                            guildId: owner[zone.zone] ?? zone.charge_guild,
                            reason,
                          }),
                        )
                      }
                    >
                      definir dono
                    </button>
                  </td>
                  <td>
                    {zone.challenge_guild_name || <span className="ui-field__hint">—</span>}
                    {zone.challenge_money > 0 && (
                      <span className="ui-field__hint">
                        {" "}
                        · {zone.challenge_money.toLocaleString("pt-BR")} gold
                      </span>
                    )}
                  </td>
                  <td style={{ width: 150 }}>
                    <input
                      className="ui-input"
                      type="number"
                      min={0}
                      max={MAX_CITY_TAX}
                      value={tax[zone.zone] ?? zone.city_tax}
                      onChange={(event) =>
                        setTax({ ...tax, [zone.zone]: Number(event.target.value) })
                      }
                    />
                    <button
                      className="ui-btn ui-btn--ghost"
                      style={{ marginTop: 4 }}
                      disabled={pending || (tax[zone.zone] ?? zone.city_tax) === zone.city_tax}
                      onClick={() =>
                        run(() =>
                          setCityTax({ zone: zone.zone, cityTax: tax[zone.zone] ?? zone.city_tax }),
                        )
                      }
                    >
                      salvar taxa
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section className="ui-card">
        <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
          <strong>Torre (GTorre)</strong>
          <Badge tone={towerOwnerGuild > 0 ? "accent" : "neutral"}>
            {guilds.find((guild) => guild.id === towerOwnerGuild)?.name ?? "sem dono"}
          </Badge>
        </div>
        <div style={{ display: "grid", gridTemplateColumns: "1fr auto", gap: 8, marginTop: 8 }}>
          <select
            className="ui-input"
            value={towerOwner}
            onChange={(event) => setTowerOwnerDraft(Number(event.target.value))}
          >
            <option value={0}>sem dono</option>
            {guilds.map((guild) => (
              <option key={guild.id} value={guild.id}>
                {guild.name}
              </option>
            ))}
          </select>
          <button
            className="ui-btn ui-btn--primary"
            disabled={pending || towerOwner === towerOwnerGuild}
            onClick={() => run(() => setTowerOwner({ guildId: towerOwner, reason }))}
          >
            definir dono da torre
          </button>
        </div>
        <p className="ui-field__hint" style={{ margin: "6px 0 0" }}>
          A torre é uma linha só (<code>guild_tower_state</code>); o dono da cidade é o{" "}
          <code>charge_guild</code> da zona.
        </p>
      </section>

      <section className="ui-card">
        <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
          <strong>Quest do castelo</strong>
          <Badge tone={castle.clear ? "ok" : "neutral"}>
            {castle.clear ? "concluída" : "em andamento"}
          </Badge>
        </div>
        <p className="ui-field__hint" style={{ margin: "6px 0 0" }}>
          nível {castle.level} · tempo restante {castle.time_left} · líder{" "}
          {castle.leader_name || "—"}
        </p>
      </section>

      <section className="ui-card">
        <div style={{ display: "flex", justifyContent: "space-between", gap: 12, flexWrap: "wrap" }}>
          <strong>Kefra</strong>
          <div style={{ display: "flex", gap: 6, flexWrap: "wrap" }}>
            <Badge tone={kefra.defeated ? "warn" : "ok"}>
              {kefra.defeated ? "morto" : "vivo"}
            </Badge>
            {!kefra.initialized && <Badge tone="neutral">sem ciclo gravado</Badge>}
            {kefra.initialized && <Badge tone="neutral">revisão {kefra.revision}</Badge>}
          </div>
        </div>

        {kefra.initialized ? (
          <>
            <p className="ui-field__hint" style={{ margin: "6px 0 0" }}>
              último spawn {formatUnix(kefra.last_spawn_unix)} · próximo{" "}
              {formatUnix(kefra.next_spawn_unix)}
            </p>
            <p className="ui-field__hint" style={{ margin: "6px 0 10px" }}>
              Marcar como morto abre o portal do Deserto; vivo o fecha. O flag sobe a{" "}
              <code>revision</code>, então o save do processo atual não desfaz a edição.
            </p>
            <div style={{ display: "flex", gap: 8, flexWrap: "wrap" }}>
              <button
                className="ui-btn ui-btn--ghost"
                disabled={pending || kefra.defeated}
                onClick={() => setKefraFlip(true)}
              >
                marcar como morto
              </button>
              <button
                className="ui-btn ui-btn--ghost"
                disabled={pending || !kefra.defeated}
                onClick={() => setKefraFlip(false)}
              >
                marcar como vivo
              </button>
            </div>
          </>
        ) : (
          <p className="ui-field__hint" style={{ margin: "6px 0 0" }}>
            <code>kefra_state</code> nasce com defeat/next_spawn vindos do tmServer; sem a linha o
            portal não inventa horário de spawn.
          </p>
        )}
      </section>
    </div>
  );
}
