import { NextResponse, type NextRequest } from "next/server";
import { jwtVerify } from "jose";

import { SESSION_COOKIE, sessionKey } from "@/server/session-key";

/**
 * Guarda de rota do portal: valida presença + assinatura do cookie (o proxy roda
 * no edge, sem backend). A autorização real continua no `wyd-admin` e, para a
 * UI, no `(admin)/layout.tsx`.
 *
 * O `matcher` do Next precisa ser literal: ao criar uma tela administrativa nova,
 * acrescente-a aqui. `proxy.test.ts` compara o matcher com as páginas em disco e
 * com o menu, então rota esquecida reprova o teste.
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
    "/contas/:path*",
    "/entregas/:path*",
    "/itens/:path*",
    "/monitoramento/:path*",
    "/configuracoes/:path*",
    "/auditoria/:path*",
    "/economia/:path*",
  ],
};
