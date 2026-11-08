# Contrato de porte — persistência (`wyd-db`)

> **Data:** 2026-11-08 · **Fontes de verdade:** schema legado `internal/migrations/*.up.sql`
> (Postgres, 56 tabelas), o código que o consome (`internal/store/*.go`,
> `tmserver/internal/dbclient/client.go`) e o schema Rust já escrito em
> `crates/wyd-db/migrations/` (28 migrations, 66 tabelas).
> **Autoridade (corrigida pela norma do projeto):** o schema canônico é
> **`docs/arquitetura/modelo-banco.md` + `crates/wyd-db/migrations/`** — é o que `AGENTS.md` §4
> declara como fonte da verdade do banco. O schema legado **não** decide o schema novo: ele é
> **entrada de requisitos** (o que o jogo precisa para rodar) e o C++ do DBSrv é referência
> histórica, não contrato (§2). Tabela legada sem espelho vira **decisão pendente registrada em
> `modelo-banco.md`**, nunca migration criada por conta deste documento.
> **Produto:** mapa tabela → uso → crate/tabela Rust, com buracos explícitos. Nenhuma migration
> foi criada nesta passada.
>
> ## Normas do projeto que regem este contrato
>
> | Norma | O que já fixa (citado, não redefinido) |
> |---|---|
> | `AGENTS.md` §3 invariante 1 | Hot Path sem I/O no tick; persistência por portas (traits) + canais async (ADR-0001/0004/0006) |
> | `AGENTS.md` §3 invariante 6 | senha/PIN com hash resistente (argon2id no legado); texto plano proibido, inclusive em log; DSN e segredos nunca versionados |
> | `AGENTS.md` §3 invariante 7 | a web nunca escreve estado vivo: grants só por `delivery_queue`, drenada pelo servidor |
> | `AGENTS.md` §4 | `modelo-banco.md` + `crates/wyd-db/` são a fonte canônica do schema |
> | `.agents/skills/wyd-rust-db/SKILL.md` | regras 1–4: migrations versionadas e **reversíveis**, integridade no banco (FK/CHECK/índice), async fora do tick com pool/timeouts, segredos fora de log e repo; árvore §12 |
> | `docs/arquitetura/modelo-banco.md` §4–§5 | convenção `NNNN_descricao.sql` com marcadores `-- migrate:up`/`-- migrate:down`, down reverte exatamente o up, nada de `ALTER TABLE` no `up`, `seeds/dev.sql` versionado |
> | `docs/arquitetura/adr/0009-conteudo-estatico.md` | conteúdo (item/skill/NPC/nível) é tabela carregada no boot, não arquivo solto |
> | `docs/arquitetura/adr/0013-down-seletivo-de-seed.md` | o `down` de seed restaura o estado anterior da fonte; nunca apaga linha de outra migration |
> | `docs/arquitetura/estrutura-e-padroes.md` §12 | árvore arquivo-a-arquivo do `wyd-db` (`model/`, `repo/`, `pool/`, `transfer/`) |
>
> **Medição nova deste documento:** o inventário das 56 tabelas legadas com linha, a frequência de
> uso por tabela no store, os 33 métodos do `dbclient` e o diff tabela-a-tabela contra as 61 do Rust
> (§5).

## 1. Duas persistências no legado (não confundir)

| Caminho | O que faz | Evidência |
|---|---|---|
| **C++ (original)** | `TMSrv` abre TCP para `DBSrv` na porta **7514** e fala `_MSG_DB*`; o DBSrv grava arquivos por conta | `Basedef.h:105`; `TMSrv/Server.cpp:4012-4017`; 48 constantes `_MSG_DB*` em `Basedef.h:1292-1513` |
| **Go (reescrita)** | `tmserver` chama `dbserver` por gRPC **no legado** — topologia que o Rust **não** reproduz (sem processo DBSrv: portas + canais, `AGENTS.md` §3 invariante 1, ADR-0001) | `tmserver/internal/dbclient/client.go` (funções em §4) |

