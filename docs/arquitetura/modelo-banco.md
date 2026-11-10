# Modelo do banco (`wyd-db`) — FASE-04

> Fonte: `roadmap-rust/issues/FASE-04.md` (+ 04-1/04-2/04-3). Implementação:
> `crates/wyd-db/{migrations,seeds,src/{model,repo,pool,transfer}}`.

## 0. Origem do modelo (arquivo Go/C++ removido em 2026-11)

O arquivo Go/C++ foi removido do repo; não há schema legado a revisar.
Este documento + `crates/wyd-db/migrations/` são a **fonte canônica** do schema.
O modelo deriva dos requisitos das issues + domínio do jogo + invariantes do
`AGENTS.md` (autoridade server-side, auditoria, `delivery_queue` como única
ponte web→jogo).

## 1. ER (texto)

```text
accounts 1───* characters 1───* character_items
   │              │  ╲
   │              │   ╲──* guild_members *───1 guilds
   │              │
   │              └── (pos_x, pos_y, level, exp, hp/mp)
   │
   ├───1 vault_items (PK account_id+slot)
   │
   └───* delivery_queue (grants pendentes do wyd-portal/admin)

characters 1───* character_affects   (um por `type`; buffs reidratados no login)
characters 1───1 character_short_skill  (ShortSkill[16] do CNFCharacterLogin)
characters 1───1 character_learned_skill (máscara LearnedSkill do _MSG_ApplyBonus)
characters 1───1 character_special       (Special[4] + SpecialBonus da maestria)
accounts   1───1 account_wallet      (Coin do CNFAccountLogin; separado de characters.gold)
accounts   1───* daily_reward_claim  (UNIQUE account_id + claim_date)
guilds     1───* guild_relations     (kind 1/2, sem auto-relação)
kingdom_cape_price (linha única)  ·  guild_tower (linha única; owner_guild 0 = sem dono)

audit_log (append-only, sem FK para nunca bloquear escrita)
```

## 2. Tabelas e integridade

| Tabela | PK | FKs | CHECKs | Índices quentes |
|---|---|---|---|---|
| `accounts` | `id` | — | `role IN ('player','moderator','admin')`, `login` não-vazio | `UNIQUE(login)` |
| `characters` | `id` | `account_id → accounts` ON DELETE CASCADE | `slot 0–3`, `level ≥ 1`, `exp ≥ 0`, `hp ≥ 0`, `name` não-vazio | `UNIQUE(account_id, slot) WHERE deleted_at IS NULL`, `UNIQUE(name)`, `idx(account_id)` |
| `character_items` | `(character_id, slot)` | `character_id → characters` ON DELETE CASCADE | `slot 0–127`, `qty ≥ 1` | `idx(character_id)` (= PK) |
| `vault_items` | `(account_id, slot)` | `account_id → accounts` ON DELETE CASCADE | `slot 0–127`, `qty ≥ 1` | `idx(account_id)` (= PK) |
| `character_short_skill` | `character_id` | `character_id → characters` CASCADE | `octet_length(short_skill) = 16` | `idx(character_id)` (= PK) |
| `account_wallet` | `account_id` | `account_id → accounts` CASCADE | `coin` 0–2147483647 (`int` do fio) | `idx(account_id)` (= PK) |
| `character_learned_skill` | `character_id` | `character_id → characters` CASCADE | `learned_skill` 0–16777215 (24 bits = `g_pSpell[24*classe+pos]`) | `idx(character_id)` (= PK) |
| `character_special` | `character_id` | `character_id → characters` CASCADE | `special_0..3` 0–255 (teto 200, 255 com skill de linha) e `special_bonus >= 0` | `idx(character_id)` (= PK) |
| `guilds` | `id` | `master_character_id → characters` DEFERRABLE | `name` não-vazio | `UNIQUE(name)` |
| `guild_members` | `(guild_id, character_id)` | ambas CASCADE | `rank 0–9` | `UNIQUE(character_id)` (um clã por char) |
| `audit_log` | `id` | — (nunca bloqueia) | `action` não-vazia | `idx(at)` |
| `account_moderation` | `account_id` | `account_id`/`blocked_by → accounts` | `blocked` exige `reason` e `blocked_at` coerentes | parcial `WHERE blocked` |
| `delivery_queue` | `id` | `account_id/character_id` SET NULL (grant sobrevive a delete? não: RESTRICT — ver §4) | coerência por `kind` (item/gold/level) | parcial `WHERE status='pending'` |
| `character_affects` | `(character_id, type)` | `character_id → characters` CASCADE | `level ≥ 0`, `time ≥ 0` | `idx(character_id)` (= PK) |
| `kingdom_cape_price` | `id` | — | `id = 1`, `version ≥ 1`, `hekalotia_cost`/`akelonia_cost` 4–32 | — (linha única) |
| `daily_reward_claim` | `id` | `account_id → accounts` CASCADE | `UNIQUE(account_id, claim_date)` | `idx(account_id, claim_date)` (= UNIQUE) |
| `guild_relations` | `(guild_id, target_guild_id)` | ambas `→ guilds` CASCADE | `kind ∈ {1,2}`, `guild_id <> target_guild_id` | `idx(target_guild_id)` |
| `guild_tower` | `id` | — (`owner_guild` sem FK: 0 = sem dono) | `id = 1`, `owner_guild ≥ 0`, `updated_at_unix ≥ 0` | — (linha única) |

Tipos: IDs `BIGINT GENERATED …` (ver `BIGSERIAL` nas migrations — forma
simples e portátil no PG17); dinheiro/EXP `BIGINT ≥ 0`; tempo `TIMESTAMPTZ`;
`role`/`kind`/`status` como `TEXT + CHECK` (enum em tabela separada é
over-engineering nesta escala; CHECK documenta o domínio).

## 3. Decisões

1. **Soft-delete de personagem** (`deleted_at`): nome e slot liberados via
   índice parcial — recriação sem violar unicidade, histórico preservado.
