# GAME-18 — Campo de Treinamento: quest, portões, buff, boss e recompensas

> **ID canônico:** `GAME-18`
> **ID histórico:** `FASE-03-18`
> **Estado 2026-11-12:** 🟡 · código/teste fechados; prova visual ainda bloqueada por captura. Ver `roadmap-rust/index.md`.

**Fase:** 3 · **Depende de:** GAME-17 (entrada no mundo), PROC-15 ✅ (migrations), WEB-05 ✅ (conteúdo) · **Bloqueia:** GAME-23 (bots) · **Esforço:** L

## Objetivo

Implementar a **primeira quest** — o Campo de Treinamento — de ponta a ponta: os quatro
treinadores, as três chaves/portões, o NPC de buff, o boss Orc e as recompensas escolhidas pelo
dono, com estado por personagem persistido e teste de fio.

## Contexto / evidência

Pesquisa completa em `docs/wyd/pesquisa-campo-de-treinamento-2026-11-10.md`. O legado **não tem
script de quest**: `_MSG_Quest` (`0x028B`, `MSG_STANDARDPARM2` com `Parm1 = npcIndex`) é um
`switch` por `Merchant` sobre a flag `STRUCT_MOBEXTRA.QuestInfo.Mortal.Newbie` (0→4).

| Etapa | Merchant | Item | Recompensa no código |
|---:|---|---|---|
| 1 | 36 | 451 | item 682 (20) + affect 44 |
| 2 | 40 | 452 | encanta `Equip[6]` |
| 3 | 41 | 453 | encanta `Equip[1..7]` + sanc 3 |
| 4 | 100 + grade 16 | 524 | sorteio 682/481/652 |

Itens 451/452/453/524 dropam de `GremlinDecoy`/`GiantCrill`/`OrcSniper`/`OrcWarrior`
(`ItemDropList.txt`). Portões usam `EF_KEYID` + `STATE_LOCKED` e **consomem a chave**. Boss
oficial: **OrcWarrior**, nível 40, HP 2.000, EXP 240, drops Emblema Orc + Âmago de Javali.
Faixa oficial **1–35** (divergência “abaixo de 35” × “1~35” em §4 D-1).

## Decisões registradas (dono, 2026-11-10)

| # | Decisão |
|---|---|
| D-2 | **Sequência A — legado OpenWYD**, fiel ao código do 7662 |
| D-1 | Limite **1 a 35** (nível 36+ recusa) |
| D-6 | Arma inicial **Unicorn oficial** — sem NPC de Sephira +9 |
| D-7 | Sequência **A** reafirmada em 2026-11-11 (legado OpenWYD) |
| D-8 | **NPC Sephira +9 entra no Campo** (decisão do dono, 2026-11-11) — reverte o “sem Sephira” de D-6; o Unicorn continua existindo |
| D-9 | Recompensa do Chefe: **um item sorteado** (Rubi do Carbúnculo / Olho Crescente / 20 Poções) |

Consequência prática: `682` (Kit de Cura 20 M, `EF_HP` 200) + affect 44 (200/200) na etapa 1;
encantamento de `Equip[6]` na etapa 2; `Equip[1..7]` + sanc 3 na etapa 3; sorteio de
`682/481/652` na etapa 4. A sequência A não inclui refino +6 (B) nem “Classe A” (C); a
Sephira +9 (D-8) é um NPC separado, ao lado do Unicorn.

## Passos

1. Mapear o **`_MSG_Quest`** na tabela do protocolo com o corpo provado (`MSG_STANDARDPARM2`:
   `npcIndex`, `confirm`) e despachar no `cpsock`; o NPC vem do id de cena e o servidor valida
   `merchant`/distância — o cliente não escolhe recompensa.
2. Persistir o estágio por personagem em tabela própria (`character_quest`, chave
   `(character_id, quest_id) UNIQUE`), migration versionada + `SCHEMA_VERSION`; nada de escrever
   em tabela de personagem vivo pela web.
