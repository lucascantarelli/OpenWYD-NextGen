# O que é editável no servidor — matriz de cobertura

> Levantamento de **tudo** que dá para editar no `OpenWYD` e qual é o caminho de edição (RPC + overlay,
> só leitura, ou nenhum). Serve para responder "a UI tem tudo?" e para não abrir issue duplicada.
>
> Colunas: **superfície** · **onde vive** · **o que muda** · **caminho hoje** · **issue**.
> ✅ = entregue · 🔵 = issue aberta · ⚪ = lacuna sem issue (ver §5).

## 1. Conteúdo em `tmserver/content/` (versionado, read-only para o portal)

| Superfície | Onde vive | O que muda | Caminho hoje | Issue |
|---|---|---|---|---|
| Catálogo de itens | `tmserver/content/Common/ItemList.csv` (3.220) | nome, display, **preço base**, grade, `slot_mask`/nPos, mesh/texture, pares `EF_*` | leitura via `ListItems`; edição = overlay no Postgres | ✅ 22 (leitura) · 🔵 33 (nome/preço) · 🔵 44 (efeitos) |
| Skills | `tmserver/content/Common/SkillData.csv` (151) | nome, classe, nível, mana, dano, alvo | **nenhum** (só o tmserver lê) | 🔵 46 |
| Templates de mob | `tmserver/content/TMsrv/run/npc/*.txt` (620) | nível, HP/MP, atributos, EXP, coin, skills, `Equip[16]` | `MobTemplateAdminService` (overlay `mob_template_stat`/`_equip`) | 🔵 25 |
| Spawns | `tmserver/content/TMsrv/run/NPCGener.txt` (6.099 geradores / 11.930 mobs) | coordenada, raio, quantidade, template | leitura no boot; sem RPC de escrita | 🔵 37 |
| Drops | `ItemDropList.txt`, `LevelItem.txt`, `Carry[]` do template | item por slot, taxa, level item | `ListDropItems`/`ListMobDrops` (leitura) | ✅/🔵 26 (relatório) · 🔵 34 (edição) |
| Taxa por slot de drop | **código** `DropRate[64]`/`DropBonus[64]` | divisor de chance por slot (0..63) | — | 🔵 34 |
| Rates | `Settings/{CompRate,SancRate,QuestsRate,CastleQuest,MobMerc}.txt` | refino, combine, quest, castelo, mercador | sem RPC | 🔵 36 |
| `gameconfig.txt` | `tmserver/content/TMsrv/run/gameconfig.txt` | evento de drop, `double`, `potioncount`, `partybonus`, **Treasure**, **Item Drop Bonus**, `PotionDelay`, `GTorreHour`/`RVRHour`/`BRHour`, `maxNightmare` | parcial (evento global via overlay) | 🔵 28 (evento) · 🔵 36 (demais chaves) · 🔵 45 (horários de guerra) |
| Itens fixos do mapa | `tmserver/content/TMsrv/run/InitItem.csv` (49: portões/baús) | posição, item, estado | leitura no boot | 🔵 41 (catálogo) · ⚪ edição |
| Item extra | `extraitem.csv` + `extraitem.bin` | item "efêmero" fora do ItemList | sem RPC | 🔵 44 |
| Mapa de altura/colisão | `HeightMap.dat` (16 MiB), `AttributeMap.dat` (1 MiB) | relevo, bloqueio, zonas de dano | `AttributeMapAdminService` (download/transform) | 🔵 27 |
| Regiões nomeadas | `tmserver/content/TMsrv/run/Regions.txt` (52 retângulos) | nome + retângulo (PvP, dungeon, Kefra…) | **nenhum** — só o tmserver lê | 🔵 41 |
| Guardas de cidade | `tmserver/content/TMsrv/run/Guard.txt` | requisitos, reward, efeitos do NPC guarda | sem RPC | 🔵 45 |
| Quest diária | `tmserver/content/TMsrv/run/QuestDiaria.txt` | missões do dia | `DailyRewardAdminService` cobre a recompensa | 🔵 30 |
| `Language.txt` / mensagens | `tmserver/content/TMsrv/run/Language.txt` | textos do sistema | sem RPC | ⚪ |
| Base de personagem | `tmserver/content/DBsrv/run/BaseMob/{TK,FM,BM,HT}` (816 B) | equip/stats iniciais por classe | sem RPC | 🔵 43 |

