/**
 * Arquivos do cliente real para o reparo do launcher.
 *
 * O portal serve o `manifest.txt` e os bytes do diretório do cliente
 * (`WYD_CLIENT_DIR`, padrão `<repo>/client`) para que um arquivo adulterado
 * possa ser rebaixado pelo hash do manifesto. É leitura pública de binário do
 * jogo (o mesmo que qualquer jogador instala), sem sessão.
 *
 * Segurança: só caminho relativo simples (`a/b/c.bin`), nunca absoluto, `..`,
 * barra invertida ou drive — o caminho resolvido tem de continuar dentro da
 * raiz. Ausente é `null` (a rota devolve 404), nunca arquivo inventado.
 */
import { existsSync, readFileSync, statSync } from "node:fs";
import path from "node:path";

const MANIFESTO = "manifest.txt";

/** Diretório do cliente (`WYD_CLIENT_DIR`, relativo ao cwd quando definido). */
export function dirCliente(): string {
  const env = process.env.WYD_CLIENT_DIR;
  return path.resolve(env && env.trim() !== "" ? env : path.join(process.cwd(), "..", "client"));
}

/** Caminho relativo seguro dentro da raiz, ou `null`. */
export function caminhoSeguro(rel: string, raiz = dirCliente()): string | null {
  if (!rel || rel.includes("\\") || rel.startsWith("/") || /^[A-Za-z]:/.test(rel)) return null;
  if (rel.split("/").some((parte) => parte === "" || parte === "." || parte === "..")) return null;
  const completo = path.resolve(raiz, rel);
  return completo.startsWith(raiz + path.sep) ? completo : null;
}

/** Manifesto do cliente (`text/plain`), ou `null` quando não existe. */
export function lerManifestoCliente(raiz = dirCliente()): string | null {
  const arquivo = path.join(raiz, MANIFESTO);
  if (!existsSync(arquivo) || !statSync(arquivo).isFile()) return null;
  return readFileSync(arquivo, "utf8");
}

/** Bytes de um arquivo do cliente, ou `null` (ausente/inseguro). */
export function abrirArquivoCliente(rel: string, raiz = dirCliente()): Buffer | null {
  const completo = caminhoSeguro(rel, raiz);
  if (!completo || !existsSync(completo) || !statSync(completo).isFile()) return null;
  return readFileSync(completo);
}
