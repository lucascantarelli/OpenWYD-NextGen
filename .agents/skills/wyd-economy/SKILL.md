---
name: wyd-economy
description: Analisa e ajusta a economia e a progressão do servidor WYD2 — preços de item, requisitos de equipamento, drops, refino com poeira (+9), combines (Odin, Ailyn, Tiny, Shany, Agatha, Lindy, Alquimia, Ehre, Extração), curva de EXP, multiplicadores de evento e Campo de Treino. Use quando a tarefa pedir balancear recompensa, mudar preço/drop/EXP, criar ou alterar receita, ou investigar por que um item não refina, não cai ou não pode ser equipado.
---

# Economia e progressão (WYD local)

Tudo que define valor no jogo vem de **dados de conteúdo** — catálogo de item/skill no Postgres
(issue 58) e taxas, drops e mapas em `tmserver/content/` — mais um punhado de flags do tmServer. Mexer em
item/preço é mexer no catálogo e reiniciar o servidor; mexer em personagem é in-game (`/gm`) ou via
save.

## 1. Os arquivos que mandam

| Arquivo | Define | Consumidor |
|---------|--------|-----------|
| `item_catalog` (Postgres) | catálogo: índice, preço, efeitos-base, requisitos, volatilidade, `nPos`, `nUnique`, grade, `Extra`, alcance | gRPC do dbServer → `content.Config.Item*`; seed em `seed_item_catalog.up.sql` |
| `skill_catalog` (Postgres) | catálogo de skills (mana, dano, affect, nomes Latin-1) | gRPC do dbServer → `content.Config.Spells`; seed em `seed_skill_catalog.up.sql` |
| `tmserver/content/Common/Settings/SancRate.txt` | taxa de sucesso do refino por anvil (0 = Ori, 1 = Lac), 12 níveis | `refine.RateTable` |
| `tmserver/content/Common/Settings/CompRate.txt` | famílias/taxas de composição | `combine.Catalog` |
| `tmserver/content/Common/Settings/QuestsRate.txt` | taxas de quest | `content.QuestRates` |
| `tmserver/content/Common/Settings/CastleQuest.txt` | quests de castelo/Zakum | `content.CastleQuest` |
| `tmserver/content/Common/Settings/MobMerc.txt` | mercadores de mob | loaders de conteúdo |
| `npc_template.Carry[]` (banco) | tabela de drop | `loot` · telas `/drops`, `/drops/carry` |
| `tmserver/content/TMsrv/run/LevelItem.txt` | itens por nível | progressão |
| `tmserver/content/TMsrv/run/gameconfig.txt` | `double`, `deadpoint`, `partybonus`, `partydif`, matriz 8×8 de bônus de drop, `DropItem`, `PotionDelay`, `maxNightmare`, horas de GTorre/RVR/BR | mapeado para flags (ver `docs/migration/config-ops.md`) |
| `npc_template` (Postgres) | template de mob/NPC (816 B) — inclui `Exp`, nível, clan, merchant, equip-modelo | gRPC do dbServer; seed em `seed_npc_template.up.sql` |

Regra de leitura do catálogo: a **coluna 0 do CSV é o índice** usado no fio e em `/gm item`. A 4ª
coluna é `ReqLvl.ReqStr.ReqInt.ReqDex.ReqCon` (confira a ordem nos comentários de
`tmserver/internal/content/catalog.go` antes de editar).

## 2. Preço, compra e venda

- Preço-base vem de `ItemPrices` (coluna de preço do catálogo). NPC compra/vende por
  `handler/shop.go` (`MsgREQShopList`, `MsgBuy`, `MsgSell`); item ausente do catálogo passa sem
  gate, então **preço errado costuma ser linha ausente**, não bug de código.
- O editor de NPC (`npc_definition`) permite mudar preço de loja de mercador sem rebuild — e desde a
  issue 57 a tabela é a **única** fonte de spawn: sem catálogo no banco o mundo sobe **sem NPCs**
  (rode `dbserver import-npcs`; o dbserver também semeia no boot).
- Ouro tem três lugares distintos: carregado (`Gold`), cargo/banco da conta
  (`_MSG_Deposit` 0x0388 / `_MSG_Withdraw` 0x0387) e chão. `/gm setgold` mexe **só no carregado**.

## 3. Refino (poeira / +9)

- Taxa por anvil em `SancRate.txt` sobre os defaults compilados (`content.DefaultSancRate()`); a
  ausência do arquivo **não** quebra o refino, só mantém os defaults — por isso o boot loga
  `sancrate_ori` e `sancrate_lac` (se estiver estranho, o mount está errado).
- Efeito de refino entra no score: o valor da instância soma sobre o efeito-base do catálogo.
  `EF_DAMAGE` de arma é tratado à parte (`weaponDamage`), e `EF_RANGE` **nunca** entra em
  `BaseEffects` (senão o alcance dobra).