3. Implementar as quatro etapas com a **sequência A** (682+affect 44; `Equip[6]`; `Equip[1..7]` +
   sanc 3; sorteio 682/481/652) e recusa tipada quando falta o item/estágio/nível.
4. Portões/chaves: tratar o uso do item-chave no `EF_KEYID`, consumir a chave, abrir
   (`STATE_OPEN`), fechar por tempo; sem layout inventado para o que não estiver provado.
5. NPC de buff: aplicar o affect definido no snapshot (o legado usa 44, 200/200) com duração.
6. Seed de conteúdo: os NPCs do campo (4 treinadores, Unicorn, Carbuncle), os guardiões, o
   `OrcWarrior` e as `drop_table` dos itens de prova — tudo em `npc_definition`/`drop_table`.
7. Arma inicial: **NPC Unicorn** (oficial, decidido) escolhendo arma física/mágica pela classe.
   O NPC **Sephira +9** (D-8) entra pela migration `0049`, ao lado do Unicorn; o item de arma
   Sephira ainda não tem linha de catálogo/loja.
8. Feedback ao cliente com `_MSG_MessagePanel`/`SendSay`/`_MSG_UpdateEtc`, nunca com uma janela
   de quest que o 7662 não tem.

## Arquivos

- Criados: `crates/wyd-protocol/src/payload/quest.rs`, `crates/wyd-server/src/quest.rs`,
  `crates/wyd-server/tests/campo_treinamento.rs`, migrations de `character_quest` e do seed.
- Alterados: `crates/wyd-server/src/cpsock.rs` (dispatch), `world.rs` (alcance/NPC),
  `crates/wyd-server/src/gameplay.rs` (recompensas), `docs/rust/08-checklist-funcionalidades.md`.
- Proibido: inventar `MSG_QuestInfo`/`QuestComplete` (não existem no 7662) e recompensa sem
  decisão registrada.

## Testes / verificação

```bash
cargo test -p wyd-server --test integracao_campo_treinamento
cargo test -p wyd-server --test integracao_campo_drops
cargo test -p wyd-server --test sistema_login
cargo test -p wyd-db --test catalogo
node tools/content/check-migrations.mjs
```

Saída esperada: 0→4 com item correto, recusa sem item, chave consumida e portão aberto, boss
dropando o emblema, recompensa uma vez por personagem. `integracao_campo_drops` usa o
pipeline real `Engine::attack_at` → `Gameplay::loot_at_death` → `add_item` → `quest_engage`
para os quatro guardiões/boss, com o HP do alvo reduzido apenas no fixture.

## Skills Relacionadas do freebuff

- `.agents/skills/wyd-rust-net` — opcode/payload provado.
- `.agents/skills/wyd-rust-db` — `character_quest` e seed.
- `.agents/skills/wyd-rust-core` — autoridade do servidor na recompensa.
- `.agents/skills/wyd-rust-testing` — harness de fio.

## Documentação de referencia

- `docs/wyd/pesquisa-campo-de-treinamento-2026-11-10.md`.
- `main:tmserver` (Go) e `Source/Code/TMSrv/_MSG_Quest.cpp`, `_MSG_UpdateItem.cpp`,
  `ItemDropList.txt` em `Jean1dev/w2pp-OpenWYD@main`.
- Guia oficial: <https://wydglobal.raidhut.com/pt-br/2402> e
  <https://feedback.kersef.com/pt-br/knowledge-bases/21/articles/20369-campo-de-treinamento-lv-135>.

## Estado real (2026-11-10)

**Entregue nesta rodada (com teste):**

- `_MSG_Quest` 0x028B no protocolo (`payload/quest.rs`, `MSG_STANDARDPARM2` = 20 B) e no
  dispatcher (`cpsock::handle_quest`), com o `merchant` vindo do mundo (`npc_merchant`).
