import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Session } from "@/server/session";

/**
 * Fronteira de autorização de verdade do portal: o `route.ts` de uma API admin.
 *
 * O proxy só olha a assinatura do cookie (roda no edge, sem sessão de
 * banco): quem sabe se o papel pode ler dado de conta é o handler. Escolhemos
 * `/api/search/accounts` porque é o caso mais sensível que **já existe** — busca
 * de contas para combobox e faturamento. Não inventamos endpoint para o teste.
 *
 * `getSession` é o único ponto trocado: o resto (401 → 403 → 200 do handler) roda
 * de verdade. Com `q` de 1 caractere o handler responde antes de tocar o gRPC,
 * então o teste não precisa de web-api nem de Postgres.
 */

const state: { session: Session | null } = { session: null };

vi.mock("@/server/session", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/server/session")>();
  return { ...actual, getSession: async () => state.session };
});

const { GET } = await import("./route");

function search(query = "a"): Request {
  return new Request(`http://127.0.0.1:3000/api/search/accounts?q=${query}`);
}

describe("GET /api/search/accounts", () => {
  beforeEach(() => {
    state.session = null;
  });

  it("sem sessão devolve 401", async () => {
    const res = await GET(search());

    expect(res.status).toBe(401);
    expect(await res.json()).toEqual({ error: "sessão expirada" });
  });

  it("jogador (papel sem permissão) devolve 403 e não consulta conta", async () => {
    state.session = { accountId: 42, role: "player" };

    const res = await GET(search());

    expect(res.status).toBe(403);
    expect(await res.json()).toEqual({ error: "sem permissão" });
  });

  it("staff passa da fronteira e responde sem cache", async () => {
    state.session = { accountId: 1, role: "admin" };

    const res = await GET(search());

    expect(res.status).toBe(200);
    expect(res.headers.get("cache-control")).toBe("private, no-store");
    expect(await res.json()).toEqual({ options: [] });
  });
});
