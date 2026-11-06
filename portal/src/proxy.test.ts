import { readdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

import { SignJWT } from "jose";
import { NextRequest } from "next/server";
import { beforeAll, describe, expect, it } from "vitest";

import { NAV, SHELL_ROUTES } from "./components/nav";
import { config, proxy } from "./proxy";
import { SESSION_COOKIE, sessionKey } from "./server/session-key";

/**
 * Prova a guarda de rota do portal. Os dois lados do contrato importam:
 *
 * 1. **comportamento** — sem cookie (ou com cookie adulterado/expirado) a request
 *    termina em `/login`, e cookie válido passa;
 * 2. **cobertura** — toda página de `(admin)` e todo link do menu sem `soon` está
 *    no `matcher`. É este segundo lado que faltava: o `matcher` é literal, então
 *    uma rota nova (ou uma lista morta, como a antiga `GUARDED`) deixa a página
 *    aberta sem nenhum teste reclamar. Em 06/11 `/precos`, `/atlas`, `/guildas` e
 *    `/castelo` viviam exatamente assim.
 */

const ORIGIN = "http://127.0.0.1:3000";

const env = process.env as Record<string, string | undefined>;
env.SESSION_SECRET = "segredo-de-teste-com-48-caracteres-para-o-hs256!!";

/** Padrão do `matcher` (`/itens/:path*`) aplicado como o Next aplica: zero ou mais. */
function covered(pathname: string, patterns: readonly string[] = config.matcher): boolean {
  return patterns.some((pattern) => {
    const base = pattern.replace(/\/:path\*$/, "");
    return pathname === base || pathname.startsWith(`${base}/`);
  });
}

/** Caminhos de página de `app/(admin)`, com `[param]` virando um valor concreto. */
function adminRoutePaths(): string[] {
  const root = fileURLToPath(new URL("./app/(admin)", import.meta.url));
  const out: string[] = [];

  const walk = (dir: string, prefix: string) => {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const segment = entry.name.startsWith("(")
        ? ""
        : `/${entry.name.replace(/^\[.+\]$/, "1")}`;
      if (entry.isDirectory()) {
        walk(path.join(dir, entry.name), prefix + segment);
      } else if (entry.name === "page.tsx" && prefix !== "") {
        out.push(prefix);
      }
    }
  };

  walk(root, "");
  return out.sort();
}

function request(pathname: string, cookie?: string): NextRequest {
  return new NextRequest(new URL(pathname, ORIGIN), {
    headers: cookie ? { cookie } : {},
  });
}

async function token(secret: Uint8Array, exp: string | number = "1h"): Promise<string> {
  return new SignJWT({ role: "admin" })
    .setProtectedHeader({ alg: "HS256" })
    .setSubject("1")
    .setIssuedAt()
    .setExpirationTime(exp)
    .sign(secret);
}

describe("proxy: visitante sem sessão", () => {
  it("redireciona para /login com 307 e sem query string", async () => {	const res = await proxy(request("/painel?tab=hoje"));

    expect(res.status).toBe(307);
    const location = new URL(res.headers.get("location")!);
    expect(location.pathname).toBe("/login");
    expect(location.search).toBe("");
  });

  it("rota aninhada também é barrada", async () => {	const res = await proxy(request("/npcs/7/editar"));

    expect(res.status).toBe(307);
    expect(new URL(res.headers.get("location")!).pathname).toBe("/login");
  });
});

describe("proxy: cookie de sessão", () => {
  it("cookie válido passa sem redirecionar", async () => {	const res = await proxy(request("/itens", `${SESSION_COOKIE}=${await token(sessionKey())}`));

    expect(res.headers.get("x-middleware-next")).toBe("1");
    expect(res.headers.get("location")).toBeNull();
  });

  it("cookie assinado com outra chave é recusado", async () => {
    const forged = await token(
      new TextEncoder().encode("chave-errada-com-48-caracteres-para-o-hs256!!!!!"),
    );	const res = await proxy(request("/itens", `${SESSION_COOKIE}=${forged}`));

    expect(res.status).toBe(307);
  });

  it("cookie expirado é recusado", async () => {
    const past = Math.floor(Date.now() / 1000) - 7200;
    const expired = await new SignJWT({ role: "admin" })
      .setProtectedHeader({ alg: "HS256" })
      .setSubject("1")
      .setIssuedAt(past)
      .setExpirationTime(past + 60)
      .sign(sessionKey());	const res = await proxy(request("/painel", `${SESSION_COOKIE}=${expired}`));

    expect(res.status).toBe(307);
  });
});

describe("cobertura do matcher", () => {
  let pages: string[];

  beforeAll(() => {
    pages = adminRoutePaths();
  });

  it("encontra as páginas de (admin) no disco", () => {
    // Guarda contra o teste passar por acidente: com a lista vazia, nenhuma
    // asserção de cobertura abaixo provaria nada.
    expect(pages.length).toBeGreaterThanOrEqual(20);
    expect(pages).toContain("/painel");
  });

  it("toda página de (admin) está no matcher", () => {
    expect(pages.filter((page) => !covered(page))).toEqual([]);
  });

  it("todo link do menu sem `soon` está no matcher", () => {
    const soon = new Map(NAV.flatMap((g) => g.items).map((item) => [item.href, item.soon === true]));
    const missing = SHELL_ROUTES.filter((href) => !soon.get(href) && !covered(href));

    expect(missing).toEqual([]);
  });

  it("nenhum padrão do matcher aponta para rota inexistente", () => {
    const stale = config.matcher
      .map((pattern) => pattern.replace(/\/:path\*$/, ""))
      .filter((base) => !pages.some((page) => page === base || page.startsWith(`${base}/`)));

    expect(stale).toEqual([]);
  });

  it("as rotas públicas continuam fora do matcher", () => {
    expect(covered("/login")).toBe(false);
    expect(covered("/sem-permissao")).toBe(false);
    expect(covered("/")).toBe(false);
  });

  it("/precos, /atlas, /guildas e /castelo estão guardados (regressão de 06/11)", () => {
    for (const route of ["/precos", "/atlas", "/guildas", "/castelo"]) {
      expect(covered(route), route).toBe(true);
    }
  });
});
