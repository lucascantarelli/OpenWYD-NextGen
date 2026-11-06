#!/usr/bin/env node
// Gera `src/content/guias.json` a partir de `docs/` — é o conteúdo que o portal
// publica em `/guias` (guia do jogador, do administrador, sistemas, história,
// guerras e reinos).
//
// POR QUE UM JSON VERSIONADO, e não leitura de `docs/` em runtime:
// a imagem do portal copia apenas `portal/` (ver portal/Dockerfile), então
// `docs/` não existe dentro do container. Ler no build do Next também não
// resolve: o build roda no mesmo contexto reduzido. Gerar aqui e versionar o
// resultado mantém uma fonte única (o markdown) e um artefato que a imagem
// carrega. O CI roda este script e reprova se o JSON divergir de `docs/`.
//
// Uso: npm run guias   (dentro de portal/)
import { mkdirSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.resolve(here, "..", "..");
const outFile = path.resolve(here, "..", "src", "content", "guias.json");

/** Seções publicadas no portal. `dir` é relativo à raiz do repositório. */
const SECTIONS = [
  { id: "jogador", label: "Guia do jogador", order: 10, dir: "docs/guias/usuario" },
  { id: "admin", label: "Guia do administrador", order: 20, dir: "docs/guias/admin" },
  { id: "sistemas", label: "Sistemas de jogo", order: 30, dir: "docs/sistemas" },
  { id: "lore", label: "História, guerras e reinos", order: 40, dir: "docs/lore" },
];

const readDoc = (abs) => readFileSync(abs, "utf8").replace(/\r\n/g, "\n");

const titleOf = (md, fallback) => md.match(/^#\s+(.+)$/m)?.[1].trim() ?? fallback;

/** O H1 vira o título da página; o corpo sai sem ele. */
const bodyOf = (md) => md.replace(/^#\s+.+\n+/, "");

/**
 * `!!! tipo "Título"` + bloco indentado → blockquote.
 *
 * O portal renderiza com `marked`, que não conhece a sintaxe de admonition do
 * mkdocs. Converter para blockquote mantém **o mesmo markdown** nos dois
 * destinos: o site renderiza admonition colorido, o portal renderiza citação.
 */
function admonitionsToQuotes(md) {
  const lines = md.split("\n");
  const out = [];

  for (let i = 0; i < lines.length; i += 1) {
    const opening = lines[i].match(/^(\s*)!!!\s+\w+(?:\s+"(.+)")?\s*$/);
    if (!opening) {
      out.push(lines[i]);
      continue;
    }

    const indent = opening[1].length;
    const body = [];
    let j = i + 1;
    for (; j < lines.length; j += 1) {
      const line = lines[j];
      if (line.trim() === "") {
        body.push("");
        continue;
      }
      const lead = line.match(/^\s*/)[0].length;
      if (lead <= indent) break;
      body.push(line.slice(indent + 4));
    }

    while (body.length > 0 && body[body.length - 1] === "") body.pop();
    if (opening[2]) out.push(`> **${opening[2]}**`, ">");
    for (const line of body) out.push(line === "" ? ">" : `> ${line}`);

    i = j - 1;
  }

  return out.join("\n");
}

// --- 1. coleta -------------------------------------------------------------

const docs = [];
for (const section of SECTIONS) {
  const dir = path.join(repo, section.dir);
  let entries;
  try {
    entries = readdirSync(dir).filter((f) => f.endsWith(".md"));
  } catch {
    throw new Error(`build-guias: diretório ausente: ${section.dir}`);
  }
  entries.sort();

  for (const file of entries) {
    const abs = path.join(dir, file);
    const base = file.replace(/\.md$/, "");
    const slug = base === "README" ? section.id : base;
    docs.push({ abs, slug, section, file, md: readDoc(abs) });
  }
}

const bySlug = new Map(docs.map((doc) => [doc.slug, doc]));
const slugs = new Map(docs.map((doc) => [path.resolve(doc.abs), doc.slug]));

// --- 2. reescreve links ----------------------------------------------------

/**
 * Link para outro guia vira rota `/guias/<slug>`. Link que sai do conjunto
 * publicado (regra de jogo, handler, portal) perde o link e mantém o texto: o
 * portal não serve `docs/migration/**`, e link morto é pior que texto.
 */
function rewriteLinks(md, fromAbs) {
  return md.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (whole, text, target) => {
    if (/^(https?:|mailto:|#)/.test(target)) return whole;

    const [rawPath, anchor] = target.split("#");
    const resolved = path.resolve(path.dirname(fromAbs), rawPath);
    const slug = slugs.get(resolved);

    if (slug) {
      const suffix = anchor ? `#${anchor}` : "";
      return `[${text}](/guias/${slug}${suffix})`;
    }
    return text;
  });
}

// --- 3. monta a saída ------------------------------------------------------

const guias = docs
  .map((doc, index) => ({
    slug: doc.slug,
    section: doc.section.id,
    sectionLabel: doc.section.label,
    sectionOrder: doc.section.order,
    order: index,
    title: titleOf(doc.md, doc.slug),
    body: rewriteLinks(admonitionsToQuotes(bodyOf(doc.md)), doc.abs).trim(),
  }))
  .sort((a, b) => a.sectionOrder - b.sectionOrder || a.order - b.order);

const seen = new Set();
for (const guia of guias) {
  if (seen.has(guia.slug)) throw new Error(`build-guias: slug duplicado: ${guia.slug}`);
  seen.add(guia.slug);
  if (bySlug.size === 0) throw new Error("build-guias: nenhum guia encontrado");
}

mkdirSync(path.dirname(outFile), { recursive: true });
writeFileSync(outFile, `${JSON.stringify(guias, null, 2)}\n`, "utf8");

console.log(`build-guias: ${guias.length} página(s) em ${path.relative(repo, outFile)}`);
