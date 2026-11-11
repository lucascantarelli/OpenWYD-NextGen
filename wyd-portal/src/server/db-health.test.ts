import { describe, expect, it } from "vitest";

import { getDbHealth } from "./db-health";

describe("saúde do banco via gateway", () => {
  it("degrada sem derrubar quando o gateway está fora", async () => {
    const health = await getDbHealth();
    expect(typeof health.reachable).toBe("boolean");
    expect(typeof health.error).toBe("string");
    if (!health.reachable) {
      expect(health.latencyMs === null || typeof health.latencyMs === "number").toBe(true);
      expect(health.database).toBeNull();
      expect(health.schema).toBeNull();
    } else {
      expect(typeof health.latencyMs).toBe("number");
    }
  });
});