Formato binário do C++ (dossiê já existente, não remedido aqui): `STRUCT_ACCOUNTFILE` (7.952 B) e
o `STRUCT_MOB` de 816 B — ver `docs/arquitetura/modelo-banco.md` e
`docs/cliente/referencia-wyd-global.md` §4. **UNVERIFIED (U-P1):** *o DBSrv escreve apenas esses
structs, ou há tabelas paralelas?* — fecha com leitura de `Source/Code/DBSrv/CFileDB.cpp`.

Para o Rust a topologia é outra (decisão já tomada em ADR-0001/0009): **não há processo DBSrv**;
`wyd-net`/`wyd-core` falam com `wyd-db` por porta assíncrona e o banco é o Postgres.

## 2. Schema legado — 56 tabelas, com linha

`internal/migrations/0001_schema.up.sql` (49 tabelas; linha do `CREATE TABLE`):

| Tabela | Linha | Papel no jogo |
|---|---:|---|
| `account` | 43 | credenciais, PIN, bloqueio, cargo (armazém) da conta |
| `account_admin_audit` | 93 | auditoria de ação admin sobre conta |
| `account_play_code` | 149 | código de resgate/jogo por conta |
| `affect` | 168 | buffs/efeitos ativos **persistidos** por personagem |
| `castle_quest_state` | 203 | estado da quest de castelo (guerra) |
| `character` | 225 | personagem (≈75 colunas) |
| `character_admin_audit` | 352 | auditoria admin sobre personagem |
| `character_pvp_stats` | 408 | PvP/duelo por personagem |
| `content_rate_audit` | 427 | auditoria de taxa de conteúdo |
| `content_rate_override` | 471 | taxa de drop/exp por fonte de conteúdo |
| `daily_reward_audit` | 493 | auditoria de recompensa diária |
| `daily_reward_claim` | 529 | quem já resgatou a recompensa diária |
| `daily_reward_item` | 563 | catálogo da recompensa diária |
| `delivery_queue` | 607 | entrega assíncrona (item/ouro/level) — web **não** escreve no personagem |
| `donate_payer_profile` | 651 | perfil de doador (loja) |
| `donate_shop_audit` | 670 | auditoria de compra |
| `donate_shop_item` | 706 | catálogo da loja de doação |
| `donate_topup_order` | 752 | pedido de recarga (dinheiro real) |
| `drop_rate_audit` | 793 | auditoria de drop |
| `drop_rate_override` | 835 | drop por região/mob |
| `guild` | 856 | guilda |
| `guild_admin_audit` | 879 | auditoria de guilda |
| `guild_member` | 922 | vínculo conta↔guilda↔cargo |
| `guild_relation` | 946 | aliança/hostilidade entre guildas |
| `guild_tower_state` | 967 | torre de guilda (evento) |
| `guild_zone` | 986 | zona dominada por guilda |
| `item` | 1012 | item possuído (personagem **ou** cargo, discriminado por `kind`) |
| `item_catalog` | 1049 | catálogo de itens |
| `item_effect_audit` | 1113 | auditoria de efeito |
| `item_effect_override` | 1156 | efeito de item sobrescrito |
| `item_price` | 1191 | preço por NPC/cidade |
| `kefra_state` | 1217 | estado do Kefra por personagem |
| `mob_carry_audit` | 1241 | auditoria de "carry" de mob |
| `mob_carry_override` | 1283 | loot/carry sobrescrito por mob |
| `mob_template_equip` | 1317 | equipamento do mob |
| `mob_template_stat` | 1342 | atributos do mob |
| `npc_audit` | 1405 | auditoria de NPC |
| `npc_config_meta` | 1441 | metadados de configuração de NPC |
| `npc_definition` | 1459 | NPC concreto (posição/mapa) |
| `npc_shop_item` | 1531 | estoque da loja do NPC |
| `npc_template` | 1558 | template de NPC |
| `sapphire_balance` | 1587 | nome enganoso: **não é saldo de moeda**, é preço da capa de reino (`modelo-banco.md` §8) |
| `server_instance_flags` | 1609 | flags de instância/evento por servidor |
| `skill_catalog` | 1647 | catálogo de skills |
| `teleport_route` | 1685 | rotas de teleporte |
| `world_event_audit` | 1736 | auditoria de evento de mundo |
| `world_event_config` | 1771 | configuração de evento de mundo |
| `world_event_meta` | 1805 | metadados de evento |
| `world_item_seed` | 1823 | itens fixos no chão (spawn de mundo) |

