import { z } from "zod";

import { accountWeb, promisify, type VerifyCredentialsResponse } from "@/server/grpc";
import { normalizeRole, setSession } from "@/server/session";

export const dynamic = "force-dynamic";

const NO_STORE = { "cache-control": "no-store" } as const;

/**
 * Limites largos de propósito: só rejeitamos o que não pode ser credencial
 * (vazio ou absurdo). Validar "4 a 16" recusaria uma conta real de 3 letras.
 */
const Body = z.object({
  name: z.string().trim().min(1).max(16),
  pass: z.string().min(1).max(64),
});

export async function POST(req: Request) {
  let parsed: z.infer<typeof Body>;
  try {
    parsed = Body.parse(await req.json());
  } catch {
    return Response.json({ error: "dados inválidos" }, { status: 422, headers: NO_STORE });
  }

  let res: VerifyCredentialsResponse;
  try {
    res = await promisify<{ name: string; password: string }, VerifyCredentialsResponse>(
      accountWeb.verifyCredentials.bind(accountWeb),
    )({ name: parsed.name, password: parsed.pass });
  } catch {
    // Falha de infra nunca vira 500 com stack no cliente.
    return Response.json({ error: "serviço indisponível" }, { status: 502, headers: NO_STORE });
  }

  if (!res.ok) {
    // Mesma resposta para conta inexistente e senha errada: não vaza existência.
    return Response.json(
      { error: "usuário ou senha inválidos" },
      { status: 401, headers: NO_STORE },
    );
  }

  if (res.blocked) {
    return Response.json({ error: "conta bloqueada" }, { status: 403, headers: NO_STORE });
  }

  const role = normalizeRole(res.role ?? "");
  await setSession(Number(res.account_id), role);

  return Response.json({ ok: true, role }, { headers: NO_STORE });
}
