#!/usr/bin/env node
/**
 * Prova que o banco construído do zero é EXATAMENTE o catálogo documentado.
 *
 * Passos: container efêmero → schema público vazio → aplica todos os `up` → lê o schema real
 * (`pg_attribute`/`information_schema`) → compara com `docs/arquitetura/banco-catalogo.md`.
 *
 * O que é comparado: conjunto de tabelas, ordem/nome/tipo/nulidade das colunas e colunas da PK.
 * O que NÃO é comparado: default, comentário e índice — o documento os lista, mas compará-los
 * traria falso positivo de formatação (`nextval(...)` vs identity) sem ganho de segurança.
 *
 * Uso: node tools/content/verify-schema.mjs [--keep]
 * Exit 1 em qualquer divergência, com a tabela/coluna divergente na mensagem.
 */
import { readFileSync, readdirSync } from 'node:fs';
import path from 'node:path';

import { blocos, parseMigrations } from './banco-parse.mjs';
import { contexto, docker, ensureContainer, psql, resetSchema } from './pg.mjs';

const ROOT = path.resolve(import.meta.dirname, '..', '..');
const DIR = path.join(ROOT, 'crates', 'wyd-db', 'migrations');
const CATALOGO = path.join(ROOT, 'docs', 'arquitetura', 'banco-catalogo.md');
const keep = process.argv.includes('--keep');