- `character_quest` (migration `0044`): etapa, `completed` e `reward_mask` (1 bit por etapa)
  — reentrega é `AlreadyRewarded`, não recompensa dupla. `SCHEMA_VERSION` 44.
- Motor puro `gameplay::{quest_engage, apply_quest_reward}`: sequência A exata (682;
  `Equip[6]` com 50 + ReqLvl; `Equip[1..7]` com 50 + ReqLvl **e** sanc 3; sorteio
  682/481/652), teto de nível 35, ordem das etapas e consumo da prova.
- `SelectionStore::quest`: carrega progressão + `character_quest`, aplica, grava inventário e
  estado; affect 44 (200,200) entra como restauração imediata de HP/MP com teto.
- **Portões com chave (`EF_KEYID`, gate real):** `gameplay::{CampGate, CAMP_GATES, GateBook}` —
  os três portões (2075,2016 / 2143,1986 / 2081,1962) com `EF_KEYID` 2/3/4 dos itens
  451/452/453, consumo da chave no aceite e fechamento por tempo (30 s, comparado no
  instante; sem tick dedicado). `Engine::use_key_at` valida ponto exato + alcance 2 com a
  posição autoritativa; `SelectionStore::use_key` lê as provas do inventário, apaga a chave e
  grava. Ligado em `_MSG_Action` no tile do portão (`cpsock::handle_action`), antes do
  diálogo. **Divergência declarada:** o legado usa `_MSG_UpdateItem` porque a porta é um item
  de mundo; o nosso snapshot não tem entidade de porta, então o clique chega por
  `_MSG_Action` (layout já provado) — sem inventar `_MSG_UpdateItem` de portão.
- Teste de integração `crates/wyd-server/tests/campo_treinamento.rs` (11 testes): fluxo
  completo das 4 etapas com valores exatos, os 3 resultados do sorteio, nível 36, prova
  errada, fora de ordem, reentrega e encantamento sem arma.
- **Loot entregue no inventário (2026-11-11):** `gameplay::add_item` empilha até
  `MAX_STACK = 120` e ocupa o primeiro slot livre `16..80`; `SelectionStore::grant_loot`
  grava com `replace_items` (mesmo caminho do inventário já persistido) e
  `cpsock::handle_attack` envia `_MSG_SendItem` por slot entregue. Mochila cheia não inventa
  slot (o drop fica no chão). Provas: `loot_empilha_ate_o_teto_e_ocupa_slot_livre` e
  `loot_sem_slot_nao_e_inventado` (2 dos 11 testes).

**Fechado em 2026-11-12 (código + teste):**

1. **Fluxo completo dos guardiões e do boss.** `integracao_campo_drops` mata
   `GremlinDecoy`/451, `GiantCrill`/452, `OrcSniper`/453 e `OrcWarrior`/524 pelo pipeline real
   de morte/loot (`Engine::attack_at` → `Gameplay::loot_at_death` → `add_item` → `quest_engage`
   → `apply_quest_reward`). Cada prova entra no inventário, é consumida na etapa certa e a
   reentrega da mesma etapa é recusada. O HP do alvo é reduzido a 1 **só no fixture** por causa
   do dano 1 do nível 1; template, nome, posição, EXP e tabela de drop continuam vindo do seed.
2. **Sephira +9 (D-8).** A migration `0049_seed_campo_sephira` (complementa `0046`, que já foi
   aplicado e não pode ser reescrito) cria o NPC `Sephira +9` (id 6311, template `Refinacoes`)
   em `(2133,2040)`, ao lado do Unicorn. `integracao_campo_drops` o encontra pelo clique e prova
   o refino `+8 → +9` com a taxa do snapshot. O teste aplica a série de migrations de forma
   idempotente antes do pool, então não depende de `wyd_test` já estar no schema 49. A arma
   Sephira +9 em si continua sem linha de catálogo/loja — o NPC existe, o item não foi inventado.