2. **Inventário normalizado, não blob**: `character_items`/`vault_items` com
   PK composta permitem conferência item-a-item no boot (anti-dupe) sem ler
   binário opaco. Efeitos de item continuam catálogo (FASE-10/economia).
   `slot` 0–15 é `Equip[i]` e 16–79 é `Carry[i − 16]` (80–127 reservado): é esse
   mapeamento que o `STRUCT_MOB` do login usa — o banco é a fonte da verdade depois
   da criação, que copia o template uma única vez.
3. **`delivery_queue` relacional**: colunas tipadas por `kind` + CHECK de
   coerência, em vez de JSONB opaco — a fila é dinheiro/item, não log.
4. **Sem FK na `audit_log`**: auditoria nunca falha por integridade alheia.
5. **`guilds.master` DEFERRABLE**: permite criar clã + líder na mesma transação.
6. **Grants restritos**: `delivery_queue.account_id/character_id` com
   `ON DELETE RESTRICT` — grant pendente impede apagar a conta/personagem
   (falha explícita em vez de item órfão).

## 4. Migrations (convenção do repo)

- `crates/wyd-db/migrations/NNNN_descricao.sql`, uma por domínio, com
  marcadores `-- migrate:up` e `-- migrate:down` (estilo dbmate/refinery).
- Down reverte exatamente o up (ordem inversa); `audit_log` incluída (ambiente
  descartável — produção futura exige política própria, ver 04-2).
- Runner embutido (`sqlx::migrate!`/`refinery`) **não** entrou na FASE-04-3: o pool e os adapters
  `pg` foram adotados (ADR-0021), mas a aplicação das migrations continua por `psql -f` com split
  nos marcadores (`tools/content/migrate-cycle.mjs`; validação estrutural em `tests/constraints.rs`).
  Trocar o runner exige issue própria.
- **Nenhum `ALTER TABLE` no `up`**: coluna nova nasce na criação da tabela. O banco ainda não
  foi para produção, então desenho corrigido se reescreve na origem, não por cima (foi o que
  aconteceu em 2026-11-08 com `0023_inventario_gema`/`0024_catalogo_skill_nota`, consolidados
  em `0003`/`0007`). `crates/wyd-db/tests/catalogo.rs` falha se um `ALTER` voltar.
- **Catálogo gerado**: `docs/arquitetura/banco-catalogo.md` vem de `make banco-catalogo`
  (migrations + `tools/content/banco-fontes.mjs`) e lista, por tabela, colunas/tipos/nulidade,
  PK/FK/CHECK/índices, arquivo de origem, fonte da decisão e o que é provisório (D-Bxx).
  Verificação: `cargo test -p wyd-db` (doc ↔ SQL) e `make banco-verify` (Postgres real ↔ doc).

## 5. Seeds

- `seeds/dev.sql` versionado: conta/personagem/clã de desenvolvimento com
  valores **obviamente falsos** (`DEV-ONLY`, nunca hash real).
- Produção **nunca** versiona seed com segredo (regra da árvore §12); o
  procedimento de seed prod vive no runbook de operação (FASE-05), não aqui.

## 6. Conteúdo estático (itens, skills, NPCs) — procedência e lacuna

Conteúdo de jogo é **dado de referência**, não estado de jogador: nunca entra em
`audit_log`, nunca é escrito pela web e nunca mistura linha de conta. O que a
árvore Rust tem hoje são só as tabelas de jogador (§2); o catálogo deconteúdo ainda **não foi portado** — esta seção registra de onde ele vem, para a FASE-03-4
(seed + validação de schema no boot) e a FASE-10 (fórmulas) não inventarem dado.

**Fonte canônica decidida na ADR-0009:** o **PostgreSQL**, em tabelas tipadas
(`item_catalog`, `skill_catalog`, `npc_*`, `level_item`, `game_setting`, …) com
`PRIMARY KEY`, `FOREIGN KEY` e `CHECK`, populadas por migrações versionadas de seed;
`wyd-db` carrega e `wyd-core` valida antes do primeiro tick. Nenhum `.csv`/`.txt`/`.bin`
é lido em runtime — o legado é insumo do gerador de seed (`tools/content/`) e base de
conhecimento. As §6.1–§6.3 abaixo ficam como histórico do import.

### 6.1 Fonte histórica (árvore Go em `origin/main`, removida da árvore em 2026-11)

`internal/migrations/seed_*.up.sql`, gerados por `tmserver/cmd/seedgen` (issue 58),
com `sha256` por linha e `ON CONFLICT DO NOTHING` (idempotente). Contagem declarada
no cabeçalho de cada arquivo:

| Seed | Linhas | Conteúdo |
|---|---:|---|
| `seed_item_catalog.up.sql` | 3.220 | `item_catalog`: preço, grade, requisitos, efeitos EF_*, `fields` cru do CSV |
| `seed_skill_catalog.up.sql` | 151 | `skill_catalog`: alvo, mana, delay, range, affect/tick, passivo |
| `seed_npc_template.up.sql` | 1.991 | `npc_template`: nome de classe, stats base |
| `seed_npc_definition.up.sql` | 6.100 | `npc_definition`: NPC por mapa/posição |
| `seed_npc_shop_item.up.sql` | 6.708 | `npc_shop_item`: mercador → item/preço |
| `seed_world_item_seed.up.sql` | 49 | `world_item_seed`: itens no chão por mapa |
| `seed_guild_zone.up.sql` | 5 | `guild_zone`: zonas de clã |
| `seed_castle_quest_state` / `seed_kefra_state` / `seed_world_event_{config,meta}` / `seed_npc_config_meta` | 1 cada | estado inicial de world event, castelo e Kefra |

