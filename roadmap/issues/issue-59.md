# Issue 59 — Migrar as configs de conteúdo restantes para o banco (guard, level item, mob merc, game settings)

**Fase:** 7 · **Depende de:** 58 · **Esforço:** M · **Estado:** 🟡 fatia entregue em 06/11 (tabelas + importador + testes); falta rodar contra o Postgres vivo e o worldatlas ler `region`

## Por que

Quatro arquivos de `Release/` não têm **nenhum** leitor Go hoje (`guard_spawn`, `level_item`,
`mob_merc`, `gameconfig`) — mas eles são configuração de jogo, não asset, e continuam sendo a única
cópia do dado. Enquanto ficam em texto, o dia em que o sistema correspondente for portado ele nasce
lendo arquivo, e a migração precisa ser refeita. Guardar o dado **agora**, com import lossless, deixa
o arquivo dispensável e o sistema futuro nascendo no banco.

> Caveat explícito: criar a tabela não implementa o sistema. Guarda de cidade, item por nível e
> restock de mercador continuam sem comportamento até a issue de cada um; aqui só garantimos que o
> dado sai do arquivo sem perda.

## Tabelas

```sql
-- Guard.txt (STRUCT_NPC_GUARD, Server.cpp:149). 699 B.
CREATE TABLE guard_spawn (
    id            bigserial PRIMARY KEY,
    slug          text NOT NULL UNIQUE,          -- "<template>-<ordem>" estável no reimport
    template_name text NOT NULL,
    display_name  text NOT NULL DEFAULT '',
    pos_x         integer, pos_y integer,
    route_type    smallint NOT NULL DEFAULT 0,
    req_level     smallint NOT NULL DEFAULT 0,
    req_class     smallint NOT NULL DEFAULT 0,   -- bitmask de arquétipo
    req_quest     smallint NOT NULL DEFAULT -1,
    reward_item   integer  NOT NULL DEFAULT 0,
    reward_qty    smallint NOT NULL DEFAULT 0,
    effects       jsonb    NOT NULL DEFAULT '[]'::jsonb,
    enabled       boolean  NOT NULL DEFAULT true,
    raw           jsonb    NOT NULL DEFAULT '{}'::jsonb  -- linhas não modeladas, sem perda
);

-- LevelItem.txt (17 KB; DoItemLevel, DEFERIDO em captura-wyd-levelup.md).
CREATE TABLE level_item (
    level      smallint NOT NULL,
    arch       smallint NOT NULL,                -- -1 = todas as classes
    slot       smallint NOT NULL,                -- ordem do item na linha
    item_index integer  NOT NULL,
    quantity   smallint NOT NULL DEFAULT 1,
    effects    jsonb    NOT NULL DEFAULT '[]'::jsonb,
    PRIMARY KEY (level, arch, slot)
);

-- MobMerc.txt (STRUCT_MERC: RenewTime/RebornTime/Stock/MaxStock/GenerateIndex).
CREATE TABLE mob_merc (
    generator_index integer PRIMARY KEY,         -- bloco NPCGener, sem FK (índice não é único global)
    template_name   text,
    renew_seconds   integer NOT NULL DEFAULT 0,
    reborn_seconds  integer NOT NULL DEFAULT 0,
    stock           jsonb   NOT NULL DEFAULT '[]'::jsonb,  -- 27 slots exibidos
    max_stock       jsonb   NOT NULL DEFAULT '[]'::jsonb
);

-- gameconfig.txt: chave/valor; `double`, `deadpoint`, `partybonus`, `partydif`,
-- `DropItem`, `PotionDelay`, `maxNightmare`, `GTorreHour`/`RVRHour`/`BRHour`, `kefrastatus`.
CREATE TABLE game_setting (
    key        text PRIMARY KEY,
    value_int  bigint,
    value_text text,
    kind       text NOT NULL DEFAULT 'int'       -- int|bool|string|matrix
);

-- Matriz 8×8 de bônus de drop do gameconfig (lida pelo loot quando o sistema chegar).
CREATE TABLE game_drop_bonus (
    party_diff smallint NOT NULL,
    level_diff smallint NOT NULL,
    pct        smallint NOT NULL,
    PRIMARY KEY (party_diff, level_diff)
);

-- Regions.txt (52 retângulos). Hoje só o worldatlas lê; P1 desta issue por ser barato.
CREATE TABLE region (
    name text PRIMARY KEY,
    x1 integer, y1 integer, x2 integer, y2 integer,
    kind text NOT NULL DEFAULT '',               -- 'pvp' | 'dungeon' | 'cidade' | 'kefra' | ...
    source text NOT NULL DEFAULT 'Regions.txt'
);
```

