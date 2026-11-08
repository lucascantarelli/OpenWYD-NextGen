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

audit_log (append-only, sem FK para nunca bloquear escrita)
```

## 2. Tabelas e integridade

| Tabela | PK | FKs | CHECKs | Índices quentes |
|---|---|---|---|---|
| `accounts` | `id` | — | `role IN ('player','moderator','admin')`, `login` não-vazio | `UNIQUE(login)` |
| `characters` | `id` | `account_id → accounts` ON DELETE CASCADE | `slot 0–3`, `level ≥ 1`, `exp ≥ 0`, `hp ≥ 0`, `name` não-vazio | `UNIQUE(account_id, slot) WHERE deleted_at IS NULL`, `UNIQUE(name)`, `idx(account_id)` |
| `character_items` | `(character_id, slot)` | `character_id → characters` ON DELETE CASCADE | `slot 0–127`, `qty ≥ 1` | `idx(character_id)` (= PK) |
| `vault_items` | `(account_id, slot)` | `account_id → accounts` ON DELETE CASCADE | `slot 0–127`, `qty ≥ 1` | `idx(account_id)` (= PK) |
| `guilds` | `id` | `master_character_id → characters` DEFERRABLE | `name` não-vazio | `UNIQUE(name)` |
| `guild_members` | `(guild_id, character_id)` | ambas CASCADE | `rank 0–9` | `UNIQUE(character_id)` (um clã por char) |
| `audit_log` | `id` | — (nunca bloqueia) | `action` não-vazia | `idx(at)` |
| `delivery_queue` | `id` | `account_id/character_id` SET NULL (grant sobrevive a delete? não: RESTRICT — ver §4) | coerência por `kind` (item/gold/level) | parcial `WHERE status='pending'` |

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
- Runner `sqlx`/`refinery` chega na FASE-05/04-3 (exige rede + ADR); até lá a
  aplicação é por `psql -f` com split nos marcadores (script de validação em
  `tests/constraints.rs` + prova ao vivo documentada na issue).

## 5. Seeds

- `seeds/dev.sql` versionado: conta/personagem/clã de desenvolvimento com
  valores **obviamente falsos** (`DEV-ONLY`, nunca hash real).
- Produção **nunca** versiona seed com segredo (regra da árvore §12); o
  procedimento de seed prod vive no runbook de operação (FASE-05), não aqui.

## 6. Conteúdo estático (itens, skills, NPCs) — procedência e lacuna

Conteúdo de jogo é **dado de referência**, não estado de jogador: nunca entra em
`audit_log`, nunca é escrito pela web e nunca mistura linha de conta. O que a
árvore Rust tem hoje são só as tabelas de jogador (§2); o catálogo de conteúdo
ainda **não foi portado** — esta seção registra de onde ele vem, para a FASE-03-4
(seed + validação de schema no boot) e a FASE-10 (fórmulas) não inventarem dado.

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

1. Nenhuma migração Rust cria as tabelas de conteúdo acima (FASE-03-4/04).
2. Não existe tabela de **lore/quests** nem de **mix/refino** na árvore Rust; o
   legado tinha `QuestsRate.txt`, `RefineRate.txt`, `Mixlist` (cliente).
3. `docs/sistemas/*` (13 arquivos), `docs/lore/*` (3) e `docs/guias/*` (4)
   existem **apenas em `origin/main`**; a árvore atual perdeu os originais e
   mantém só `wyd-portal/src/content/guias.json` (19 slugs gerados) — o gerador
   `scripts/build-guias.mjs` não tem mais fonte para regenerar.
4. Texto de sistema em `docs/` hoje é só protocolo/cliente/arquitetura; não
   existe documento de combate/drop/refino na árvore Rust.
