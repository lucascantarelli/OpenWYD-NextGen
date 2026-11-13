import { lerConfiguracao } from "@/server/launcher-dist";

export const dynamic = "force-dynamic";

/**
 * Configuração assinada do launcher (`launcher.conf`).
 *
 * Público e somente leitura: são bytes assinados, não segredo. O launcher baixa,
 * confere o HMAC com a chave que já tem (`--key-file`) e só então usa marca,
 * canais e URLs — por isso a rota não interpreta nem re-assina nada.
 *
 * Sem publicação, `404`: um `launcher.conf` vazio seria aceito como "sem
 * sobreposição" e esconderia a falta de publicação.
 */
export async function GET() {
  const texto = lerConfiguracao();
  if (texto === null) {
    return Response.json(
      { error: "configuração do launcher não publicada" },
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
