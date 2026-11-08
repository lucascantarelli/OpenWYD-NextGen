/**
 * Parser da série de migrations — uma leitura só, usada pelo gerador do catálogo e pela
 * verificação de `information_schema`.
 *
 * Ele não é um parser de SQL genérico: lê o **estilo** que a série usa (`CREATE TABLE x (`
 * numa linha, `);` sozinho no fim, elementos separados por vírgula de nível zero). Fora desse
 * estilo falha alto, em vez de adivinhar — migration fora do padrão quebraria o contrato do
 * catálogo em silêncio.
 *
 * Por que fatiar por vírgula de nível zero e não por linha: constraint longa (`CHECK (x IN (`)
 * quebra em várias linhas, e um leitor linha-a-linha trataria `'mortal', 'god'` como coluna.
 */

import { readFileSync, readdirSync } from 'node:fs';
import path from 'node:path';

/** Tipo SQL → nome que `format_type()` devolve no Postgres (comparação byte-a-byte). */
const TIPOS = {
  BIGSERIAL: 'bigint',
  BIGINT: 'bigint',
  SMALLSERIAL: 'smallint',
  SMALLINT: 'smallint',
  SERIAL: 'integer',
  INTEGER: 'integer',
  TEXT: 'text',
  BOOLEAN: 'boolean',
  TIMESTAMPTZ: 'timestamp with time zone',
  TIME: 'time without time zone',
  DATE: 'date',
  JSONB: 'jsonb',
  JSON: 'json',
  BYTEA: 'bytea',
  REAL: 'real',
  NUMERIC: 'numeric',
};

const MODIFICADORES =
  /\s+(?:GENERATED|NOT\s+NULL|NULL|DEFAULT|PRIMARY\s+KEY|REFERENCES|UNIQUE|CHECK|CONSTRAINT)\b/;

function normalizarTipo(bruto) {
  const tipo = bruto.trim().replace(/\s+/g, ' ');
  const array = tipo.endsWith('[]');
  const base = (array ? tipo.slice(0, -2) : tipo).toUpperCase();
  const nome = TIPOS[base];
  if (!nome) throw new Error(`tipo SQL não mapeado: ${bruto}`);
  return array ? `${nome}[]` : nome;
}

/** Remove comentário de linha preservando o resto (o `--` pode citar vírgula/parentese). */
const semComentarios = (texto) => texto.replace(/--[^\n]*/g, '');

/** Separa o corpo da tabela em elementos, ignorando vírgula dentro de `()` e de `'...'`. */
export function elementosDoCorpo(corpo) {
  const texto = semComentarios(corpo);
  const partes = [];
  let atual = '';
  let profundidade = 0;
  let aspas = false;
  for (const ch of texto) {
    if (ch === "'") aspas = !aspas;
    if (!aspas) {
      if (ch === '(') profundidade += 1;
      else if (ch === ')') profundidade -= 1;
      else if (ch === ',' && profundidade === 0) {
        partes.push(atual);
        atual = '';
        continue;
      }
    }
    atual += ch;
  }
  if (atual.trim()) partes.push(atual);
  return partes.map((p) => p.replace(/\s+/g, ' ').trim()).filter(Boolean);
}

const ehConstraint = (elemento) =>
  /^(CONSTRAINT|PRIMARY KEY|UNIQUE|FOREIGN KEY|CHECK)\b/.test(elemento);

export function parseColuna(texto) {
  const m = /^([a-z_][a-z0-9_]*)\s+/.exec(texto);
  if (!m) throw new Error(`elemento de coluna não reconhecido: ${texto}`);
  const nome = m[1];
  const resto = texto.slice(m[0].length);
  const corte = resto.search(MODIFICADORES);
  const tipo = normalizarTipo(corte >= 0 ? resto.slice(0, corte) : resto);
  const referencia = /REFERENCES\s+([a-z_]+)\s*(?:\(([^)]*)\))?/.exec(resto);
  return {
    nome,
    tipo,
    nulo: !/NOT\s+NULL/.test(resto) && !/PRIMARY\s+KEY/.test(resto),
    pk: /PRIMARY\s+KEY/.test(resto),
    gerado: /GENERATED/.test(resto),
    fk: referencia ? { tabela: referencia[1], colunas: referencia[2] ?? '' } : null,
  };
}

/** Separa up/down pelo marcador que ocupa a linha inteira (o cabeçalho cita os dois). */
export function blocos(sql, arquivo) {
  const ups = [...sql.matchAll(/^-- migrate:up\s*$/gm)];
  const downs = [...sql.matchAll(/^-- migrate:down\s*$/gm)];
  if (ups.length === 0 || downs.length === 0) {
    throw new Error(`${arquivo}: sem linha -- migrate:up / -- migrate:down`);
  }
  const inicio = ups[0].index + ups[0][0].length;
  const fim = downs[downs.length - 1].index;
  if (fim < inicio) throw new Error(`${arquivo}: down antes do up`);
  return { up: sql.slice(inicio, fim), down: sql.slice(fim) };
}