## Importação (`dbserver import-content --sets=...`)

- Parsers portados do legado com paridade de campo: `CReadFiles::ReadMobMerc` (blocos começam em `#`;
  `ParseMobMercString`), leitura de `Guard.txt` (`Server.cpp:149`), `LevelItem.txt` e
  `gameconfig.txt` (`Server.cpp:1115-1342`, 8 linhas fixas + matriz 8×8).
- Upsert idempotente por chave natural (`slug`, `(level,arch,slot)`, `generator_index`, `key`,
  `(party_diff,level_diff)`, `name`), `ON CONFLICT DO NOTHING` no seed e `DO UPDATE` no reimport
  manual.
- `--check` compara o banco com o arquivo e sai 1 se divergir (gate de CI).
- Chave/token desconhecido em `gameconfig` **não** é descartado: grava em `game_setting` com
  `kind='string'` e o token cru, para o importador ser lossless.

## Ordem das migrations (padrão da issue 58)

```
0044_guard_spawn.up / 0045_guard_spawn_seed.up / 0046_guard_spawn_constraints.up
0047_level_item…     0050_mob_merc…      0053_game_setting + game_drop_bonus…
0056_region…
```

## Entregue em 06/11/2026

| Peça | Onde | Prova |
|---|---|---|
| Tabelas `guard_spawn`, `level_item`, `mob_merc`, `game_setting`, `game_drop_bonus`, `region` + `content_source_file` | `internal/migrations/0008_platform_content.up.sql` | migração aplicada no `go test` do store? não: sem banco neste host — ver `test/issue-59.md` |
| Parsers lossless | `tmserver/internal/content/settings.go` | `go test -race ./tmserver/internal/content/` ok |
| Upserts idempotentes | `internal/store/contentsettings.go` | compila e `go vet` ok; escrita provada só com banco |
| Conjuntos no importador | `tmserver/cmd/importcontent/main.go` (`guard,level-item,mob-merc,game-settings,regions`, `all`, `-check` com hash) | dry-run: `guards=5 level_items=318 mob_mercs=1 settings=26 drop_matrix=64 regions=52` |
| Golden de paridade | `tmserver/internal/content/settings_test.go` | fixa 5/318/1/26/64/52 e a forma 4×16 da matriz |

Decisões que a medição forçou (e que corrigem o texto original desta issue):

- `level_item` não pode ter `(level, arch, slot)` como chave: o arquivo tem 318 linhas com 159 pares
  `classe/seq` distintos (blocos repetidos, linha 13 = linha 256). A chave é o **ordinal** da linha.
- A matriz de drop é **4×16** (64 células), não 8×8.
- `MobMerc.txt` está em `Common/Settings/`, não em `TMsrv/run/`.
- `Guard.txt` declara 13 campos no cabeçalho e traz 8 por regra; por isso `rules` é jsonb + `raw`, em vez
  de colunas `req_*` inventadas.

## O que ainda falta

1. Rodar `importcontent -dsn` contra o Postgres e validar contagens + `-check` (comando e critério em
   [`test/issue-59.md`](../test/issue-59.md), porque este host não sobe container).
2. ~~`worldatlas` passar a ler `region` em vez de `Regions.txt`~~ — **feito em 06/11**: o atlas consulta
   `SELECT name, x1, y1, x2, y2 FROM region ORDER BY name`, o parser de arquivo ficou só como referência
   de paridade, e o `source_hash` passou a cobrir as regiões do banco (editar a tabela invalida o cache
   do portal). Prova offline: `go test -race ./tmserver/cmd/worldatlas/` (paridade arquivo × importador,
   mapeamento puro e teste de banco gated por `W2PP_TEST_DSN`).
3. Comportamento de guarda de cidade, item por nível e restock: continuam sem consumidor Go — a
   tabela guarda o dado, não implementa o sistema (caveat mantido).

## Critérios de aceite

- [ ] `dbserver import-content --sets=all --check` verde contra `Release/`
- [ ] Nenhum dos 4 arquivos é lido por código Go (`grep` só em testes/importador)
- [ ] `game_setting` cobre todas as linhas do `gameconfig.txt` (incluindo a matriz 8×8)
- [ ] `region` = 52 linhas e o `worldatlas` passa a lê-la
- [ ] Documentado, por linha, o que ainda não tem consumidor Go (guard/level item/restock)

## Riscos

- **Modelar o que não roda:** mitigado pelo campo `raw jsonb` e pelo critério de paridade do import;
  o dado fica fiel mesmo antes de existir leitor.
- **`gameconfig` é 8 linhas fixas + matriz:** um parser tolerante demais mascara arquivo corrompido;
  o `--check` compara contagem de linhas e a matriz inteira.
