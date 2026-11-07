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
    include: ["src/**/*.test.ts"],
  },
});
