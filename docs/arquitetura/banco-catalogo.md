# Catálogo do banco (`wyd-db`) — FASE-04

> **GERADO.** Não edite à mão: a fonte da verdade do schema é
> `crates/wyd-db/migrations/*.sql`; a fonte de cada decisão é `tools/content/banco-fontes.mjs`.
> Regenere com `make banco-catalogo`.
>
> Duas verificações prendem este documento ao banco:
> 1. `cargo test -p wyd-db --test catalogo` — documento ↔ migrations, sem Postgres;
> 2. `node tools/content/verify-schema.mjs` — `information_schema` de um Postgres real
>    aplicado do zero ↔ documento (tabela/coluna/tipo/nulidade/PK, sem sobra nem falta).

## 1. Resumo

- migrations: **31** · última versão: **31**
- tabelas: **69** · colunas: **519** · índices: **33** · funções: **1**
- domínios: **17** · tabelas com dado provisório: **24**

| Domínio | Tabelas |
|---|---|
| acesso | `accounts`, `character_session` |
| arena | `arena`, `arena_match`, `arena_round`, `arena_score` |
| auditoria | `audit_log` |
| canal de servidor | `server_channel` |
| catálogo | `content_version`, `game_system`, `game_class`, `skill_lineage`, `item_kind`, `item_effect`, `item_catalog`, `skill_catalog` |
| combate | `mob_template_stat`, `mob_template_equip`, `drop_rate_override`, `content_rate_override` |
| economia | `city`, `item_price`, `personal_shop`, `city_tax_ledger`, `donate_shop_item`, `daily_reward_item`, `kingdom_cape_price`, `daily_reward_claim` |
| eventos | `world_event`, `event_schedule`, `kefra_state`, `castle_state`, `event_participation` |
| inventário | `character_items`, `vault_items` |
| moderação | `account_moderation` |
| mundo | `content_source`, `npc_template`, `npc_definition`, `npc_shop_item`, `level_item`, `guard_spawn`, `guard_spawn_reward`, `mob_merc`, `game_setting`, `game_matrix`, `region`, `world_item_seed`, `guild_zone`, `map_definition`, `teleport_route` |
| personagem | `characters` |
| ponte web→jogo | `delivery_queue` |
| progressão | `character_status`, `character_lineage`, `character_skills`, `character_affects` |
| quests | `quest`, `quest_step`, `quest_reward`, `character_quest` |
| refino | `refine_rate`, `refine_gem_effect`, `sanc_rate`, `combine_value` |
| social | `guilds`, `guild_members`, `guild_relations`, `guild_tower` |

## 2. Catálogo

### `accounts`

- **Arquivo:** `0001_contas.sql` · **Domínio:** acesso · **Estado:** confirmado
- **Fonte:** docs/arquitetura/modelo-banco.md §2
- **Nota:** Hash de senha/PIN opaco (argon2id); nada em texto plano.
- **PK:** `id`
- **FK:** —
- **CHECK:** `accounts_login_nonempty`, `accounts_role`, `accounts_pin_failures`
- **UNIQUE de tabela:** —
- **Índices:** `accounts_login_unique` (login) UNIQUE ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `bigint` | não |
| `login` | `text` | não |
| `password_hash` | `text` | não |
| `pin_hash` | `text` | sim |
| `pin_failures` | `smallint` | não |
| `pin_locked_until` | `timestamp with time zone` | sim |
| `role` | `text` | não |
| `created_at` | `timestamp with time zone` | não |
| `last_login_at` | `timestamp with time zone` | sim |

### `characters`

- **Arquivo:** `0002_personagens.sql` · **Domínio:** personagem · **Estado:** confirmado
- **Fonte:** modelo-banco.md §2 + docs/wyd/progressao.md §1 (camadas e Arch)
- **Nota:** `tier`/`arch_of_id` vêm das camadas CONFIRMADAS de progressão (D-B02).
- **PK:** `id`
- **FK:** `account_id` → `accounts` · `arch_of_id` → `characters`
- **CHECK:** `characters_slot`, `characters_name_nonempty`, `characters_tier`, `characters_arch_not_self`, `characters_level`, `characters_exp`, `characters_hp`, `characters_mp`, `characters_gold`, `characters_hp_max`, `characters_mp_max`, `characters_pvp_points`
- **UNIQUE de tabela:** —
- **Índices:** `characters_account_idx` (account_id) ; · `characters_name_unique` (name) UNIQUE WHERE deleted_at IS NULL;

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `bigint` | não |
| `account_id` | `bigint` | não |
| `slot` | `smallint` | não |
| `name` | `text` | não |
| `class` | `smallint` | não |
| `tier` | `text` | não |
| `arch_of_id` | `bigint` | sim |
| `level` | `integer` | não |
| `exp` | `bigint` | não |
| `hp` | `integer` | não |
| `mp` | `integer` | não |
| `gold` | `bigint` | não |
| `hp_max` | `integer` | não |
| `mp_max` | `integer` | não |
| `pvp_points` | `integer` | não |
| `pos_x` | `integer` | não |
| `pos_y` | `integer` | não |
| `created_at` | `timestamp with time zone` | não |
| `updated_at` | `timestamp with time zone` | não |
| `deleted_at` | `timestamp with time zone` | sim |

### `character_items`

- **Arquivo:** `0003_inventario.sql` · **Domínio:** inventário · **Estado:** confirmado
- **Fonte:** modelo-banco.md §2 + docs/wyd/sistemas/refino.md §2–§3
- **Nota:** Refino/Anct/gema na criação da tabela: CHECK de gema espelha `Item` do core (D-B03).
- **PK:** `character_id`, `slot`
- **FK:** `character_id` → `characters`
- **CHECK:** `character_items_slot`, `character_items_qty`, `character_items_refine`, `character_items_gem_effect`, `character_items_gem_value`, `character_items_gem_pair`, `character_items_gem_refine`
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `character_id` | `bigint` | não |
| `slot` | `smallint` | não |
| `item_id` | `integer` | não |
| `qty` | `integer` | não |
| `flags` | `integer` | não |
| `refine` | `smallint` | não |
| `ancient` | `boolean` | não |
| `gem_effect` | `smallint` | não |
| `gem_value` | `smallint` | não |

### `vault_items`

- **Arquivo:** `0003_inventario.sql` · **Domínio:** inventário · **Estado:** confirmado
- **Fonte:** modelo-banco.md §2 + docs/wyd/sistemas/refino.md §2–§3
- **Nota:** Mesmas colunas e CHECKs de `character_items`, por conta.
- **PK:** `account_id`, `slot`
- **FK:** `account_id` → `accounts`
- **CHECK:** `vault_items_slot`, `vault_items_qty`, `vault_items_refine`, `vault_items_gem_effect`, `vault_items_gem_value`, `vault_items_gem_pair`, `vault_items_gem_refine`
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `account_id` | `bigint` | não |
| `slot` | `smallint` | não |
| `item_id` | `integer` | não |
| `qty` | `integer` | não |
| `flags` | `integer` | não |
| `refine` | `smallint` | não |
| `ancient` | `boolean` | não |
| `gem_effect` | `smallint` | não |
| `gem_value` | `smallint` | não |

### `guilds`

- **Arquivo:** `0004_guildas.sql` · **Domínio:** social · **Estado:** confirmado
- **Fonte:** modelo-banco.md §2
- **Nota:** `master_character_id` DEFERRABLE: cria clã e líder na mesma transação.
- **PK:** `id`
- **FK:** `master_character_id` → `characters`
- **CHECK:** `guilds_name_nonempty`
- **UNIQUE de tabela:** —
- **Índices:** `guilds_name_unique` (name) UNIQUE ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `bigint` | não |
| `name` | `text` | não |
| `master_character_id` | `bigint` | sim |
| `created_at` | `timestamp with time zone` | não |

### `guild_members`