const normalizar = (s) => s.replace(/\s+/g, ' ').trim();

export function parseMigrations(dir) {
  const arquivos = readdirSync(dir)
    .filter((f) => f.endsWith('.sql'))
    .sort();
  const tabelas = [];
  const funcoes = [];
  const indices = [];

  for (const arquivo of arquivos) {
    const { up } = blocos(readFileSync(path.join(dir, arquivo), 'utf8'), arquivo);

    // Regra da série: DDL nasce na criação da tabela. `ALTER TABLE` no up é desenho
    // corrigido por cima — se aparecer, a série está errada e o catálogo não deve fingir.
    if (/^ALTER TABLE/m.test(up)) {
      throw new Error(`${arquivo}: ALTER TABLE no up — consolide a coluna na criação da tabela`);
    }

    for (const inicio of up.matchAll(/^CREATE TABLE ([a-z_][a-z0-9_]*) \($/gm)) {
      const nome = inicio[1];
      const corpoInicio = inicio.index + inicio[0].length;
      const corpoFim = up.indexOf('\n);', corpoInicio);
      if (corpoFim < 0) throw new Error(`${arquivo}: CREATE TABLE ${nome} sem ");"`);
      const elementos = elementosDoCorpo(up.slice(corpoInicio, corpoFim));

      const colunas = elementos.filter((e) => !ehConstraint(e)).map(parseColuna);
      const constraints = elementos.filter(ehConstraint);
      const pk = [
        ...colunas.filter((c) => c.pk).map((c) => c.nome),
        ...constraints.flatMap((c) => {
          const m = /^PRIMARY KEY \(([^)]*)\)/.exec(c);
          return m ? m[1].split(',').map((x) => x.trim()) : [];
        }),
      ];
      const fks = [
        ...colunas.filter((c) => c.fk).map((c) => ({ de: c.nome, ...c.fk })),
        ...constraints.flatMap((c) => {
          const m = /^FOREIGN KEY \(([^)]*)\) REFERENCES ([a-z_]+)/.exec(c);
          return m ? [{ de: m[1].trim(), tabela: m[2], colunas: '' }] : [];
        }),
      ];
      const checks = constraints
        .filter((c) => c.startsWith('CONSTRAINT') && c.includes('CHECK'))
        .map((c) => /^CONSTRAINT ([a-z_]+)/.exec(c)?.[1] ?? c);

      // Coluna de PK é NOT NULL no Postgres mesmo sem a palavra na definição; o catálogo
      // descreve o schema que existe, não o texto do SQL.
      for (const coluna of colunas) {
        if (pk.includes(coluna.nome)) coluna.nulo = false;
      }

      tabelas.push({
        nome,
        arquivo,
        colunas,
        pk,
        fks,
        checks,
        unicidade: constraints.filter((c) => c.startsWith('UNIQUE')).map(normalizar),
        seed: arquivo.includes('_seed_'),
      });
    }

    for (const linha of up.split('\n')) {
      const funcao = /^CREATE FUNCTION ([a-z_][a-z0-9_]*)/.exec(linha);
      if (funcao) funcoes.push({ nome: funcao[1], arquivo });

      const indice =
        /^CREATE (UNIQUE )?INDEX ([a-z_][a-z0-9_]*) ON ([a-z_][a-z0-9_]*)(?:\s+USING ([A-Z]+))?\s*\(([^)]*)\)\s*(.*)$/.exec(
          linha,
        );
      if (indice) {
        indices.push({
          nome: indice[2],
          tabela: indice[3],
          unico: Boolean(indice[1]),
          metodo: indice[4] ?? 'btree',
          colunas: normalizar(indice[5]),
          predicado: indice[6] ? normalizar(indice[6]) : '',
          arquivo,
        });
      }
    }
  }

  for (const indice of indices) {
    if (!tabelas.some((t) => t.nome === indice.tabela)) {
      throw new Error(`${indice.arquivo}: índice ${indice.nome} em tabela inexistente ${indice.tabela}`);
    }
  }

  return {
    arquivos,
    tabelas,
    funcoes,
    indices: indices.sort((a, b) => a.nome.localeCompare(b.nome)),
  };
}

/** Número da última migration — é ele que `SCHEMA_VERSION` deve declarar. */
export function ultimaVersao(modelo) {
  return Number(modelo.arquivos[modelo.arquivos.length - 1].slice(0, 4));
}