- A partir de +9 há bônus por `nPos` (limiar), e `nUnique` libera `EF_DAMAGEADD` em joias.
- O `+9` da capa é passo obrigatório do Sub-Celestial (`docs/game.md`), então mexer em `SancRate`
  afeta progressão de endgame, não só economia.

## 4. Combines

Um motor só (`handler/combine.go`) atende todas as variantes roteadas em `dispatch.go`:
`CombItem`, `Extracao`, `Odin`, `Ailyn`, `Tiny`, `Shany`, `Agatha`, `Lindy`, `Alquimia`, `Ehre`.
O que muda por variante é a **família** (`combineFamily`: receita + taxa) instalada por
`handler.DefaultCombineFamilies(odinCatalog)`.

- Receita que "não existe" é sintoma de `CompRate.txt` sem a família, ou item sem entrada no
  catálogo (Odin precisa de `Extra` para resolver o resultado).
- Combines de Anct e Adamantita usam `ItemExtra` (índice do item resultante).
- O catálogo real de receitas é **UNVERIFIED** em parte (`docs/migration/game-rules.md` §3): ao
  alterar, valide por teste de handler e, se possível, contra captura do servidor legado.

## 5. EXP e progressão

Pipeline: mob morto → `level.ExpApply` (escala por nível) → modificadores globais → `ScoreBonus`/HP/MP
por nível → `MSG_UpdateScore`.

- Curva Mortal/Arch: `g_pNextLevel[0..400]` (`tmserver/internal/level`). Celestial usa
  `g_pNextLevel_2` (MAX_CLEVEL), ainda com **placeholder** — captura pendente
  (`docs/migration/prompts/agent-prompt-celestial-curve.md`).
- Modificadores globais (`-double-exp`, `-newbie-event`, estado do Kefra): `Kefra` vivo **divide a
  EXP por 2**; Campo de Treino ignora esses modificadores e aplica `-training-field-exp` (padrão 6)
  para mortais abaixo do nível armazenado 35.
- O valor de EXP de cada mob está gravado **no template** (`STRUCT_MOB.Exp`). O boot avisa
  `monster template has unbalanced Exp (run cmd/exptool)` quando o catálogo não foi restampado.
  Ferramenta: `make exp-data` (= `go run ./tmserver/cmd/exptool -npc tmserver/content/TMsrv/run/npc -quiet`;
  use `-dry-run` primeiro).
- EXP de party usa divisores que a documentação marca como não confiáveis
  (`captura-wyd-levelup.md`): não "conserte" balanceamento de party assumindo que a regra está certa.

## 6. Drops

- `loot` consome o `Carry[]` do template (banco) + bônus por slot da matriz 8×8 do `gameconfig.txt`; o relatório item↔mob é a tela `/drops`, não um arquivo
  (`g_pDropBonus`), com paridade de RNG (ordem de chamadas do LCG).
- `DropItem 0` no `gameconfig` **desliga** drop de item — se "nada cai" e o gold cai, comece por aí.
- A recompensa de kill é server-authoritative e testada com golden cases (`loot`/`combat`). Mudar
  drop sem atualizar golden quebra `go test -race ./tmserver/internal/...` — atualize o teste com o
  número novo, nunca remova a asserção.
- Mob template sem `Exp`/`Exp` absurdo: ver §5. Para stats de mob (HP, dano, defesa), a ferramenta de
  moderação é `-mob-stat-editing` + overlay `mob_template_stat` (aplicado **uma vez no boot**;
  reiniciar o tmServer para valer).

## 7. Rotina de balanceamento (segura e reproduzível)

1. **Meça antes.** Suba a stack, mate N mobs do tipo alvo (bot/harness), registre EXP/min, gold/min,
   itens/min. Sem número inicial não há como provar melhora.
2. **Mude dado, não código.** Prefira CSV/template/flag. Se precisar de fórmula, mude em
   `tmserver/internal/level`/`loot`/`refine` com teste table-driven.
3. **Valide paridade.** `go test -race ./tmserver/internal/...`; se mexeu em RNG/ordem, rode os
   golden de `loot`, `combat` e `refine`.
4. **Reinicie só o tmServer** e confirme no log: `content loaded`, `NPCs spawned`, e ausência de
   `unbalanced Exp`.
5. **Registre** o antes/depois e o motivo em `docs/migration/ingame-bugs.md` ou no roadmap.

## 8. Nunca

- Não acrescente item ao `item_catalog` sem checar que o **cliente** tem o item correspondente em
  `ItemList.bin`/`itemname.bin` — item novo só de servidor fica invisível ou com nome errado.
- Não edite preço à mão no banco de dev: o catálogo vem do seed (`seed_item_catalog.up.sql`) ou do
  `importcontent`, e a alteração manual some no próximo baseline.
- Não rode `make exp-data` em produção sem `-dry-run` e sem backup da tabela `npc_template`.
- Não trate valor **UNVERIFIED** (curva celestial, coef. Dex/Str por classe×arma, DoT) como verdade
  ao calibrar.