O schema legado (`0001_schema.up.sql`) tem 54 tabelas; as de conteúdo incluem
`item`, `item_catalog`, `item_price`, `item_effect_override`,
`item_effect_audit`, `skill_catalog`, `npc_template`, `npc_definition`,
`npc_shop_item`, `npc_config_meta`, `mob_template_stat`,
`mob_template_equip`, `mob_carry_override`, `drop_rate_override`,
`content_rate_override`, `teleport_route`, `daily_reward_item`,
`donate_shop_item`, `world_item_seed`, `guild_zone`, `castle_quest_state`,
`kefra_state`, `world_event_{config,meta}`.

### 6.2 Configuração de jogo sem tabela própria

`0008_platform_content.up.sql` (issue 59) trouxe os arquivos sem leitor Go, com
proveniência em `content_source_file` (`path`, `sha256`, `size_bytes`, `rows`):
`Guard.txt` (699 B → `guard_spawn.rules` jsonb), `LevelItem.txt` (17.468 B →
`level_item` por **ordinal**, porque o arquivo repete blocos), `MobMerc.txt`
(187 B → `mob_merc`), `gameconfig.txt` (2.467 B → `game_setting`) e
`Regions.txt` (1.922 B → 52 retângulos). Onde a semântica era UNVERIFIED, o dado
ficou em `jsonb`/`raw` em vez de coluna inventada.

### 6.3 Binários do cliente (`client/`, fora do git)

O build 7662 carrega o catálogo do disco: `ItemList.bin` 910.004 B,
`Itemname.bin` 215.084 B, `SkillData.bin` 23.812 B, `Mixlist.bin` 272.000 B,
`extraitem.bin` 106 B, `cdata.bin` 8.192 B, `InitItem.bin` 768 B, `object.bin`
524.292 B (`docs/cliente/inventario.md`). Vários abrem com bytes repetidos
(`ItemList.bin` = `5A 5A 5A …`, `SkillData.bin` = `42 5A 5A …`) e `Itemname.bin`
tem ASCII cifrado byte-a-byte — isto é, são **XOR/estrutura própria**, não CSV.
Nenhum leitor existe em `tools/client/` (`formats.mjs` só mediu `.wyt`/`.msa` e
deixou `serverlist.bin`, `object.bin`, `curse.bin`, `itemicon.bin` opacos).

**Regra de import:** regenerar o conteúdo a partir das linhas `item_catalog` /
`skill_catalog` / `npc_*` (texto + `sha256` de proveniência), **não** do `.bin`.
O `.bin` serve para conferência de `item_index`/nome e para o launcher, não como
fonte de verdade do servidor.

### 6.4 Lacunas registradas

1. ~~Nenhuma migração Rust cria as tabelas de conteúdo~~ — **fechado em 2026-11-08**: o
   conteúdo entra em `wyd-db` como DDL + seed idempotente, com `content_version` e
   proveniência (ADR-0009); nada de arquivo em runtime (§6.5/§6.8).
2. ~~Não existe tabela de lore/quests nem de mix/refino~~ — **fechado em 2026-11-08**: os
   domínios viraram `0009_refino`, `0013_quests` e `0010_economia` (mix fica em
   `game_system` + receita a modelar quando a fórmula for fechada, ver §6.8).
3. `docs/sistemas/*` (13 arquivos), `docs/lore/*` (3) e `docs/guias/*` (4)
   existem **apenas em `origin/main`**; a árvore atual perdeu os originais e
   mantém só `wyd-portal/src/content/guias.json` (19 slugs gerados) — o gerador
   `scripts/build-guias.mjs` não tem mais fonte para regenerar.
4. Texto de sistema em `docs/` hoje é só protocolo/cliente/arquitetura; não
   existe documento de combate/drop/refino na árvore Rust.

### 6.5 Sequência de migrações (revisada em 2026-11-08)

> **Exceção registrada (2026-11-08):** `0028_estado_de_jogo.sql` traz **DDL depois** do bloco de
> seed (`0017–0027`), porque as cinco tabelas da §8 só foram decididas depois da série de seed.
> A regra que continua valendo é a que importa: nenhum seed insere em tabela que não existe —
> cobrado por `crates/wyd-db/tests/catalogo.rs` (`seed_insere_so_em_tabela_ja_criada`). Renumerar
> a série para manter a ordem posicional não vale o custo em histórico e em referências de ADR.

Regra da série: **DDL primeiro (0001–0016), seed depois (0017–0024)**, um domínio por
arquivo, nenhuma tabela partida entre arquivos, nenhuma FK para tabela criada depois e
nenhum `ALTER TABLE` no `up`. Verificação em três camadas:
`node tools/content/check-migrations.mjs` (ordem/FK/down/INSERT, sem banco),
`cargo test -p wyd-db` (`tests/catalogo.rs`: doc ↔ SQL, `SCHEMA_VERSION`, seed depois do DDL)
e `node tools/content/verify-schema.mjs` (banco vazio → `up` → `information_schema` ↔ catálogo).
Números atuais: **42 arquivos, 83 tabelas, 152 `INSERT`s**, sem problema (medidos pelo `check-migrations`, 2026-11-10; colunas não recontadas aqui).

