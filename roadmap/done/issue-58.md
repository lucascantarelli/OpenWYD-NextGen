# Issue 58 — Baseline do banco: uma inicialização completa, sem arquivos de conteúdo

**Fase:** 7 · **Depende de:** 57 · **Esforço:** L · **Estado:** ✅ fases 1–5 concluídas (05/11): o banco é a fonte única de catálogo/NPC, o baseline está squasheado e os arquivos de conteúdo saíram do repositório

## Resposta curta

Sim — o "dump inicial" é um **seed versionado** de `npc_definition` + `npc_shop_item` (6.100 + 3.538
linhas, ~2,5 MB de SQL) gerado a partir do conteúdo; depois disso o `NPCGener.txt` sai do repositório.

Mas o squash só vale a pena **depois** de completar o esquema: hoje faltam 4 tabelas que o Go já lê
(item, skill, template de NPC, itens do mundo) e mais 4 que fecham o resto dos arquivos de config
(guard, level item, mob merc, game settings — issue 59). Squashear antes produz um baseline que nasce
incompleto.

Ordem: **esquema → leitores → seeds → squash → apagar arquivos**. O squash é o penúltimo passo.

## Inventário real (04/11/2026)

```
migrations aplicadas ................. 50   (schema_migrations.version = nome do arquivo)
números repetidos .................... 0007 ×2, 0008 ×2, 0012 ×4, 0013 ×2, 0022 ×2
tabelas .............................. 45 de domínio + schema_migrations
banco ................................ 15 MB
```

Referência (entra no seed): `npc_definition`=6.100, `npc_shop_item`=3.538, `guild_zone`=5,
`castle_quest_state`=1, `kefra_state`=1, `world_event_config`=1, `world_event_meta`=1,
`npc_config_meta`=1, `server_instance_flags`=10.

Jogador (NUNCA entra no dump): `account`=1, `character`=2, `item`=62, `affect`=2 e todas as
`*_audit` (28 linhas de histórico).

## Arquivos de conteúdo: destino de cada um

| Arquivo | Tamanho | Leitor Go | Destino |
|---|---|---|---|
| `Common/ItemList.csv` | 361 KB / 3.220 itens | tmServer, `itemcatalog`, `itemeffectadmin`, `characteradmin`, `droptool` | **`item_catalog`** (fase 1) |
| `Common/SkillData.csv` | 14 KB / 151 | tmServer, `skillcatalog` | **`skill_catalog`** (fase 1) |
| `TMsrv/run/npc/*` | 1.991 arquivos (620 usados) | tmServer, `mobtemplates`, `mobstat`, `mobcarry` | **`npc_template`** BYTEA (fase 1) |
| `TMsrv/run/InitItem.csv` | 911 B / 48 | tmServer (`seedWorldItems`), `worldatlas` | **`world_item_seed`** (fase 1) |
| `TMsrv/run/NPCGener.txt` | 2,68 MB / 6.099 blocos | só o importador do dbserver | ✅ `npc_definition` (issue 57), seed na fase 3 |
| `Settings/{Sanc,Comp,Quests}Rate.txt` | pequenos | tmServer | ✅ `content_rate_override` |
| `Settings/CastleQuest.txt` | pequeno | tmServer | ✅ `castle_quest_state` |
| `TMsrv/run/Guard.txt` | 699 B | **nenhum** (sistema de guarda não portado) | **`guard_spawn`** — issue 59 |
| `TMsrv/run/LevelItem.txt` | 17 KB | **nenhum** (`DoItemLevel` DEFERIDO) | **`level_item`** — issue 59 |
| `Common/Settings/MobMerc.txt` | pequeno | **nenhum** (restock não portado) | **`mob_merc`** — issue 59 |
| `TMsrv/run/gameconfig.txt` | pequeno | **nenhum** (viram flags) | **`game_setting` + `game_drop_bonus`** — issue 59 |
| `TMsrv/run/Regions.txt` | 1,9 KB / 52 | só `worldatlas` | **`region`** — P1 |
| `TMsrv/run/ItemDropList.txt`, `MobDropList.txt` | 242 KB | **nenhum** | **apagar** — artefato do DropTool, substituído por `/drops` (issue 34) — issue 61 |
| `HeightMap.dat` / `AttributeMap.dat` | 16 MiB / 1 MiB | tmServer | **asset binário**: decisão e trade-off na issue 60 |