## 2. Estado morto no Postgres (overlay/config — escrita permitida)

| Tabela | Conteúdo | RPC | Issue |
|---|---|---|---|
| `npc_definition`, `npc_shop_item`, `npc_audit` | NPC, loja, trilha | `NpcAdminService` | ✅ 23 |
| `item_price` | preço global | `SetItemPrice`/`ListItemPrices` | ✅ 24 |
| `mob_template_stat`, `mob_template_equip` | stats/equip de mob | `MobTemplateAdminService` | 🔵 25 |
| `world_event_config`, `world_event_meta`, `world_event_audit` | evento global | `WorldEventAdminService` | 🔵 28 |
| `donate_shop_item`, `donate_shop_audit` | loja de donate | `DonateAdminService` | 🔵 29 |
| `daily_reward_item`, `_claim`, `_audit` | recompensa diária | `DailyRewardAdminService` | 🔵 30 |
| `donate_payer_profile`, `donate_topup_order`, `sapphire_balance` | faturamento | `DonateTopupService`/`RevenueAdminService` | 🔵 31 |
| `account` (role, blocked) | papel e ban | **sem RPC** | 🔵 32 |
| `guild`, `guild_member`, `guild_relation`, `guild_tower_state` | guildas e torre | **sem RPC** | 🔵 45 |
| `castle_quest_state` | castelo/Zakum | **sem RPC** | 🔵 45 |
| `kefra_state` | Kefra vivo/morto | **sem RPC** | 🔵 45 |
| `delivery_queue` | concessões ao jogador | escrita pela web, drenada pelo tmserver | 🔵 43 |
| `character`, `item`, `affect` | save de personagem | **sem RPC** | 🔵 43 |

## 3. Estado vivo (tmserver — goroutine única)

| Superfície | Como editar sem violar o dono único | Issue |
|---|---|---|
| Quem está online, kick, teleporte de jogador, aviso global | RPC no tmserver (hoje só CPSock/HTTP de status) | 🔵 38 |
| Inventário/status de personagem online | **só** por `delivery_queue`; edição direta no banco corrompe o save em memória | 🔵 43 |
| Portões abertos, castelo no ar, Kefra em curso | evento no mundo + restart/flag | 🔵 45 |

## 4. Constantes no código Go (precisam de overlay + RPC para serem editáveis)

| Constante | Arquivo | Issue |
|---|---|---|
| 5 cidades (limit/spawn/war rects) | `tmserver/internal/world/city.go:18` | 🔵 41 |
| `teleportTable` (23 rotas, custo, requisito) | `tmserver/internal/world/teleport.go:22` | 🔵 42 |
| `DropRate`/`DropBonus` (64 slots) | `tmserver/internal/loot` | 🔵 34 |
| Curva de EXP `g_pNextLevel[0..400]` | `tmserver/internal/level` | 🔵 35 |
| Regras de refino (cap, custo) | `tmserver/internal/refine` | 🔵 44 |
| Receitas de combine (Odin/Anct) | `tmserver/internal/combine/*.go` | 🔵 44 |
| Taxa de cidade (`CityTax`) e horários de guerra | `world/city.go`, `gameconfig.txt` | 🔵 45 |

## 5. Lacunas ainda sem issue (candidatas)

1. `Language.txt` — mensagens do sistema (tradução/revisão de texto).
2. `InitItem.csv` — escrita de portões/baús fixos do mapa (hoje só catálogo na 41).
3. Eventos sazonais "hardcoded" (`Chall_00_00.txt`, `Guilds.txt`, `ChampionCity_*.txt`) — placar de campeões.

## 6. Ordem sugerida

`41` (atlas) → `42` (teleportes, usa o atlas) → `25` (mobs) → `44` (adds/refino) → `43` (personagens) →
`45` (guildas/castelo) → `46` (skills) → `33`/`34`/`35`/`36`/`37` (conteúdo) → `38`/`39` (operação/QA).

> Regra que vale para todas: o portal **nunca** escreve em `tmserver/content/` nem no estado vivo. Escrita é
> overlay no Postgres (aplicado pelo tmserver no boot/poll) ou `delivery_queue`.
