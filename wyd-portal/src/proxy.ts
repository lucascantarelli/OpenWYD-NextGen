import { NextResponse, type NextRequest } from "next/server";
import { jwtVerify } from "jose";

import { SESSION_COOKIE, sessionKey } from "@/server/session-key";

/**
 * Guarda de rota do portal: valida presença + assinatura do cookie, sem gRPC
 * (o proxy roda no edge). A autorização real continua no web-api e, para a UI,
 * no `(admin)/layout.tsx`.
 *
 * Next 16 renomeou a convenção `middleware.ts` para `proxy.ts`; o arquivo e o
 * export seguem o nome novo, senão o build avisa que a convenção está obsoleta.
 *
 * O `matcher` do Next precisa ser literal, então a lista não pode ser derivada
 * em runtime — antes havia aqui um array `GUARDED` morto (`void GUARDED`) que
 * prometia cobrir as rotas e não cobria nada. `/precos` e `/atlas` ficaram de
 * fora do matcher por isso; ao criar uma rota nova, acrescente-a aqui e teste
 * com `curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:3000/<rota>`
 * sem cookie (esperado 307 para /login).
 */

export async function proxy(req: NextRequest) {
  const token = req.cookies.get(SESSION_COOKIE)?.value;
  if (token) {
    try {
      await jwtVerify(token, sessionKey(), { algorithms: ["HS256"] });
      return NextResponse.next();
    } catch {
      // cai no redirect abaixo
    }
  }

  const url = req.nextUrl.clone();
  url.pathname = "/login";
  url.search = "";
  return NextResponse.redirect(url);
}

export const config = {
  matcher: [
    "/painel/:path*",
    "/npcs/:path*",
    "/mobs/:path*",
    "/itens/:path*",
    "/precos/:path*",
    "/atlas/:path*",
    "/teleportes/:path*",
    "/personagens/:path*",
    "/drops/:path*",
    "/mapa/:path*",
    "/eventos/:path*",
    "/loja/:path*",
    "/diaria/:path*",
    "/faturamento/:path*",
    "/contas/:path*",
    "/guildas/:path*",
    "/castelo/:path*",
    "/config/:path*",
    "/logs/:path*",
  ],
};
