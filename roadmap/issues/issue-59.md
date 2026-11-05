# Issue 59 — Migrar as configs de conteúdo restantes para o banco (guard, level item, mob merc, game settings)

**Fase:** 7 · **Depende de:** 58 · **Esforço:** M · **Estado:** ⏳ proposta

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