function parseCatalogo(markdown) {
  const tabelas = new Map();
  const inicio = markdown.indexOf('## 2. Catálogo');
  const fim = markdown.indexOf('## 3. Decisões provisórias');
  if (inicio < 0 || fim < 0) throw new Error('catálogo sem seção 2/3');
  let atual = null;
  for (const linha of markdown.slice(inicio, fim).split('\n')) {
    const titulo = /^### `([a-z_]+)`$/.exec(linha.trim());
    if (titulo) {
      atual = { nome: titulo[1], colunas: [], pk: [] };
      tabelas.set(atual.nome, atual);
      continue;
    }
    if (!atual) continue;
    const coluna = /^\| `([a-z_0-9]+)` \| `([^`]+)` \| (sim|não) \|$/.exec(linha.trim());
    if (coluna) {
      atual.colunas.push({ nome: coluna[1], tipo: coluna[2], nulo: coluna[3] === 'sim' });
      continue;
    }
    const pk = /^- \*\*PK:\*\* (.+)$/.exec(linha.trim());
    if (pk) {
      atual.pk = [...pk[1].matchAll(/`([a-z_0-9]+)`/g)].map((m) => m[1]);
    }
  }
  return tabelas;
}

function lerSchemaReal() {
  const ctx = contexto();
  ensureContainer(ctx);
  resetSchema(ctx);

  const arquivos = readdirSync(DIR).filter((f) => f.endsWith('.sql')).sort();
  for (const arquivo of arquivos) {
    const { up } = blocos(readFileSync(path.join(DIR, arquivo), 'utf8'), arquivo);
    psql(ctx, up, `up ${arquivo}`);
  }

  const colunas = new Map();
  const bruto = psql(
    ctx,
    // Nulidade vem como 'sim'/'não' do próprio SQL: `boolean::text` devolve 'true'/'false'
    // (o 't'/'f' é só formato de exibição do psql) e comparar com 't' reprovava tudo.
    `SELECT c.relname, a.attname, format_type(a.atttypid, a.atttypmod),
            CASE WHEN a.attnotnull THEN 'não' ELSE 'sim' END
       FROM pg_class c
       JOIN pg_namespace n ON n.oid = c.relnamespace
       JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
      WHERE n.nspname = 'public' AND c.relkind = 'r'
      ORDER BY c.relname, a.attnum;`,
    'leitura de pg_attribute',
  );
  for (const linha of bruto.split('\n').filter(Boolean)) {
    const [tabela, coluna, tipo, nulidade] = linha.split('|');
    if (!colunas.has(tabela)) colunas.set(tabela, []);
    colunas.get(tabela).push({ nome: coluna, tipo, nulo: nulidade === 'sim' });
  }

  const pk = new Map();
  const brutoPk = psql(
    ctx,
    `SELECT tc.table_name, kcu.column_name
       FROM information_schema.table_constraints tc
       JOIN information_schema.key_column_usage kcu
         ON kcu.constraint_name = tc.constraint_name AND kcu.table_schema = tc.table_schema
      WHERE tc.constraint_type = 'PRIMARY KEY' AND tc.table_schema = 'public'
      ORDER BY tc.table_name, kcu.ordinal_position;`,
    'leitura das PKs',
  );
  for (const linha of brutoPk.split('\n').filter(Boolean)) {
    const [tabela, coluna] = linha.split('|');
    if (!pk.has(tabela)) pk.set(tabela, []);
    pk.get(tabela).push(coluna);
  }
  return { ctx, colunas, pk, arquivos };
}

function comparar(documentado, real) {
  const problemas = [];

  for (const tabela of documentado.keys()) {
    if (!real.colunas.has(tabela)) problemas.push(`tabela documentada e ausente no banco: ${tabela}`);
  }
  for (const tabela of real.colunas.keys()) {
    if (!documentado.has(tabela)) problemas.push(`tabela do banco fora do catálogo: ${tabela}`);
  }

  for (const [tabela, doc] of documentado) {
    const banco = real.colunas.get(tabela);
    if (!banco) continue;
    const nomesDoc = doc.colunas.map((c) => c.nome);
    const nomesBanco = banco.map((c) => c.nome);
    if (nomesDoc.join(',') !== nomesBanco.join(',')) {
      problemas.push(
        `${tabela}: colunas divergem\n    catálogo: ${nomesDoc.join(', ')}\n    banco:    ${nomesBanco.join(', ')}`,
      );
      continue;
    }
    for (const [i, coluna] of doc.colunas.entries()) {
      const vivo = banco[i];
      if (coluna.tipo !== vivo.tipo) {
        problemas.push(`${tabela}.${coluna.nome}: tipo ${coluna.tipo} no catálogo, ${vivo.tipo} no banco`);
      }
      if (coluna.nulo !== vivo.nulo) {
        problemas.push(
          `${tabela}.${coluna.nome}: nulidade ${coluna.nulo ? 'sim' : 'não'} no catálogo, ${vivo.nulo ? 'sim' : 'não'} no banco`,
        );
      }
    }
    const pkDoc = [...doc.pk].sort().join(',');
    const pkBanco = [...(real.pk.get(tabela) ?? [])].sort().join(',');
    if (pkDoc !== pkBanco) {
      problemas.push(`${tabela}: PK ${pkDoc || '—'} no catálogo, ${pkBanco || '—'} no banco`);
    }
  }
  return problemas;
}

try {
  const documento = parseCatalogo(readFileSync(CATALOGO, 'utf8'));
  const modelo = parseMigrations(DIR);
  if (documento.size !== modelo.tabelas.length) {
    console.error(`catálogo com ${documento.size} tabelas e migrations com ${modelo.tabelas.length}`);
    process.exitCode = 1;
  }
  const real = lerSchemaReal();
  const problemas = comparar(documento, real);
  const totalColunas = [...real.colunas.values()].reduce((s, c) => s + c.length, 0);

  console.log(
    `\nbanco: ${real.colunas.size} tabelas, ${totalColunas} colunas, ${real.pk.size} PKs | ` +
      `catálogo: ${documento.size} tabelas | migrations: ${real.arquivos.length}`,
  );
  if (problemas.length) {
    for (const problema of problemas) console.error(`  ✗ ${problema}`);
    console.error(`\n${problemas.length} divergência(s) entre o banco e o catálogo`);
    process.exitCode = 1;
  } else {
    console.log('ok: schema real == catálogo documentado (tabelas, colunas, tipos, nulidade e PK)');
  }
} catch (error) {
  console.error(`\nFALHOU: ${error.message}`);
  process.exitCode = 1;
} finally {
  if (!keep) {
    const { container } = contexto();
    const removido = docker(['rm', '-f', container]);
    if (removido.status === 0) console.log(`container: ${container} removido`);
  }
}
