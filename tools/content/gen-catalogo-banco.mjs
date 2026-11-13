#!/usr/bin/env node
/**
 * Gera `docs/arquitetura/banco-catalogo.md` a partir das migrations + curadoria de fontes.
 *
 * Por que gerar em vez de escrever à mão: 59 tabelas e ~600 colunas desatualizam em silêncio.
 * Aqui a migration é a fonte da verdade do schema, `banco-fontes.mjs` é a fonte da decisão, e
 * o documento é a leitura humana das duas. O `cargo test` prende doc ↔ SQL e o
 * `verify-schema.mjs` prende SQL ↔ Postgres real.
 *
 * Uso:
 *   node tools/content/gen-catalogo-banco.mjs           # escreve o documento
 *   node tools/content/gen-catalogo-banco.mjs --check   # falha se estiver desatualizado
 */
import { readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';

import { parseMigrations, ultimaVersao } from './banco-parse.mjs';
import { DECISOES, FONTES, LEGADO } from './banco-fontes.mjs';

const ROOT = path.resolve(import.meta.dirname, '..', '..');
const DIR = path.join(ROOT, 'crates', 'wyd-db', 'migrations');
const OUT = path.join(ROOT, 'docs', 'arquitetura', 'banco-catalogo.md');
const B = '`';

const modelo = parseMigrations(DIR);
const nomes = new Set(modelo.tabelas.map((t) => t.nome));

for (const tabela of modelo.tabelas) {
  if (!FONTES[tabela.nome]) {
    throw new Error(`tabela ${tabela.nome} (${tabela.arquivo}) sem entrada em banco-fontes.mjs`);
  }
}
for (const nome of Object.keys(FONTES)) {
  if (!nomes.has(nome)) throw new Error(`banco-fontes.mjs cita tabela inexistente: ${nome}`);
}

const porDominio = new Map();
for (const tabela of modelo.tabelas) {
  const dominio = FONTES[tabela.nome].dominio;
  if (!porDominio.has(dominio)) porDominio.set(dominio, []);
  porDominio.get(dominio).push(tabela.nome);
}

const provisorias = modelo.tabelas.filter((t) => FONTES[t.nome].estado === 'provisorio');
const colunas = modelo.tabelas.reduce((soma, t) => soma + t.colunas.length, 0);
const indicePorTabela = new Map();
for (const indice of modelo.indices) {
  if (!indicePorTabela.has(indice.tabela)) indicePorTabela.set(indice.tabela, []);
  indicePorTabela.get(indice.tabela).push(indice);
}

const linhas = [];
const push = (...itens) => linhas.push(...itens);

push(
  '# Catálogo do banco (`wyd-db`) — FASE-04',
  '',
  '> **GERADO.** Não edite à mão: a fonte da verdade do schema é',
  '> `crates/wyd-db/migrations/*.sql`; a fonte de cada decisão é `tools/content/banco-fontes.mjs`.',
  '> Regenere com `make banco-catalogo`.',
  '>',
  '> Duas verificações prendem este documento ao banco:',
  '> 1. `cargo test -p wyd-db --test contrato_catalogo` — documento ↔ migrations, sem Postgres;',
  '> 2. `node tools/content/verify-schema.mjs` — `information_schema` de um Postgres real',
  '>    aplicado do zero ↔ documento (tabela/coluna/tipo/nulidade/PK, sem sobra nem falta).',
  '',
  '## 1. Resumo',
  '',
  `- migrations: **${modelo.arquivos.length}** · última versão: **${ultimaVersao(modelo)}**`,
  `- tabelas: **${modelo.tabelas.length}** · colunas: **${colunas}** · índices: **${modelo.indices.length}** · funções: **${modelo.funcoes.length}**`,
  `- domínios: **${porDominio.size}** · tabelas com dado provisório: **${provisorias.length}**`,
  '',
  '| Domínio | Tabelas |',
  '|---|---|',
  ...[...porDominio.entries()]
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([dominio, tabelas]) => `| ${dominio} | ${tabelas.map((t) => `${B}${t}${B}`).join(', ')} |`),
  '',
  '## 2. Catálogo',
  '',
);

