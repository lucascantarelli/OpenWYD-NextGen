# Banco de dados — auditoria e baseline

> Auditoria de 04/11 com `scripts/db-audit.sql` (somente leitura) contra o banco de dev: **50 tabelas**,
> 59 migrations aplicadas, 15.059 linhas de referência. Este documento é o diagnóstico que a fase 4 da
> issue 58 usa para reescrever o esquema — cada afirmação abaixo tem a consulta correspondente no
> script, para poder ser reconferida a qualquer momento.

## 1. Veredito curto

O banco está **estruturalmente saudável** e acima da média de projeto pequeno: 50/50 tabelas com PK,
24 FKs todas válidas e com ação definida (`cascade` em dado de jogador, `set null` em ponteiro de
auditoria), 62 CHECKs, 57 UNIQUEs, `timestamptz` em todo campo de tempo, `jsonb` onde a forma varia,
`bytea` só para blob binário. Não há tabela órfã, não há coluna sem tipo, não há dado de jogador em
seed.

O que está **abaixo** do padrão que o projeto merece: padronização de tipos entre tabelas que
representam a mesma coisa (16 nomes com mais de um tipo), cobertura de comentários (17/50 tabelas e
**3 colunas** no banco inteiro), oito FKs sem índice na origem, 22 índices sem uma única leitura,
ausência de backup automatizado e — o maior de todos — **um único superusuário `postgres` para todos
os serviços**.

## 2. Estrutura e normalização

- **PK**: 50/50. Nenhuma tabela sem chave. ✔
- **FK**: 24, todas `convalidated = t`. Ações: `cascade` para o que pertence ao jogador
  (`character → account`, `item → character`, `affect`, `delivery_queue`, `guild_member`…),
  `set null` para `updated_by`/`updated_by` de catálogo. ✔
- **Forma normal**: 3NF na prática. `item` guarda efeitos como colunas (`eff1/effv1…`) porque espelha
  `STRUCT_ITEM` de 8 bytes — é decisão de paridade de fio, não desnormalização acidental. `character`
  tem 75 colunas pelo mesmo motivo (layout MSVC x86). O que **não** é aceitável é `eff*` ter dois
  tipos diferentes (ver §4).
- **Catálogo de conteúdo (issue 58)**: `item_catalog.fields` guarda a linha crua em `jsonb` **de
  propósito** — é o que torna o import lossless enquanto as colunas derivadas servem ao runtime. Isso
  é staging documentado, não modelagem preguiçosa: quando o conjunto de colunas usado estabilizar,
  `fields` pode sair.

## 3. Índices

| Achado | Número | Detalhe |
|---|---:|---|
| Índices nunca lidos (`idx_scan = 0`, fora de constraint) | **22** | 19 são de tabelas de auditoria e de override ainda sem leitor; 3 são dos catálogos novos |
| FK de coluna única sem índice na origem | **8** | `guild_relation.target_guild_id` (join real), `daily_reward_claim.reward_item_id`, e 6 `updated_by` que o `SET NULL` varre no delete do pai |
| Possível duplicidade | 2 | `account(name)` tem UNIQUE + prefixo; `skill_catalog` são dois índices de **expressão** — o audit agrupa por `indkey` e expressão aparece como `0`, então esse par é falso positivo por construção |
| Seq scan em tabela grande | 139 | `npc_definition`: é o snapshot inteiro lido no boot, não falta de índice |

Decisões para o baseline:

- **Manter** os índices dos catálogos (`item_catalog_name_idx`, `skill_catalog_name_idx`): o portal vai
  lê-los; "nunca usado" hoje é ausência de consumidor, não de caso de uso. Documentar no comentário.
- **Adicionar** índice em `guild_relation.target_guild_id` e nos `updated_by` que têm `SET NULL` —
  sem eles, apagar uma conta varre a tabela filha inteira.
- **Não** criar índice "preventivo" novo. O critério é leitor identificado.

## 4. Padronização (o achado mais acionável)

16 nomes de coluna têm mais de um tipo no banco. Os que importam:

| Coluna | Tipos hoje | Canônico | Por quê |
|---|---|---|---|
| `item_index` | `integer`, `smallint` | **integer** | índice do catálogo é `int` em `STRUCT_ITEMLIST`; `smallint` estoura em 32k e há 3.220 itens com espaço |
| `eff1..eff3`, `effv1..effv3` | `integer`, `smallint` | **smallint** | são `short` no fio (`STRUCT_ITEM`), 8 bytes totais |
| `pos_x`, `pos_y` | `integer`, `smallint` | **smallint** | grid 1024/4096 cabe em 16 bits |
| `price` | `integer`, `bigint` | **bigint** em `character`/`account` (moeda), **integer** em catálogo (preço base) — documentar a diferença |
| `slot` | `integer`, `smallint` | **smallint** (0–15 equip, 0–27 carry) |
| `id` | `bigint`, `integer`, `smallint`, `boolean` | `bigint` para PK; os `boolean`/`smallint` chamados `id` são flags e devem ser renomeados |
| `kind`, `status`, `version` | `smallint`/`text`, `bigint`/`text` | decidir por tabela; `version` é `bigint` só nos metadados de config |