| Migração | Domínio | Objetos |
|---|---|---|
| `0001_contas.sql` | acesso | `accounts` |
| `0002_personagens.sql` | personagem | `characters` |
| `0003_inventario.sql` | inventário | `character_items`, `vault_items` |
| `0004_guildas.sql` | social | `guilds`, `guild_members` |
| `0005_auditoria.sql` | auditoria | `audit_log` |
| `0006_delivery_queue.sql` | Cold Path → Hot Path | `delivery_queue` |
| `0007_conteudo_catalogo.sql` | catálogo | `content_version`, `game_system`, `game_class`, `skill_lineage`, `item_kind`, `item_effect`, `item_catalog`, `skill_catalog` — PK/FK/CHECK + índice GIN em `effect_names` |
| `0008_conteudo_mundo.sql` | mundo | `content_source`, `npc_template`, `npc_definition`, `npc_shop_item`, `level_item`, `guard_spawn`, `guard_spawn_reward`, `mob_merc`, `game_setting`, `game_matrix`, `region`, `world_item_seed`, `guild_zone` |
| `0009_refino.sql` | refino | `refine_rate` (taxa por passo em bps), `refine_gem_effect` (4 gemas: drop/pierce/exp/absorb) |
| `0010_economia.sql` | economia | `city`, `personal_shop`, `city_tax_ledger`, `item_price`, `donate_shop_item`, `daily_reward_item` |
| `0011_rotas_mundo.sql` | mundo | `map_definition`, `teleport_route` |
| `0012_combate.sql` | combate | `mob_template_stat`, `mob_template_equip`, `drop_rate_override`, `content_rate_override` |
| `0013_quests.sql` | quests | `quest`, `quest_step`, `quest_reward`, `character_quest` |
| `0014_eventos.sql` | eventos | `world_event`, `event_schedule`, `kefra_state`, `castle_state`, `event_participation` |
| `0015_arena.sql` | arena | `arena`, `arena_match`, `arena_round`, `arena_score` |
| `0016_progressao.sql` | progressão | `character_status` (atributos, pontos, fama), `character_lineage` (maestria), `character_skills` (aprendidas) |
| `0031_moderacao.sql` | moderação | `account_moderation` (bloqueio com motivo/autor; ADR-0023) |
| `0017_seed_taxonomia.sql` | seed | 20 sistemas, 6 classes, 13 linhagens, 16 categorias de item |
| `0018_seed_item_catalog.sql` (gerado) | seed | 3.220 itens + 55 símbolos `EF_*` |
| `0019_seed_skill_catalog.sql` (gerado) | seed | 151 skills |
| `0020_seed_mundo.sql` (gerado, 5,5 MB) | seed | NPCs, lojas, mundo, settings (§6.7) |
| `0021_seed_skill_lineage.sql` (gerado) | seed | classe/linhagem de 63 skills curadas, com fonte e confiança por linha (cobertura 69/154) |
| `0022_seed_refino.sql` | seed | 14 passos de `refine_rate` (9 de pó, +10 e +12..+15; +11 sem fonte publicada) + 4 gemas de `refine_gem_effect` |
| `0023_seed_mundo_fixo.sql` | seed | 4 `city`, 2 `arena`, 3 `world_event` + 3 `event_schedule` (Kefra sábado, Noatun quinzenal, guerra de cidades) |
| `0024_seed_skill_catalog_kersef.sql` | seed | 3 skills publicadas pela Kersef que o seed do cliente não trazia (`Roubar_Dinheiro`, `Evocar_Lobo_Selvagem`, `Metamorfose_Superior`), com linhagem, fonte, confiança e índice provisório declarado; `content_version = 7` |

Uma migração por domínio, não uma por tabela: o `down` de `0013_quests`, por exemplo,
derruba `quest` (CASCADE em `quest_step`/`quest_reward`/`character_quest`) numa unidade —
meia quest não existe.

Campos que respondem "para que serve este item": `item_catalog.kind`,
`primary_system`, `event_only`, `tradeable`, `stackable`, `classification_source`,
`confidence`; do lado de skill: `class_code`, `lineage_code`, `skill_role`. A
classificação vem de regra de nome (`rule:name_v1`, confiança 60) ou fica `NULL` —
**1.532 itens permanecem sem categoria** porque o nome não prova nada; isso é
informação registrada, não falha a esconder.

Prova (2026-11-08, Postgres do compose, banco descartável `wyd_seedcheck`): bloco `up`
das quatro migrações aplicado com `ON_ERROR_STOP=1`; resultado das consultas → 3.220
itens, 151 skills, 55 efeitos, 20 sistemas, 1.688 itens classificados, 99 `event_only`,
156 não negociáveis e `content_version = 3`. O bloco `down` derruba as quatro tabelas
na ordem inversa.

### 6.6 Sanitização e normalização do insumo (2026-11-08)

Regras aplicadas por `tools/content/gen-seed.mjs` **antes** de emitir SQL. O gerador
verifica e falha; não conserta dado por chute. O relatório `sanitizacao` conta cada regra
que disparou, então "está limpo" é uma afirmação medida, não uma suposição.

| Regra | Por quê | Medido nos 12 insumos |
|---|---|---|
| UTF-8 estrito, por arquivo | Postgres só guarda UTF-8 válido; `?` silencioso perde nome | 11 válidos; `LevelItem.txt` com 4 bytes truncados (`0xC3` antes de CRLF), **todos em linha `#`** → tolerado, contado e descartado com o comentário |
| CRLF → LF | `\r` fantasma quebra `raw` e comparação de linha | 5 TXT em CRLF (`LevelItem` 546, `Regions` 51, `gameconfig` 30, `Guard` 8, `MobMerc` 7) — normalizado no parse |
| BOM, U+200B–200F, U+2060, U+00AD, controle C0/C1 | invisível quebra busca, log e diff | 0 |
| NBSP (U+00A0, U+2007, U+202F) → espaço | `foo bar` ≠ `foo bar` numa comparação | 0 |
| NFC como forma canônica | `ã` (U+00E3) e `a`+U+0303 são textos diferentes para `=` | todos já NFC; a regra impede NFD entrar |
| espaços colapsados em rótulo | `"Guarda   Real "` → `"Guarda Real"` | 0 (dado já aparado) |
| dobra de acento em chave de busca | busca do portal sem digitar acento | ver §6.7 (`content_fold`) |

Três correções **de dado**, todas registradas no relatório:

1. `level_item`: 318 linhas → **161 de conteúdo** (o bloco inteiro aparece repetido; 157
   linhas idênticas descartadas). Dois `seq` têm duas linhas distintas — preservados.