`internal/migrations/0008_platform_content.up.sql` (7 tabelas):

| Tabela | Linha |
|---|---:|
| `content_source_file` | 10 |
| `guard_spawn` | 23 |
| `level_item` | 32 |
| `mob_merc` | 49 |
| `game_setting` | 60 |
| `game_drop_bonus` | 71 |
| `region` | 79 |

## 3. Quem lê e quem escreve (frequência medida no código legado)

Contagem de ocorrências de `FROM <tabela>` / `INTO <tabela>` em `internal/store/*.go`:
`account` 35/20, `characters`→`character` 18/7, `npc_definition` 16, `npc_shop_item` 10,
`item` 7, `guild_member` 7, `mob_template_stat` 6, `mob_template_equip` 6, `delivery_queue` 6.

Funções do store que definem o contrato real (`internal/store/store_live.go`):

| Função | Linha | Tabelas tocadas |
|---|---:|---|
| `AccountByName` | 39 | `account` |
| `AccountAuthByID` | 55 | `account` |
| `PinHashByID` / `SetPinHash` | 72 / 86 | `account` (PIN com hash) |
| `SetBlockedByName` | 101 | `account` |
| `ListCharacters` | 115 | `character` |
| **`LoadCharacter`** | **154** | `character` + `item` + `affect` (via `loadItems` 213, `loadAffects` 234) |
| `CreateCharacter` | 255 | `character` + `item` inicial |
| `CreateArchCharacter` | 276 | `character` (arconte) |
| `DeleteCharacter` | 347 | `character` (soft delete) |
| **`SaveCharacter`** | **388** | `character` + `item` + `affect` |
| `LoadCargo` | 482 | `account` (moedas) + `item` `kind='vault'` (`loadAccountItems` 501) |
| `SaveCargo` | 526 | `account` + `item` |

**Detalhe que qualquer implementação precisa respeitar:** `SaveCharacter` faz `UPDATE` de ~64
colunas em `character` (linha 397) e termina com `RETURNING id` (linha 420); `LoadCharacter` lê a
mesma tabela em uma única linha (171). Ou seja o legado **grava o personagem inteiro por UPDATE
monolítico** — é o oposto do SoA do Hot Path e o motivo do plano de saída por `Result` +
`delivery_queue`.

## 4. Fronteira jogo ↔ banco no legado Go (o "port" que o Rust precisa)

`tmserver/internal/dbclient/client.go` é o contrato explícito entre o loop de jogo e o banco —
cada método é uma intenção, não SQL:

| Método | Linha | Uso no tick? |
|---|---:|---|
| `AccountLogin` | 34 | não (handshake) |
| `ListCharacters` | 71 | não (seleção) |
| `CreateCharacter` | 108 | não |
| `CreateArchCharacter` | 122 | não |
| `DeleteCharacter` | 138 | não |
| `SetPin` / `VerifyPin` | 151 / 160 | não (login) |
| `LoadCharacter` | 182 | não (entrada no mundo) |
| `SaveOnShutdown` | 198 | não (desligamento) |
| `QuoteKingdomCape` / `PurchaseKingdomCape` | 210 / 219 | não (compra) |
| `LoadCargo` / `SaveCargo` | 230 / 254 | não (armazém) |
| `ListPendingDeliveries` | 268 | não (login/cargo) |
| `SaveCargoWithDeliveries` | 282 | não (resgate atômico) |
| `SetAccountBlocked` | 297 | admin |
| `RecordDuelResult` | 309 | fim de duelo |
| guilda (7 métodos) | 325-424 | ação de jogador |
| `LoadGuildZones` / `SaveGuildZone` | 450 / 463 | evento |
| `LoadGuildTowerState` / `SaveGuildTowerState` | 475 / 484 | evento |
| `LoadCastleQuestState` / `SaveCastleQuestState` | 496 / 505 | evento |