Também: **12 identidades (`GENERATED ALWAYS`) e 12 `BIGSERIAL`** convivem. O baseline padroniza
identidade; `BIGSERIAL` fica só onde já existe dado e a troca exigiria reescrever a coluna.

Nomenclatura: `character` era **palavra reservada** e obrigava aspas em toda query; `class`, `int`,
`con`, `level`, `name`, `value`, `type`, `time`, `kind` também são reservadas ou ambíguas. O rename
`character` → `characters` foi feito em 06/11 pela migration `0003_rename_character_to_characters`
(tabela + três índices) junto com o `store`, na mesma entrega: hoje nenhuma query precisa de aspas e
não existe alias de compatibilidade.

## 5. Comentários e documentação viva

17/50 tabelas comentadas (34%) e **3 colunas** no banco inteiro (`item_catalog.fields`,
`item_catalog.effects`, `npc_template.source_size`). As tabelas de conteúdo (as novas) estão bem;
as antigas de domínio (`character`, `account`, `item`, `affect`, `npc_definition`) não têm nada.

Regra para o baseline: comentar **tabela** em toda tabela de domínio e **coluna** onde o nome não
basta (`pk_point` = 75 neutro, `celestial_arch_level`, `soul`, `fame`, `mob_template_stat.updated_by`).
Comentário de coluna é o único lugar onde "por que 75 e não 0" sobrevive ao próximo dev.

## 6. Recursos do Postgres em uso

| Recurso | Hoje | Recomendação |
|---|---:|---|
| Funções | 0 | Nenhuma necessária. Lógica de jogo pertence ao Go (dono único do estado). |
| Views | 0 | **Usar** para leitura de relatório no portal: `v_character_overview`, `v_guild_roster`, `v_item_price_effective` (catálogo + override). View evita duplicar JOIN em 3 telas. |
| Triggers | 0 | **Manter zero** em tabela de jogo: trigger dispararia também no seed e na migration, e a auditoria do projeto é escrita na mesma transação do `store` com `before/after` explícito. Trigger de `updated_at` é aceitável só em tabela de configuração. |
| Materialized view | 0 | Só quando houver ranking caro; hoje `character` tem 2 linhas. |
| Enums | 1 | Manter; é melhor que `text` livre para estado. |
| Extensions | 1 | Só `plpgsql`. `pg_trgm` fica de fora: 3.220 itens não justificam a extensão. |
| Sequences | 10 | Resquício de `serial`; o baseline usa identidade. |

## 7. Segurança

**Achado P1:** todos os serviços usam o mesmo `postgres` **superusuário** via
`W2PP_DB_DSN`. Um bug de SQL no importador, no portal ou no seed tem poder de `DROP DATABASE`.

Recomendação (fase 5 do baseline, não bloqueia o squash):

- `openwyd_app` — dono do esquema, DML completo; usado por dbserver/webserver.
- `openwyd_ro` — só `SELECT`, para `importcontent -check` e ferramentas de diagnóstico.
- `postgres` fica para migration/backup e não aparece em `.env` de serviço.
- `pg_hba`: `scram-sha-256` (o default do PG 17 já é), conexão só pela rede do compose.

## 8. Backup e recuperação

Não havia **nenhum** script (`scripts/` só tinha preflight/run-local/gen-certs). Criado
`scripts/db-backup.sh`: dump `--format=custom` (restaurável com `pg_restore`) + esquema em texto para
diff, com destino **fora do repositório** por padrão — dump carrega conta e personagem.

O que ainda falta para produção: retenção (7 diários + 4 semanais), cópia fora da máquina e teste de
restauração agendado. Um backup que nunca foi restaurado não é backup.

## 9. Migrations

59 arquivos `.up.sql` para 50 tabelas, com números **duplicados** (`0007×2`, `0008×2`, `0012×4`,
`0013×2`, `0022×2`) porque duas frentes criaram migrations na mesma semana. Funciona (a versão é o
nome do arquivo, não o número), mas o histórico deixou de ser legível — não dá para dizer "o esquema
na versão 22" e saber o que existe.

O squash resolve isso com um corte limpo:

1. `0001_schema.up.sql` — `pg_dump --schema-only`, sem `schema_migrations` (o runner é dono dela),
   sem meta-comandos de `psql` (`\restrict`, que `tx.Exec` não entende).