- **Arquivo:** `0004_guildas.sql` · **Domínio:** social · **Estado:** confirmado
- **Fonte:** modelo-banco.md §2 + docs/wyd/sistemas/guerras.md §1
- **Nota:** Um clã por personagem (índice único) e rank 0–9.
- **PK:** `guild_id`, `character_id`
- **FK:** `guild_id` → `guilds` · `character_id` → `characters`
- **CHECK:** `guild_members_rank`
- **UNIQUE de tabela:** —
- **Índices:** `guild_members_character_unique` (character_id) UNIQUE ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `guild_id` | `bigint` | não |
| `character_id` | `bigint` | não |
| `rank` | `smallint` | não |
| `joined_at` | `timestamp with time zone` | não |

### `audit_log`

- **Arquivo:** `0005_auditoria.sql` · **Domínio:** auditoria · **Estado:** confirmado
- **Fonte:** modelo-banco.md §3.4
- **Nota:** Sem FK de propósito: auditoria não falha por integridade alheia.
- **PK:** `id`
- **FK:** —
- **CHECK:** `audit_log_action_nonempty`
- **UNIQUE de tabela:** —
- **Índices:** `audit_log_at_idx` (at) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `bigint` | não |
| `at` | `timestamp with time zone` | não |
| `actor` | `text` | não |
| `action` | `text` | não |
| `target` | `text` | não |
| `detail` | `text` | não |

### `delivery_queue`

- **Arquivo:** `0006_delivery_queue.sql` · **Domínio:** ponte web→jogo · **Estado:** confirmado
- **Fonte:** modelo-banco.md §3.3
- **Nota:** Única escrita do Cold Path no estado de jogo; o servidor drena.
- **PK:** `id`
- **FK:** `account_id` → `accounts` · `character_id` → `characters` · `created_by` → `accounts`
- **CHECK:** `delivery_queue_kind`, `delivery_queue_status`, `delivery_queue_attempts`, `delivery_queue_item_coherent`, `delivery_queue_gold_coherent`, `delivery_queue_level_coherent`
- **UNIQUE de tabela:** —
- **Índices:** `delivery_queue_pending_idx` (status) WHERE status = 'pending';

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `bigint` | não |
| `created_at` | `timestamp with time zone` | não |
| `kind` | `text` | não |
| `account_id` | `bigint` | sim |
| `character_id` | `bigint` | sim |
| `item_id` | `integer` | sim |
| `qty` | `integer` | sim |
| `gold_amount` | `bigint` | sim |
| `level_delta` | `integer` | sim |
| `status` | `text` | não |
| `attempts` | `integer` | não |
| `claimed_at` | `timestamp with time zone` | sim |
| `idempotency_key` | `text` | sim |
| `created_by` | `bigint` | sim |

### `content_version`

- **Arquivo:** `0007_conteudo_catalogo.sql` · **Domínio:** catálogo · **Estado:** confirmado
- **Fonte:** ADR-0009
- **Nota:** Uma linha (`id = 1`); cada seed de conteúdo avança a versão.
- **PK:** `id`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `smallint` | não |
| `version` | `integer` | não |
| `note` | `text` | não |
| `applied_at` | `timestamp with time zone` | não |

### `game_system`

- **Arquivo:** `0007_conteudo_catalogo.sql` · **Domínio:** catálogo · **Estado:** confirmado
- **Fonte:** ADR-0009 + docs/wyd/sistemas.md
- **Nota:** Taxonomia de sistemas do jogo (20 linhas no seed).
- **PK:** `code`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `code` | `text` | não |
| `label` | `text` | não |
| `category` | `text` | não |
| `description` | `text` | não |

### `game_class`

- **Arquivo:** `0007_conteudo_catalogo.sql` · **Domínio:** catálogo · **Estado:** confirmado
- **Fonte:** ADR-0009 + docs/wyd/classes.md
- **Nota:** Seis classes jogáveis.
- **PK:** `code`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `code` | `text` | não |
| `label` | `text` | não |
| `role` | `text` | não |
| `description` | `text` | não |

### `skill_lineage`

- **Arquivo:** `0007_conteudo_catalogo.sql` · **Domínio:** catálogo · **Estado:** confirmado
- **Fonte:** ADR-0009 + docs/wyd/classes.md §2
- **Nota:** Linhagem por classe (13 no seed).
- **PK:** `code`
- **FK:** `class_code` → `game_class`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `code` | `text` | não |
| `class_code` | `text` | não |
| `label` | `text` | não |
| `focus` | `text` | não |

### `item_kind`

- **Arquivo:** `0007_conteudo_catalogo.sql` · **Domínio:** catálogo · **Estado:** confirmado
- **Fonte:** docs/wyd/itens.md
- **Nota:** Categoria descritiva do item; 1.532 itens ficam sem categoria por falta de prova.
- **PK:** `code`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `code` | `text` | não |
| `label` | `text` | não |
| `slot` | `text` | sim |
| `description` | `text` | não |

### `item_effect`

- **Arquivo:** `0007_conteudo_catalogo.sql` · **Domínio:** catálogo · **Estado:** provisorio
- **Fonte:** docs/wyd/itens.md + ADR-0009
- **Nota:** O símbolo `EF_*` é a chave; o código numérico do `ItemEffect.h` não foi provado (`confirmed`).
- **PK:** `name`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `name` | `text` | não |
| `label` | `text` | não |
| `category` | `text` | não |
| `description` | `text` | não |
| `confirmed` | `boolean` | não |
| `first_seen` | `text` | não |

### `item_catalog`

- **Arquivo:** `0007_conteudo_catalogo.sql` · **Domínio:** catálogo · **Estado:** provisorio
- **Fonte:** ADR-0009 + modelo-banco.md §6.5
- **Nota:** 3.220 itens do legado; nome não é chave (488 grupos repetem) e `class_mask` é UNVERIFIED.
- **PK:** `item_index`
- **FK:** `kind` → `item_kind` · `primary_system` → `game_system`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** `item_catalog_effects_gin` (effect_names) USING GIN ; · `item_catalog_kind_idx` (kind) ; · `item_catalog_name_fold_idx` (name_fold) ; · `item_catalog_name_lower_idx` (lower(name) ); · `item_catalog_system_idx` (primary_system) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `item_index` | `integer` | não |
| `name` | `text` | não |
| `name_fold` | `text` | sim |
| `price` | `integer` | não |
| `grade` | `smallint` | não |
| `unique_id` | `smallint` | não |
| `extra` | `integer` | não |
| `pos` | `smallint` | não |
| `req_level` | `smallint` | não |
| `req_str` | `smallint` | não |
| `req_int` | `smallint` | não |
| `req_dex` | `smallint` | não |
| `req_con` | `smallint` | não |
| `volatile` | `smallint` | sim |
| `attack_range` | `smallint` | sim |
| `effects` | `jsonb` | não |
| `effect_names` | `text[]` | não |
| `fields` | `jsonb` | não |
| `kind` | `text` | sim |
| `primary_system` | `text` | sim |
| `event_only` | `boolean` | sim |
| `tradeable` | `boolean` | sim |
| `stackable` | `boolean` | sim |
| `class_mask` | `text` | sim |
| `classification_source` | `text` | não |
| `confidence` | `smallint` | sim |
| `notes` | `text` | não |
| `sha256` | `text` | não |
| `updated_at` | `timestamp with time zone` | não |

### `skill_catalog`

- **Arquivo:** `0007_conteudo_catalogo.sql` · **Domínio:** catálogo · **Estado:** provisorio
- **Fonte:** ADR-0009 + docs/wyd/skills.md
- **Nota:** 154 linhas; 3 têm `skill_index` provisório e isso fica declarado em `notes`.
- **PK:** `skill_index`
- **FK:** `class_code` → `game_class` · `lineage_code` → `skill_lineage` · `primary_system` → `game_system`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** `skill_catalog_class_idx` (class_code, lineage_code) ; · `skill_catalog_name_lower_idx` (lower(name) );

