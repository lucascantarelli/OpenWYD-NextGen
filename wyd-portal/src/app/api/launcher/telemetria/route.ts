import { esquemaEvento, LIMITE_CORPO, registrar } from "@/server/launcher-telemetria";

export const dynamic = "force-dynamic";

const SEM_CACHE = { "cache-control": "no-store" } as const;

function erro(status: number, mensagem: string): Response {
  return Response.json({ error: mensagem }, { status, headers: SEM_CACHE });
}

/**
 * Telemetria anônima do launcher (pública por necessidade: o launcher ainda não
 * tem sessão). O contrato é fechado (`strictObject`) e o corpo é limitado antes
 * de qualquer parse: caminho absoluto, login, IP ou campo extra é 400/413, não
 * "dado a mais guardado".
 *
 * Resposta `204` sem corpo — o launcher nunca depende dela para abrir o jogo.
 */
export async function POST(request: Request) {
  const tipo = request.headers.get("content-type") ?? "";
  if (!tipo.toLowerCase().includes("application/json")) {
    return erro(415, "corpo deve ser application/json");
  }
  const declarado = Number(request.headers.get("content-length") ?? "0");
  if (Number.isFinite(declarado) && declarado > LIMITE_CORPO) {
    return erro(413, "telemetria acima do limite");
  }
  const texto = await request.text();
  if (texto.length > LIMITE_CORPO) {
    return erro(413, "telemetria acima do limite");
  }

  let bruto: unknown;
  try {
    bruto = JSON.parse(texto);
  } catch {
    return erro(400, "telemetria não é JSON");
  }

  const lido = esquemaEvento.safeParse(bruto);
  if (!lido.success) {
    return erro(400, "telemetria fora do contrato");
  }

  registrar(lido.data);
  return new Response(null, { status: 204, headers: SEM_CACHE });
}
