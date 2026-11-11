import { describe, expect, it } from "vitest";

import { embedUrlFor, openUrlFor } from "@/server/grafana";

describe("embed do Grafana", () => {
  it("pede kiosk com menu e controles escondidos", () => {
    const url = new URL(embedUrlFor("wyd-saude"));

    expect(url.pathname).toBe("/d/wyd-saude");
    // `kiosk=tv` não esconde a navegação no Grafana 12 (medido); `kiosk=true` sim.
    expect(url.searchParams.get("kiosk")).toBe("true");
    expect(url.searchParams.get("_dash.hideTimePicker")).toBe("true");
    expect(url.searchParams.get("_dash.hideVariables")).toBe("true");
    expect(url.searchParams.get("_dash.hideLinks")).toBe("true");
    expect(url.searchParams.get("theme")).toBe("dark");
    expect(url.searchParams.get("from")).toBe("now-1h");
  });

  it("a URL de abrir no Grafana mantém o chrome completo", () => {
    expect(openUrlFor("wyd-saude")).not.toContain("kiosk");
  });
});
