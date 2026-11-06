# Persistência: arquivos de `tmserver/content/` versus banco de dados

Pergunta central: **é possível parar de ler texto/CSV no boot e viver só de Postgres?** Resposta:
sim para o conteúdo de jogo, não para mapas binários e assets do cliente — e há um caminho de
migração incremental (issue 50) que não quebra a paridade.

## 1. O que o servidor lê hoje (evidência: grep no Go)

| Arquivo | Referências no Go | Uso | Já existe equivalente em banco? |
|---------|------------------:|-----|---------------------------------|
| `ItemList.csv` | 18 | catálogo de itens, preços base, ícones, efeitos | **sim**: `item_catalog` (issue 58, importado; leitores em migração) |
| `NPCGener.txt` | 13 | 6.099 geradores de spawn | **sim**: `npc_definition` (issue 57) |
| `CompRate.txt` / `SancRate.txt` / `QuestsRate.txt` | 6 / 3 / 3 | taxas de refino, combine e quests | sim: `content_rate_override` |
| `SkillData.csv` | 5 | 151 skills (dano, affects, delay) | **sim**: `skill_catalog` (issue 58) |
| `InitItem.csv` | 4 | itens iniciais | não |
| `AttributeMap.dat` / `HeightMap.dat` | 4 / 3 | colisão e altura (17 MiB) | **não deve ir** (ver §4) |
| `Regions.txt` | 3 | zonas/regiões | parcial: `guild_zone`, `teleport_route` |
| `CastleQuest.txt` | 2 | quest de castelo | sim: `castle_quest_state` (estado), conteúdo não |
| `ItemDropList.txt` | 1 | **relatório gerado**, não lido em runtime | n/a |
| `itemicon.bin` | 2 | índice do atlas de ícones | não (metadado de asset) |
| `gameconfig.txt`, `Guard.txt`, `LevelItem.txt`, `MobMerc.txt` | vários | flags, guardas, itens por nível, mercadores | parcial (`server_instance_flags`; resto não) |

## 2. O que já vive no banco (46 tabelas)

- **Contas/personagens**: `account`, `character`, `item`, `affect`, `delivery_queue`,
  `account_play_code`, `sapphire_balance`, `character_pvp_stats`.
- **Conteúdo com overlay + auditoria**: `npc_definition`/`npc_shop_item`, `item_price`,
  `item_effect_override`, `mob_template_stat`/`mob_template_equip`/`mob_carry_override`,
  `drop_rate_override`, `content_rate_override`, `teleport_route`, `world_event_config`,
  `daily_reward_item`, `donate_shop_item`.
- **Estado de mundo**: `guild`, `guild_member`, `guild_relation`, `guild_zone`, `guild_tower_state`,
  `kefra_state`, `castle_quest_state`, `server_instance_flags`.
- **Auditoria**: `*_audit` (conta, personagem, NPC, drop, carry, rate, efeito, evento, guilda, loja,
  faturamento, recompensa diária).
- **Meta**: `schema_migrations`, `npc_config_meta`, `world_event_meta`, `donate_payer_profile`,
  `donate_topup_order`.

Ou seja: **o padrão de "arquivo base + overlay auditado" já está provado** em 8 áreas. O que falta é
tornar o arquivo *seed* e o banco a fonte de verdade.

## 3. Desenho recomendado: importar uma vez, servir do banco

```
tmserver/content/ (read-only)                      Postgres
  ItemList.csv  ─┐                    ┌─ content_item (seed + versão)
  NPCGener.txt  ─┼─ importador ──────►├─ content_spawn
  SkillData.csv ─┤  (idempotente,     ├─ content_skill
  rates/*.txt   ─┘   com hash/versão) └─ content_*_audit (overlay do portal)
                        │
                        └─► tmserver carrega SÓ do banco (cache em memória no boot)
```

Regras que preservam o que já funciona:

1. **Seed idempotente**: o importador grava `content_seed(arquivo, sha256, versão)` e só reescreve
   quando o hash muda. Rodar duas vezes não duplica nada.
2. **Overlay sempre vence**: a linha do portal continua sendo "o que o operador mudou", com auditoria;
   o seed nunca sobrescreve overlay.
3. **Boot com banco fora**: se o banco não responder, o `tmserver` falha alto (não sobe "vazio") — hoje
   ele já falha quando falta conteúdo obrigatório.
4. **Sem releitura de CSV em produção**: arquivo vira *fixture* de importação, usado por
   `make content-import` e pelos testes.
5. **Paridade**: o conteúdo importado tem de gerar exatamente os mesmos valores de hoje — golden test
   comparando `catalog hash` antes/depois.

## 4. O que **não** deve ir para o banco

| Item | Por quê |
|------|---------|
| `HeightMap.dat` (16 MiB) / `AttributeMap.dat` (1 MiB) | são grades binárias 4096²/1024² lidas inteiras no boot e *baked* em memória; BLOB no banco não ajuda (nem cache, nem diff, nem backup incremental) |
| `itemicon.bin` + atlas `.wyt/.wys` | assets de cliente; o banco guarda apenas URL/metadata do ícone (já é o caso) |
| `serverlist.bin` | artefato do cliente, gerado por IP/porta (ver [cliente-launcher.md](cliente-launcher.md)) |
| templates `npc/<nome>` (816 B cada) | podem ir para o banco, mas atenção: são o **template de mob** (não o spawn). Migrar exige codec de 816 B no importador — viável, mas é a fatia mais arriscada; deixar para a fase 2 da issue 50 |

## 5. Limitações reais

1. **Compatibilidade com o cliente não é afetada** — o cliente nunca lê `tmserver/content/`; só o servidor.
2. **Ferramentas de terceiros** (editores antigos do fórum) esperam CSV. Se a operação abandonar essas
   ferramentas (já abandonou: portal cobre), o CSV pode ser apenas export.
3. **Diagnóstico**: hoje um valor estranho se investiga abrindo CSV. Com banco, é preciso uma tela/SQL
   e um export de volta — o portal já tem `/itens`, `/mobs`, `/drops`, `/npcs`.
4. **Backup**: hoje `tmserver/content/` + dump. Depois, só dump (mais simples, mas o dump cresce).
5. **Tempo de boot**: sai de ~40 s de scan (logs atuais: 1.991 templates + 20.680 drops) para leitura
   de banco; medir antes/depois — é ganho, mas precisa de número (issue 50 exige medição).

## 6. Entregável da issue 50

- Migrações `content_item`, `content_skill`, `content_spawn` (+ `content_seed`).
- `make content-import` + comando `tmserver -import-content=tmserver/content` (idempotente).
- `tmserver` lendo do banco com flag `-content-source=db|files` (default `db` após validação).
- Golden test de hash de catálogo + comparação de 10 itens/mobs/skills sorteados.
- Volume único `./tmserver/content:/content:ro` (LEGACY-02); o portal só lê o CSV de preço, quando existe.
