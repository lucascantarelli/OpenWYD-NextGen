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
  },
});