## Etapas de migration (ordem pedida: tabela → seed → constraints/índices)

Cada tabela nova entra em **três migrations separadas**, para que a falha apareça na etapa certa e o
seed rode sem custo de índice:

```
0044_<tabela>.up.sql            CREATE TABLE ... (sem índices além da PK)
0045_<tabela>_seed.up.sql       INSERT ... ON CONFLICT DO NOTHING  (dados gerados)
0046_<tabela>_constraints.up.sql  FK, CHECK, UNIQUE e índices de leitura
```

Isso também torna o seed revisável sozinho no PR (é o arquivo grande) e permite `SET CONSTRAINTS ALL
DEFERRED` quando a ordem natural de importação não respeitar as FKs.

## Layout alvo do baseline (fase 3–4)

```
0001_schema.up.sql            -- pg_dump --schema-only consolidado
0002_seed_world_defaults.up   -- guild_zone, castle_quest_state, kefra_state, world_event_*, npc_config_meta
0003_seed_npc_catalog.up      -- npc_definition (6.099) + npc_shop_item
0004_seed_item_catalog.up     -- item_catalog (3.220)
0005_seed_skill_catalog.up    -- skill_catalog (151)
0006_seed_npc_templates.up    -- npc_template (620 blobs)
0007_seed_world_items.up      -- world_item_seed (48)
0008_seed_game_config.up      -- guard_spawn, level_item, mob_merc, game_setting, game_drop_bonus, region
```

Regra: versão única de 4 dígitos, sem down para seed (o `down` do baseline é `DROP SCHEMA public
CASCADE`, documentado como destrutivo), nenhum dado de jogador.

## Fases de execução

**Fase 1 — completar o esquema (0044+). ✅ CONCLUÍDA (04/11).**
`item_catalog`, `skill_catalog`, `npc_template`, `world_item_seed` + stores de leitura + importador
idempotente. Leitores ainda nos arquivos: nada de runtime mudou.

Entregue:

- Migrations segmentadas em tabela → seed → índices: `0044/0045` item_catalog, `0046/0047`
  skill_catalog, `0048/0049` npc_template, `0050/0051` world_item_seed (os índices ficam na última
  etapa, então o seed não paga b-tree por linha).
- `internal/domain/content.go` (tipos), `internal/store/contentcatalog.go` (upsert transacional,
  `List*` para a fase 2 e `CountContentCatalog` para o `--check`).
- `tmserver/cmd/importcontent` — flags `-content`, `-dsn` (ou `W2PP_DB_DSN`), `-sets`, `-check`;
  dry-run sem DSN. Mora em `tmserver/cmd` e não em `dbserver` porque os parsers estão em
  `tmserver/internal/content` e a regra de visibilidade do `internal` impede o irmão de importá-los.
- `internal/textenc` — o CSV é **Windows-1252**, não UTF-8: o primeiro import morreu em
  `invalid byte sequence for encoding "UTF8": 0xe3` no item 25. Conversão explícita (com a tabela
  0x80–0x9F, que em CP1252 é pontuação tipográfica e não caractere de controle) + teste.

Evidência do import real no banco de dev:

```
conteúdo lido items=3220 skills=151 templates=1991 world_items=49 templates_ignorados=0
itens importados rows=3220 · skills importadas rows=151 · templates importados rows=1991
objetos do mundo importados rows=49
--check: item_catalog 3220=3220 · skill_catalog 151=151 · npc_template 1991=1991 · world_item_seed 49=49
catálogos em sincronia com o conteúdo (CHECK_EXIT=0)
```

Notas de modelagem que valem para as próximas tabelas: `item_catalog.fields` guarda a linha crua
(JSON) para o import ser lossless; `npc_template.blob` tem `CHECK (length(blob) = 816)` porque o
layout canônico é a única forma aceita; `world_item_seed` usa PK sintética com
`UNIQUE (item_index, pos_x, pos_y)` — o mesmo índice de item aparece em várias posições e reimportar
não pode duplicar o objeto no mundo.

