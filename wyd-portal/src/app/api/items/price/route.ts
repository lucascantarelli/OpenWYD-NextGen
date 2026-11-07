import { priceSchema } from "@/lib/price";
import { setItemPrice } from "@/server/actions/prices";

export const dynamic = "force-dynamic";

/**
 * Porta HTTP do mesmo caso de uso de `/precos` (issue 24).
 *
 * A tela usa a Server Action; esta rota existe para que a regra
 * `AdminResult → HTTP` seja verificável por requisição — item existente 200,
 * inexistente 422 — sem depender de navegador. Sessão, auditoria e o
 * `moderator_id` da sessão continuam dentro do `adminCall`; aqui não há
 * autorização própria.
 */
export async function POST(request: Request) {
  let body: unknown;
  try {
    body = await request.json();
  } catch {
    return Response.json({ error: "corpo inválido" }, { status: 422 });
  }

  const parsed = priceSchema.safeParse(body);
  if (!parsed.success) {
    return Response.json(
      { error: parsed.error.issues[0]?.message ?? "dados inválidos" },
      { status: 422 },
    );
  }

  const result = await setItemPrice(parsed.data);
  return Response.json(result, {
    status: result.ok ? 200 : result.status,
    headers: { "cache-control": "no-store" },
  });
}