| Coluna | Tipo | Nulo |
|---|---|---|
| `skill_index` | `integer` | não |
| `name` | `text` | não |
| `skill_point` | `smallint` | não |
| `target_type` | `smallint` | não |
| `mana_spent` | `smallint` | não |
| `delay_ticks` | `smallint` | não |
| `range_tiles` | `smallint` | não |
| `instance_type` | `smallint` | não |
| `instance_value` | `smallint` | não |
| `tick_type` | `smallint` | não |
| `tick_value` | `smallint` | não |
| `affect_type` | `smallint` | não |
| `affect_value` | `smallint` | não |
| `affect_time` | `smallint` | não |
| `instance_attribute` | `smallint` | não |
| `tick_attribute` | `smallint` | não |
| `aggressive` | `smallint` | não |
| `max_target` | `smallint` | não |
| `party` | `smallint` | não |
| `affect_resist` | `smallint` | não |
| `passive` | `smallint` | não |
| `class_code` | `text` | sim |
| `lineage_code` | `text` | sim |
| `primary_system` | `text` | sim |
| `skill_role` | `text` | sim |
| `classification_source` | `text` | não |
| `confidence` | `smallint` | sim |
| `notes` | `text` | não |
| `updated_at` | `timestamp with time zone` | não |

### `content_source`

- **Arquivo:** `0008_conteudo_mundo.sql` · **Domínio:** mundo · **Estado:** confirmado
- **Fonte:** ADR-0009 + modelo-banco.md §6.7
- **Nota:** Proveniência por insumo (path/sha256/linhas); apagar o arquivo não muda o banco.
- **PK:** `path`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `path` | `text` | não |
| `sha256` | `text` | não |
| `size_bytes` | `integer` | não |
| `row_count` | `integer` | não |
| `loaded_by` | `text` | não |
| `transform` | `text` | não |
| `imported_at` | `timestamp with time zone` | não |

### `npc_template`

- **Arquivo:** `0008_conteudo_mundo.sql` · **Domínio:** mundo · **Estado:** provisorio
- **Fonte:** modelo-banco.md §6.7 (STRUCT_MOB 816 B)
- **Nota:** 20 nomes vieram de CP949 lido como latin1; a chave não é renomeada (ver §6.6).
- **PK:** `template_name`
- **FK:** —
- **CHECK:** `npc_template_recovered_needs_mojibake`
- **UNIQUE de tabela:** —
- **Índices:** `npc_template_source_size_idx` (source_size) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `template_name` | `text` | não |
| `file_name` | `text` | não |
| `source_size` | `smallint` | não |
| `sha256` | `text` | não |
| `blob` | `bytea` | não |
| `encoding_origin` | `text` | não |
| `name_recovered` | `text` | sim |

### `npc_definition`

- **Arquivo:** `0008_conteudo_mundo.sql` · **Domínio:** mundo · **Estado:** provisorio
- **Fonte:** modelo-banco.md §6.7 (NPCGener)
- **Nota:** `merchant` acima de 0 e `generator_data` são UNVERIFIED e não viram gate.
- **PK:** `id`
- **FK:** `template_name` → `npc_template`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** `npc_definition_generator_idx` (generator_index) ; · `npc_definition_map_idx` (map_id, pos_x, pos_y) ; · `npc_definition_merchant_idx` (merchant) WHERE merchant > 0; · `npc_definition_template_idx` (template_name) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `integer` | não |
| `slug` | `text` | não |
| `template_name` | `text` | não |
| `display_name` | `text` | não |
| `enabled` | `boolean` | não |
| `map_id` | `smallint` | não |
| `pos_x` | `smallint` | não |
| `pos_y` | `smallint` | não |
| `route_type` | `smallint` | não |
| `merchant` | `smallint` | não |
| `origin` | `text` | não |
| `generator_index` | `integer` | sim |
| `follower_template` | `text` | não |
| `minute_generate` | `integer` | não |
| `min_group` | `smallint` | não |
| `max_group` | `smallint` | não |
| `max_num_mob` | `smallint` | não |
| `formation` | `smallint` | não |
| `generator_data` | `jsonb` | não |

### `npc_shop_item`

- **Arquivo:** `0008_conteudo_mundo.sql` · **Domínio:** mundo · **Estado:** provisorio
- **Fonte:** modelo-banco.md §6.7
- **Nota:** Sem FK em `item_index`: 22 itens vendidos não existem no catálogo do legado.
- **PK:** `npc_id`, `slot`
- **FK:** `npc_id` → `npc_definition`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** `npc_shop_item_item_idx` (item_index) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `npc_id` | `integer` | não |
| `slot` | `smallint` | não |
| `item_index` | `integer` | não |
| `eff1` | `smallint` | não |
| `effv1` | `smallint` | não |
| `eff2` | `smallint` | não |
| `effv2` | `smallint` | não |
| `eff3` | `smallint` | não |
| `effv3` | `smallint` | não |
| `quantity` | `smallint` | não |

### `level_item`

- **Arquivo:** `0008_conteudo_mundo.sql` · **Domínio:** mundo · **Estado:** confirmado
- **Fonte:** modelo-banco.md §6.6 (`LevelItem.txt`)
- **Nota:** 161 linhas de conteúdo após deduplicar o bloco repetido; `type_id` documentado no insumo.
- **PK:** `ordinal`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** `level_item_item_idx` (item_index) ; · `level_item_level_class_idx` (level, class_id) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `ordinal` | `smallint` | não |
| `source_seq` | `smallint` | não |
| `source_line` | `integer` | não |
| `level` | `smallint` | não |
| `class_id` | `smallint` | não |
| `type_id` | `smallint` | não |
| `item_index` | `integer` | não |
| `extras` | `jsonb` | não |

### `guard_spawn`

- **Arquivo:** `0008_conteudo_mundo.sql` · **Domínio:** mundo · **Estado:** provisorio
- **Fonte:** modelo-banco.md §6.6 (`Guard.txt`)
- **Nota:** O bloco de 8 números é preservado cru (o cabeçalho descreve 13 campos).
- **PK:** `guard_index`
- **FK:** `npc_id` → `npc_definition`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `guard_index` | `smallint` | não |
| `npc_id` | `integer` | não |
| `source_line` | `integer` | não |

### `guard_spawn_reward`

- **Arquivo:** `0008_conteudo_mundo.sql` · **Domínio:** mundo · **Estado:** provisorio
- **Fonte:** modelo-banco.md §6.6 (`Guard.txt`)
- **Nota:** `values` posicional, sem semântica inventada.
- **PK:** `guard_index`, `req_slot`
- **FK:** `guard_index` → `guard_spawn`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `guard_index` | `smallint` | não |
| `req_slot` | `smallint` | não |
| `values` | `smallint[]` | não |

### `mob_merc`

- **Arquivo:** `0008_conteudo_mundo.sql` · **Domínio:** mundo · **Estado:** confirmado
- **Fonte:** modelo-banco.md §6.6 (`MobMerc.txt`)
- **Nota:** Uma linha no insumo atual.
- **PK:** `generator_index`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `generator_index` | `integer` | não |
| `template_name` | `text` | não |
| `renew_seconds` | `integer` | não |
| `reborn_seconds` | `integer` | não |
| `items` | `jsonb` | não |

### `game_setting`

- **Arquivo:** `0008_conteudo_mundo.sql` · **Domínio:** mundo · **Estado:** confirmado
- **Fonte:** modelo-banco.md §6.6 (`gameconfig.txt`)
- **Nota:** `evstart` tem 4 números: fica em `value_text` e `value_int` só existe com inteiro único.
- **PK:** `section`, `key`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `section` | `text` | não |
| `key` | `text` | não |
| `value_int` | `bigint` | sim |
| `value_text` | `text` | não |
| `source_line` | `integer` | não |