**Fase 2 — virar os leitores. ✅ tmServer (04/11); portal/web-api pendentes.**
O tmServer lê os quatro catálogos por gRPC do dbserver, com fallback para `Release/` quando a tabela
está vazia ou o RPC falha. `-content` continua obrigatório para mapas, ícones e taxas.

Entregue:

- `api/db/v1/content.proto` — `ContentCatalogService` com um RPC por catálogo (não um snapshot único:
  `npc_template` sozinho são 1,6 MB e o limite padrão de mensagem gRPC é 4 MiB). Gerado por `protoc`
  no container (o host não tem protoc), arquivos novos em vez de reescrever `db.pb.go` — evita diff de
  milhares de linhas por causa da versão do protoc.
- `dbserver/internal/grpcsrv/contentcatalog.go` — implementação read-only sobre `store.List*`, com
  `-1` como sentinela de coluna nula (`volatile`, `attack_range`) para o tmServer distinguir "sem
  alcance" de "alcance 0"; registrado em `serve` junto dos outros serviços.
- `tmserver/internal/dbclient/contentcatalog.go` — `Fetch` dos quatro conjuntos, devolvendo os tipos
  de `content` (`ItemEntry`, `Spell`, `InitItem`) para o mundo não mudar.
- `tmserver/cmd/tmserver/main.go` — busca o catálogo logo depois de abrir o gRPC e só então instala
  itens/skills (`content.NewItemList`/`NewSkillData`), o resolver de templates (banco antes do
  arquivo, inclusive para o `Vine` do Muro de Espinhos) e os objetos do mundo; `seedWorldItems` agora
  recebe a lista pronta e cai no `InitItem.csv` só quando ela vem vazia.
- `dbserver/internal/grpcsrv/contentcatalog_test.go` — fixa o mapeamento (colunas cruas, sentinela
  -1, blob de 816 B).

Evidência (dbserver e tmServer novos, portas 17514/18281, build e boot reais):

```
content catalogs loaded from the database items=3220 skills=151 templates=1991 world_items=49
vine template loaded source=database
world items seeded count=49
npc generators registered generators=6099 slots=6099 event_owned=199 training_grounds=104
tmserver listening addr=:18281
```

Fallback também exercitado (dbserver fora do ar):
`content catalogs not loaded from the database; using Release/ files … connection refused`, e o mundo
subiu com `vine template loaded source=file` + `world items seeded count=49`.

**Achado que muda a fase 4:** o `dbserver serve` ainda **aborta no boot** lendo
`NPCGener.txt` (`build NPC generator catalog: open /Release/TMsrv/run/NPCGener.txt: no such file or
directory`), e reconcilia as 6.099 definições contra o arquivo a cada start
(`NPC generator catalog reconciled expected=6099 processed=0`). Enquanto isso existir, o arquivo não
pode sair do repositório — é exatamente o que o seed do baseline substitui.

**Fase 3 — seeds gerados. ✅ (04/11)**
`tmserver/cmd/seedgen` lê as tabelas de referência e emite um `.sql` por tabela em
`internal/migrations/seeds/`. Determinismo vem da ordenação pelo próprio JSON da linha
(`ORDER BY to_jsonb(t)::text`), não por lista de colunas — assim o dump não quebra quando uma migration
adiciona coluna, e o arquivo não tem timestamp no cabeçalho (regerar sem mudança = diff vazio).

Como o SQL é gerado:

```sql
INSERT INTO item_catalog
SELECT * FROM json_populate_record(NULL::item_catalog, $json${"item_index": 1, …}$json$::json)
ON CONFLICT DO NOTHING;
```

`row_to_json`/`json_populate_record` é o que mantém o dump **lossless para qualquer coluna** (jsonb,
bytea como `\x…`, arrays) sem uma segunda cópia do esquema no código do gerador.

Allowlist explícita em `internal/store/seed.go` (12 tabelas): `guild_zone`, `castle_quest_state`,
`kefra_state`, `world_event_config`, `world_event_meta`, `npc_config_meta`, `npc_definition`,
`npc_shop_item`, `item_catalog`, `skill_catalog`, `npc_template`, `world_item_seed`. Fora dela:
`account`, `character`, `item`, `affect`, auditorias e `server_instance_flags` (estado de boot, não
conteúdo). Um `pg_dump` genérico copiaria dado de jogador — é o risco número um do squash.

