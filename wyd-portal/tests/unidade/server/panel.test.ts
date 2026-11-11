import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/server/session", () => ({
  getSession: vi.fn(),
}));

import { AdminError } from "@/server/errors";
import { getPanelAccount, getPanelCharacters, getPanelRankings } from "@/server/panel";
import { getSession } from "@/server/session";

const getSessionMock = vi.mocked(getSession);

interface Captured {
  url: string;
  init: RequestInit;
}

function json(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "content-type": "application/json" },
  });
}

/** Gruda um `fetch` que grava as chamadas (sem depender de `.mock.calls`). */
function stubFetch(body: unknown, calls: Captured[] = []) {
  vi.stubGlobal("fetch", async (url: string, init: RequestInit) => {
    calls.push({ url: String(url), init });
    return json(body);
  });
  return calls;
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.clearAllMocks();
});

describe("painel do jogador", () => {
  it("injeta o account_id da sessão, nunca o do browser", async () => {
    getSessionMock.mockResolvedValue({ accountId: 42, role: "player" });
    const calls = stubFetch({ result: 1, characters: [] });

    await getPanelCharacters();

    expect(calls[0].url).toContain("/admin/v1/panel/characters");
    expect(JSON.parse(String(calls[0].init.body))).toEqual({ account_id: 42 });
    expect(calls[0].init.cache).toBe("no-store");
    expect(calls[0].init.signal).toBeInstanceOf(AbortSignal);
  });

  it("sem sessão não chama o backend", async () => {
    getSessionMock.mockResolvedValue(null);
    const calls = stubFetch({ result: 1 });

    await expect(getPanelAccount()).rejects.toBeInstanceOf(AdminError);
    expect(calls).toHaveLength(0);
  });

  it("ranking limita a faixa aceita pelo backend", async () => {
    getSessionMock.mockResolvedValue({ accountId: 7, role: "player" });
    const calls = stubFetch({ result: 1, ranking: [] });

    await getPanelRankings(10_000);

    expect(JSON.parse(String(calls[0].init.body))).toEqual({ limit: 10 });
  });

  it("recusa de negócio vira AdminError com o status do resultado", async () => {
    getSessionMock.mockResolvedValue({ accountId: 7, role: "player" });
    stubFetch({ result: 4, message: "não encontrado" });

    await expect(getPanelAccount()).rejects.toMatchObject({ status: 404 });
  });
});