### `game_matrix`

- **Arquivo:** `0008_conteudo_mundo.sql` · **Domínio:** mundo · **Estado:** provisorio
- **Fonte:** modelo-banco.md §6.6 (`gameconfig.txt`)
- **Nota:** Matrizes (`Treasure`, `Item Drop Bonus`) preservadas na ordem do arquivo.
- **PK:** `section`, `block_index`, `line_index`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `section` | `text` | não |
| `block_index` | `smallint` | não |
| `line_index` | `smallint` | não |
| `values` | `smallint[]` | não |
| `source_line` | `integer` | não |

### `region`

- **Arquivo:** `0008_conteudo_mundo.sql` · **Domínio:** mundo · **Estado:** confirmado
- **Fonte:** modelo-banco.md §6.6 (`Regions.txt`)
- **Nota:** Nome repetido é legítimo (Erion/Armia têm retângulos diferentes).
- **PK:** `id`
- **FK:** —
- **CHECK:** `region_order`
- **UNIQUE de tabela:** —
- **Índices:** `region_name_fold_idx` (name_fold) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `smallint` | não |
| `name` | `text` | não |
| `name_fold` | `text` | sim |
| `x1` | `integer` | não |
| `y1` | `integer` | não |
| `x2` | `integer` | não |
| `y2` | `integer` | não |
| `source_line` | `integer` | não |

### `world_item_seed`

- **Arquivo:** `0008_conteudo_mundo.sql` · **Domínio:** mundo · **Estado:** confirmado
- **Fonte:** modelo-banco.md §6.1 (`seed_world_item_seed`)
- **Nota:** Itens no chão por mapa (49 linhas).
- **PK:** `id`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `integer` | não |
| `item_index` | `integer` | não |
| `pos_x` | `smallint` | não |
| `pos_y` | `smallint` | não |
| `rotate` | `smallint` | não |
| `enabled` | `boolean` | não |

### `guild_zone`

- **Arquivo:** `0008_conteudo_mundo.sql` · **Domínio:** mundo · **Estado:** confirmado
- **Fonte:** modelo-banco.md §6.1 (`seed_guild_zone`) + guerras.md §2
- **Nota:** Zonas de clã com imposto e desafio (5 linhas).
- **PK:** `clan`, `zone`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `clan` | `smallint` | não |
| `zone` | `smallint` | não |
| `victory` | `smallint` | não |
| `city_tax` | `smallint` | não |
| `tax_vault` | `smallint` | não |
| `charge_guild` | `smallint` | não |
| `challenge_guild` | `smallint` | não |
| `challenge_money` | `smallint` | não |

### `refine_rate`

- **Arquivo:** `0009_refino.sql` · **Domínio:** refino · **Estado:** provisorio
- **Fonte:** docs/wyd/sistemas/refino.md §2
- **Nota:** +11 não tem taxa publicada (furo declarado, não chutado).
- **PK:** `step`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `step` | `smallint` | não |
| `rate_bps` | `integer` | não |
| `kind` | `text` | não |
| `content_source` | `text` | não |
| `confidence` | `smallint` | não |
| `notes` | `text` | não |

### `refine_gem_effect`

- **Arquivo:** `0009_refino.sql` · **Domínio:** refino · **Estado:** confirmado
- **Fonte:** docs/wyd/sistemas/refino.md §3
- **Nota:** Quatro gemas (3386–3389), escala 40 × (nível − 9), drop 8% e EXP 2% fixos.
- **PK:** `item_index`
- **FK:** `item_index` → `item_catalog`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `item_index` | `integer` | não |
| `effect_code` | `text` | não |
| `flat_value` | `smallint` | não |
| `per_level_value` | `smallint` | não |
| `min_refine` | `smallint` | não |
| `max_refine` | `smallint` | não |
| `allows_ancient` | `boolean` | não |
| `content_source` | `text` | não |
| `confidence` | `smallint` | não |
| `notes` | `text` | não |

### `sanc_rate`

- **Arquivo:** `0009_refino.sql` · **Domínio:** refino · **Estado:** confirmado
- **Fonte:** origin/main:tmserver/content/Common/Settings/SancRate.txt + Basedef.cpp g_pSancRate
- **Nota:** 36 células efetivas: as 28 do arquivo (PO 0–5, PL 0–9, Âmago 0–11) e as 8 que o loader mantém do default compilado (PO 6–11, PL 10–11 = 0). O arquivo sobrescreve o default; ausência de linha não é ausência de taxa.
- **PK:** `powder`, `refine_index`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `powder` | `text` | não |
| `refine_index` | `smallint` | não |
| `percent` | `smallint` | não |
| `content_source` | `text` | não |
| `confidence` | `smallint` | não |
| `notes` | `text` | não |

### `combine_value`