Regra derivada (e já invariante do projeto, `AGENTS.md` §3 invariantes 1 e 7): **nenhum desses métodos pode ser
chamado dentro do tick**; eles são a fronteira que `wyd-db` expõe como porta assíncrona.

## 5. Mapa legado → Rust (66 tabelas, 28 migrations)

Estado medido do esquema Rust: `crates/wyd-db/migrations/` com 27 arquivos, sendo 16 de DDL
(61 `CREATE TABLE`) e 11 de seed sem DDL (`0017_seed_taxonomia` … `0027_seed_combine`).
DDL por arquivo: `0001` 1 · `0002` 1 · `0003` 2 · `0004` 2 · `0005` 1 · `0006` 1 · `0007` 8 ·
`0008` 13 · `0009` 4 · `0010` 6 · `0011` 2 · `0012` 4 · `0013` 4 · `0014` 5 · `0015` 4 · `0016` 3.

| Tabela legada | Rust | Situação |
|---|---|---|
| `account` (43) | `accounts` (`0001_contas.sql`) | ✅ direta (nome pluralizado) |
| `account_play_code` (149) | — | ⛔ **fora do `wyd-db`** — código de login do site é Cold Path (`modelo-banco.md` §8) |
| `affect` (168) | `character_affects` (`0028`) | ✅ MIGRADO |
| `castle_quest_state` (203) | `castle_state` | ⚠️ renomeada |
| `character` (225) | `characters` (`0002_personagens.sql`, **16 colunas**) + `character_status` (`0016_progressao.sql:14`) + `character_skills` + `character_lineage` + `character_quest` | ⚠️ **dividida em 5** |
| `character_pvp_stats` (408) | `arena_score` (`0015`) | ⚠️ outra semântica (arena ≠ duelo) |
| `character_admin_audit` / `account_admin_audit` (352/93) | `audit_log` (`0005_auditoria.sql`) | ⚠️ audit genérico |
| `content_rate_override` (471) | `content_rate_override` | ✅ |
| `content_rate_audit` / `drop_rate_audit` / `daily_reward_audit` / `donate_shop_audit` / `guild_admin_audit` / `item_effect_audit` / `mob_carry_audit` / `npc_audit` / `world_event_audit` (427/793/493/670/879/1113/1241/1405/1736) | `audit_log` | ⚠️ consolidados |
| `daily_reward_claim` (529) | `daily_reward_claim` (`0028`) | ✅ MIGRADO (`UNIQUE(account_id, claim_date)`) |
| `daily_reward_item` (563) | `daily_reward_item` | ✅ |
| `delivery_queue` (607) | `delivery_queue` (`0006_delivery_queue.sql`) | ✅ com `CHECK` de coerência por `kind` |
| `donate_payer_profile` (651) | — | ❌ **falta** |
| `donate_shop_item` (706) | `donate_shop_item` | ✅ |
| `donate_topup_order` (752) | — | ⛔ **fora do `wyd-db`** — recarga comercial é Cold Path (`modelo-banco.md` §8) |
| `drop_rate_override` (835) | `drop_rate_override` | ✅ |
| `guild` (856) / `guild_member` (922) | `guilds` (`0004:5`) / `guild_members` (`0004:14`) | ✅ |
| `guild_relation` (946) | `guild_relations` (`0028`) | ✅ MIGRADO |
| `guild_tower_state` (967) | `guild_tower` (`0028`) | ✅ MIGRADO |
| `guild_zone` (986) | `guild_zone` (`0008:188`) | ✅ |
| `item` (1012) | `character_items` (`0003:10`) + `vault_items` (`0003:32`) | ✅ separação explícita (melhor que `kind`) |
| `item_catalog` (1049) | `item_catalog` (`0007`) | ✅ |
| `item_effect_override` (1156) | `item_effect` (`0007`) | ⚠️ catálogo, sem histórico de override |
| `item_price` (1191) | `item_price` (`0010`) | ✅ |
| `kefra_state` (1217) | `kefra_state` | ✅ |
| `mob_carry_override` (1283) | `mob_merc` (`0008:130`) | ⚠️ semântica diferente (mercenário ≠ carry) |
| `mob_template_equip` (1317) / `mob_template_stat` (1342) | `mob_template_equip` / `mob_template_stat` | ✅ |
| `npc_config_meta` (1441) | — | ❌ **falta** |
| `npc_definition` (1459) | `npc_definition` (`0008:47`) | ✅ |
| `npc_shop_item` (1531) | `npc_shop_item` (`0008:77`) | ✅ |
| `npc_template` (1558) | `npc_template` (`0008:23`) | ✅ |
| `sapphire_balance` (1587) | `kingdom_cape_price` (`0028`) | ✅ MIGRADO — é preço com `version`, não saldo |
| `server_instance_flags` (1609) | — | ❌ **falta** |
| `skill_catalog` (1647) | `skill_catalog` (`0007`) | ✅ |
| `teleport_route` (1685) | `teleport_route` (`0011`) | ✅ |
| `world_event_config` (1771) / `world_event_meta` (1805) | `world_event` (`0014`) + `event_schedule` | ⚠️ normalizado |
| `world_item_seed` (1823) | `world_item_seed` (`0008:179`) | ✅ |
| `content_source_file` (0008:10) | `content_source` (`0008:10`) | ✅ renomeada |
| `guard_spawn` (0008:23) | `guard_spawn` (`0008:116`) + `guard_spawn_reward` (`0008:122`) | ✅ melhor |
| `level_item` (0008:32) | `level_item` (`0008:99`) | ✅ |
| `mob_merc` (0008:49) | `mob_merc` (`0008:130`) | ✅ |
| `game_setting` (0008:60) | `game_setting` (`0008:140`) | ✅ |
| `game_drop_bonus` (0008:71) | — | ❌ **falta** |

