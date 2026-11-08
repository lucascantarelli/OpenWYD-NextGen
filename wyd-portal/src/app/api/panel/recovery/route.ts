import { z } from "zod";

import { toResponse } from "@/server/errors";
import { requestRecovery } from "@/server/panel";

export const dynamic = "force-dynamic";

const NO_STORE = { "cache-control": "no-store" } as const;

/** Mesmo limite do login: só recusa o que não pode ser credencial. */
const Body = z.object({
  login: z.string().trim().min(1).max(16),
});

/**
 * Solicita recuperação de conta. A resposta é **sempre** a mesma mensagem —
 * exista ou não o login — para não vazar quais contas existem.
 */
export async function POST(req: Request) {
  let parsed: z.infer<typeof Body>;
  try {
    parsed = Body.parse(await req.json());
  } catch {
    return Response.json({ error: "dados inválidos" }, { status: 422, headers: NO_STORE });
  }

  try {
    await requestRecovery(parsed.login);
  } catch (err) {
    return toResponse(err);
  }

  return Response.json(
    {
      ok: true,
      message:
        "Se a conta existir, a solicitação ficou registrada para atendimento da equipe.",
    },
    { headers: NO_STORE },
  );
}
