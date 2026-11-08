# Migrations (`crates/wyd-db`)

Série única, aplicada em ordem pelo nome do arquivo. Cada arquivo é um domínio e tem dois
blocos: `-- migrate:up` e `-- migrate:down`. O marcador só vale quando ocupa a linha inteira
(o cabeçalho cita os dois nomes na mesma frase e já enganou dois runners).

## Regra da sequência

**DDL primeiro (`0001`–`0016`), seed depois (`0017`–`0024`).** Um seed nunca roda antes de a
tabela que ele preenche existir; separar as fases deixa o `down` de seed (só `DELETE`) e o de
DDL (`DROP TABLE`) em blocos previsíveis.

| Faixa | Conteúdo |
|---|---|
| `0001`–`0006` | base de jogo: contas, personagens, inventário (com refino/Anct/gema na criação), guildas, auditoria, delivery_queue |
| `0007`–`0008` | catálogo (item/skill/efeito, com `skill_catalog.notes`) e mundo (NPC, loja, região, settings) |
| `0009`–`0015` | refino, economia, rotas, combate, quests, eventos, arena |
| `0016` | progressão por personagem: `character_status`, `character_lineage`, `character_skills` |
| `0017`–`0024` | seeds: taxonomia, itens, skills, mundo, linhagem de skills, refino, mundo fixo, skills da Kersef |

Regras que o verificador cobra:

1. **Nenhum `ALTER TABLE` no `up`.** Coluna nova nasce na criação da tabela. `ALTER` corretivo
   esconde desenho errado e foi removido da série em 2026-11-08 (`0023_inventario_gema` e
   `0024_catalogo_skill_nota` foram consolidados em `0003` e `0007`); o gerador do catálogo
   falha se um `ALTER` reaparecer.
2. **Nenhuma FK para tabela criada depois** — é o que obriga `0016_progressao` a vir depois de
   `0007`, já que referencia `skill_lineage`/`skill_catalog`.
3. **Toda tabela criada cai no `down` do próprio arquivo.**
4. **`down` de seed restaura o estado anterior, não o apaga** (ADR-0013): reverter uma migração
   de seed devolve as linhas herdadas ao valor da migração anterior (ex.: `0021` restaura as 6
   linhas da `0019`) e só zera o que ela própria classificou.
4. Numeração contígua a partir de `0001`, sem buraco.

## Verificação em três camadas

```bash
make migrations-check   # ordem/FK/down/INSERT, sem Postgres  (segundos)
make banco-catalogo     # regenera docs/arquitetura/banco-catalogo.md
make banco-verify       # banco vazio → up → compara com information_schema (Docker)
make migrations-apply   # up → down → up, contagens iguais (Docker)
```

| Camada | Comando | O que prova |
|---|---|---|
| estrutural | `node tools/content/check-migrations.mjs` | ordem, tabela duplicada, FK para tabela futura, `down` esquecido, `INSERT` órfão |
| contrato | `cargo test -p wyd-db` (`tests/catalogo.rs`) | catálogo documentado == migrations; `SCHEMA_VERSION` == última; seed só depois do DDL |
| banco real | `node tools/content/verify-schema.mjs` | `pg_attribute`/`information_schema` == catálogo (tabela, coluna, tipo, nulidade, PK), sem sobra nem falta |
| ciclo | `node tools/content/migrate-cycle.mjs` | `up` completo, `down` completo (0 tabelas), `up` de novo com contagens idênticas |

Última execução (2026-11-08, `postgres:17-alpine`):

| Medida | Valor |
|---|---|
| migrations | 24 (16 DDL + 8 seeds) |
| tabelas / colunas | 59 / 457 |
| tabelas após o `down` | 0 |
| `item_catalog` / `skill_catalog` | 3.220 / 154 (72 com linhagem) |
| `npc_definition` / `npc_shop_item` / `level_item` | 6.100 / 6.706 / 161 |
| `region` / `game_system` / `item_effect` | 52 / 20 / 55 |
| `refine_rate` / `refine_gem_effect` / `sanc_rate` | 15 / 4 / 36 |
| `content_version` | 7 |
| ciclo up/down/up | 24 migrations, ~55 s, contagens idênticas |

## Lacunas declaradas (tabela criada ≠ conteúdo pronto)

| Tabela | Estado | Por quê |
|---|---|---|
| `quest`, `character_quest` | vazia | cadeia do 7662 não está fechada; seed seria invenção de quest |
| `mob_template_stat`, `mob_template_equip` | vazia | stats vêm do `STRUCT_MOB` (blob de `npc_template`); extração do struct ainda não foi feita |
| `character_status`, `character_lineage`, `character_skills` | vazia | schema pronto para o save; a fórmula de pontos está em `docs/wyd/progressao.md` e entra com a FASE-10 |

| `item_price`, `donate_shop_item`, `daily_reward_item` | vazia | preço é configuração do dono do servidor, não dado do legado |
| `item_effect_override`, `castle_quest_state`, `mob_carry_override` | fora do schema | pendências D-B05/D-B06/D-B07 em `docs/arquitetura/banco-catalogo.md` §3 |

## Catálogo

`docs/arquitetura/banco-catalogo.md` é **gerado** (`make banco-catalogo`) a partir destas
migrations + `tools/content/banco-fontes.mjs`. Ele traz, por tabela, colunas/tipos/nulidade,
PK/FK/CHECK/índices, o arquivo que a cria, a fonte da decisão e o que é provisório. Não edite
à mão: rode o gerador.