for (const tabela of modelo.tabelas) {
  const fonte = FONTES[tabela.nome];
  const fks = tabela.fks.map((fk) => `${B}${fk.de}${B} → ${B}${fk.tabela}${B}`);
  const indices = (indicePorTabela.get(tabela.nome) ?? []).map(
    (i) =>
      `${B}${i.nome}${B} (${i.colunas})${i.unico ? ' UNIQUE' : ''}${i.metodo !== 'btree' ? ` USING ${i.metodo}` : ''}${i.predicado ? ` ${i.predicado}` : ''}`,
  );
  push(
    `### ${B}${tabela.nome}${B}`,
    '',
    `- **Arquivo:** ${B}${tabela.arquivo}${B} · **Domínio:** ${fonte.dominio} · **Estado:** ${fonte.estado}`,
    `- **Fonte:** ${fonte.fonte}`,
    `- **Nota:** ${fonte.nota}`,
    `- **PK:** ${tabela.pk.length ? tabela.pk.map((c) => `${B}${c}${B}`).join(', ') : '—'}`,
    `- **FK:** ${fks.length ? fks.join(' · ') : '—'}`,
    `- **CHECK:** ${tabela.checks.length ? tabela.checks.map((c) => `${B}${c}${B}`).join(', ') : '—'}`,
    `- **UNIQUE de tabela:** ${tabela.unicidade.length ? tabela.unicidade.join(' · ') : '—'}`,
    `- **Índices:** ${indices.length ? indices.join(' · ') : '—'}`,
    '',
    '| Coluna | Tipo | Nulo |',
    '|---|---|---|',
    ...tabela.colunas.map((c) => `| ${B}${c.nome}${B} | ${B}${c.tipo}${B} | ${c.nulo ? 'sim' : 'não'} |`),
    '',
  );
}

push(
  '## 3. Decisões provisórias (D-Bxx)',
  '',
  'Mesmo padrão do registro `D-xx` de `docs/arquitetura/matematica-sim.md`: decisão, escolha,',
  'por quê e como confirmar. `trace` vazio significa decisão de organização, não de semântica.',
  '',
  '| ID | Decisão | Escolha | Por quê | Como confirmar |',
  '|---|---|---|---|---|',
  ...DECISOES.map((d) => `| ${d.id} | ${d.decisao} | ${d.escolha} | ${d.porque} | ${d.trace} |`),
  '',
  '## 4. Mapa legado → novo',
  '',
  'Tabelas do schema Go antigo citadas em `modelo-banco.md` §6.1 e onde elas foram parar. O que',
  'não virou tabela está declarado como pendência com ID, nunca sumiu em silêncio.',
  '',
  '| Tabela legada | Destino |',
  '|---|---|',
  ...LEGADO.map(([legado, destino]) => `| ${B}${legado}${B} | ${destino} |`),
  '',
  '## 5. Fila de migrations',
  '',
  '| # | Arquivo | Tipo | Objetos |',
  '|---|---|---|---|',
  ...modelo.arquivos.map((arquivo) => {
    const doArquivo = modelo.tabelas.filter((t) => t.arquivo === arquivo);
    const funcoes = modelo.funcoes.filter((f) => f.arquivo === arquivo).map((f) => `${B}${f.nome}()${B}`);
    const objetos = [...doArquivo.map((t) => `${B}${t.nome}${B}`), ...funcoes];
    // Tipo vem do nome: arquivo de seed não cria tabela nenhuma, então `doArquivo` ficaria
    // vazio e todo seed apareceria como DDL.
    const tipo = arquivo.includes('_seed_') ? 'seed' : 'DDL';
    return `| ${arquivo.slice(0, 4)} | ${B}${arquivo}${B} | ${tipo} | ${objetos.length ? objetos.join(', ') : '—'} |`;
  }),
  '',
  'Regra da série: DDL inteiro antes do primeiro seed, um domínio por arquivo, nenhuma tabela',
  'partida entre arquivos, nenhuma FK para tabela criada depois e **nenhum `ALTER TABLE` no',
  '`up`** — coluna nova nasce na criação da tabela (o gerador falha se aparecer um).',
  '',
);

const texto = `${linhas.join('\n')}`;
const existente = (() => {
  try {
    return readFileSync(OUT, 'utf8');
  } catch {
    return null;
  }
})();

if (process.argv.includes('--check')) {
  if (existente !== texto) {
    console.error(`catálogo desatualizado: rode node tools/content/gen-catalogo-banco.mjs`);
    process.exitCode = 1;
  } else {
    console.log(`ok: catálogo em dia (${modelo.tabelas.length} tabelas, ${modelo.arquivos.length} migrations)`);
  }
} else {
  writeFileSync(OUT, texto);
  console.log(
    `catálogo: ${path.relative(ROOT, OUT)} — ${modelo.tabelas.length} tabelas, ${colunas} colunas, ${modelo.indices.length} índices, ${modelo.arquivos.length} migrations`,
  );
}
