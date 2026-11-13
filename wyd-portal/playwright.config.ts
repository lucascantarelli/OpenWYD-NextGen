import { defineConfig } from "@playwright/test";

/**
 * E2E do portal: o `scripts/e2e.mjs` provisiona banco, gateway, canal e portal
 * efêmeros e publica o portal em `127.0.0.1:3110` antes de chamar o Playwright.
 * Este config só fixa a pasta dos specs, a baseURL relativa usada pelos testes e
 * a serialidade (todos compartilham o mesmo gateway/banco efêmero).
 */
export default defineConfig({
  testDir: "./tests/e2e",
  fullyParallel: false,
  workers: 1,
  timeout: 60_000,
  expect: { timeout: 10_000 },
  use: {
    baseURL: "http://127.0.0.1:3110",
    trace: "retain-on-failure",
  },
});