**Só no Rust** (não existem no legado, decisão de normalização): `character_lineage`,
`city`, `city_tax_ledger`, `combine_value`, `content_version`, `game_class`, `game_matrix`,
`game_system`, `map_definition`, `personal_shop`, `quest`/`quest_step`/`quest_reward`,
`refine_rate`/`refine_gem_effect`, `sanc_rate`, `skill_lineage`, `event_participation`,
`arena`/`arena_match`/`arena_round`.

### O que fazer com as ❌ (norma, não iniciativa)

Nenhuma tabela ❌ vira migration por causa deste contrato. O rito é o do projeto
(`AGENTS.md` §7):

1. decidir em `docs/arquitetura/modelo-banco.md` **se** a tabela entra (o schema canônico é ele,
   `AGENTS.md` §4) e com que nome — `banco-catalogo.md` é gerado por `make banco-catalogo` e
   precisa passar (`cargo test -p wyd-db`, `make banco-verify`, ver `modelo-banco.md` §4);
2. se a mudança for estrutural (nova entidade, novo `kind`, mudança de autoridade), ADR antes do
   SQL (`docs/arquitetura/adr/`, rito FASE-13-1);
3. só então a migration `NNNN_descricao.sql` **com** `-- migrate:up` e `-- migrate:down`, onde o
   down reverte exatamente o up (`modelo-banco.md` §4; skill `wyd-rust-db` regra 1) — sem
   `ALTER TABLE` no `up` (coluna nova nasce na criação da tabela).

