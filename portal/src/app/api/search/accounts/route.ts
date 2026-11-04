import { accountOptionHint, accountOptionLabel } from "@/lib/accountsearch";
import { AdminError } from "@/server/errors";
import { searchAccounts } from "@/server/revenue";
import { getSession, isStaff } from "@/server/session";

export const dynamic = "force-dynamic";

const LIMIT = 20;

/**
 * Busca de contas para o `Combobox` (crédito manual da loja e filtro do
 * faturamento). Minúsculo e com sessão de staff: login de jogador não é dado
 * público, e o web-api recusa prefixo com menos de 2 caracteres.
 */
export async function GET(request: Request) {
  const session = await getSession();
  if (!session) {
    return Response.json({ error: "sessão expirada" }, { status: 401 });
  }
  if (!isStaff(session.role)) {
    return Response.json({ error: "sem permissão" }, { status: 403 });
  }

  const { searchParams } = new URL(request.url);
  const query = (searchParams.get("q") ?? "").trim().toLowerCase();

  if (query.length < 2) {
    return Response.json({ options: [] }, { headers: { "cache-control": "private, no-store" } });
  }

  try {
    const accounts = await searchAccounts(query, LIMIT);
    return Response.json(
      {
        options: accounts.map((account) => ({
          value: String(account.id),
          label: accountOptionLabel(account),
          hint: accountOptionHint(account),
        })),
      },
      { headers: { "cache-control": "private, no-store" } },
    );
  } catch (error) {
    const forbidden = error instanceof AdminError;
    return Response.json(
      { error: forbidden ? "sem permissão para consultar contas" : "contas indisponíveis" },
      { status: forbidden ? 403 : 502 },
    );
  }
}
