import { defineConfig } from "@playwright/test";

/**
 * E2E da FASE-07-4 (headless, contra a stack real).
 *
 * O banco dedicado (`wyd_e2e`) e os containers do gateway/canal são provisionados
 * por `scripts/e2e.mjs` — **nunca** o banco `wyd` de desenvolvimento. O portal
 * sobe aqui apontando para o gateway efêmero em `127.0.0.1:7601`.
 */
// 3110: o dev server do projeto já usa 3100; o E2E não disputa porta com ele.
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
  webServer: {
    command: `npx next dev -p ${PORTAL_PORT}`,
    url: `http://127.0.0.1:${PORTAL_PORT}/login`,
    reuseExistingServer: false,
    timeout: 120_000,
    env: {
      WEB_API_ADDR: process.env.E2E_GATEWAY_URL ?? "http://127.0.0.1:7601",
      WEB_API_TOKEN: process.env.E2E_GATEWAY_TOKEN ?? "e2e-token-do-gateway",
      SESSION_SECRET: "e2e-session-secret-com-32-caracteres-ok!!",
    },
  },
});
