#!/usr/bin/env node
/**
 * Gate de skills: valida o que o agente carrega e o que os docs/issues citam.
 *
 * Regras (spec Agent Skills / Freebuff):
 *  - um SKILL.md por diretório, com `name` e `description` no frontmatter;
 *  - `name` igual ao diretório, minúsculo, até 64 caracteres;
 *  - `description` de 1 a 1024 caracteres;
 *  - corpo do SKILL.md com no máximo 500 linhas;
 *  - recursos `references/`, `scripts/`, `assets/` e `agents/` citados existem;
 *  - caminhos `.agents/skills/<slug>` em docs/issues existem;
 *  - issues não citam aliases removidos nesta auditoria.
 */

import { existsSync, readFileSync, readdirSync, statSync } from "node:fs";
import path from "node:path";

const ROOT = process.cwd();
const SKILLS_DIR = path.join(ROOT, ".agents", "skills");
const RETIRED = [
  "github-actions-docs",
  "solid-design-patterns",
];
const problems = [];

function fail(message) {
  problems.push(message);
}

function walkMarkdown(dir, out = []) {
  if (!existsSync(dir)) return out;
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.name === "node_modules" || entry.name === "target" || entry.name === ".git") {
      continue;
    }
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) walkMarkdown(full, out);
    else if (entry.isFile() && entry.name.endsWith(".md")) out.push(full);
  }
  return out;
}

function withoutFences(text) {
  return text.replace(/```[\s\S]*?```/g, "");
}

function frontmatterValue(meta, key) {
  const match = meta.match(new RegExp(`^${key}:\\s*(.+)$`, "m"));
  if (!match) return undefined;
  return match[1].trim().replace(/^["']|["']$/g, "");
}

if (!existsSync(SKILLS_DIR)) {
  console.error("skills-check: .agents/skills não existe");
  process.exit(1);
}

const skills = readdirSync(SKILLS_DIR, { withFileTypes: true })
  .filter((entry) => entry.isDirectory() && existsSync(path.join(SKILLS_DIR, entry.name, "SKILL.md")))
  .map((entry) => entry.name)
  .sort();

if (skills.length === 0) fail(".agents/skills não tem nenhum SKILL.md");

const names = new Map();
let skillLines = 0;
let skillBytes = 0;

for (const dir of skills) {
  const file = path.join(SKILLS_DIR, dir, "SKILL.md");
  const text = readFileSync(file, "utf8");
  const fm = text.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n/);

  if (!fm) {
    fail(`${dir}: sem frontmatter delimitado por ---`);
    continue;
  }

  const meta = fm[1];
  const name = frontmatterValue(meta, "name");
  const description = frontmatterValue(meta, "description");

  if (!name) fail(`${dir}: campo name ausente`);
  else if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(name) || name.length > 64) {
    fail(`${dir}: name inválido (minúsculo/hífen, até 64): ${name}`);
  } else if (name !== dir) {
    fail(`${dir}: name "${name}" difere do diretório`);
  } else if (names.has(name)) {
    fail(`${dir}: name duplicado também usado em ${names.get(name)}`);
  } else {
    names.set(name, dir);
  }

  if (!description) fail(`${dir}: campo description ausente`);
  else if (description.length < 1 || description.length > 1024) {
    fail(`${dir}: description com ${description.length} caracteres (1..1024)`);
  }

  const body = text.slice(fm[0].length);
  const lines = body.split(/\r?\n/).length;
  skillLines += lines;
  skillBytes += statSync(file).size;
  if (lines > 500) fail(`${dir}: corpo com ${lines} linhas (máximo 500)`);

  const visible = withoutFences(body);
  for (const match of visible.matchAll(/`((?:references|scripts|assets|agents)\/[^`\s]+)`/g)) {
    const resource = match[1].replace(/[.,;:]+$/, "");
    if (resource.endsWith("/")) continue;
    if (!existsSync(path.join(SKILLS_DIR, dir, resource))) {
      fail(`${dir}: recurso citado não existe: ${resource}`);
    }
  }

  for (const match of visible.matchAll(/`((?:docs|wyd-portal)\/[^`\s]+)`/g)) {
    const resource = match[1].replace(/[.,;:]+$/, "");
    if (resource.endsWith("/")) continue;
    if (!existsSync(path.join(ROOT, resource))) {
      fail(`${dir}: caminho de doc citado não existe: ${resource}`);
    }
  }
}

const docs = ["README.md", "AGENTS.md", path.join(".agents", "skills", "README.md")]
  .map((file) => path.join(ROOT, file))
  .filter((file) => existsSync(file));
for (const dir of ["docs", "roadmap-rust"]) {
  docs.push(...walkMarkdown(path.join(ROOT, dir)));
}

const known = new Set(skills);
for (const file of docs) {
  const text = withoutFences(readFileSync(file, "utf8"));
  for (const match of text.matchAll(/\.agents[\\/]skills[\\/]([a-z0-9]+(?:-[a-z0-9]+)*)(?![a-z0-9.-])/g)) {
    const slug = match[1];
    if (!known.has(slug)) fail(`${path.relative(ROOT, file)}: skill citada não existe: ${slug}`);
  }
}

for (const file of walkMarkdown(path.join(ROOT, "roadmap-rust", "issues"))) {
  const text = readFileSync(file, "utf8");
  for (const name of RETIRED) {
    if (new RegExp(`\\b${name}\\b`).test(text)) {
      fail(`${path.relative(ROOT, file)}: issue cita skill removida: ${name}`);
    }
  }
}

if (problems.length > 0) {
  console.error(`skills-check: ${problems.length} problema(s)`);
  for (const problem of problems) console.error(`  - ${problem}`);
  process.exit(1);
}

console.log(
  `skills-check ....................... ${skills.length} SKILL.md, ${skillLines} linhas, ` +
    `${skillBytes} bytes, ${docs.length} docs, 0 problemas`,
);