2. `region`: um retângulo vem invertido (`x1 > x2`); canonizar é perda zero (mesmos tiles).
   Duas regiões passam de 4095 (`Dugeon_1_Andar_Hidra` tem `6841`) — preservadas, porque o
   teto do cliente é regra do loader, não do banco.
3. `npc_template`: **20 nomes em CP949 lido como ISO-8859-1** (arquivo coreano decodificado
   como latin1), ex. `ÇÇ´Ð½º` = 피닉스. A chave **não** é renomeada — é referência do
   cliente e de `npc_definition`; `encoding_origin` marca as 20 e `name_recovered` guarda o
   coreano quando a recuperação é total (**12 de 20**; 8 perderam byte no legado).
   `Foema_Anciã` e `Troll_Caçador@@` são português legítimo e ficam `utf8`.

#### 6.6.1 Duplicidades — o que é e por que existe

| Conjunto | Grupos | Itens | Causa | Ação |
|---|---|---|---|---|
| `item_catalog.name` (exato) | 488 | 1.367 | nomes comerciais repetidos | nenhuma |
| `item_catalog.name` (dobrado) | 489 | 1.370 | idem, com acento/caixa variando | nenhuma |
| `item_catalog` (nome+grade+preço+`effects`+`fields`) | **0** | 0 | — | **nenhuma duplicata acidental** |
| `npc_template.template_name` (exato) | 0 | — | é PK | — |
| `npc_template` (nome normalizado) | 345 | — | variantes `@Nome`, `Nome_`, `Nome__` | mantidas (são chaves) |
| `npc_definition` (`id`, `slug`) | 0 | — | `slug` = nome + id | — |
| `npc_shop_item` (`npc_id`, `slot`) | 0 | — | é PK | — |
| `region.name` | 2 | 4 | `Erion` e `Armia` com **retângulos diferentes** | id sintético |
| `game_setting` (`section`,`key`) | 0 | — | é PK | — |

Por que os nomes de item repetem (medido, não suposto):

- **tier antigo**: `Espada_Vorpal(Anct)` em `g5,g6,g7,g8` — mesmo nome, quatro grades;
- **evento**: `Cupom_da_Sorte` ×78, cada um com `item_index` próprio;
- **placeholder**: `NONAME` ×12, `not_used` ×7 (índice reservado no `ItemList`);
- **item de classe**: `TransKnight`, `Foema`, `BeastMaster`, `Huntress` ×6 cada.

Ou seja: nome **não é chave** de item e não deve ser único. O que identifica é
`item_index`; o que agrupa visualmente é `name_fold`.

### 6.7 Mundo entregue (migração 0008 + seed 0019, 2026-11-08)

| Migração | Conteúdo |
|---|---|
| `0008_conteudo_mundo.sql` | `content_source`, `npc_template`, `npc_definition`, `npc_shop_item`, `level_item`, `guard_spawn` + `guard_spawn_reward`, `mob_merc`, `game_setting` + `game_matrix`, `region`, `world_item_seed`, `guild_zone` |
| `0019_seed_mundo.sql` (gerado, 5,5 MB) | 1.991 templates, 6.100 NPCs, 6.706 slots de loja, 161 `level_item`, 5 guardas/27 exigências, 1 mercador, 25 settings, 20 matrizes, 52 regiões, 49 itens de chão, 5 zonas de guilda, 12 linhas de `content_source`, `content_version = 4` |

Mapa insumo → tabela/coluna (o "mapeado completamente"):

| Insumo | Tabela | Colunas |
|---|---|---|
| `seed_npc_template.up.sql` | `npc_template` | `template_name`, `file_name`, `source_size`, `sha256`, `blob` (STRUCT_MOB 816 B) 1:1; `encoding_origin`/`name_recovered` derivados (§6.6) |
| `seed_npc_definition.up.sql` | `npc_definition` | `id`, `slug`, `template_name`, `display_name`, `enabled`, `map_id`, `pos_x`, `pos_y`, `route_type`, `merchant`, `origin`, `generator_index`, `follower_template`, `minute_generate`, `min_group`, `max_group`, `max_num_mob`, `formation`, `generator_data` 1:1 |
| `seed_npc_shop_item.up.sql` | `npc_shop_item` | `npc_id`, `slot`, `item_index`, `eff1..3`, `effv1..3`, `quantity` 1:1 |
| `LevelItem.txt` | `level_item` | `source_seq`←`seq`, `level`, `class_id`←`classe`, `type_id`←`type`, `item_index`←`item`, `extras`; `ordinal`/`source_line` derivados |
| `Guard.txt` | `guard_spawn` / `guard_spawn_reward` | `guard_index`, `npc_id`←`npc_index`, `source_line`; `req_slot`+`values[8]`←blocos |
| `MobMerc.txt` | `mob_merc` | `generator_index`, `template_name`←`MobName`, `renew_seconds`, `reborn_seconds`, `items` |
| `gameconfig.txt` (escalares) | `game_setting` | `section`, `key`, `value_int`, `value_text`, `source_line` |
| `gameconfig.txt` (matrizes) | `game_matrix` | `section`, `block_index`, `line_index`, `values[]`, `source_line` |
| `Regions.txt` | `region` | `name`, `x1..y2`, `source_line`; `id` identity, `name_fold` gerado |
| `seed_world_item_seed.up.sql` | `world_item_seed` | `id`, `item_index`, `pos_x`, `pos_y`, `rotate`, `enabled` 1:1 |
| `seed_guild_zone.up.sql` | `guild_zone` | `clan`, `zone`, `victory`, `city_tax`, `tax_vault`, `charge_guild`, `challenge_guild`, `challenge_money` 1:1 |
| os 12 anteriores | `content_source` | `path`, `sha256`, `size_bytes`, `row_count`, `loaded_by`, `transform` |

