import { readFileSync, readdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

/**
 * Regra de segurança do portal: **página pública não anuncia a administração**.
 *
 * A guarda real é do `proxy.ts` + `(admin)/layout` + gateway; este teste impede a
 * regressão mais provável, que é alguém "ajudar" o staff com um atalho no header
 * público. Ele varre as páginas públicas, a home, o header e o footer.
 */
// Teste em `wyd-portal/src/`: `../..` sobe até a raiz do repositório.
const raiz = fileURLToPath(new URL("../..", import.meta.url));

/** Todos os `.tsx` de uma pasta, recursivo. */
function tsx(dir: string): string[] {
  const out: string[] = [];
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const abs = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...tsx(abs));
    else if (entry.name.endsWith(".tsx")) out.push(abs);
  }
  return out;
}

const portal = `${raiz}/wyd-portal`;
const ARQUIVOS = [
  `${portal}/src/app/page.tsx`,
  `${portal}/src/components/SiteHeader.tsx`,
  `${portal}/src/components/SiteFooter.tsx`,
  ...tsx(`${portal}/src/app/(public)`),
];

describe("site público não expõe a administração", () => {
  it("nenhum link para rotas de staff", () => {
    const infratores: string[] = [];
    for (const arquivo of ARQUIVOS) {
      const fonte = readFileSync(arquivo, "utf8");
      // `login-form.tsx` decide o destino do staff no cliente; ali o caminho é
      // permitido. O resto do site público não pode citar a administração.
      if (arquivo.endsWith("login-form.tsx")) continue;
      for (const rota of ["/painel", "/contas", "/monitoramento", "/auditoria", "/economia"]) {
        if (fonte.includes(`"${rota}`) || fonte.includes(`'${rota}`)) {
          infratores.push(`${path.relative(raiz, arquivo)}: ${rota}`);
        }
      }
    }
    expect(infratores).toEqual([]);
  });

  it("o menu público tem as páginas do jogador", () => {
    const header = readFileSync(`${portal}/src/components/SiteHeader.tsx`, "utf8");
    for (const rota of ["/noticias", "/ranking", "/downloads", "/guias", "/enciclopedia"]) {
      expect(header).toContain(`"${rota}"`);
    }
  });

  it("robots desindexa as rotas de staff", () => {
    const robots = readFileSync(`${portal}/src/app/robots.ts`, "utf8");
    for (const rota of ["/painel", "/contas", "/monitoramento", "/auditoria", "/login"]) {
      expect(robots).toContain(`"${rota}"`);
    }
  });
});
