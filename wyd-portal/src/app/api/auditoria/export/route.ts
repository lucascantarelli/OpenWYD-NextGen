import { filterAuditTrail, getAuditTrail } from "@/server/moderation";
import { getSession, isStaff } from "@/server/session";

export const dynamic = "force-dynamic";

const NO_STORE = { "cache-control": "no-store" } as const;

/** Uma célula CSV com aspas escapadas; nunca quebra a linha. */
function cell(value: string): string {
  return `"${value.replace(/"/g, '""')}"`;
}

/**
 * Exporta a trilha de auditoria em CSV (janela das últimas 200 linhas).
 *
 * Server-only + staff: os mesmos filtros da tela (`ator`, `acao`, `alvo`,
 * `dias` 1..90). Nada de PII além do id — a linha já nasce sem senha, hash ou PIN.
 */
export async function GET(req: Request) {
  const session = await getSession();
  if (!session) {
    return Response.json({ error: "sessão expirada: entre novamente" }, { status: 401, headers: NO_STORE });
  }
  if (!isStaff(session.role)) {
    return Response.json(
      { error: "sua conta não tem permissão para esta ação" },
      { status: 403, headers: NO_STORE },
    );
  }

  const url = new URL(req.url);
  const actor = (url.searchParams.get("ator") ?? "").slice(0, 32);
  const action = (url.searchParams.get("acao") ?? "").slice(0, 64);
  const target = (url.searchParams.get("alvo") ?? "").slice(0, 64);
  const diasRaw = Number(url.searchParams.get("dias") ?? "");
  if (url.searchParams.get("dias") && !(Number.isInteger(diasRaw) && diasRaw >= 1 && diasRaw <= 90)) {
    return Response.json({ error: "dias deve ser 1..90" }, { status: 422, headers: NO_STORE });
  }
  const sinceMs =
    Number.isInteger(diasRaw) && diasRaw >= 1 ? Date.now() - diasRaw * 86_400_000 : undefined;

  let csv: string;
  try {
    const entries = filterAuditTrail(await getAuditTrail(200), {
      ...(actor ? { actor } : {}),
      ...(action ? { action } : {}),
      ...(target ? { target } : {}),
      ...(sinceMs !== undefined ? { sinceMs } : {}),
    });
    const lines = ["quando,autor_id,autor_papel,acao,alvo,detalhe"];
    for (const entry of entries) {
      lines.push(
        [
          cell(entry.at.toISOString()),
          cell(String(entry.actorId)),
          cell(entry.actorRole),
          cell(entry.action),
          cell(entry.target),
          cell(entry.detail),
        ].join(","),
      );
    }
    csv = `${lines.join("\n")}\n`;
  } catch {
    return Response.json({ error: "trilha de auditoria indisponível" }, { status: 502, headers: NO_STORE });
  }

  const stamp = new Date().toISOString().replace(/[:.]/g, "-");
  return new Response(csv, {
    headers: {
      ...NO_STORE,
      "content-type": "text/csv; charset=utf-8",
      "content-disposition": `attachment; filename="auditoria-${stamp}.csv"`,
    },
  });
}