3. **`ReqLvl` do equipamento.** O catálogo já tinha `req_level`; agora ele é consumido de fato:
   `SelectionStore::quest` lê o `ReqLvl` do item em `Equip[6]` (`PgBackend::item_req_level`) em
   vez de usar o nível do personagem, e `EquipmentCatalog` implementa `ItemRules` sobre
   `CatalogItemRow` para a validação de equipagem. Testes: bônus `50 + ReqLvl` com arma real e
   `equip` recusando nível 1 para item `ReqLvl` 319.

**Aberto (bloqueado por captura no cliente 7662 — não marcar como concluído):**

1. **Portões como objeto visível.** A mecânica está implementada (chave por `EF_KEYID`,
   consumo, fechamento por tempo), mas o portão não é uma entidade da cena: o cliente 7662
   não vê a porta nem recebe o `EF_KEYID` no `_MSG_UpdateItem` do legado. Falta o item de
   mundo/porta no snapshot para o fluxo visual.
2. **Buff do `Cabuncle`.** As três instâncias estão no seed, mas o affect não tem pacote de
   ícone/duração no fio; falta o caminho de aplicação/reaplicação visível ao cliente.
3. **Prova visual no 7662.** Os guardiões/boss já têm prova de código/teste no pipeline real,
   mas ainda falta observar entrada, drop, portão e conclusão na tela do cliente.

## Pesquisa 2026-11-11 (guardas, drops, sequências)

Detalhe em `docs/wyd/pesquisa-campo-treinamento-2026-11-11.md`: stats e drops oficiais dos três
guardas (`GremlinDecoy` 30/600/120, `GiantCrill` 50/2000/140, `OrcSniper` 50/2500/160) e do
`OrcWarrior` (40/2000/240, Âmago de Javali + Emblema Orc), as **quatro sequências** de
recompensa (A legado, B oficial PT, C oficial EN, D privado com Sephira +9), as três instâncias
do `CabuncleWind`, e as regras de chave/portão (consumo no uso, portão fecha depois, guardião
volta a dropar). O Emblema Orc, no texto oficial do item, dá **recompensa aleatória** entre Rubi
do Carbúnculo, Olho Crescente e Poção de Cura (M) — diferente do guia PT.

Escopo pedido pelo dono em 2026-11-11 (depende de decisão registrada antes do código):

- NPC com **armas Sephira +9** (dano full) para iniciantes — D-8 decidiu o NPC; a arma em si
  ainda não tem linha de `item_catalog`/loja.
- Buff NPC ao longo da quest nas **três** instâncias, reaplicável.
- Guardas com recompensas corretas e lista de sequências para escolha do dono.
- Boss final completo, drops/limites/recompensas configurados.
- 4 bots autônomos (um por raça) executando a quest inteira (`GAME-23`).

## Seed do campo (2026-11-11)

`crates/wyd-db/migrations/0046_seed_campo_treinamento.sql` coloca os dez NPCs do campo no mapa
com **templates reais** já presentes em `npc_template` (blobs `STRUCT_MOB` de 816 B do snapshot
legado, com `sha256` gravado) e os quatro drops das provas/emblema em `drop_rate_override`.
A migration `0049_seed_campo_sephira.sql` complementa o seed com o NPC `Sephira +9` (id 6311),
sem reescrever a `0046` já aplicada. Nenhum blob foi inventado: os seeds referenciam
`npc_template.template_name` por FK.

| NPC | Template | Coord. | Merchant |
|---|---|---|---|
| Primeiro/Segundo/Terceiro Treinador | `FirstTrainer` / `SecondTrainer` / `ThirdTrainer` | 2077,2017 · 2144,1989 · 2082,1966 | 36 / 40 / 41 |
| Chefe de Treino | `TrainerChief` | 2122,2041 | 100 |
| Unicorn / Cabuncle | `Unicorn` / `Cabuncle` | 2130,2040 · 2131,2033 | 0 |
| Sephira +9 (`0049`) | `Refinacoes` | 2133,2040 | 0 |
| Guardiões | `GremlinDecoy` / `GiantCrill` / `OrcSniper` | 2076,2022 · 2146,1997 · 2078,1971 | 0 |
| Boss | `OrcWarrior` | 2135,1948 | 0 |