Semântica não provada, marcada e não promovida a gate: `merchant` acima de 0
(UNVERIFIED; o campo é `BYTE` no `STRUCT_MOB`), efeitos numéricos `eff/effv` de loja,
o bloco de 8 campos do `Guard.txt` (o cabeçalho do arquivo descreve 13), as seções
`Treasure`/`Drop Item Event`, `class_mask` do item.

Prova (2026-11-08, Postgres do compose, banco descartável `wyd_seedcheck`, `ON_ERROR_STOP=1`):
`up` de 0007–0008 → contagens da tabela acima; `sha256(blob) = sha256` em **1.991/1.991**;
`content_fold('Capa_de_Akelônia') = 'capa de akelonia'`, `('Arco_Æsir') = 'arco aesir'`,
`('Baú_de_10ª_Skill') = 'bau de 10a skill'`; `down` na ordem inversa → **0 tabelas** e
`content_fold` removida.

### 6.8 Domínios modelados depois do mundo (0009–0015)

| Domínio | Por que virou tabela (e não constante em Rust) | Onde a regra vive |
|---|---|---|
| Refino | taxa muda por servidor; o portal e a auditoria precisam ler o mesmo número | `refine_rate.rate_bps`; fórmula em `crates/wyd-sim/src/formula/refine.rs` (+ `wyd-core` como autoridade) |
| Gemas | efeito é dado de catálogo (item 3386–3389), não `match` no código | `refine_gem_effect`; escala `40 × (nível − 9)` em `GEM_*` no core |
| Economia | preço, taxa por cidade e fatia de Noatun são configuração do dono | `city`, `personal_shop`, `city_tax_ledger`; `item_price` sobrepõe o preço do catálogo sem reescrevê-lo |
| Rotas | teleporte tem requisito (nível, item, gold) e destino por mapa | `map_definition`, `teleport_route` |
| Combate | stats/equip de monstro e overrides de drop/EXP são por servidor | `mob_template_stat`, `mob_template_equip`, `drop_rate_override`, `content_rate_override` |
| Quests | passo, recompensa e estado por personagem não cabem em `characters` | `quest`, `quest_step`, `quest_reward`, `character_quest` |
| Eventos | Kefra/Noatun/castelo têm estado e agenda próprios | `world_event`, `event_schedule`, `kefra_state`, `castle_state`, `event_participation` |
| Arena | partida e round são histórico, não estado vivo | `arena_match`, `arena_round`, `arena_score` |

Fronteira mantida: **nada aqui é escrito pela web** e nada entra no tick. Overlay de
cold path (preço, taxa, agenda) chega ao servidor por `delivery_queue` (0006); o core lê
uma cópia em memória carregada no boot (FASE-04-3). `character_quest` e `arena_match`
são escrita **do servidor**, nunca de portal.

Pendente declarado: receita de **mix/craft** (`Mixlist` 272 KB) ainda não tem tabela porque
a fórmula de resultado não está fechada — modelar antes seria inventar coluna. O insumo
está inventariado em `docs/cliente/` e a decisão fica registrada em
`docs/wyd/decisoes-servidor.md`.

### 6.9 Prova no Postgres (2026-11-08) — up/down/up

A série deixou de ser "validada só estruturalmente": roda inteira num Postgres 17 real.

```bash
make migrations-apply     # = node tools/content/migrate-cycle.mjs
```

O runner (`tools/content/migrate-cycle.mjs`) sobe um container efêmero `wyd-migcheck`
(`postgres:17-alpine`, o mesmo do compose), espera **duas consultas reais seguidas** —
`pg_isready` responde ao servidor temporário do init e daria falso positivo —, aplica os 22
`up` com `ON_ERROR_STOP=1`, mede, aplica os 22 `down` em ordem inversa, exige **0 tabelas** e
reaplica tudo comparando as contagens dos dois ciclos.

Resultado real (24 migrations, ~55 s): **59 tabelas / 457 colunas** no fim do `up`, **0** após
o `down`; `item_catalog` 3.220, `item_effect` 55, `skill_catalog` **154** (**63** com
`class_code`), `game_system` 20, `region` 52, `npc_definition` 6.100, `npc_shop_item` 6.706,
`level_item` 161, `city` 4, `arena` 2, `world_event` 3, `refine_rate` 14,
`refine_gem_effect` 4, `content_version` **7**. Ciclo 1 e ciclo 2 idênticos.

Além do ciclo, `node tools/content/verify-schema.mjs` compara o banco recém-criado com
`banco-catalogo.md` via `pg_attribute`/`information_schema`: mesma contagem de tabelas, mesma
ordem/nome/tipo/nulidade de coluna e mesma PK em todas as 59 — é o que impede o catálogo de
virar ficção. Ver §6.10.

Três defeitos apareceram **só** ao aplicar de verdade, e foram corrigidos na origem:

1. runner lia o marcador citado no cabeçalho (`-- migrate:up / -- migrate:down`) e montava um
   bloco inválido — passou a exigir o marcador sozinho na linha, e o `check-migrations` foi
   corrigido do mesmo jeito;
2. `0021` (então `0020`) anunciava 61 skills mapeadas e o banco tinha 60: a curadoria tinha um índice
   duplicado e a cobertura somava "já mapeadas + novas" em vez da união. O gerador agora
   descarta e **reporta** a duplicata e calcula a união;
3. `refine_rate` tinha 9 passos com o doc dizendo 15, e `city`/`arena`/`world_event` estavam
   vazias (FK sem dado, loja pessoal impossível de abrir). Entraram os passos +10 e +12..+15,
   o seed `0022` e `city.map_id` virou anulável — o build 7662 não publica o mapa de cada
   cidade, e inventar número quebraria o teleporte.

#### 6.10 Catálogo do banco (fonte única da leitura humana)

