import { describe, expect, it } from "vitest";

import { computeEconomyAlerts, type EconomyView } from "@/server/economy";

/**
 * Alertas derivam só de agregado já resumido (sem linha por jogador):
 * volume alto de ouro na fila e cidade zerada com outras ativas.
 */
describe("alertas de economia", () => {
  it("silêncio total não é alerta", () => {
    expect(computeEconomyAlerts({ taxByCity: [], grants: [] })).toEqual([]);
  });

  it("ouro somado acima de 1M vira aviso de volume", () => {
    const summary: EconomyView = {
      taxByCity: [{ city: "armia", total: 500 }],
      grants: [{ kind: "gold", count: 3, gold: 1_500_000 }],
    };
    const alerts = computeEconomyAlerts(summary);
    expect(alerts).toHaveLength(1);
    expect(alerts[0]?.tone).toBe("warn");
  });

  it("abaixo do limiar não alerta", () => {
    const summary: EconomyView = {
      taxByCity: [],
      grants: [{ kind: "gold", count: 1, gold: 250 }],
    };
    expect(computeEconomyAlerts(summary)).toEqual([]);
  });

  it("cidade zerada com outras ativas vira nota de dreno", () => {
    const summary: EconomyView = {
      taxByCity: [
        { city: "armia", total: 1200 },
        { city: "azran", total: 0 },
      ],
      grants: [],
    };
    const alerts = computeEconomyAlerts(summary);
    expect(alerts).toHaveLength(1);
    expect(alerts[0]?.detail).toContain("azran");
  });

  it("cidade única zerada não é anomalia", () => {
    const summary: EconomyView = {
      taxByCity: [{ city: "armia", total: 0 }],
      grants: [],
    };
    expect(computeEconomyAlerts(summary)).toEqual([]);
  });
});