Drops: `GremlinDecoy→451`, `GiantCrill→452`, `OrcSniper→453`, `OrcWarrior→524`.

### Prova viva (2026-11-11)

- `0047_seed_limpeza_fixtures_campo.sql` removeu as fixtures `teste-campo-*` (ids 6201–6208) e os
drops `fixture-campo-teste`; o `down` as restaura. A cena passou a ter **uma** instância de
cada NPC do Campo (`crates/wyd-bots/tests/cena_campo.rs` cobra isso por coordenada).
- Aceite implementado no servidor: `gameplay::quest_accept` + `SelectionStore::quest`. Antes,
o primeiro clique de um personagem novo morria em `falta a prova 451` e nenhuma linha nascia
em `character_quest`; agora o primeiro contato grava `step=1, progress=0` e responde
`Campo de Treinamento: etapa 1/4 (aceita)`.
- Prova no servidor rodando (`crates/wyd-server/tests/campo_vivo.rs`, personagem criado pelo
`_MSG_CreateCharacter` do bot):

```
PASS(campo vivo): conta=qacampo01 personagem=QACAMPO01 id=140
  quest=campo_de_treinamento step=1 progress=0 npc=5980
```

O segundo clique, sem a chave 451, é recusado e o estado permanece `step=1, progress=0`.
Atualização de 2026-11-12: a entrega das provas pelos quatro guardiões/boss passou a ser
provada no pipeline real (`integracao_campo_drops`). Continuam abertos o movimento
(`_MSG_Move`), o affect do buff (`Cabuncle`) e a arma Sephira +9 no `item_catalog`/loja — o
NPC `Sephira +9` existe pela `0049`, o item não foi inventado.

## Teste manual do dono (2026-11-11)

Achados do cliente real, com causa e estado. Nenhum foi silenciado:

| Sintoma | Causa no código | Estado |
|---|---|---|
| 5 pontos livres no nível 1 | `starting_scores` somava `STARTING_LEVEL * SCORE_POINTS_PER_LEVEL` | **corrigido**: pontos são por nível subido (0 no nível 1) |
| nome vermelho (PK) recém-criado | `ChaosRate@15` do `BaseMob` da classe é lixo do dump (`0xCC`) e era copiado para o personagem | **corrigido**: `base_score`/`current_score` com `chaos_rate = 0` |
| personagem sem gold | o valor existe (`BaseMob.Coin@28` = 5 000 000) e o banco o gravava; o login não mandava `_MSG_UpdateEtc`, que é quem carrega `Coin@28` para o cliente | **corrigido**: par `UpdateScore` + `UpdateEtc` no `CNFCharacterLogin`; `tests/gold_inicial.rs` |
| nenhum NPC na tela | o canal envia os 6 110 `_MSG_CreateMob`; log: `event=scene result=budget limit=4096 created=4096 removed=0 pending=1` | aberto (cliente recebeu só a 1ª varredura; o teto é `WYD_SCENE_MAX_PER_REFRESH`) |
| não distribui os 5 pontos | `_MSG_UpdateScore` existe só J→C; a alocação C→J não está portada | aberto |
| baú de XP sem efeito | `_MSG_UseItem` cobre refino/pó (`BlockedByCatalog`); não há affect/efeito visual | aberto |
| lojas de NPC em Armia | **dados já no snapshot**: `npc_shop_item` com 6 706 linhas, 718 itens distintos, até 27 slots por NPC; opcodes `_MSG_REQShopList` 0x027B / `_MSG_ShopList` 0x017C só no catálogo | aberto (falta portar os opcodes e o handler) |

### Causa raiz dos "NPCs invisíveis" (2026-11-11)

