#!/usr/bin/env node
/**
 * Gate de consistência documental (PROC-13).
 *
 * Verifica, sem dependências:
 * - membros de `Cargo.toml` × diretórios com `crates/<nome>/Cargo.toml`;
 * - contagem e nomes das crates citados no README e na tabela do AGENTS;
 * - links relativos de README.md e AGENTS.md;
 * - conta issues ativas/migrations para o resumo do gate.
 *
 * Uso: node tools/dev/check-docs-consistency.mjs
 * Exit 1 com a lista de divergências; nada é corrigido automaticamente.
 */
import { existsSync, readFileSync, readdirSync } from "node:fs";
import path from "node:path";

const problems = [];
const fail = (message) => problems.push(message);
const read = (file) => readFileSync(file, "utf8");
const uniq = (values) => [...new Set(values)].sort();

// 1. Workspace × árvore de crates.
const cargo = read("Cargo.toml");
const membersBlock = cargo.match(/members\s*=\s*\[([\s\S]*?)\]/);
if (!membersBlock) {
  fail("Cargo.toml: bloco [workspace].members não encontrado");
}
const members = membersBlock
  ? [...membersBlock[1].matchAll(/"([^"]+)"/g)].map((m) => m[1])
  : [];
for (const member of members) {
  if (!existsSync(member)) fail(`Cargo.toml: membro inexistente (${member})`);
}
const memberCrates = uniq(
  members
    .filter((member) => member.startsWith("crates/"))
    .map((member) => path.basename(member)),
);
const crateDirs = uniq(
  readdirSync("crates", { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && existsSync(path.join("crates", entry.name, "Cargo.toml")))
    .map((entry) => entry.name),
);

for (const name of crateDirs) {
  if (!memberCrates.includes(name)) fail(`crates/${name}: diretório com Cargo.toml fora do workspace`);
}
for (const name of memberCrates) {
  if (!crateDirs.includes(name)) fail(`crates/${name}: membro do workspace sem Cargo.toml`);
}

// 2. README: frase de contagem + árvore de crates.
const readme = read("README.md");
const readmeCount = readme.match(/compila as (\d+) crates/);
if (!readmeCount) {
  fail("README.md: frase 'compila as N crates' não encontrada");
} else if (Number(readmeCount[1]) !== memberCrates.length) {
  fail(`README.md: diz ${readmeCount[1]} crates, workspace tem ${memberCrates.length}`);
}

const treeBlock = [...readme.matchAll(/```[\s\S]*?```/g)]
  .map((match) => match[0])
  .find((block) => block.includes("crates/"));
if (!treeBlock) {
  fail("README.md: bloco de árvore com crates/ não encontrado");
} else {
  const treeCrates = uniq(
    [...treeBlock.matchAll(/^\s*│\s+[├└]── (wyd-[a-z0-9-]+)\//gm)].map((match) => match[1]),
  );
  for (const name of treeCrates) {
    if (!memberCrates.includes(name)) fail(`README.md: crate na árvore fora do workspace (${name})`);
  }
  for (const name of memberCrates) {
    if (!treeCrates.includes(name)) fail(`README.md: crate do workspace ausente na árvore (${name})`);
  }
}

// 3. AGENTS: tabela de crates.
const agents = read("AGENTS.md");
const agentsCrates = uniq(
  [...agents.matchAll(/^\|\s*`crates\/([a-z0-9-]+)\/`[^|]*\|/gm)].map((match) => match[1]),
);
for (const name of agentsCrates) {
  if (!memberCrates.includes(name)) fail(`AGENTS.md: crate na tabela fora do workspace (${name})`);
}
for (const name of memberCrates) {
  if (!agentsCrates.includes(name)) fail(`AGENTS.md: crate do workspace ausente na tabela (${name})`);
}

// 4. Links relativos de README/AGENTS + toda a documentação viva.
const markdownFiles = (directory) => {
  if (!existsSync(directory)) return [];
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const child = path.join(directory, entry.name);
    if (entry.isDirectory()) return markdownFiles(child);
    return entry.isFile() && entry.name.endsWith(".md") ? [child] : [];
  });
};
const linkFiles = [
  ...new Set(["README.md", "AGENTS.md", ...markdownFiles("docs"), ...markdownFiles("roadmap-rust")]),
].sort();

const linkTargets = (file) => {
  const text = read(file).replace(/```[\s\S]*?```/g, "");
  return [...text.matchAll(/\]\(([^)]+)\)/g)]
    .map((match) => match[1].trim())
    .filter((target) => target && !target.startsWith("#") && !/^(https?:|mailto:|tel:)/i.test(target))
    .map((target) => target.replace(/#.*$/, "").replace(/\?.*$/, ""))
    .filter((target) => target && !target.includes("<") && !target.includes(">") && !target.includes("*") && !target.startsWith("/"));
};

let linkCount = 0;
for (const file of linkFiles) {
  for (const target of linkTargets(file)) {
    linkCount += 1;
    let decoded;
    try {
      decoded = decodeURIComponent(target);
    } catch {
      fail(`${file}: link com escape inválido (${target})`);
      continue;
    }
    const resolved = path.resolve(path.dirname(file), decoded);
    if (!existsSync(resolved)) fail(`${file}: link relativo não resolve (${target})`);
  }
}

// 5. Resumo medido, sem substituir os gates específicos de migrations/status.
const issues = readdirSync("roadmap-rust/issues").filter(
  (file) => file.endsWith(".md") && !/^FASE-\d\d\.md$/.test(file),
);
const migrations = readdirSync("crates/wyd-db/migrations").filter((file) => file.endsWith(".sql"));

if (problems.length) {
  console.error(`docs: ${problems.length} divergência(s) | ${memberCrates.length} crates | ${linkFiles.length} docs | ${linkCount} links`);
  for (const problem of problems) console.error(`  ✗ ${problem}`);
  process.exit(1);
}

console.log(
  `docs: ${issues.length} issues, ${memberCrates.length} crates, ${members.length} membros, ${migrations.length} migrations, ${linkFiles.length} docs, ${linkCount} links — consistente`,
);
