import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

/**
 * O portal **consome** a identidade única; não tem paleta própria.
 *
 * `docs/arquitetura/identidade.tokens` é a fonte (a mesma que o launcher embute no binário
 * para gerar `Launcher.bmp`/`Launcher.hat` e o `sn.bin`). Este teste falha se o CSS divergir
 * do arquivo, se um token não chegar ao CSS ou se aparecer uma cor solta no CSS que não
 * exista na identidade — as três formas de nascer uma paleta paralela.
 */

const raiz = fileURLToPath(new URL("../..", import.meta.url));
const tokensPath = `${raiz}/docs/arquitetura/identidade.tokens`;
const cssPath = `${raiz}/wyd-portal/src/app/globals.css`;

/** `chave = valor` do arquivo canônico (mesmo formato do parser Rust). */
function tokens(): Map<string, string> {
  const out = new Map<string, string>();
  for (const raw of readFileSync(tokensPath, "utf8").split(/\r?\n/)) {
    const line = raw.trim();
    if (!line || line.startsWith("#")) continue;
    const [chave, ...resto] = line.split("=");
    if (!chave || resto.length === 0) continue;
    out.set(chave.trim(), resto.join("=").trim());
  }
  return out;
}

/** Variáveis CSS `--nome: valor;` declaradas no arquivo inteiro. */
function variaveisCss(): Map<string, string> {
  const out = new Map<string, string>();
  for (const raw of readFileSync(cssPath, "utf8").split(/\r?\n/)) {
    const line = raw.trim();
    if (!line.startsWith("--")) continue;
    const [nome, ...resto] = line.split(":");
    if (!nome || resto.length === 0) continue;
    out.set(nome.trim(), resto.join(":").trim().replace(/;$/, "").trim());
  }
  return out;
}

describe("identidade única: portal consome os tokens", () => {
  it("toda cor da identidade existe no CSS com o mesmo valor", () => {
    const css = variaveisCss();
    const divergentes: string[] = [];
    for (const [chave, valor] of tokens()) {
      if (!chave.startsWith("cor.")) continue;
      const nome = `--${chave.slice("cor.".length)}`;
      if (css.get(nome) !== valor) divergentes.push(`${nome}: css=${css.get(nome)} tokens=${valor}`);
    }
    expect(divergentes).toEqual([]);
  });

  it("a escala de espaço é a mesma", () => {
    const css = variaveisCss();
    const divergentes: string[] = [];
    for (const [chave, valor] of tokens()) {
      if (!chave.startsWith("espaco.")) continue;
      const nome = `--space-${chave.slice("espaco.".length)}`;
      if (css.get(nome) !== `${valor}px`) divergentes.push(`${nome}: css=${css.get(nome)} tokens=${valor}px`);
    }
    expect(divergentes).toEqual([]);
  });

  it("raio e tipografia saem dos tokens", () => {
    const t = tokens();
    const css = variaveisCss();
    expect(css.get("--radius")).toBe(`${t.get("raio")}px`);
    expect(css.get("--radius-sm")).toBe(`${t.get("raio.pequeno")}px`);
    const arquivo = readFileSync(cssPath, "utf8");
    expect(arquivo).toContain(`font-family: ${t.get("tipografia.familia")};`);
    expect(arquivo).toContain(`font-size: ${t.get("tipografia.tamanho")}px;`);
    expect(arquivo).toContain(`line-height: ${t.get("tipografia.altura")};`);
  });

  it("nenhuma cor de 6 dígitos no CSS fora da identidade", () => {
    const cores = new Set(
      [...tokens()]
        .filter(([chave]) => chave.startsWith("cor."))
        .map(([, valor]) => valor.toLowerCase()),
    );
    const soltas: string[] = [];
    readFileSync(cssPath, "utf8")
      .split(/\r?\n/)
      .forEach((linha, indice) => {
        for (const achado of linha.toLowerCase().matchAll(/#([0-9a-f]{6})\b/g)) {
          const cor = `#${achado[1]}`;
          if (!cores.has(cor)) soltas.push(`linha ${indice + 1}: ${cor}`);
        }
      });
    expect(soltas).toEqual([]);
  });

  it("a identidade nomeia os servidores que o cliente desenha", () => {
    const nomes = [...tokens()].filter(([chave]) => /^servidor\.\d+$/.test(chave));
    expect(nomes.length).toBeGreaterThan(0);
    for (const [, nome] of nomes) {
      // Campo de 9 B no sn.bin: NUL no fim, logo no máximo 8 caracteres.
      expect(Buffer.byteLength(nome, "latin1")).toBeLessThanOrEqual(8);
    }
  });

  it("a tela de login usa as classes da identidade, sem estilo inline", () => {
    const arquivos = [
      "wyd-portal/src/app/(public)/login/page.tsx",
      "wyd-portal/src/app/(public)/login/login-form.tsx",
    ];
    for (const relativo of arquivos) {
      const fonte = readFileSync(`${raiz}/${relativo}`, "utf8");
      expect(fonte.toLowerCase()).not.toMatch(/#[0-9a-f]{6}/);
      expect(fonte).not.toContain("style={{");
    }
    const css = readFileSync(cssPath, "utf8");
    for (const classe of [
      ".login-shell",
      ".login-title",
      ".login-sub",
      ".login-hint",
      ".login-form",
      ".login-field",
      ".login-error",
    ]) {
      expect(css).toContain(classe);
    }
    // Título e legenda têm tamanho próprio, mas continuam declarados só no CSS.
    expect(css).toContain("--title-size: 22px;");
    expect(css).toContain("--legend-size: 13px;");
  });
});