O servidor entregava **todos** os NPCs do mapa no login: 6 110 `_MSG_CreateMob` de 232 B
(~1,4 MB) em duas varre duras de 4 096 + 2 014. O cliente 7662 logava `Init Field Scene` e não
desenhava nenhum. Correção na entrega, não no sintoma:

- `world::{scene_around, ENV_SCENE_VIEW_TILES}` (raio padrão 24 tiles, Chebyshev) e
  `Engine::scene_around`; a sessão passou a pedir a cena em volta do personagem a cada poll
  (a geração não muda quando o personagem anda, então o delta é recalculado a cada varredura:
  quem saiu do raio recebe `RemoveMob`, quem entrou recebe `CreateMob`).
- Prova viva (`crates/wyd-server/tests/cena_armia.rs`, personagem criado pelo protocolo):
  `PASS(cena): 33 mobs` e `PASS(armia): 32 NPCs em volta de (2096, 2096), 32 com merchant`,
  ids 1080..6941. O teste falha se a cena voltar a vir inteira (`total < 500`) ou se algum NPC
  chegar sem nome/coordenada fora do raio. Unitário: `cena_por_proximidade_corta_o_que_esta_longe`.
- `crates/wyd-bots/tests/cena_campo.rs` passou a andar até o Campo antes de conferir as dez
  instâncias (o Campo fica a ~33 tiles do spawn e agora só chega quando o char se aproxima).

O abate no Campo continua impossível para nível 1 no cliente real: o log mostra
`damage=1 hp_left=597` por golpe (`GremlinDecoy` com 598 HP), porque o ataque usa o
`CurrentScore.Damage` do template sem o dano do equipamento — o mesmo motivo pelo qual a
arma Sephira +9 é necessária no jogo real. No teste de integração, o HP do alvo é reduzido a
1 **só no fixture**; a morte, a tabela de drop, o inventário e a quest são os caminhos reais.
**Origem dos blobs:** o cliente 7662 **não** traz tabela de NPC (`docs/cliente/tabelas-bin.md` §61:
nenhum `.bin` do cliente tem tabela por monstro; `STRUCT_MOB` é dado de servidor). A extração
possível é a que já foi feita: o dump legado em `0020_seed_mundo.sql` (1991 templates).

## Revisão de prova (2026-11-12)

**Decisão:** manter 🟡. O código que não depende do cliente está fechado; o que resta é
**captura/observação visual**, não implementação de servidor.

- **Provado por teste:** a migration `0046` cria os dez NPCs do Campo com templates reais e os
  drops em `drop_rate_override`; `cena_campo` exige uma instância por coordenada. O motor de
  quest cobre as quatro etapas, o teto de nível, a ordem, o consumo da chave e os três portões;
  `integracao_campo_treinamento` (12) prova o fluxo e as recusas. O `campo_vivo` prova o
  **GremlinDecoy** dropando a chave 451 pelo fio; `integracao_campo_drops` (3) prova os quatro
  guardiões/boss pelo pipeline real de morte/loot e o consumo único de cada prova. A migration
  `0049` cria o NPC `Sephira +9` em `(2133,2040)` e o refino `+8 → +9` usa a taxa do snapshot.
  `EquipmentCatalog` aplica o `ReqLvl` real na validação de equipagem e o bônus do treinador usa
  `50 + ReqLvl` do catálogo.
- **Bloqueado por captura:** portão como objeto visível (`_MSG_UpdateItem` + `EF_KEYID`), buff
  do `Cabuncle` (affect/duração no fio) e a prova visual no cliente 7662 (entrada, drops,
  portões e conclusão na tela). Nada disso deve ser marcado como concluído sem a captura.
- **Consequência:** “NPC existe”, “os quatro guardiões/boss fecham o fluxo no pipeline real” e
  “ReqLvl é usado” estão provados. “O cliente 7662 desenha o fluxo completo” continua aberto e
  identificado como dependência de captura.

