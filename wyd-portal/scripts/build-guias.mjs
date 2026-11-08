#!/usr/bin/env node
// Gera `src/content/guias.json` a partir de `docs/` — é o conteúdo que o portal
// publica em `/guias` (guia do jogador, do administrador, sistemas, história,
// guerras e reinos).
//
// POR QUE UM JSON VERSIONADO, e não leitura de `docs/` em runtime:
// a imagem do portal copia apenas `wyd-portal/` (ver wyd-portal/Dockerfile), então
// `docs/` não existe dentro do container. Ler no build do Next também não
// resolve: o build roda no mesmo contexto reduzido. Gerar aqui e versionar o
// resultado mantém uma fonte única (o markdown) e um artefato que a imagem
// carrega. O CI roda este script e reprova se o JSON divergir de `docs/`.
//
// Uso: npm run guias   (dentro de wyd-portal/)
import { mkdirSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const here = path.dirname(fileURLToPath(import.meta.url));
const repo = path.resolve(here, "..", "..");
const outFile = path.resolve(here, "..", "src", "content", "guias.json");

/** Seções publicadas no portal. `dir` é relativo à raiz do repositório. */
// `docs/` foi reorganizado em 2026-11: as pastas `guias/`, `sistemas/` e `lore/`
// foram substituídas por `docs/wyd/` (manual de domínio). Manter seções apontando
// para diretórios inexistentes quebrava `npm run guias` — e o JSON antigo seguia
// publicado, escondendo o problema.
const SECTIONS = [
  {
    id: "manual",
    label: "Manual completo de WYD",
    order: 5,
    dir: "docs/wyd",
    prefix: "manual-",
    recursive: true,
  },
];

/**
 * Ordem do manual: história primeiro (dá contexto), depois personagem, depois
 * os sistemas na ordem em que o jogador os encontra, e as fontes por último.
 */
const MANUAL_ORDER = [
  "README",
  "lore",
  "classes",
  "skills",
  "progressao",
  "itens",
  "economia",
  "sistemas",
  "sistemas/refino",
  "sistemas/drops",
  "sistemas/guerras",
  "sistemas/mundo",
  "sistemas/quests",
  "sistemas/eventos",
  "fontes",
];

/**
 * Ordem de leitura das páginas de sistema. A ordem alfabética jogaria "castelos"
 * antes de "personagem"; a leitura começa pelo que o jogador vive primeiro.
 */
const SYSTEM_ORDER = [
  "personagem",
  "combate",
  "skills",
  "drops",
  "refino",
  "quests",
  "comercio",
  "guildas",
  "castelos",
  "eventos",
  "canais",
  "mapa",
];

/** README primeiro (é a introdução da seção); o resto segue a ordem declarada. */
function rankOf(sectionId, file) {
  const base = file.replace(/\.md$/, "").split(path.sep).join("/");
  if (base === "README") return -1;
  const order = sectionId === "sistemas" ? SYSTEM_ORDER : sectionId === "manual" ? MANUAL_ORDER : null;
  if (!order) return 0;
  const index = order.indexOf(base);
  return index === -1 ? order.length : index;
}

/** Caminhos relativos (com `/`) de todos os `.md`, recursivo quando pedido. */
function collect(dirAbs, recursive, prefix = "") {
  const found = [];
  for (const entry of readdirSync(dirAbs, { withFileTypes: true })) {
    const rel = prefix ? `${prefix}/${entry.name}` : entry.name;
    if (entry.isDirectory()) {
      if (recursive) found.push(...collect(path.join(dirAbs, entry.name), true, rel));
      continue;
    }
    if (entry.name.endsWith(".md")) found.push(rel);
  }
  return found;
}

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
    entries = collect(dir, Boolean(section.recursive));
  } catch {
    throw new Error(`build-guias: diretório ausente: ${section.dir}`);
  }
  entries.sort(
    (a, b) =>
      rankOf(section.id, a) - rankOf(section.id, b) || a.localeCompare(b),
  );

  for (const file of entries) {
    const abs = path.join(dir, file);
    // `collect` sempre usa "/" (mesmo no Windows), então não dá para confiar em `path.sep`.
    const base = file.replace(/\.md$/, "").replace(/[/\\]/g, "-");
    const prefix = section.prefix ?? "";
    // README vira a porta da seção: `manual`, `jogador`, `sistemas`…
    const slug = base === "README" ? prefix.replace(/-$/, "") || section.id : `${prefix}${base}`;
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