**U-P7 — resolvida (2026-11-08).** O grep dos consumidores no legado separou as sete:

| Veredito | Tabelas |
|---|---|
| **jogo** (entram no `wyd-db`) | `affect` → `character_affects`, `sapphire_balance` → `kingdom_cape_price` (é preço com `version`, não saldo), `daily_reward_claim`, `guild_relation` → `guild_relations`, `guild_tower_state` → `guild_tower` |
| **portal/Cold Path** (não entram) | `account_play_code` (código de login do site) e `donate_topup_order` (recarga em dinheiro real) |

Decisão registrada em `docs/arquitetura/modelo-banco.md` §8 (schema canônico) e **entregue** em
`crates/wyd-db/migrations/0028_estado_de_jogo.sql`, com `-- migrate:up`/`-- migrate:down` e prova
de ciclo (`migrations-apply` + `banco-verify`).

## 6. O que o `characters` do Rust **não** carrega (e por quê importa)

`characters` tem **16 colunas**: `id, account_id, slot, name, class, tier, arch_of_id, level,
exp, hp, mp, pos_x, pos_y, created_at, updated_at, deleted_at`
(`crates/wyd-db/migrations/0002_personagens.sql`). O legado tinha ~75 colunas em `character`
(225-350). A diferença está normalizada em:

- `character_status` (`0016:14`): `attr_str/int/dex/con`, `points_status/learn/skill`, `fame`.
- `character_skills`, `character_lineage`, `character_quest`, `character_items`.

**Buracos a confirmar antes de portar o save (UNVERIFIED):**

| # | Pergunta | Fecha com |
|---:|---|---|
| U-P2 | *`magic`/lista de skills e `save_celestial`/`nightmare_entries`/`kefra_ticket`/`pk_point` do legado têm coluna no Rust?* | comparação coluna-a-coluna de `character` (legado 225-350) × `characters`+`character_status` |
| U-P3 | *os `affect` (buffs) precisam persistir no login, ou são recalculados?* | `LoadCharacter` (legado 154) + `loadAffects` (234) provam que persistia |
| U-P4 | *`sapphire` (moeda premium) tem onde morar no Rust?* | existe `donate_shop_item`, mas não há saldo/ledger |
| U-P5 | *`delivery_queue` do Rust cobre os 3 `kind` do legado?* | `0006` tem `CHECK kind IN ('item','gold','level')` — confirmar que o legado não tinha 4º tipo |
| U-P6 | *`characters.deleted_at` reproduz o soft delete + restauração do legado?* | `DeleteCharacter` (legado 347) |

## 7. `sqlx` e o caminho de execução

O crate tem **28 migrations e 66 tabelas** e, desde a FASE-04-3, **usa `sqlx`** (ADR-0021): pool
real, adapters `pg` das três portas e fila de save drenada fora do tick. Destino normativo
(`visao-pastas.md` §2): `wyd-db` = `model/`, `repo/`, `pool/`, `transfer/`, `migrations/`.

Nomes conforme a árvore §12 de `estrutura-e-padroes.md` (não inventar arquivo: `AGENTS.md` §7):