## Seções de prova no 7662 (roteiro 2026-11-12)

> Roteiro consolidado de execução: [`docs/cliente/roteiro-prova-visual-7662.md`](../../docs/cliente/roteiro-prova-visual-7662.md).
>
> **A** é a prova automatizada já fechada; **B** é o roteiro visual que ainda depende de
> captura. Não usar A para marcar B. Cada etapa visual só conta com print + log/estado do
> servidor; se o cliente não renderizar, registrar o ponto exato e parar.

### A. Prova automatizada (regressão)

```bash
cargo test -p wyd-server --test integracao_campo_treinamento
cargo test -p wyd-server --test integracao_campo_drops
cargo test -p wyd-server --test sistema_campo_vivo
node tools/content/check-migrations.mjs
```

- [x] Quatro etapas, teto 35, ordem, recusa sem prova e recompensa única
- [x] Portões: `EF_KEYID` distinto, consumo da chave, abertura e fechamento por tempo
- [x] Loot no inventário: empilhamento até 120 e recusa quando não há slot
- [x] `GremlinDecoy`/451 no fio vivo (`sistema_campo_vivo`)
- [x] `GiantCrill`/452, `OrcSniper`/453 e `OrcWarrior`/524 no pipeline real de morte/loot
- [x] NPC `Sephira +9` (`0049`, id 6311 em 2133,2040) e refino `+8 → +9`
- [x] `ReqLvl` do catálogo no bônus e na validação de equipagem
- [x] Migration `0049` aplicada; `/health` local em `schema:50` (após a `0050`)

### B. Roteiro visual no cliente (campo)

Executar na ordem. Anotar coordenada e horário em cada print; usar a conta de teste, não a do
dono. O Campo fica a ~33 tiles do spawn — andar até lá e confirmar a cena por
proximidade antes de interagir. **Pré-condição de B3–B7:** o personagem precisa ter dano
suficiente para abater os guardiões; se o nível 1 continuar batendo 1, registrar o bloqueio de
dano/equipamento e não simular a morte no cliente.

| # | Ação | Resultado que precisa aparecer | Evidência | [ ] |
|---:|---|---|---|---|
| B1 | Entrar no Campo e olhar em volta | 11 NPCs do seed visíveis: 4 treinadores, Unicorn, Cabuncle, Sephira +9, 3 guardiões e OrcWarrior | print amplo + log de cena | |
| B2 | Falar com o Primeiro Treinador | Primeiro clique **aceita** a etapa 1 (`etapa 1/4`), sem “falta a prova 451” | print + `character_quest step=1` | |
| B3 | Matar `GremlinDecoy` e conferir o inventário | Chave 451 entra no inventário; se a mochila estiver cheia, o drop fica no chão | print do drop/inventário + log `quest`/loot | |
| B4 | Usar a chave 451 no portão | Portão abre visualmente, a chave é consumida e fecha depois do tempo | print antes/depois + log do portão | |
| B5 | Repetir etapas 2 e 3 com `GiantCrill`/452 e `OrcSniper`/453 | Segundo/terceiro treinadores aceitam, encantam `Equip[6]`/`Equip[1..7]` + sanc 3; portões 2 e 3 abrem | prints + estado do equipamento | |
| B6 | Falar com `Cabuncle` | Buff/affect visível e reaplicável; HP/MP ou ícone mudam conforme o snapshot | print antes/depois + log do affect | |
| B7 | Matar `OrcWarrior` e falar com o Chefe | Emblema 524 entra; recompensa final sorteada (682/481/652) aparece **uma vez** | print do inventário + log da recompensa | |
| B8 | Sair e relogar | Etapa/conclusão persistem; repetir a entrega é recusado, sem recompensa dupla | print + `character_quest completed` | |

### C. Coleta de evidência e gate

