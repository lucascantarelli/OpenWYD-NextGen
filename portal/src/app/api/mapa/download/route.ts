import { MAP_BYTES, parseTransformQuery, transformError } from "@/lib/attributemap";
import { transformAttributeMapBytes } from "@/server/attributemap";
import { AdminError } from "@/server/errors";
import { getSession } from "@/server/session";

export const dynamic = "force-dynamic";

/**
 * Download do `AttributeMap_New.dat` (issue 27).
 *
 * Refaz a transformação com os mesmos parâmetros da pré-visualização e devolve
 * os bytes crus. A checagem de sessão é explícita aqui (a rota não passa pelo
 * layout do admin) e o `adminCall`, dentro de `transformAttributeMapBytes`,
 * revalida o papel no web-api e audita `map.download`.
 */
export async function GET(request: Request) {
  const session = await getSession();
  if (!session) {
    return Response.json({ error: "sessão expirada: entre novamente" }, { status: 401 });
  }

  const params = parseTransformQuery(new URL(request.url).searchParams);
  const invalid = transformError(params);
  if (invalid) {
    return Response.json({ error: invalid }, { status: 422 });
  }

  try {
    const out = await transformAttributeMapBytes(params);
    if (out.bytes.length !== MAP_BYTES) {
      return Response.json(
        { error: `payload com ${out.bytes.length} bytes; esperado ${MAP_BYTES}` },
        { status: 502 },
      );
    }
    return new Response(new Uint8Array(out.bytes), {
      headers: {
        "content-type": "application/octet-stream",
        "content-disposition": `attachment; filename="${out.filename}"`,
        "content-length": String(out.bytes.length),
        "x-map-sha256": out.sha256,
        "x-map-changed-count": String(out.changedCount),
        "cache-control": "private, no-store",
      },
    });
  } catch (error) {
    if (error instanceof AdminError) {
      return Response.json({ error: error.message }, { status: error.status });
    }
    console.error(
      "[attributemap] falha no download:",
      error instanceof Error ? error.message : String(error),
    );
    return Response.json({ error: "serviço de administração indisponível" }, { status: 502 });
  }
}
