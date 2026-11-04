import { NextResponse, type NextRequest } from "next/server";
import { jwtVerify } from "jose";

import { SESSION_COOKIE, sessionKey } from "@/server/session-key";

/**
 * Guarda de rota do portal: valida presença + assinatura do cookie, sem gRPC
 * (middleware roda no edge). A autorização real continua no web-api e, para a
 * UI, no `(admin)/layout.tsx`.
 *
 * A lista abaixo espelha `SHELL_ROUTES` em `src/components/nav.ts` — o matcher
 * do Next precisa ser literal, então a sincronia é manual e vale a pena revisar
 * as duas ao adicionar uma seção.
 */
const GUARDED = [
  "/painel",
  "/npcs",
  "/mobs",
  "/precos", "/atlas", "/itens",
  "/teleportes",
  "/personagens",
  "/drops",
  "/mapa",
  "/eventos",
  "/loja",
  "/diaria",
  "/faturamento",
  "/contas",
  "/config",
  "/operacao",
  "/logs",
];

void GUARDED;

export async function middleware(req: NextRequest) {
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
    "/teleportes/:path*",
    "/personagens/:path*",
    "/drops/:path*",
    "/mapa/:path*",
    "/eventos/:path*",
    "/loja/:path*",
    "/diaria/:path*",
    "/faturamento/:path*",
    "/contas/:path*",
    "/config/:path*",
    "/operacao/:path*",
    "/logs/:path*",
  ],
};
