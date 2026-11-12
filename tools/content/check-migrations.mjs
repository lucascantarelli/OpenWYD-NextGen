#!/usr/bin/env node
/**
 * Valida a série de migrations **sem Postgres**: ordem, FKs, duplicidade e `down`.
 *
 * O que ele NÃO faz: julgar SQL. Ele pega o que quebra a aplicação em série — arquivo fora de
 * ordem, tabela criada duas vezes, FK apontando para tabela que ainda não existe, `down` que
 * esquece uma tabela, e `INSERT` em tabela inexistente.
 *
 * Uso: node tools/content/check-migrations.mjs
 * Saída: uma linha por problema e o resumo; exit 1 se houver problema.
 */
import { readdirSync, readFileSync } from "node:fs";
import path from "node:path";

const ROOT = path.resolve(import.meta.dirname, "..", "..");
const DIR = path.join(ROOT, "crates", "wyd-db", "migrations");

const files = readdirSync(DIR)
  .filter((f) => f.endsWith(".sql"))
  .sort();

const problems = [];
const created = new Map(); // table -> file
const droppedInDown = new Map(); // file -> Set
const inserts = [];

let lastNumber = 0;
for (const file of files) {
  const number = Number(file.slice(0, 4));
  if (!Number.isFinite(number)) {
    problems.push(`${file}: nome não começa com número de 4 dígitos`);
    continue;
  }
  if (number <= lastNumber) {
    problems.push(`${file}: número ${number} não é maior que o anterior (${lastNumber})`);
  }
  lastNumber = Math.max(lastNumber, number);

  const sql = readFileSync(path.join(DIR, file), "utf8");
  if (!sql.includes("-- migrate:up")) problems.push(`${file}: sem bloco -- migrate:up`);
  if (!sql.includes("-- migrate:down")) problems.push(`${file}: sem bloco -- migrate:down`);

  // O cabeçalho dos arquivos cita "-- migrate:down" na frase da convenção, então o bloco
  // real é sempre o ÚLTIMO marcador do arquivo, nunca o primeiro.
  const down = sql.slice(sql.lastIndexOf("-- migrate:down"));
  droppedInDown.set(file, new Set([...down.matchAll(/DROP TABLE (?:IF EXISTS )?([a-z_]+)/g)].map((m) => m[1])));

  for (const match of sql.matchAll(/CREATE TABLE (?:IF NOT EXISTS )?([a-z_]+)/g)) {
    const table = match[1];
    if (created.has(table)) {
      problems.push(`${file}: tabela ${table} já criada em ${created.get(table)}`);
    } else {
      created.set(table, file);
    }
  }

  for (const match of sql.matchAll(/INSERT INTO ([a-z_]+)/g)) {
    inserts.push({ file, table: match[1] });
  }
}

// FK: alvo tem de existir na mesma migration (auto-referência) ou em anterior.
for (const file of files) {
  if (!created.size) continue;
  const sql = readFileSync(path.join(DIR, file), "utf8");
  for (const match of sql.matchAll(/REFERENCES ([a-z_]+)\s*\(/g)) {
    const target = match[1];
    const owner = created.get(target);
    if (!owner) {
      problems.push(`${file}: FK para tabela inexistente ${target}`);
    } else if (owner > file) {
      problems.push(`${file}: FK para ${target} criada depois (${owner})`);
    }
  }
}

// Touta tabela criada precisa aparecer no down do mesmo arquivo.
for (const [table, file] of created) {
  if (!droppedInDown.get(file)?.has(table)) {
    problems.push(`${file}: down não derruba ${table}`);
  }
}

for (const { file, table } of inserts) {
  const owner = created.get(table);
  if (!owner) problems.push(`${file}: INSERT em tabela inexistente ${table}`);
  else if (owner > file) problems.push(`${file}: INSERT em ${table} antes de ser criada (${owner})`);
}

const tables = created.size;

// Gate de deriva: a contagem do código tem de bater com a contagem registrada nos documentos.
// Ao criar uma migration, atualize os números abaixo e os documentos citados (FASE-02-6).
const DOCUMENTED = { migrations: 48, tables: 86 };
if (files.length !== DOCUMENTED.migrations) {
  problems.push(`${files.length} migrations no código, ${DOCUMENTED.migrations} documentadas`);
}
if (tables !== DOCUMENTED.tables) {
  problems.push(`${tables} tabelas no código, ${DOCUMENTED.tables} documentadas`);
}

console.log(`migrations: ${files.length} | tabelas: ${tables} | INSERTs: ${inserts.length}`);
if (problems.length === 0) {
  console.log("ok: ordem, FKs, duplicidade e down consistentes");
} else {
  for (const problem of problems) console.log(`  ✗ ${problem}`);
  console.log(`\n${problems.length} problema(s)`);
  process.exitCode = 1;
}
