import { beforeEach, describe, expect, it, vi } from "vitest";

import { adminCall } from "./admin";
import { getSession } from "./session";

/**
 * Prova o invariante do `adminCall`: o `moderator_id` enviado ao web-api vem da
 * sessão assinada, **nunca** do corpo da request.
 *
 * A ameaça é concreta: o browser monta o JSON que vira o corpo da Server Action,
 * então sem essa injeção um moderador poderia escrever `moderator_id` de outra
 * conta e plantar auditoria em nome de terceiro. O web-api revalida o papel,
 * mas quem atribui a autoria é este ponto.
 */
vi.mock("./session", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./session")>();
  return { ...actual, getSession: vi.fn() };
});

interface Body {
  npc_id: number;
  moderator_id?: number;
  moderatorId?: number;
}

interface Reply {
  result: number;
  message?: string;
}

/** Chamada admin falsa (HTTP/JSON, ADR-0008): registra o payload enviado. */
function captureRpc(reply: Reply = { result: 1 }) {
  const seen: Body[] = [];
  const fn = async (req: Body): Promise<Reply> => {
    seen.push(req);
    return reply;
  };
  return { seen, fn };
}

const STAFF = { accountId: 42, role: "moderator" as const };
const PLAYER = { accountId: 7, role: "player" as const };

describe("adminCall: moderator_id vem da sessão", () => {
  beforeEach(() => {
    vi.mocked(getSession).mockReset();
  });

  it("sobrescreve o moderator_id plantado no corpo com o da sessão", async () => {
    vi.mocked(getSession).mockResolvedValue(STAFF);
    const { seen, fn } = captureRpc();

    // Corpo hostil: o browser tenta se passar pela conta 999.
    const body = { npc_id: 7, moderator_id: 999, moderatorId: 999 };
    await adminCall(fn, body as never, { action: "npc.update", target: "npc:7" });

    expect(seen).toHaveLength(1);
    expect(seen[0].moderator_id).toBe(STAFF.accountId);
    expect(seen[0].moderator_id).not.toBe(999);
    // O nome em camelCase também não pode sobreviver: o proto usa keepCase.
    expect(seen[0].moderatorId).toBeUndefined();
  });

  it("injeta o id da sessão mesmo quando o corpo não traz nenhum", async () => {
    vi.mocked(getSession).mockResolvedValue(STAFF);
    const { seen, fn } = captureRpc();

    await adminCall(fn, { npc_id: 8 } as never, { action: "npc.update" });

    expect(seen[0]).toEqual({ npc_id: 8, moderator_id: STAFF.accountId });
  });

  it("não chama a RPC sem sessão (401) e não vaza o id do corpo", async () => {
    vi.mocked(getSession).mockResolvedValue(null);
    const { seen, fn } = captureRpc();

    await expect(
      adminCall(fn, { npc_id: 9, moderator_id: 999 } as never, { action: "npc.delete" }),
    ).rejects.toMatchObject({ status: 401 });

    expect(seen).toHaveLength(0);
  });

  it("não chama a RPC para conta sem papel de staff (403)", async () => {
    vi.mocked(getSession).mockResolvedValue(PLAYER);
    const { seen, fn } = captureRpc();

    await expect(
      adminCall(fn, { npc_id: 9 } as never, { action: "npc.delete" }),
    ).rejects.toMatchObject({ status: 403 });

    expect(seen).toHaveLength(0);
  });
});