**Boot do dbserver sem `NPCGener.txt`:** o `serve` não aborta mais quando o arquivo falta — registra
WARN e segue; o requisito real passou a ser "existe catálogo no banco", verificado por
`ContentNPCDefinitionCount` (`origin='content'`), que falha fechado se for zero. Isso é o que
destrava remover o arquivo do repositório.

Dois bugs reais apareceram ao aplicar os seeds pela primeira vez — ambos invisíveis até existir um banco
limpo para restaurar:

1. **`effects` como jsonb `null`.** O importador gravava `json.Marshal(nil)` = `null`. O `NOT NULL` não
   pegava (jsonb `null` não é SQL NULL), mas `json_populate_record` relê JSON null como SQL NULL e o seed
   morria no item 4138. Corrigido no importador (slice vazio) e fechado por constraint na 0052.
2. **Colunas de jogador e de identidade.** `npc_definition.id` é `GENERATED ALWAYS` (recusa valor
   explícito) e `npc_definition.updated_by` referencia `account` — um banco novo não tem contas e a FK
   abortava o seed inteiro. O gerador agora emite **lista de colunas explícita** (lida do catálogo, sem
   identidade/geradas) e **omite** as colunas que apontam para jogador, em vez de gravar NULL: lista de
   colunas de INSERT não aceita expressão.

Evidência:

```
seed gerado tabela=npc_definition linhas=6100 · npc_shop_item 3538 · item_catalog 3220
npc_template 1991 · skill_catalog 151 · world_item_seed 49 · world defaults 10
seeds gerados tabelas=12 linhas=15059
DETERMINISTIC (duas execuções, diff -r vazio)
```

Restauração real: banco `seedcheck` criado só com `pg_dump --schema-only` do banco de dev (sem dado
nenhum), os 12 seeds aplicados com `ON_ERROR_STOP=1` → **zero falhas** e contagens idênticas ao original
em todas as tabelas (`npc_definition 6100`, `npc_shop_item 3538`, `item_catalog 3220`, `npc_template
1991`, `skill_catalog 151`, `world_item_seed 49`, defaults 10). Integridade amostrada: item 25 =
`Titã` (acento preservado), blobs de `npc_template` todos com 816 B e 3.538 linhas de loja com FK
resolvida. O banco de teste é descartado no fim.

**Fase 4 — o squash. ✅ CONCLUÍDA (04/11)**

Antes de arquivar, o esquema recebeu os achados P0/P1 da auditoria
(`docs/analysis/banco-de-dados.md` §11) como migration normal, `0053_baseline_normalization`: tipos
canônicos, os 8 índices de FK que faltavam e comentário em 49/49 tabelas de domínio. O dump só foi
tirado **depois** disso — o baseline é a fotografia de um estado real, não SQL editado à mão.

`scripts/db-baseline.sh` faz, nesta ordem:
1. backup completo (`scripts/db-backup.sh`, formato custom, destino fora do repositório);
2. `pg_dump --schema-only` → `0001_schema.up.sql` (sem `OWNER`/`GRANT`, sem `schema_migrations`,
   sem meta-comando de `psql` — o runner executa SQL puro e `\restrict` do pg_dump 17 o quebraria);
3. `0001_schema.down.sql` destrutivo e explícito + os 12 seeds gerados;
4. validação em banco vazio (`baselinecheck`): aplica tudo com `ON_ERROR_STOP=1`, compara contagem por
   tabela e número de tabelas com a origem, e **falha** (sem promover nada) se divergir;
5. `--apply`: as 59 migrations antigas (`up` **e** `down`) saem da raiz do pacote embedded (vão para
   `_archived/`, gitignored), o baseline e os seeds passam a ser os únicos arquivos da raiz,
   `internal/migrations/seeds/` deixa de existir, e o banco atual recebe
   `INSERT INTO schema_migrations(version) VALUES ('0001_schema')`.

