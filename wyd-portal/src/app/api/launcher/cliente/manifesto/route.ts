import { lerManifestoCliente } from "@/server/launcher-cliente";

export const dynamic = "force-dynamic";

/**
 * Manifesto do cliente real (`client/manifest.txt`).
 *
 * Público e somente leitura: é o mesmo arquivo que o launcher confere antes de
 * abrir o jogo e a fonte do reparo. Sem manifesto, `404` — nunca um manifesto
 * vazio que faria o launcher aceitar qualquer arquivo.
 */
export function GET() {
  const texto = lerManifestoCliente();
  if (texto === null) {
    return Response.json(
      { error: "manifesto do cliente não disponível" },
      { status: 404, headers: { "cache-control": "no-store" } },
    );
  }
  return new Response(texto, {
    headers: {
      "content-type": "text/plain; charset=utf-8",
      "cache-control": "no-store",
    },
  });
}
