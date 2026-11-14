import { fileURLToPath } from "node:url";

import { defineConfig } from "vitest/config";

/**
 * Vitest não lê `paths` do tsconfig: sem este alias, qualquer módulo que importe
 * `@/…` (o middleware, por exemplo) não carrega no teste. O ambiente é `node`
 * porque o middleware só usa Web API (`Request`/`Response`/`cookies`), sem DOM.
 */
export default defineConfig({
  resolve: {
    alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
  },
  test: {
    environment: "node",
    // Testes por tipo em `tests/`: `unidade`, `contrato`, `integracao` e
    // `componente`. `.tsx` entra por causa do teste de componente da tela de
    // conta (jsdom por arquivo, via `@vitest-environment jsdom` no topo). O E2E
    // é Playwright (`tests/e2e`, `npm run test:e2e`), fora do Vitest.
    include: ["tests/**/*.test.ts", "tests/**/*.test.tsx"],
    // 29 arquivos no Windows: com quatro forks o startup do worker jsdom passa
    // do timeout e o arquivo de componente (`conta/pin-form`) nem roda — a suíte
    // saía com 144/28 e código 1. Dois workers rodam os 154/29 e mantêm o
    // isolamento por arquivo (medido em 2026-11-13).
    pool: "forks",
    maxWorkers: 2,
  },
});