Evidência do dry-run e da promoção:

```
dump do esquema .......... 2.949 linhas, 84 KB
seeds .................... 12 arquivos (18.232 linhas de referência)
validação em banco vazio . contagens idênticas ao banco de origem, 49/49 tabelas
arquivadas ............... 110 arquivos (59 up + 51 down)
adoção ................... schema_migrations(0001_schema) registrado
```

Dois cuidados que o script resolveu: o embed é `//go:embed *.sql` **na raiz** do pacote (por isso o
`seedgen` passou a gravar em `internal/migrations`, não em `seeds/`), e a contagem de tabelas exclui
`schema_migrations` — ela é criada pelo runner, então a origem sempre teria uma a mais.

Gate final executado com o schema **dropado** (`DROP SCHEMA public CASCADE`) e as imagens reconstruídas a
partir do código: `store.Migrate` aplicou os 13 arquivos no boot, sem intervenção manual.

```
schema_migrations ........ 13 versões (0001_schema + 12 seeds)
tabelas .................. 50 (49 de domínio + schema_migrations criada pelo runner)
linhas de referência ..... 18.232 (npc_definition 6.100 · npc_shop_item 6.711 · item_catalog 3.220
                           · npc_template 1.991 · skill_catalog 151 · world_item_seed 49 · defaults 10)
dbserver ................. npc catalogue ready content_definitions=6099 · dbserver serving :7514
tmserver ................. content catalogs loaded from the database items=3220 skills=151
                           templates=1991 world_items=49 · vine template source=database
                           · world items seeded count=49
                           · npc generators registered generators=6099 slots=6099
                           · npc config applied at boot version=19 npcs=5162
                           · tmserver listening addr=:8281 · world loop started grid=4096
e2e ...................... TestE2ESmokeLogin PASS (LOGIN OK, CNFAccountLogin, 1996 bytes)
```

Equivalência de esquema conferida de forma mecânica, não por inspeção: o dump do esquema que estava
vivo (`db-backup.sh`, 04/11 20:52) foi restaurado num banco descartável e comparado coluna a coluna
com o banco criado só pelo baseline — **509/509 colunas idênticas** (nome, tipo, nulabilidade), 49
tabelas em ambos. `grep` nos seeds por `pass_hash`, `account` e `account_id`: nenhum resultado.

Dois bugs reais só apareceram nesse gate — ambos invisíveis enquanto o banco era o mesmo de sempre:

1. **`set_config('search_path','')` no dump.** O `pg_dump` 17 emite esse `SET` de sessão logo no
   cabeçalho. Como `store.Migrate` reaproveita uma conexão para todas as migrations, o `INSERT INTO
   schema_migrations` do runner passava a rodar com `search_path` vazio e falhava com
   `42P01 relation "schema_migrations" does not exist` — dbserver em crash-loop a cada boot. O
   `db-baseline.sh` agora filtra a linha (junto com `\restrict`); o dump qualifica tudo com `public.`,
   então nada depende dela.
2. **`npc_shop_item` com 6.711 linhas, não 3.538.** O inventário de 04/11 estava velho: o import da
   issue 57 regravou as lojas. O baseline acompanha o banco (a validação do script é por tabela); o
   número é que estava errado no documento.

Os 739 `npc content generator produced no entities` no boot são esperados e já documentados na issue
57 (`max_num_mob=0` e posição bloqueada no `AttributeMap`) — não são regressão do squash.

**Fase 5 — apagar os arquivos. ✅ CONCLUÍDA (05/11)**

Saíram da árvore de conteúdo: `Common/ItemList.csv`, `Common/SkillData.csv`,
`TMsrv/run/NPCGener.txt`, `TMsrv/run/InitItem.csv` e `TMsrv/run/npc/` (1.991 templates).
Continuam em arquivo, com motivo: `Guard.txt`, `LevelItem.txt`, `MobMerc.txt` e `gameconfig.txt`
(ainda sem tabela — issue 59), `Regions.txt` (vira `region` na issue 61), os mapas (issue 60), as
taxas, `CastleQuest.txt`, `serv00.htm` e `DBsrv/run/BaseMob`/`TMsrv/run/BaseSummon`.

