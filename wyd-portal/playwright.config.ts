import { defineConfig } from "@playwright/test";

/**
 * E2E da FASE-07-4 (headless, contra a stack real).
 *
 * `scripts/e2e.mjs` provisiona o banco dedicado (`wyd_e2e`, nunca o `wyd` de
 * dev), o gateway/canal efêmeros e um container do **portal do compose**
 * (`wyd/portal:dev`) na 3110 apontando para esse gateway. O compose continua
 * dono da 3000 para o teste manual — o E2E não disputa porta nem o lock de
 * `next dev` com ele.
 */
const PORTAL_PORT = Number(process.env.E2E_PORTAL_PORT ?? 3110);

export default defineConfig({
  testDir: "./e2e",
  // Um spec por vez: todos compartilham o mesmo banco efêmero e o mesmo canal.
  fullyParallel: false,
  workers: 1,
  timeout: 60_000,
  expect: { timeout: 15_000 },
  reporter: [["list"]],
  use: {
    baseURL: `http://127.0.0.1:${PORTAL_PORT}`,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
});