`docs/arquitetura/banco-catalogo.md` é **gerado** por `make banco-catalogo` a partir das
migrations + `tools/content/banco-fontes.mjs`. Por tabela ele traz: arquivo que a cria,
domínio, estado (`confirmado`/`provisorio`), a fonte da decisão, PK/FK/CHECK/UNIQUE/índices e
a lista de colunas com tipo e nulidade. As seções finais registram as decisões provisórias
(`D-B01`..`D-B08`, mesmo padrão `D-xx` de `matematica-sim.md`) e o mapa tabela legada → nova,
com as pendências nomeadas em vez de silenciadas.

O catálogo **não** é escrito à mão: `cargo test -p wyd-db --test catalogo` falha se ele
divergir das migrations (tabela, coluna, PK, `SCHEMA_VERSION`, seed antes do DDL) e
`make banco-verify` falha se o Postgres real divergir dele. É por isso que os números deste
documento podem ser cobrados.

## 7. Revisão de uso do PostgreSQL (2026-11-08)

Pergunta: "estamos usando os recursos do Postgres e as boas práticas?" No volume atual
(maior tabela: 6.706 linhas) o que dá ganho é **integridade declarativa, proveniência e
chave de busca derivada** — não índice exótico. O que foi adotado:

| Recurso | Onde | Por quê |
|---|---|---|
| `CHECK` derivado do dado | `slot`, `class_id`, `type_id`, `pos_x/y`, `quantity`, `level`, `source_size` | falha na migração, não no tick |
| `FOREIGN KEY` real | `npc_definition→npc_template`, `npc_shop_item→npc_definition`, `guard_spawn→npc_definition` | 0 órfãos medidos |
| ausência de FK, documentada | `npc_shop_item.item_index`, `level_item.item_index` | 22 + 1 itens fora do catálogo; FK quebraria o seed |
| função `IMMUTABLE` | `content_fold(text)` | indexável e determinística sem extensão |
| `GENERATED ALWAYS AS … STORED` | `item_catalog.name_fold`, `region.name_fold` | a dobra é derivada, nunca digitada |
| `GENERATED … AS IDENTITY` | `region.id` | padrão SQL, sem `SERIAL` |
| índice parcial | `npc_definition (merchant) WHERE merchant > 0` | 549 de 6.100 linhas |
| tabela filha em vez de JSON | `guard_spawn_reward`, `game_matrix` | consultável por `values[1]`/coluna |
| `jsonb` | `generator_data`, `extras`, `mob_merc.items` | estrutura aberta/UNVERIFIED |
| `smallint[]` | `guard_spawn_reward.values`, `game_matrix.values` | bloco posicional sem semântica |

Descartado, com motivo — para não reabrir a discussão:

| Recurso | Motivo |
|---|---|
| `unaccent` | STABLE: não entra em índice nem em coluna gerada; exigiria wrapper + extensão para 29 letras que `translate()` resolve |
| `citext` | ignora caixa e **não** acento; o `UNIQUE` implícito dele é case-sensitive (armadilha documentada) |
| `pg_trgm` / trigram | 3.220 itens: varredura sequencial responde em menos de 1 ms; reavaliar acima de 100 mil |
| BRIN / particionamento | maior tabela tem 6.706 linhas |
| `ENUM` nativo | valor novo viraria migração de tipo; `item_kind`/`game_system` já dão FK + rótulo |
| FK `NOT VALID` em `item_index` | barraria item novo de loja sem catálogo no overlay |
| `SERIAL` | `IDENTITY` é o padrão SQL |
| migração única de conteúdo | 6 arquivos por domínio mantêm diff e rollback revisáveis |

Pendências que **não** são de arquitetura, são de entrega: carga tipada no boot
(FASE-04-3), `ContentError { tabela, pk, coluna, motivo }`, e o servidor logando
`content_version` + `content_source.sha256` para provar que duas máquinas têm o mesmo mundo.

## 8. Tabelas do legado sem espelho — decisão (2026-11-08)

Sete tabelas do schema legado (`origin/main`, `internal/migrations/0001_schema.up.sql`) estavam sem
tabela correspondente em `crates/wyd-db/migrations/` quando esta decisão foi tomada. A pergunta era
separar **requisito do jogo** de **operação do portal**. Critério: quem consome a tabela — o loop de
jogo (`tmserver` + `dbclient` + handler de protocolo) ou o caminho web/admin/comercial
(`internal/store/play_code.go`, `donate_revenue.go`, portal).

**Fontes de evidência usadas (e o que cada uma não prova):**

- **schema legado** (`internal/migrations/0001_schema.up.sql`, com a linha citada) — existência e
  forma da tabela; não diz quem usa.
- **código do servidor legado** (`tmserver/internal/store/*.go`, `dbclient/client.go`) — o consumidor
  real; é a prova que classifica jogo × portal.
- **nosso schema** (`crates/wyd-db/migrations/0028_estado_de_jogo.sql`) — o lado Rust da decisão, com
  `-- migrate:up`/`-- migrate:down` e `COMMENT ON TABLE` citando o nome legado.
- **nosso cliente `client/`** (o oficial) — não foi possível usar como prova aqui: nenhuma dessas
  sete tabelas aparece na superfície capturada do cliente (`docs/cliente/superficie-conhecida.md`),
  porque as sete são de **bookkeeping do servidor** (buff persistido, preço, idempotência de resgate,
  relação de clã, dono da torre, código de site, pedido de pagamento). O cliente consome o **efeito**
  (buff aplicado, item entregue, relação de clã desenhada), não a linha.
- **WYD Global (`blogal`, RaidHut)** — é cliente real em funcionamento, mas **terceiro**: serve para
  conferir comportamento observável, nunca como fonte de requisito do nosso servidor. Nenhuma linha
  abaixo depende dele.

**Estado aplicado:** as cinco de jogo são exatamente as cinco tabelas de
`0028_estado_de_jogo.sql`; as duas de portal **não** existem em `crates/wyd-db/migrations/`.

