# Qualidade de dados do Postgres

Auditoria de texto (06/11/2026) e o que passou a ser garantido por código. Medido no banco de
desenvolvimento (`postgres:17-alpine`, `server_encoding = UTF8`), com `scripts/check-data-quality.sql`.

## O que foi medido

| Medida | Resultado |
|---|---|
| Tabelas no schema `public` | 50 |
| `item_catalog` | 3.220 linhas |
| Fora de NFC (forma decomposta) | **0** |
| Espaço nas pontas ou espaço duplo | **0** |
| Caracteres de controle (C0/C1) | **0** |
| Nome vazio (`''` onde é identificador) | **0** |
| "Mojibake" pela primeira versão da consulta | 37 — **falso positivo** |

Os 37 casos eram `Cajado de Âmbar` e `Baú do Âmago`: **português correto**. A heurística original
marcava qualquer `Â` seguido de letra; a versão atual só marca **sequências completas** de duplo
encoding (`Ã©`, `Ã§`, `â€”`, `Âº`…). Conclusão: o dado está limpo, o detector estava errado — e um
detector errado é pior que nenhum, porque treina a equipe a ignorar o alerta.

## Por que normalizar mesmo com o dado limpo

O banco é alimentado por três caminhos, e nenhum deles garante forma sozinho:

1. **Importação** de `Release/` (`tmserver/cmd/importcontent`), onde os CSVs são Windows-1252.
2. **Seeds** versionados (`internal/migrations/seed_*.up.sql`), regerados por `seedgen`.
3. **Portal**, que grava texto digitado por moderador.

`internal/textenc.ToUTF8` cobre (1) na conversão de codificação — mas UTF-8 válido **não** implica NFC
nem texto sem espaço invisível. Daí o pacote novo.

## O que foi implementado

| Peça | Papel |
|---|---|
| `internal/textnorm` | `Clean`: reparo de duplo encoding → NFC → remoção de controle → colapso de espaço |
| `importcontent` | `textnorm.Clean`/`Slice` em nome de item, campos, nome de skill e nome de template |
| `0002_normalize_text.up.sql` | saneia o que já está gravado nas tabelas de conteúdo (idempotente) |
| `scripts/check-data-quality.sql` | auditoria reexecutável, sem alterar nada |

Decisões que valem registro:

- **NFC, não NFKC.** NFKC é *lossy* (funde ligaduras, larguras, sobrescritos) e mudaria o que o cliente
  desenha. NFC preserva o texto e resolve a equivalência canônica
  ([UAX #15](https://www.unicode.org/reports/tr15), [EDB: normalização no PG 13+](https://www.enterprisedb.com/blog/unicode-normalization-postgresql-13)).
- **Reparo de duplo encoding é conservador.** Só quando há marcador `Ã`/`Â` **e** os bytes
  reinterpretados formam UTF-8 válido. É o mesmo raciocínio do `ftfy`: reparar enquanto os bytes
  originais existem; depois de U+FFFD, não há o que recuperar.
- **U+FFFD não é removido.** Apagar esconderia a evidência de dano de codificação.
- **Texto de jogador fica fora.** `account`, `character`, `guild` e auditoria não passam pela
  migration: normalizar ali mudaria o que a pessoa digitou. A validação desses campos é regra de
  entrada (tamanho, charset), não saneamento retroativo.
- **Sem CHECK CONSTRAINT ainda.** `CHECK (col IS NFC NORMALIZED)` existe no PG 13+, mas só faz sentido
  depois de uma rodada com dados reais de produção; hoje a garantia é importador + migration.

## Como reexecutar

```bash
docker compose exec -T db psql -U postgres -d postgres < scripts/check-data-quality.sql
go test ./internal/textnorm/ -v
```

Esperado: `OK: nenhuma coluna de texto com problema` e todos os testes verdes. Se aparecer
`PROBLEMA`, a consulta diz tabela e coluna — a correção é importador/reparo, não `UPDATE` manual.

## O que continua aberto

- Ampliar a auditoria para **duplicidade** (mesmo item com espaço/acento diferente) e para
  **integridade referencial** de `npc_shop_item.item_index` contra `item_catalog.index` — hoje o gate
  existente é o `importcontent -check`.
- Rodar a auditoria no banco com conteúdo de produção e, se ficar em zero por duas rodadas, promover
  as constraints de NFC/trim nas colunas de nome.
- **Texto de jogador fica fora.**: estamos em fase de teste, pode matar o banco e subir tudo do zero, importante é garantir que tudo esteja corretamente normalizado.