Código ajustado na mesma entrega (migração sem remoção do legado é o defeito que a fase 4 veio
corrigir):

- **tmServer** — `loadContent` perdeu os loaders de ItemList/SkillData; os catálogos vêm só do
  gRPC e o **fallback por arquivo foi removido**: banco sem catálogo agora recusa o boot
  (`load content catalogs from the database` / `item_catalog is empty`). Saíram também o fallback de
  `Vine`, o resolvedor de template por `npc/` (agora `npc_template` + overlays de stat/carry) e o
  fallback de `InitItem.csv` em `seedWorldItems`. As flags `-item-effect-editing`/`-mob-stat-editing`
  /`-mob-carry-editing` passaram a exigir só `-dbserver`.
- **dbServer** — fora `import-npcs`, `buildNPCDefinitions`, `parseNPCGener` e o **reconcile de
  `NPCGener.txt` a cada boot** (era a origem da divergência 6.711 × 3.538 no `npc_shop_item`). O
  boot só exige catálogo no banco (`npc catalogue ready content_definitions=6099`).
- **worldatlas** — passou a ler `world_item_seed` e `npc_definition` por `-dsn`; o `SourceHash`
  agora cobre arquivos + payload do banco, então mudar um NPC invalida o cache do portal.
- **`importcontent`** — fica, deliberadamente: é o único caminho para reprocessar uma árvore de
  conteúdo **externa** (backup), nunca a do repositório.
- **Testes** — os que dependem de conteúdo real agora fazem `t.Skip` quando o arquivo não existe
  (`readRelease`/`requireItemList`/`releasePath` em `tmserver/internal/handler`, `release()` em
  `tmserver/internal/content`), em vez de falhar num checkout sem a árvore proprietária.

- **`_archived/` apagado** — 111 arquivos (474 KB) que o squash tinha deixado em disco. Eles nunca
  eram lidos (`//go:embed *.sql` só enxerga a raiz) e estão inteiros no git até `45eeb20`; manter
  uma segunda cópia executável só criava a chance de alguém rodar SQL pré-baseline. A pasta continua
  no `.gitignore` porque `db-baseline.sh --apply` ainda a usa como staging local.

Docs: `AGENTS.md` §2/§4 e `docs/migration/config-ops.md` §2.5 reescritos para "catálogos vêm do
banco".

## Critérios de aceite

- [x] Banco vazio + `store.Migrate` = esquema completo (12 tabelas novas) e todos os seeds aplicados
- [x] Esquema do banco novo == esquema do banco vivo (509/509 colunas; o dump do esquema antigo foi
      restaurado e comparado coluna a coluna — o hash de `--data-only` virou comparação por contagem,
      porque a origem foi dropada)
- [x] Contagem por tabela das 12 tabelas de referência igual à do banco de origem (18.232 linhas)
- [x] Nenhum `INSERT` de `account`/`character`/`item`/`affect`/`*_audit` nos seeds
- [x] Banco criado do zero (schema dropado, volume preservado) + stack sobe com o mundo completo
- [ ] `grep -rn "ItemList.csv\|SkillData.csv\|InitItem.csv\|NPCGener\|Guard.txt\|LevelItem\|MobMerc\|gameconfig" --include=*.go` só em testes (fase 5)
- [x] `TestE2ESmokeLogin` verde no banco criado só pelo baseline

## Riscos

- **Dump com dado de jogador:** seed gerado por allowlist de tabelas, nunca `pg_dump` inteiro.
- **Migration 0043 irreversível:** o baseline a torna definitiva (o bloco importado é o dono).
- **Baseline divergente do código:** por isso a ordem das fases — completar o esquema antes de squashear.
- **Tabela sem leitor (issue 59):** `guard_spawn`/`level_item`/`mob_merc` guardam o dado fielmente, mas
  nenhum sistema Go os consome ainda; o import é lossless e o comportamento entra na issue de cada
  sistema. Sem isso, o arquivo continuaria sendo a única cópia.
- **Adoção em banco existente:** esquecer o `INSERT` da versão `0001_schema` reaplica o baseline e falha
  em `CREATE TABLE` existente — o script roda o diff antes de marcar.
