#!/usr/bin/env node
// Gera `src/content/noticias.json` a partir de `docs/noticias/*.md`.
//
// Mesma razão do `build-guias.mjs`: a imagem do portal copia só `wyd-portal/`,
// então `docs/` não existe dentro do container. O markdown é a fonte única; o
// JSON é o artefato versionado que a imagem carrega.
//
// Uso: npm run noticias   (dentro de wyd-portal/)
import { mkdirSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.resolve(here, "..", "..");
const sourceDir = path.join(repo, "docs", "noticias");
const outFile = path.resolve(here, "..", "src", "content", "noticias.json");

const REQUIRED = ["title", "date", "author", "tag", "summary"];

/** Frontmatter `chave: valor` entre duas linhas `---`; nada de YAML completo. */
function parse(file, raw) {
  const text = raw.replace(/\r\n/g, "\n");
  const match = text.match(/^---\n([\s\S]*?)\n---\n?([\s\S]*)$/);
  if (!match) throw new Error(`noticias: ${file} sem frontmatter`);
  const meta = {};
  for (const line of match[1].split("\n")) {
    if (line.trim() === "") continue;
    const split = line.indexOf(":");
    if (split < 0) throw new Error(`noticias: ${file} linha inválida: ${line}`);
    meta[line.slice(0, split).trim()] = line.slice(split + 1).trim();
  }
  for (const key of REQUIRED) {
    if (!meta[key]) throw new Error(`noticias: ${file} sem "${key}"`);
  }
  if (!/^\d{4}-\d{2}-\d{2}$/.test(meta.date)) {
    throw new Error(`noticias: ${file} data fora de AAAA-MM-DD`);
  }
  return {
    slug: file.replace(/\.md$/, ""),
    title: meta.title,
    date: meta.date,
    author: meta.author,
    tag: meta.tag,
    summary: meta.summary,
    body: match[2].trim(),
  };
}

let entries;
try {
  entries = readdirSync(sourceDir, { withFileTypes: true })
    .filter((entry) => entry.isFile() && entry.name.endsWith(".md"))
    .map((entry) => parse(entry.name, readFileSync(path.join(sourceDir, entry.name), "utf8")));
} catch (error) {
  throw new Error(`build-noticias: ${error instanceof Error ? error.message : error}`);
}

if (entries.length === 0) throw new Error("build-noticias: nenhuma notícia encontrada");

const seen = new Set();
for (const item of entries) {
  if (seen.has(item.slug)) throw new Error(`build-noticias: slug duplicado: ${item.slug}`);
  seen.add(item.slug);
}

entries.sort((a, b) => b.date.localeCompare(a.date) || a.slug.localeCompare(b.slug));

mkdirSync(path.dirname(outFile), { recursive: true });
writeFileSync(outFile, `${JSON.stringify(entries, null, 2)}\n`, "utf8");
console.log(`build-noticias: ${entries.length} notícia(s) em ${path.relative(repo, outFile)}`);