| Tabela legada | Classificação | Evidência (arquivo/consulta) | Implicação no nosso modelo |
|---|---|---|---|
| `affect` (168) | **requisito do jogo** | `loadAffects` e `SaveCharacter` em `store_live.go:234,388`; struct `affect[4][32]` em `savefmt.go:110` (offset 4572) | `character_affects` em `0028`, PK `(character_id, type)`, reidratado no login — nunca escrito por tick |
| `sapphire_balance` (1587) | **requisito do jogo** (nome enganoso: é **preço**, não saldo) | `kingdom_cape.go:25,56,61,103`; `dbclient.Client.QuoteKingdomCape/PurchaseKingdomCape` (`client.go:210,219`) | `kingdom_cape_price` em `0028`, linha única com `version` (revisão otimista) e custo 4..32 por reino |
| `daily_reward_claim` (529) | **requisito do jogo** | `dailyreward.go:17,23,205,233` — `unique_violation` 23505 era o sinal de "já resgatou" | `daily_reward_claim` em `0028`, `UNIQUE(account_id, claim_date)`; o item sai por `delivery_queue` |
| `guild_relation` (946) | **requisito do jogo** | `guild.go:351,361,395`; `dbclient` `:408,437` | `guild_relations` em `0028`, PK composta, `kind IN (1,2)`, CHECK contra auto-relação |
| `guild_tower_state` (967) | **requisito do jogo** | `guild.go:460,477`; `dbclient:475,484` (linha única `id=1`) | `guild_tower` em `0028`, linha única `CHECK (id = 1)`, `owner_guild = 0` = sem dono (sem FK: o sentinela não é clã) |
| `account_play_code` (149) | **exclusiva do portal legado** | só `store/play_code.go:22-73` (código de login de uso único, máx. 5 ativos); **zero** ocorrências em `tmserver` | fora do schema do jogo: se o portal precisar, persiste por ADR própria no Cold Path (`AGENTS.md` §3 invariante 7) |
| `donate_topup_order` (752) | **exclusiva do portal legado** (comercial) | `domain.go:408,437`; `donate_revenue.go:14-19` — recarga em dinheiro real, só vira receita depois do gateway | fora do schema do jogo, pelo mesmo motivo: dinheiro não é estado de personagem |

**Decisão.** As **5 primeiras entraram no `wyd-db`** (são estado de jogo que o tick precisa
reidratar) e as **2 últimas não entram**: código de login do site e pedido de recarga são do Cold
Path comercial e não têm relação com o estado do personagem — se o portal precisar persistí-los,
entra por ADR própria, fora do schema do jogo (`AGENTS.md` §3 invariante 7).

**Regra que encerra a ambiguidade de "sem espelho"** (vale para qualquer tabela legada daqui em
diante):

- **"Sem espelho" é um estado do schema, não um veredito.** Significa exatamente "existe em
  `internal/migrations/*.up.sql` e não existe em `crates/wyd-db/migrations/` **hoje**". Não significa
  "sem uso", "obsoleta" nem "fora do jogo" — era essa confusão que deixava as sete pendentes.
- **Entra no modelo (jogo)** quando houver consumidor **medido no caminho de jogo**: leitura/escrita
  em `tmserver/internal/{store,dbclient}` ou handler de protocolo que precise do estado reidratado,
  ou estado que o cliente observe após o login. Entrar = migration nova com `-- migrate:up` e
  `-- migrate:down` + entrada em `tools/content/banco-fontes.mjs` + seção neste documento. Sem
  consumidor medido, **não entra** por conveniência.
- **Fica fora (portal/Cold Path)** quando todos os consumidores estiverem no caminho web/admin/
  comercial. Se o portal precisar persistir, isso é decisão do portal (ADR própria) e não amplia o
  schema do jogo.
- **Sem evidência conclusiva** a tabela permanece **UNVERIFIED nomeada** (com a pergunta que a
  fecha, como U-P1..U-P7 no contrato de persistência): não vira tabela, não vira "portal por
  intuição" e não vira gate.
- **Saída do modelo** só por migration com `down` que reverte exatamente o `up` (nunca `DELETE` de
  dado como efeito colateral de decisão de modelagem).

**Nomes no Rust** (o legado tem dois nomes enganosos; o schema canônico é este documento):

| Legado | Nome no Rust | Por quê |
|---|---|---|
| `affect` | `character_affects` | é por personagem (`character_id`), não estado global |
| `sapphire_balance` | `kingdom_cape_price` | as colunas são `version`, `hekalotia_cost`, `akelonia_cost` — é **preço** com revisão otimista, não saldo de moeda |
| `daily_reward_claim` | `daily_reward_claim` | nome já correto; `UNIQUE(account_id, claim_date)` é o coração da idempotência |
| `guild_relation` | `guild_relations` | plural, como as demais |
| `guild_tower_state` | `guild_tower` | linha única (`id = 1` com `CHECK`), sem histórico no legado |

**Consequências:**

1. As 5 tabelas entraram em **`0028_estado_de_jogo.sql`** (DDL com `-- migrate:up` e
   `-- migrate:down`), com a exceção à ordem "DDL antes de seed" registrada na §6.5 e cobrada por
   `tests/catalogo.rs`.
2. `kingdom_cape_price` precisa de `version` (revisão otimista) para reproduzir
   `PurchaseKingdomCape(expectedRevision)` — sem isso duas compras concorrentes pagam o preço velho.
3. `character_affects` só faz sentido se o save do personagem for por agregado (contrato de
   persistência §7); gravar buff a cada tick é I/O dentro do tick, proibido (`AGENTS.md` §3
   invariante 1).
4. **Prova de ciclo:** `make migrations-apply` (up → down → up num Postgres efêmero) e
   `make banco-verify` (`information_schema` ↔ `banco-catalogo.md`); resultado registrado em
   `docs/rust/03-progresso.md`.

**Rastreabilidade:** `docs/arquitetura/contrato-persistencia.md` §5 (U-P7, agora resolvida),
`docs/rust/03-progresso.md` (registro de 2026-11-08).