- [ ] Print de cada B1–B8 com coordenada/horário
- [ ] Log do canal no mesmo intervalo, com `npc`, `merchant`, `quest step`, loot e portão
- [ ] Estado do banco (`character_quest`, inventário) anexado sem dados pessoais
- [ ] B1–B8 marcados individualmente; o que falhar vira bloqueio com o ponto exato
- [ ] Se o portão não aparecer, registrar `_MSG_UpdateItem`/`EF_KEYID` ausente — não simular visual
- [ ] Se o buff não aparecer, registrar affect/duração ausentes no fio — não inventar pacote
- [ ] Se o abate nível 1 não for possível, registrar dano/equipamento e usar a arma prevista, sem
  reduzir HP no cliente

### D. Definition of done visual

- [ ] Entrada, aceite, quatro provas, três portões, buff e boss observados na tela
- [ ] `Sephira +9` e `ReqLvl` confirmados no cliente, além do teste
- [ ] Relogin/persistência e recusa de recompensa repetida confirmados
- [ ] Capturas e logs anexados à issue; GAME-18 só então passa de 🟡

## Subtasks

- [x] Decidir D-1 (nível 35), D-2 (sequência A) e D-6 (Unicorn)
- [x] Codificar `_MSG_Quest` + `character_quest` + motor/recompensas
- [x] Testar (fluxo + recusas) e documentar o estado real
- [x] Seed dos NPCs/drops no snapshot (`0046`; 10 NPCs, um por coordenada)
- [x] `GremlinDecoy` dropa a chave 451 no combate real (`campo_vivo`)
- [x] GiantCrill/OrcSniper/OrcWarrior no fluxo completo de combate/drop (`integracao_campo_drops`)
- [ ] Portões como objeto de mundo (`_MSG_UpdateItem` + `EF_KEYID`) — **bloqueado por captura**
- [x] Chave consumida no uso; portão abre e fecha (testes de `gameplay`)
- [x] NPC `Sephira +9` no seed (`0049`, id 6311 em 2133,2040) e refino do +9 no teste
- [x] `ReqLvl` do catálogo usado no bônus `50 + ReqLvl` e na validação de equipagem
- [ ] Buff do `Cabuncle` aplicado e reaplicável — **bloqueado por captura**
- [ ] Provar com o cliente 7662 aberto — **bloqueado por captura**

## Critérios de aceite

- [x] Personagem novo (nível 1..=35) conclui as 4 etapas no motor e recebe cada recompensa.
- [x] A recompensa de cada etapa é exatamente a da sequência A; nível 36 é recusado.
- [x] NPCs/guardiões/boss existem no snapshot e na cena (`0046` + `cena_campo`).
- [x] `GremlinDecoy` dropa a chave 451 no combate real e a entrega no inventário (`campo_vivo`).
- [x] GiantCrill/OrcSniper/OrcWarrior dropam no fluxo completo de combate (`integracao_campo_drops`; HP do alvo reduzido só no fixture).
- [x] Chave consumida no uso; portão abre e fecha (testes de `gameplay`).
- [x] Chave volta a dropar se perdida no combate dos demais guardiões (tabela do seed, um drop por abate).
- [ ] Buff do `Cabuncle` aplicado e reaplicável — **bloqueado por captura**.
- [x] Boss dropa Emblema; recompensa final uma vez por personagem (`integracao_campo_drops`).
- [x] NPC `Sephira +9` no seed (`0049`) e refino do +9 encontrando a coordenada.
- [x] `ReqLvl` do catálogo usado no bônus e na validação de equipagem.
- [x] Nenhum pacote inventado na entrega atual; valores de balanceamento vêm do snapshot.

## Riscos / rollback

- Risco: recompensa duplicada → mitigação: `UNIQUE (character_id, quest_id)` + teste de replay.
- Risco: seed com ids errados → mitigação: conferir `item_catalog` antes (ids 451/452/453/524/682).
- Rollback: desligar o NPC da quest no snapshot (nenhuma migration destrutiva).