2. `seed_<tabela>.up.sql` — os 12 seeds determinísticos do `seedgen`, aplicados depois do esquema.
3. Migrations antigas removidas do working tree: o histórico pré-squash é o git (commits até
   `45eeb20`), não uma segunda cópia em disco — `_archived/` é apenas staging local, gitignored.
4. Adoção no banco existente: `INSERT INTO schema_migrations(version) VALUES ('0001_schema')` — sem
   isso o próximo boot tentaria criar tabelas que já existem.

Regra nova: daqui pra frente, **três arquivos por tabela** (tabela → seed → constraints/índices) e
nunca reutilizar número.

## 10. Plano priorizado

| Prio | Ação | Onde | Estado |
|---|---|---|---|
| **P0** | Baseline `0001_schema` + seeds + adoção + validação em banco vazio | fase 4 | ✅ 04/11 |
| **P0** | Índices das 8 FKs sem origem (`guild_relation.target_guild_id` primeiro) | baseline | ✅ 04/11 (0053) |
| **P1** | Tipos canônicos (`item_index`, `eff*`, `slot`, `pos_x/y`) | baseline | ✅ 04/11 (0053) |
| **P1** | Comentários de tabela em todo o domínio + colunas ambíguas | baseline | ✅ 04/11 (0053: 49/49 tabelas, 18 colunas) |
| **P1** | Roles separadas (`openwyd_app`, `openwyd_ro`) e DSN sem superusuário | após baseline | ⏳ pendente |
| **P1** | Retenção de backup + teste de restauração | scripts | ⏳ pendente |
| **P2** | `character` → `characters` (quebra código, precisa de uma entrega só para isso) | `0003_rename_character_to_characters` | ✅ 06/11 |
| **P2** | Views de relatório do portal | com as telas | ⏳ |
| **P2** | Revisar os 19 índices de auditoria sem leitura (manter ou remover com critério) | após portal ler | ⏳ |

## 11. O que a fase 4 aplicou (04/11/2026)

O squash não é só "juntar arquivos": o esquema do dump passou a ser o **esquema corrigido**. As
mudanças foram escritas como migration normal (`0053_baseline_normalization`), aplicada no banco de
dev e depois capturada pelo `pg_dump` — assim o baseline é consequência de um estado real, não de SQL
editado à mão no dump.

- **Tipos** (`ALTER COLUMN … TYPE`): `mob_carry_override.eff*/effv*/slot` e `drop_rate_*.slot`
  `integer → smallint`; `item.item_index` e `world_item_seed.item_index` `smallint → integer`;
  `npc_definition.pos_x/pos_y` `integer → smallint`. Faixas conferidas antes (posição ≤ 4079,
  slot ≤ 12, maior `item_index` = 5134).
- **Índices**: os 8 das FKs sem origem (7 `updated_by` com `SET NULL`, `daily_reward_claim.reward_item_id`
  e `guild_relation.target_guild_id` com `CASCADE`).
- **Comentários**: 49/49 tabelas de domínio (todas menos `schema_migrations`) e 18 colunas ambíguas
  (`pk_point`, `soul`, `fame`, `celestial_arch_level`, `origin`, `price`, coordenadas, `status`…).
- **Layout**: `0001_schema.up.sql` + 12 `seed_*.up.sql` no topo do pacote embedded; as 59 migrations
  antigas saíram do working tree na fase 5 (o histórico é o git); `0001_schema.down.sql` destrutivo
  e explícito. O staging `_baseline/` e `seeds/` deixaram de existir — dois lugares com o mesmo SQL
  divergem, e era exatamente esse o defeito que o squash veio corrigir.
- **Gate**: banco vazio → `store.Migrate` aplica 13 arquivos → mesmas 49 tabelas e as mesmas
  **18.232 linhas** de referência do banco de origem (comparação por tabela no `db-baseline.sh`).
  O runner cria a 50ª (`schema_migrations`), por isso ela fica fora da contagem do script.

Dois bugs reais só apareceram nesse gate, ambos invisíveis enquanto o banco era o mesmo de sempre:

1. **`set_config('search_path','')` no dump.** O `pg_dump` 17 emite esse `SET` de sessão; como o
   runner reaproveita a conexão, toda migration seguinte rodava com `search_path` vazio e o
   `INSERT INTO schema_migrations` falhava com `42P01 relation does not exist` — o dbserver entrava
   em crash-loop. O script agora filtra essa linha (junto com `\restrict`), e o dump qualifica os
   objetos com `public.`, então nada se perde.
2. **`npc_shop_item` com 6.711 linhas, não 3.538.** O inventário de 04/11 estava velho: o import da
   issue 57 regravou as lojas. O baseline acompanha o banco (validação por tabela), o número é que
   precisava ser corrigido no documento.
