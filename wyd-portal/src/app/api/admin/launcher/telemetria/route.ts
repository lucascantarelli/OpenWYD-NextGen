import { AdminError } from "@/server/errors";
import { resumoTabela, resumo } from "@/server/launcher-telemetria";
import { requireStaff } from "@/server/session";

export const dynamic = "force-dynamic";

/**
 * Telemetria agregada do launcher (somente staff).
 *
 * O launcher manda eventos anônimos para `/api/launcher/telemetria`; este endpoint
 * é o outro lado, autenticado: números por `versao + operacao`, nunca uma trilha
 * por jogador. A agregação vive no processo do BFF, então o painel mostra a janela
 * desde o último boot — o log estruturado segue sendo a série histórica.
 */
export async function GET() {
  try {
    await requireStaff();
  } catch (erro) {
    const status = erro instanceof AdminError ? erro.status : 500;
    const mensagem = erro instanceof AdminError ? erro.message : "telemetria indisponível";
    return Response.json(
      { error: mensagem },
      { status, headers: { "cache-control": "no-store" } },
    );
  }
  const foto = resumo();
  return Response.json(
    {
      chaves: resumoTabela(),
      totalEventos: foto.totalEventos,
      descartados: foto.descartados,
    },
    { headers: { "cache-control": "no-store" } },
  );
}
