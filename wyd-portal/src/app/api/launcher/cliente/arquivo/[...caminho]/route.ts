import { abrirArquivoCliente } from "@/server/launcher-cliente";

export const dynamic = "force-dynamic";

/**
 * Bytes de um arquivo do cliente (`/api/launcher/cliente/arquivo/<caminho>`).
 *
 * O launcher confere tamanho + SHA-256 contra o manifesto antes de escrever;
 * a rota só transporta o que está em `client/`. Caminho inseguro ou ausente é
 * `404` com `no-store`, nunca leitura fora da raiz.
 */
export async function GET(
  _req: Request,
  { params }: { params: Promise<{ caminho: string[] }> },
) {
  const { caminho } = await params;
  const rel = (caminho ?? []).join("/");
  const bytes = abrirArquivoCliente(rel);
  if (bytes === null) {
    return Response.json(
      { error: "arquivo do cliente não encontrado" },
      { status: 404, headers: { "cache-control": "no-store" } },
    );
  }
  return new Response(new Uint8Array(bytes), {
    headers: {
      "content-type": "application/octet-stream",
      "content-length": String(bytes.byteLength),
      "cache-control": "no-store",
    },
  });
}
