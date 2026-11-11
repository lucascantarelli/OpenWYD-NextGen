import { contentRates, promisify, type GetRatesResponse } from "@/server/admin-client";

export const dynamic = "force-dynamic";

/**
 * Versão que o launcher lê antes de abrir o jogo (LAUNCHER-01, pública).
 *
 * - `client_build`/`protocol_version` são constantes documentadas (não
 *   inventadas): build 7662 (`docs/arquitetura/protocolo.md`) e fio 12000
 *   (`wyd-protocol::PROTOCOL_VERSION` — o que o cliente envia e o gate confere).
 * - `config_revision` é viva: lida do gateway com o token de processo do BFF
 *   (mesmo padrão de `/api/health` — o browser nunca vê o token). Com o gateway
 *   fora do ar, as constantes continuam servidas e `gateway_up` é `false`.
 *
 * Cache curto no emissor (60 s): versão não muda a cada request.
 */
export async function GET() {
  let revision: number | null = null;
  let gateway = false;
  try {
    const res = await promisify<Record<string, never>, GetRatesResponse>(
      contentRates.getRates.bind(contentRates),
    )({});
    revision = res.revision ?? null;
    gateway = true;
  } catch {
    gateway = false;
  }

  return Response.json(
    {
      client_build: 7662,
      protocol_version: 12000,
      config_revision: revision,
      gateway_up: gateway,
    },
    { headers: { "cache-control": "public, max-age=60" } },
  );
}
