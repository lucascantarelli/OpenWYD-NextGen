import { readRegionMapPng, readWorldMapHdPng, readWorldMapPng } from "@/server/atlas";
import { getSession } from "@/server/session";

export const dynamic = "force-dynamic";

/**
 * Relevo do mundo: visão geral por padrão, detalhe nativo com `?detail=1`.
 *
 * O detalhe existe porque 1024² sobre um mundo de 4096 blocos deixa 1 pixel para
 * cada 4×4 blocos: ampliar aquilo só estica blocos. O cliente pede o detalhe
 * depois de 2× e mantém a visão geral embaixo, então não há tela vazia enquanto
 * o arquivo grande carrega.
 */
export async function GET(request: Request) {
  const session = await getSession();
  if (!session) return new Response("sessão expirada", { status: 401 });

  const params = new URL(request.url).searchParams;
  const detail = params.get("detail") === "1";
  const regionLayer = params.get("layer") === "regions";

  const png = regionLayer
    ? await readRegionMapPng()
    : detail
      ? await readWorldMapHdPng()
      : await readWorldMapPng();
  if (!png) {
    return new Response(
      regionLayer ? "máscara de regiões não gerada" : detail ? "detalhe do mapa não gerado" : "mapa não gerado",
      { status: 404 },
    );
  }

  return new Response(new Uint8Array(png), {
    headers: {
      "content-type": "image/png",
      "cache-control": "private, max-age=300",
    },
  });
}
