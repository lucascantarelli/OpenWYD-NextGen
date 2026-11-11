import { lerManifesto } from "@/server/launcher-dist";

export const dynamic = "force-dynamic";

/**
 * Manifesto de distribuição do launcher (`manifest.txt`).
 *
 * Público e somente leitura: é o arquivo que o launcher baixa para conferir os
 * hashes antes de aplicar patch. Sem publicação, `404` — nunca um manifesto vazio
 * que faria o launcher aceitar qualquer arquivo.
 */
export async function GET() {
  const texto = lerManifesto();
  if (texto === null) {
    return Response.json(
      { error: "manifesto do launcher não publicado" },
      { status: 404, headers: { "cache-control": "no-store" } },
    );
  }
  return new Response(texto, {
    headers: {
      "content-type": "text/plain; charset=utf-8",
      "cache-control": "public, max-age=60",
    },
  });
}
