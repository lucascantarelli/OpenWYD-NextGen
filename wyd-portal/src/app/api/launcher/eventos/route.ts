import { promisify, worldEventAdmin, type WorldEventResponse } from "@/server/admin-client";

export const dynamic = "force-dynamic";

/**
 * Grade pública de eventos para o launcher (CLIENT-11).
 *
 * Lê `world_event_admin/get_world_event_config` com o token de processo do BFF
 * (mesmo padrão de `/api/health`): a rota do gateway exige staff, e aqui quem
 * decide o que é público é o portal — só saem código, nome, estado e horários,
 * nunca alvo interno ou trilha. Com o gateway fora do ar, a grade vem vazia em
 * vez de 502: launcher sem grade ainda abre o jogo.
 *
 * Cache curto no emissor (60 s): grade não muda a cada request.
 */
export async function GET() {
  try {
    const res = await promisify<Record<string, never>, WorldEventResponse>(
      worldEventAdmin.getWorldEventConfig.bind(worldEventAdmin),
    )({});
    return Response.json(
      { events: res.events ?? [] },
      { headers: { "cache-control": "public, max-age=60" } },
    );
  } catch {
    return Response.json(
      { events: [] },
      { headers: { "cache-control": "public, max-age=60" } },
    );
  }
}
