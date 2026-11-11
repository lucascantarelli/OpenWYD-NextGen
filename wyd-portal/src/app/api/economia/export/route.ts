import { getEconomySummary } from "@/server/economy";
import { getSession, isStaff } from "@/server/session";

export const dynamic = "force-dynamic";

const NO_STORE = { "cache-control": "no-store" } as const;

function cell(value: string): string {
  return `"${value.replace(/"/g, '""')}"`;
}

/**
 * Exporta o resumo de economia em CSV (imposto por cidade + concessões por tipo).
 *
 * Server-only + staff, mesmas fontes auditadas da tela (`city_tax_ledger` e
 * `delivery_queue`). Agregados por cidade/tipo, sem linha por jogador.
 */
export async function GET() {
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

  let csv: string;
  try {
    const summary = await getEconomySummary();
    const lines = ["secao,chave,total"];
    for (const row of summary.taxByCity) {
      lines.push(["imposto_por_cidade", row.city, String(row.total)].map(cell).join(","));
    }
    for (const row of summary.grants) {
      lines.push(
        [`concessoes_${row.kind}_linhas`, row.kind, String(row.count)].map(cell).join(","),
      );
      lines.push(
        [`concessoes_${row.kind}_ouro`, row.kind, String(row.gold)].map(cell).join(","),
      );
    }
    csv = `${lines.join("\n")}\n`;
  } catch {
    return Response.json({ error: "economia indisponível" }, { status: 502, headers: NO_STORE });
  }

  const stamp = new Date().toISOString().replace(/[:.]/g, "-");
  return new Response(csv, {
    headers: {
      ...NO_STORE,
      "content-type": "text/csv; charset=utf-8",
      "content-disposition": `attachment; filename="economia-${stamp}.csv"`,
    },
  });
}
