import { abrirPacote } from "@/server/launcher-dist";

export const dynamic = "force-dynamic";

/**
 * Download do pacote do launcher.
 *
 * O nome vem da configuração do servidor (`LAUNCHER_PACKAGE`), nunca da URL:
 * não existe caminho escolhido pelo browser, então não há travessia de diretório
 * a validar. O SHA-256 real vai no header para conferência após o download.
 */
export async function GET() {
  const pacote = abrirPacote();
  if (pacote === null) {
    return Response.json(
      { error: "pacote do launcher não publicado" },
      { status: 404, headers: { "cache-control": "no-store" } },
    );
  }

  return new Response(new Uint8Array(pacote.conteudo), {
    headers: {
      "content-type": "application/octet-stream",
      "content-disposition": `attachment; filename="${pacote.nome}"`,
      "content-length": String(pacote.conteudo.byteLength),
      "x-checksum-sha256": pacote.sha256,
      "cache-control": "public, max-age=300",
    },
  });
}