| Peça | Módulo (árvore §12) | Contrato |
|---|---|---|
| pool + timeouts + transações curtas | `pool/{mod,config,pg}.rs` — **executado** | `acquire_timeout` + `SET LOCAL statement_timeout` por transação; nenhuma query no tick (skill `wyd-rust-db` regra 3) |
| repositórios por agregado | `repo/{accounts,characters,grants,content}.rs` + `repo/pg.rs` — **executado** | uma função = uma intenção (espelha os 33 métodos do `dbclient`, §4); `create_account`/`create_character` evitam ambiguidade de trait no mesmo adapter |
| tipos de domínio sem `sqlx` | `model/{account,character,guild,content,audit}.rs` | `model` não conhece SQL |
| drenagem de `delivery_queue` | `transfer/delivery_queue.rs` | grants do portal → dreno do servidor; leitura por `repo/grants.rs` (`AGENTS.md` §3 invariante 7) |
| migrações | `migrations/NNNN_descricao.sql` | 28 arquivos hoje, **todos** com `-- migrate:up` e `-- migrate:down` (medido); down reverte exatamente o up; runner embutido (`sqlx::migrate!`/`refinery`) **não** entrou — aplicação por `tools/content/migrate-cycle.mjs`/`psql -f` (ADR-0021) |
| carga de conteúdo | `repo/content.rs` | `content_source`/`content_version` (`0008:10`) dão versão ao boot; conteúdo inconsistente aborta o boot (ADR-0009) |
| segredos e DSN | `error.rs` + `pool/config.rs` | `DbError`/configuração nunca imprimem DSN; PIN só como hash (argon2id; `AGENTS.md` §3 invariante 6) |

Correção de rumo registrada: o legado **grava personagem inteiro** por `UPDATE` de ~64 colunas
(§3); o Rust deve gravar por **diff/agregado** e fora do tick — a diferença não é cosmética, é o
que impede o tick de bloquear em I/O.

### 7.1 Fila de save e política (FASE-04-3)

| Peça | Onde | Contrato |
|---|---|---|
| produtor (Hot Path) | `transfer/save.rs` → `SaveQueue::enqueue` | `try_send` em canal limitado, **não é `async`**: não existe `await` possível no tick |
| consumidor (Cold Path) | `SaveDrain::{flush, run, shutdown}` | transação curta com `UPDATE ... WHERE id AND deleted_at IS NULL` |
| coalescência | `SaveDrain::coalesce` | último snapshot por personagem vence; lote sai ordenado por `character_id` |
| backpressure | `DbError::SaveQueueFull` | fila cheia recusa na hora; quem chama decide reagendar (nunca espera) |
| perda | `DbError::SaveTimeout` | shutdown que não esvazia no prazo é **erro**, não perda silenciosa |
| personagem sumido | `FlushReport::missing` | contado e reportado; não derruba o lote |

Gatilhos: periódico (`periodic_ms`, padrão 5 s), por lote (`max_batch`, padrão 256) e shutdown
(`shutdown_timeout_ms`, padrão 3 s). `SavePolicy` é validada — zero/desligado é recusado.

Medido em 2026-11-08 (Postgres 17 descartável, 28 migrations + seed, integração `tests/pg.rs` 6/6):
`enqueue` com N=20 000 e dreno real gravando em paralelo → **p50 500 ns, p99 1 600 ns**
(p99 = 0,003% do tick de 50 ms); shutdown gravou 2/2 sem perda e o personagem apagado contou como
`missing`, não como sucesso.

## 8. Reprodução

```bash
rm -rf /tmp/legacy && mkdir -p /tmp/legacy
git archive origin/main tmserver internal | tar -x -C /tmp/legacy
grep -nE "^CREATE TABLE" /tmp/legacy/internal/migrations/0001_schema.up.sql | wc -l   # 49
grep -nE "^CREATE TABLE" /tmp/legacy/internal/migrations/0008_platform_content.up.sql | wc -l  # 7
grep -nE "^func \(s \*Store\)" /tmp/legacy/internal/store/store_live.go
grep -nE "^func \(c \*Client\)" /tmp/legacy/tmserver/internal/dbclient/client.go
grep -rhoE "CREATE TABLE( IF NOT EXISTS)? [a-z_]+" crates/wyd-db/migrations/ | wc -l  # 66
for f in crates/wyd-db/migrations/*.sql; do echo "$(basename $f) $(grep -c 'CREATE TABLE' $f)"; done
```