- **Arquivo:** `0009_refino.sql` · **Domínio:** refino · **Estado:** provisorio
- **Fonte:** origin/main:tmserver/content/Common/Settings/CompRate.txt + internal/combine/*.go (g_pOdinRate)
- **Nota:** Taxa/custo por (família, chave) lidos pelos handlers de composição. Onde código e arquivo divergem (Celestial 35 × 5), `notes` diz qual vale — não há número inventado.
- **PK:** `family`, `key`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `family` | `text` | não |
| `key` | `text` | não |
| `value` | `integer` | não |
| `unit` | `text` | não |
| `content_source` | `text` | não |
| `confidence` | `smallint` | não |
| `notes` | `text` | não |

### `city`

- **Arquivo:** `0010_economia.sql` · **Domínio:** economia · **Estado:** provisorio
- **Fonte:** docs/wyd/economia.md + decisoes-servidor.md §3
- **Nota:** `map_id` nulo: o build 7662 não publica o mapa de cada cidade.
- **PK:** `code`
- **FK:** `region_id` → `region`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `code` | `text` | não |
| `name` | `text` | não |
| `map_id` | `smallint` | sim |
| `region_id` | `smallint` | sim |
| `is_capital` | `boolean` | não |
| `tax_rate_bps` | `integer` | não |
| `content_source` | `text` | não |

### `item_price`

- **Arquivo:** `0010_economia.sql` · **Domínio:** economia · **Estado:** confirmado
- **Fonte:** docs/wyd/economia.md
- **Nota:** Sobrepõe o preço do catálogo por NPC sem reescrever `item_catalog`.
- **PK:** `item_index`, `npc_id`
- **FK:** `item_index` → `item_catalog` · `npc_id` → `npc_definition`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `item_index` | `integer` | não |
| `npc_id` | `integer` | não |
| `buy_price` | `bigint` | não |
| `sell_price` | `bigint` | não |
| `content_source` | `text` | não |

### `personal_shop`

- **Arquivo:** `0010_economia.sql` · **Domínio:** economia · **Estado:** confirmado
- **Fonte:** docs/wyd/economia.md + decisoes-servidor.md §3
- **Nota:** Imposto de 10% da loja pessoal; `tax_paid` acumula o já recolhido.
- **PK:** `id`
- **FK:** `character_id` → `characters` · `city_code` → `city`
- **CHECK:** `personal_shop_closed_after_open`
- **UNIQUE de tabela:** —
- **Índices:** `personal_shop_city_open_idx` (city_code) WHERE is_open;

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `bigint` | não |
| `character_id` | `bigint` | não |
| `city_code` | `text` | não |
| `name` | `text` | não |
| `is_open` | `boolean` | não |
| `tax_bps` | `integer` | não |
| `tax_paid` | `bigint` | não |
| `opened_at` | `timestamp with time zone` | não |
| `closed_at` | `timestamp with time zone` | sim |

### `city_tax_ledger`

- **Arquivo:** `0010_economia.sql` · **Domínio:** economia · **Estado:** confirmado
- **Fonte:** docs/wyd/economia.md
- **Nota:** Livro-caixa por cidade; 40% do acumulado vai para Noatun.
- **PK:** `id`
- **FK:** `city_code` → `city` · `shop_id` → `personal_shop` · `collected_by` → `characters`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** `city_tax_ledger_city_at_idx` (city_code, at) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `bigint` | não |
| `city_code` | `text` | não |
| `shop_id` | `bigint` | sim |
| `amount` | `bigint` | não |
| `reason` | `text` | não |
| `collected_by` | `bigint` | sim |
| `at` | `timestamp with time zone` | não |

### `donate_shop_item`

- **Arquivo:** `0010_economia.sql` · **Domínio:** economia · **Estado:** confirmado
- **Fonte:** docs/wyd/economia.md
- **Nota:** Só crédito já conciliado; dinheiro real nunca entra aqui.
- **PK:** `id`
- **FK:** `item_index` → `item_catalog`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `integer` | não |
| `item_index` | `integer` | não |
| `price_credits` | `integer` | não |
| `duration_days` | `smallint` | sim |
| `enabled` | `boolean` | não |

### `daily_reward_item`

- **Arquivo:** `0010_economia.sql` · **Domínio:** economia · **Estado:** confirmado
- **Fonte:** docs/wyd/economia.md
- **Nota:** `day` 1..7 ou NULL para todo dia; item e/ou ouro.
- **PK:** `id`
- **FK:** `item_index` → `item_catalog`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `integer` | não |
| `day` | `smallint` | sim |
| `item_index` | `integer` | sim |
| `gold` | `bigint` | não |
| `qty` | `smallint` | não |
| `account_wide` | `boolean` | não |

### `map_definition`

- **Arquivo:** `0011_rotas_mundo.sql` · **Domínio:** mundo · **Estado:** provisorio
- **Fonte:** docs/wyd/sistemas/mundo.md
- **Nota:** Lista de mapas do 7662 é UNVERIFIED; a tabela recebe o dado, não o inventa.
- **PK:** `map_id`
- **FK:** `region_id` → `region`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `map_id` | `smallint` | não |
| `name` | `text` | não |
| `region_id` | `smallint` | sim |
| `is_pvp` | `boolean` | não |
| `is_safe` | `boolean` | não |
| `is_dungeon` | `boolean` | não |
| `content_source` | `text` | não |

### `teleport_route`

- **Arquivo:** `0011_rotas_mundo.sql` · **Domínio:** mundo · **Estado:** provisorio
- **Fonte:** docs/wyd/sistemas/mundo.md
- **Nota:** Rota sem `content_source` não entra: procedência é obrigatória.
- **PK:** `id`
- **FK:** `from_map_id` → `map_definition` · `to_map_id` → `map_definition` · `npc_id` → `npc_definition` · `req_item_index` → `item_catalog`
- **CHECK:** —
- **UNIQUE de tabela:** UNIQUE (from_map_id, from_x, from_y, to_map_id, to_x, to_y)
- **Índices:** `teleport_route_npc_idx` (npc_id) WHERE enabled;

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `integer` | não |
| `from_map_id` | `smallint` | não |
| `from_x` | `smallint` | não |
| `from_y` | `smallint` | não |
| `to_map_id` | `smallint` | não |
| `to_x` | `smallint` | não |
| `to_y` | `smallint` | não |
| `npc_id` | `integer` | sim |
| `cost_gold` | `integer` | não |
| `req_level` | `smallint` | não |
| `req_item_index` | `integer` | sim |
| `enabled` | `boolean` | não |
| `content_source` | `text` | não |
| `confidence` | `smallint` | não |

### `mob_template_stat`

- **Arquivo:** `0012_combate.sql` · **Domínio:** combate · **Estado:** provisorio
- **Fonte:** modelo-banco.md §6.1 + docs/wyd/sistemas/drops.md
- **Nota:** Vazia: stats vêm do `STRUCT_MOB` e a extração do struct ainda não foi feita.
- **PK:** `template_name`
- **FK:** `template_name` → `npc_template`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `template_name` | `text` | não |
| `level` | `smallint` | não |
| `max_hp` | `bigint` | não |
| `attack` | `integer` | não |
| `defense` | `integer` | não |
| `absorb` | `integer` | não |
| `magic_attack` | `integer` | não |
| `magic_defense` | `integer` | não |
| `exp_reward` | `bigint` | não |
| `gold_min` | `bigint` | não |
| `gold_max` | `bigint` | não |
| `content_source` | `text` | não |

### `mob_template_equip`

- **Arquivo:** `0012_combate.sql` · **Domínio:** combate · **Estado:** provisorio
- **Fonte:** modelo-banco.md §6.1
- **Nota:** Vazia pelo mesmo motivo de `mob_template_stat`.
- **PK:** `template_name`, `slot`
- **FK:** `template_name` → `npc_template` · `item_index` → `item_catalog`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `template_name` | `text` | não |
| `slot` | `smallint` | não |
| `item_index` | `integer` | não |

### `drop_rate_override`

- **Arquivo:** `0012_combate.sql` · **Domínio:** combate · **Estado:** confirmado
- **Fonte:** docs/wyd/sistemas/drops.md + decisoes-servidor.md §1
- **Nota:** Drop é PESO comparado ao roll 1..1.000.000, não probabilidade absoluta.
- **PK:** `template_name`, `item_index`
- **FK:** `template_name` → `npc_template` · `item_index` → `item_catalog`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `template_name` | `text` | não |
| `item_index` | `integer` | não |
| `weight` | `bigint` | não |
| `min_qty` | `smallint` | não |
| `max_qty` | `smallint` | não |
| `content_source` | `text` | não |
| `confidence` | `smallint` | não |

### `content_rate_override`

- **Arquivo:** `0012_combate.sql` · **Domínio:** combate · **Estado:** confirmado
- **Fonte:** docs/wyd/sistemas/drops.md
- **Nota:** Multiplicadores em bps (10000 = 100%).
- **PK:** `kind`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `kind` | `text` | não |
| `rate_bps` | `integer` | não |
| `content_source` | `text` | não |
| `notes` | `text` | não |

### `quest`

- **Arquivo:** `0013_quests.sql` · **Domínio:** quests · **Estado:** provisorio
- **Fonte:** docs/wyd/sistemas/quests.md
- **Nota:** Tabela vazia: a cadeia do 7662 não está fechada; seed seria invenção.
- **PK:** `code`
- **FK:** `class_code` → `game_class`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** `quest_kind_level_idx` (kind, min_level) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `code` | `text` | não |
| `name` | `text` | não |
| `kind` | `text` | não |
| `min_level` | `smallint` | não |
| `class_code` | `text` | sim |
| `repeatable` | `boolean` | não |
| `cooldown_seconds` | `integer` | não |
| `requires` | `text[]` | não |
| `description` | `text` | não |
| `content_source` | `text` | não |
| `confidence` | `smallint` | não |

### `quest_step`

- **Arquivo:** `0013_quests.sql` · **Domínio:** quests · **Estado:** provisorio
- **Fonte:** docs/wyd/sistemas/quests.md
- **Nota:** Objetivo tipado; posição obrigatória só em `reach`.
- **PK:** `quest_code`, `step`
- **FK:** `quest_code` → `quest` · `map_id` → `map_definition`
- **CHECK:** `quest_step_reach_needs_position`
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `quest_code` | `text` | não |
| `step` | `smallint` | não |
| `objective` | `text` | não |
| `target_id` | `integer` | sim |
| `count` | `integer` | não |
| `map_id` | `smallint` | sim |
| `pos_x` | `smallint` | sim |
| `pos_y` | `smallint` | sim |
| `description` | `text` | não |

### `quest_reward`

- **Arquivo:** `0013_quests.sql` · **Domínio:** quests · **Estado:** provisorio
- **Fonte:** docs/wyd/sistemas/quests.md
- **Nota:** Recompensa por tipo; item exige `item_index`.
- **PK:** `quest_code`, `kind`, `amount`, `item_index`
- **FK:** `quest_code` → `quest` · `item_index` → `item_catalog`
- **CHECK:** `quest_reward_item_needs_index`
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `quest_code` | `text` | não |
| `kind` | `text` | não |
| `item_index` | `integer` | não |
| `amount` | `bigint` | não |

### `character_quest`

- **Arquivo:** `0013_quests.sql` · **Domínio:** quests · **Estado:** confirmado
- **Fonte:** docs/wyd/sistemas/quests.md
- **Nota:** Escrita só do servidor; a web nunca toca progresso de personagem.
- **PK:** `character_id`, `quest_code`
- **FK:** `character_id` → `characters` · `quest_code` → `quest`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `character_id` | `bigint` | não |
| `quest_code` | `text` | não |
| `step` | `smallint` | não |
| `progress` | `integer` | não |
| `completed_at` | `timestamp with time zone` | sim |
| `started_at` | `timestamp with time zone` | não |

### `world_event`

- **Arquivo:** `0014_eventos.sql` · **Domínio:** eventos · **Estado:** confirmado
- **Fonte:** docs/wyd/sistemas/eventos.md + decisoes-servidor.md §4
- **Nota:** Kefra, Noatun, guerra de cidades e RvR.
- **PK:** `code`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `code` | `text` | não |
| `name` | `text` | não |
| `kind` | `text` | não |
| `enabled` | `boolean` | não |
| `requires_city` | `boolean` | não |
| `description` | `text` | não |
| `content_source` | `text` | não |

### `event_schedule`

- **Arquivo:** `0014_eventos.sql` · **Domínio:** eventos · **Estado:** confirmado
- **Fonte:** docs/wyd/decisoes-servidor.md §4
- **Nota:** `timezone` explícito para o evento não andar quando o host muda de fuso.
- **PK:** `id`
- **FK:** `event_code` → `world_event`
- **CHECK:** —
- **UNIQUE de tabela:** UNIQUE (event_code, weekday, start_time)
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `integer` | não |
| `event_code` | `text` | não |
| `weekday` | `smallint` | não |
| `start_time` | `time without time zone` | não |
| `duration_minutes` | `smallint` | não |
| `timezone` | `text` | não |
| `every_n_weeks` | `smallint` | não |
| `enabled` | `boolean` | não |

### `kefra_state`

- **Arquivo:** `0014_eventos.sql` · **Domínio:** eventos · **Estado:** confirmado
- **Fonte:** docs/wyd/sistemas/eventos.md
- **Nota:** Uma linha (`id = 1`); Kefra semanal.
- **PK:** `id`
- **FK:** `last_guild_id` → `guilds`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `smallint` | não |
| `last_kill_at` | `timestamp with time zone` | sim |
| `next_spawn_at` | `timestamp with time zone` | sim |
| `kill_count` | `integer` | não |
| `last_guild_id` | `bigint` | sim |

### `castle_state`

- **Arquivo:** `0014_eventos.sql` · **Domínio:** eventos · **Estado:** confirmado
- **Fonte:** docs/wyd/sistemas/guerras.md §3
- **Nota:** Dono do castelo, cidade e cofre; cascata cidade→castelo roda no `SETTLED`.
- **PK:** `id`
- **FK:** `owner_guild_id` → `guilds` · `city_code` → `city`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `smallint` | não |
| `owner_guild_id` | `bigint` | sim |
| `city_code` | `text` | sim |
| `vault_gold` | `bigint` | não |
| `last_collected_at` | `timestamp with time zone` | sim |
| `last_war_at` | `timestamp with time zone` | sim |

### `event_participation`

- **Arquivo:** `0014_eventos.sql` · **Domínio:** eventos · **Estado:** confirmado
- **Fonte:** docs/wyd/sistemas/eventos.md
- **Nota:** Histórico por evento/personagem/clã.
- **PK:** `id`
- **FK:** `event_code` → `world_event` · `character_id` → `characters` · `guild_id` → `guilds`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** `event_participation_event_at_idx` (event_code, at) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `bigint` | não |
| `event_code` | `text` | não |
| `character_id` | `bigint` | sim |
| `guild_id` | `bigint` | sim |
| `result` | `text` | não |
| `score` | `bigint` | não |
| `at` | `timestamp with time zone` | não |

### `arena`

- **Arquivo:** `0015_arena.sql` · **Domínio:** arena · **Estado:** confirmado
- **Fonte:** docs/wyd/sistemas/battle-royale.md
- **Nota:** Arena Real e Battle Royale (2 equipes por round).
- **PK:** `code`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `code` | `text` | não |
| `name` | `text` | não |
| `kind` | `text` | não |
| `team_size` | `smallint` | não |
| `rounds` | `smallint` | não |
| `round_ms` | `integer` | não |
| `respawns` | `smallint` | não |
| `min_level` | `smallint` | não |
| `no_drop` | `boolean` | não |
| `enabled` | `boolean` | não |
| `content_source` | `text` | não |

### `arena_match`

- **Arquivo:** `0015_arena.sql` · **Domínio:** arena · **Estado:** confirmado
- **Fonte:** docs/wyd/sistemas/battle-royale.md
- **Nota:** Partida é histórico, não estado vivo.
- **PK:** `id`
- **FK:** `arena_code` → `arena`
- **CHECK:** `arena_match_ends_after_start`
- **UNIQUE de tabela:** —
- **Índices:** `arena_match_arena_started_idx` (arena_code, started_at DESC) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `bigint` | não |
| `arena_code` | `text` | não |
| `started_at` | `timestamp with time zone` | não |
| `ended_at` | `timestamp with time zone` | sim |
| `winner_team` | `smallint` | sim |

### `arena_round`

- **Arquivo:** `0015_arena.sql` · **Domínio:** arena · **Estado:** confirmado
- **Fonte:** docs/wyd/sistemas/battle-royale.md
- **Nota:** Round com desempate próprio.
- **PK:** `match_id`, `round_no`
- **FK:** `match_id` → `arena_match`
- **CHECK:** `arena_round_ends_after_start`
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `match_id` | `bigint` | não |
| `round_no` | `smallint` | não |
| `started_at` | `timestamp with time zone` | não |
| `ended_at` | `timestamp with time zone` | sim |
| `winner_team` | `smallint` | sim |
| `is_tiebreak` | `boolean` | não |

### `arena_score`

- **Arquivo:** `0015_arena.sql` · **Domínio:** arena · **Estado:** confirmado
- **Fonte:** docs/wyd/sistemas/battle-royale.md
- **Nota:** Placar por jogador/round; kills/dano nascem do combate validado no servidor.
- **PK:** `match_id`, `round_no`, `character_id`
- **FK:** `character_id` → `characters` · `match_id, round_no` → `arena_round`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** `arena_score_character_idx` (character_id) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `match_id` | `bigint` | não |
| `round_no` | `smallint` | não |
| `character_id` | `bigint` | não |
| `team` | `smallint` | não |
| `kills` | `smallint` | não |
| `assists` | `smallint` | não |
| `damage` | `bigint` | não |
| `deaths` | `smallint` | não |

### `character_status`

- **Arquivo:** `0016_progressao.sql` · **Domínio:** progressão · **Estado:** provisorio
- **Fonte:** docs/wyd/progressao.md §2–§3 (pontos, atributos, fama)
- **Nota:** Nomes de coluna são nossos (a fonte descreve o dado, não o schema); regra 6/2 por nível até 200.
- **PK:** `character_id`
- **FK:** `character_id` → `characters`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `character_id` | `bigint` | não |
| `attr_str` | `integer` | não |
| `attr_int` | `integer` | não |
| `attr_dex` | `integer` | não |
| `attr_con` | `integer` | não |
| `points_status` | `integer` | não |
| `points_learn` | `integer` | não |
| `points_skill` | `integer` | não |
| `fame` | `bigint` | não |
| `updated_at` | `timestamp with time zone` | não |

### `character_lineage`

- **Arquivo:** `0016_progressao.sql` · **Domínio:** progressão · **Estado:** provisorio
- **Fonte:** docs/wyd/progressao.md §3 (maestria 200/255/320)
- **Nota:** CHECK cobre o maior teto conhecido; o teto por classe é regra do core (D-B05).
- **PK:** `character_id`, `lineage_code`
- **FK:** `character_id` → `characters` · `lineage_code` → `skill_lineage`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `character_id` | `bigint` | não |
| `lineage_code` | `text` | não |
| `mastery` | `smallint` | não |

### `character_skills`

- **Arquivo:** `0016_progressao.sql` · **Domínio:** progressão · **Estado:** provisorio
- **Fonte:** docs/wyd/skills.md §1 + docs/wyd/classes.md §2
- **Nota:** `skill_index` sem FK (catálogo parcial) e limite de 25 skills é regra, não CHECK (D-B01).
- **PK:** `character_id`, `skill_index`
- **FK:** `character_id` → `characters`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** `character_skills_skill_idx` (skill_index) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `character_id` | `bigint` | não |
| `skill_index` | `integer` | não |
| `mastery` | `smallint` | não |
| `learned_at` | `timestamp with time zone` | não |

### `character_affects`

- **Arquivo:** `0028_estado_de_jogo.sql` · **Domínio:** progressão · **Estado:** provisorio
- **Fonte:** modelo-banco.md §8 + legado `affect` (savefmt.go offset 4572)
- **Nota:** Semântica de `type`/`time` vem do struct do cliente; a faixa não foi provada (UNVERIFIED).
- **PK:** `character_id`, `type`
- **FK:** `character_id` → `characters`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `character_id` | `bigint` | não |
| `type` | `smallint` | não |
| `value` | `smallint` | não |
| `level` | `integer` | não |
| `time` | `bigint` | não |

### `kingdom_cape_price`

- **Arquivo:** `0028_estado_de_jogo.sql` · **Domínio:** economia · **Estado:** confirmado
- **Fonte:** modelo-banco.md §8 + legado `sapphire_balance` (kingdom_cape.go)
- **Nota:** Linha única: preço por reino (CHECK 4..32) e `version` para concorrência otimista.
- **PK:** `id`
- **FK:** —
- **CHECK:** `kingdom_cape_price_singleton`
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `smallint` | não |
| `version` | `bigint` | não |
| `hekalotia_cost` | `integer` | não |
| `akelonia_cost` | `integer` | não |

### `daily_reward_claim`

- **Arquivo:** `0028_estado_de_jogo.sql` · **Domínio:** economia · **Estado:** confirmado
- **Fonte:** modelo-banco.md §8 + legado `daily_reward_claim` (dailyreward.go)
- **Nota:** Um resgate por conta e dia UTC (UNIQUE); o item sai por delivery_queue.
- **PK:** `id`
- **FK:** `account_id` → `accounts`
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `bigint` | não |
| `account_id` | `bigint` | não |
| `reward_item_id` | `bigint` | sim |
| `claim_date` | `date` | não |
| `created_at` | `timestamp with time zone` | não |

### `guild_relations`

- **Arquivo:** `0028_estado_de_jogo.sql` · **Domínio:** social · **Estado:** confirmado
- **Fonte:** modelo-banco.md §8 + legado `guild_relation` (guild.go)
- **Nota:** `kind` 1/2 (aliado/hostil), sem auto-relação.
- **PK:** `guild_id`, `target_guild_id`
- **FK:** `guild_id` → `guilds` · `target_guild_id` → `guilds`
- **CHECK:** `guild_relations_kind`, `guild_relations_distinct`
- **UNIQUE de tabela:** —
- **Índices:** `guild_relations_target_idx` (target_guild_id) ;

| Coluna | Tipo | Nulo |
|---|---|---|
| `guild_id` | `bigint` | não |
| `target_guild_id` | `bigint` | não |
| `kind` | `smallint` | não |
| `updated_at` | `timestamp with time zone` | não |

### `guild_tower`

- **Arquivo:** `0028_estado_de_jogo.sql` · **Domínio:** social · **Estado:** confirmado
- **Fonte:** modelo-banco.md §8 + legado `guild_tower_state` (guild.go)
- **Nota:** Linha única; `owner_guild` 0 = sem dono (por isso sem FK).
- **PK:** `id`
- **FK:** —
- **CHECK:** `guild_tower_singleton`
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `smallint` | não |
| `owner_guild` | `bigint` | não |
| `updated_at_unix` | `bigint` | não |

### `server_channel`

- **Arquivo:** `0029_canal_de_servidor.sql` · **Domínio:** canal de servidor · **Estado:** confirmado
- **Fonte:** ADR-0022 §7 + docs/wyd/canais-e-cidadania.md §5 + tools/client/serverlist-servers.txt
- **Nota:** Canal 0 = URL de status (≤ 32 chars); 1..10 = host que o cliente disca no 8281 fixo. Coloracao verde/normal e papel do processo, nao da linha: o mesmo host pode ser verde em outro grupo.
- **PK:** `group_id`, `channel`
- **FK:** —
- **CHECK:** —
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `group_id` | `smallint` | não |
| `channel` | `smallint` | não |
| `host` | `text` | não |
| `status_url` | `text` | não |
| `enabled` | `boolean` | não |
| `note` | `text` | não |

### `character_session`

- **Arquivo:** `0030_sessao_unica.sql` · **Domínio:** acesso · **Estado:** provisorio
- **Fonte:** docs/wyd/canais-e-cidadania.md §5 + migration 0030
- **Nota:** Cerca de sessão por `account_login` (a autenticação da FASE-08-2 troca por account_id/character_id); `renewed_at` é o sinal de vida que retoma sessão órfã.
- **PK:** `id`
- **FK:** `character_id` → `characters`
- **CHECK:** `character_session_release_after_open`
- **UNIQUE de tabela:** —
- **Índices:** —

| Coluna | Tipo | Nulo |
|---|---|---|
| `id` | `bigint` | não |
| `account_login` | `text` | não |
| `character_id` | `bigint` | sim |
| `group_id` | `smallint` | não |
| `channel` | `smallint` | não |
| `client_version` | `integer` | não |
| `opened_at` | `timestamp with time zone` | não |
| `renewed_at` | `timestamp with time zone` | não |
| `released_at` | `timestamp with time zone` | sim |

### `account_moderation`

- **Arquivo:** `0031_moderacao.sql` · **Domínio:** moderação · **Estado:** confirmado
- **Fonte:** pre-release-rust.md FASE 7 §7.2
- **Nota:** Bloqueio com motivo + autor; recusado no login do portal e do jogo.
- **PK:** `account_id`
- **FK:** `account_id` → `accounts` · `blocked_by` → `accounts`
- **CHECK:** `account_moderation_reason`, `account_moderation_blocked_at`
- **UNIQUE de tabela:** —
- **Índices:** `account_moderation_blocked_idx` (account_id) WHERE blocked;

| Coluna | Tipo | Nulo |
|---|---|---|
| `account_id` | `bigint` | não |
| `blocked` | `boolean` | não |
| `reason` | `text` | não |
| `blocked_by` | `bigint` | sim |
| `blocked_at` | `timestamp with time zone` | sim |

## 3. Decisões provisórias (D-Bxx)

Mesmo padrão do registro `D-xx` de `docs/arquitetura/matematica-sim.md`: decisão, escolha,
por quê e como confirmar. `trace` vazio significa decisão de organização, não de semântica.

| ID | Decisão | Escolha | Por quê | Como confirmar |
|---|---|---|---|---|
| D-B01 | `skill_index`/`item_index` referencia o catálogo sem FK? | sem FK em `character_skills`, `npc_shop_item`, `level_item` | o catálogo cobre 154 dos 248 índices de skill e 22 itens vendidos não existem nele; FK barraria dado legítimo do cliente | fechar a cobertura do catálogo pelo `SkillData.bin`/`ItemList.bin` do 7662 |
| D-B02 | Onde mora a camada de progressão (Mortal→Arch→God2)? | `characters.tier` (CHECK de 6 valores) + `characters.arch_of_id` DEFERRABLE | as camadas são CONFIRMADAS em progressao.md §1 e o vínculo Mortal↔Arch é 1:1 opcional; DEFERRABLE permite criar os dois na mesma transação | trace de transformação Arch no cliente para provar o momento da troca de `tier` |
| D-B03 | Refino/Anct/gema como colunas do item | colunas na criação de `character_items`/`vault_items` com CHECK de par efeito/valor e de +10-exceto-Anct | é estado da linha, não do catálogo (refino.md §2–§3); o CHECK espelha `Item` do core e impede linha inconsistente | inventário do cliente após refinar/aplicar gema (D-12/D-13 de matematica-sim.md) |
| D-B04 | Tabelas `npc_config_meta`, `world_event_config`, `world_event_meta` do legado | consolidadas em `content_source` + colunas tipadas de `world_event`/`event_schedule` | eram metadado de import; `content_source` já guarda path/sha256/linhas e o evento virou tabela com semântica | — |
| D-B05 | Tabelas `item_effect_override`/`item_effect_audit` do legado | pendentes (não modeladas) | `item_effect.confirmed` cobre o símbolo, não o override por servidor; sem captura do efeito numérico não há coluna honesta | derivar código EF_* do cliente e então modelar override com auditoria |
| D-B06 | Tabela `castle_quest_state` do legado | pendente; `castle_state` + `quest` cobrem dono/cofre e quest genérica, não progresso de quest de castelo | a cadeia de quest do castelo não está fechada em docs/wyd/sistemas/guerras.md §3 | fechar a cadeia e então modelar estado por castelo |
| D-B07 | Tabela `mob_carry_override` do legado | pendente; `drop_rate_override` cobre peso de drop, não item carregado | sem o layout de `STRUCT_MOB` extraído, nomear colunas seria invenção | extrair o struct na FASE-10 e unificar com `mob_template_equip` |
| D-B08 | Estado de personagem do legado ainda não modelado (chaos/PK, correio, macro/hotbar) | fora do schema até haver fonte | nenhum documento da árvore atual descreve as regras desses sistemas | capturar pacote do cliente e documentar antes de criar tabela |

## 4. Mapa legado → novo

Tabelas do schema Go antigo citadas em `modelo-banco.md` §6.1 e onde elas foram parar. O que
não virou tabela está declarado como pendência com ID, nunca sumiu em silêncio.

| Tabela legada | Destino |
|---|---|
| `item` | absorvido por `item_catalog` (catálogo único por `item_index`) |
| `item_catalog` | `item_catalog` |
| `item_price` | `item_price` |
| `item_effect_override` | pendente — D-B05 |
| `item_effect_audit` | pendente — D-B05 (evento de auditoria vai em `audit_log`) |
| `skill_catalog` | `skill_catalog` |
| `npc_template` | `npc_template` |
| `npc_definition` | `npc_definition` |
| `npc_shop_item` | `npc_shop_item` |
| `npc_config_meta` | consolidado em `content_source` — D-B04 |
| `mob_template_stat` | `mob_template_stat` (vazia até extrair o `STRUCT_MOB`) |
| `mob_template_equip` | `mob_template_equip` |
| `mob_carry_override` | pendente — D-B07 |
| `drop_rate_override` | `drop_rate_override` |
| `content_rate_override` | `content_rate_override` |
| `teleport_route` | `teleport_route` |
| `daily_reward_item` | `daily_reward_item` |
| `donate_shop_item` | `donate_shop_item` |
| `world_item_seed` | `world_item_seed` |
| `guild_zone` | `guild_zone` |
| `castle_quest_state` | pendente — D-B06 |
| `kefra_state` | `kefra_state` |
| `world_event_config` | consolidado em `world_event` — D-B04 |
| `world_event_meta` | consolidado em `world_event` + `event_schedule` — D-B04 |

## 5. Fila de migrations

| # | Arquivo | Tipo | Objetos |
|---|---|---|---|
| 0001 | `0001_contas.sql` | DDL | `accounts` |
| 0002 | `0002_personagens.sql` | DDL | `characters` |
| 0003 | `0003_inventario.sql` | DDL | `character_items`, `vault_items` |
| 0004 | `0004_guildas.sql` | DDL | `guilds`, `guild_members` |
| 0005 | `0005_auditoria.sql` | DDL | `audit_log` |
| 0006 | `0006_delivery_queue.sql` | DDL | `delivery_queue` |
| 0007 | `0007_conteudo_catalogo.sql` | DDL | `content_version`, `game_system`, `game_class`, `skill_lineage`, `item_kind`, `item_effect`, `item_catalog`, `skill_catalog`, `content_fold()` |
| 0008 | `0008_conteudo_mundo.sql` | DDL | `content_source`, `npc_template`, `npc_definition`, `npc_shop_item`, `level_item`, `guard_spawn`, `guard_spawn_reward`, `mob_merc`, `game_setting`, `game_matrix`, `region`, `world_item_seed`, `guild_zone` |
| 0009 | `0009_refino.sql` | DDL | `refine_rate`, `refine_gem_effect`, `sanc_rate`, `combine_value` |
| 0010 | `0010_economia.sql` | DDL | `city`, `item_price`, `personal_shop`, `city_tax_ledger`, `donate_shop_item`, `daily_reward_item` |
| 0011 | `0011_rotas_mundo.sql` | DDL | `map_definition`, `teleport_route` |
| 0012 | `0012_combate.sql` | DDL | `mob_template_stat`, `mob_template_equip`, `drop_rate_override`, `content_rate_override` |
| 0013 | `0013_quests.sql` | DDL | `quest`, `quest_step`, `quest_reward`, `character_quest` |
| 0014 | `0014_eventos.sql` | DDL | `world_event`, `event_schedule`, `kefra_state`, `castle_state`, `event_participation` |
| 0015 | `0015_arena.sql` | DDL | `arena`, `arena_match`, `arena_round`, `arena_score` |
| 0016 | `0016_progressao.sql` | DDL | `character_status`, `character_lineage`, `character_skills` |
| 0017 | `0017_seed_taxonomia.sql` | seed | — |
| 0018 | `0018_seed_item_catalog.sql` | seed | — |
| 0019 | `0019_seed_skill_catalog.sql` | seed | — |
| 0020 | `0020_seed_mundo.sql` | seed | — |
| 0021 | `0021_seed_skill_lineage.sql` | seed | — |
| 0022 | `0022_seed_refino.sql` | seed | — |
| 0023 | `0023_seed_mundo_fixo.sql` | seed | — |
| 0024 | `0024_seed_skill_catalog_kersef.sql` | seed | — |
| 0025 | `0025_seed_sanc_rate.sql` | seed | — |
| 0026 | `0026_seed_sanc_rate_defaults.sql` | seed | — |
| 0027 | `0027_seed_combine.sql` | seed | — |
| 0028 | `0028_estado_de_jogo.sql` | DDL | `character_affects`, `kingdom_cape_price`, `daily_reward_claim`, `guild_relations`, `guild_tower` |
| 0029 | `0029_canal_de_servidor.sql` | DDL | `server_channel` |
| 0030 | `0030_sessao_unica.sql` | DDL | `character_session` |
| 0031 | `0031_moderacao.sql` | DDL | `account_moderation` |

Regra da série: DDL inteiro antes do primeiro seed, um domínio por arquivo, nenhuma tabela
partida entre arquivos, nenhuma FK para tabela criada depois e **nenhum `ALTER TABLE` no
`up`** — coluna nova nasce na criação da tabela (o gerador falha se aparecer um).
