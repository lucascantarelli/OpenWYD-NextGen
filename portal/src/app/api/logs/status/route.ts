import { composeStatus } from "@/server/docker";
import { getSession, isStaff } from "@/server/session";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const NO_STORE = { "cache-control": "no-store" } as const;

/** `GET /api/logs/status` — estado de cada serviço para os chips da tela. */
export async function GET() {
  const session = await getSession();
  if (!session) {
    return Response.json({ error: "sessão expirada: entre novamente" }, { status: 401, headers: NO_STORE });
  }
  if (!isStaff(session.role)) {
    return Response.json(
      { error: "sua conta não tem permissão para ver os logs dos containers" },
      { status: 403, headers: NO_STORE },
    );
  }

  const status = await composeStatus();
  return Response.json(status, { headers: NO_STORE });
}
