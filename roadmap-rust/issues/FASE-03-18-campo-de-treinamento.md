# FASE-03-18 — Campo de Treinamento: quest, portões, buff, boss e recompensas

**Fase:** 3 · **Depende de:** FASE-03-17 (entrada no mundo), FASE-04-2 ✅ (migrations), FASE-07-5 ✅ (conteúdo) · **Bloqueia:** FASE-15-4 (bots) · **Esforço:** L

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

Consequência prática: `682` (Kit de Cura 20 M, `EF_HP` 200) + affect 44 (200/200) na etapa 1;
encantamento de `Equip[6]` na etapa 2; `Equip[1..7]` + sanc 3 na etapa 3; sorteio de
`682/481/652` na etapa 4. Nada de refino +6 (B), “Classe A” (C) ou Sephira (D).

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
   Não há NPC de Sephira +9 nesta entrega.
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
cargo test -p wyd-server --test campo_treinamento
cargo test -p wyd-db --test catalogo
node tools/content/check-migrations.mjs
```

Saída esperada: 0→4 com item correto, recusa sem item, chave consumida e portão aberto, boss
dropando o emblema, recompensa uma vez por personagem.

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

**Aberto (não implementado, e por quê):**

1. **NPCs do campo no mundo.** `npc_definition` exige `template_name` de `npc_template`
   (blob `STRUCT_MOB` de 816 B vindo do cliente). Não há template dos treinadores do Campo
   no snapshot (`FirstTrainer`/`SecondTrainer`/… ausentes), e inventar blob é proibido. Falta
   extrair os templates do cliente 7662 (`tools/client/`) antes do seed. **Fixture de teste
   (pendente):** o caminho pretendido é um loader de teste que monte `NpcSpec` em memória
   (sem tocar `npc_definition`) e um `DropConfig` sintético para as provas — o teste ponta a
   ponta com os 4 bots depende disso.
2. **Guardiões e boss no mundo.** Mesmo motivo para `GremlinDecoy`/`GiantCrill`/`OrcSniper`/
   `OrcWarrior`; sem template não há `npc_definition`, combate, HP 2.000 nem `drop_table`.
   O emblema 524 entra no teste como drop já obtido (`ItemDropList.txt`).
3. **Portões como objeto visível.** A mecânica está implementada (chave por `EF_KEYID`,
   consumo, fechamento por tempo), mas o portão não é uma entidade da cena: o cliente 7662
   não vê a porta nem recebe o `EF_KEYID` no `_MSG_UpdateItem` do legado. Falta o item de
   mundo/porta no snapshot para o fluxo visual.
4. **NPCs Unicorn (2130,2040) e Cabuncle (2131,2033).** Também dependem de template/posição;
   a arma inicial já chega pelo `BaseMob` do template de classe, e o affect 44 tem semântica
   mas não tem ícone/duração no fio (não há pacote de affect portado).
5. **`ReqLvl` do equipamento.** Não está no modelo (`ItemRow`); o bônus usa o nível do
   personagem como base até o catálogo expor a coluna.

## Subtasks

- [x] Decidir D-1 (nível 35), D-2 (sequência A) e D-6 (Unicorn)
- [x] Codificar `_MSG_Quest` + `character_quest` + motor/recompensas
- [x] Testar (fluxo + recusas) e documentar o estado real
- [ ] Seed dos NPCs/monstros/drops (depende dos templates do cliente)
- [ ] Portões como objeto de mundo (`_MSG_UpdateItem` + `EF_KEYID`)
- [ ] Provar com o cliente 7662 aberto
- [ ] Versionar issue, mergear branch e fechar issue

## Critérios de aceite

- [x] Personagem novo (nível 1..=35) conclui as 4 etapas no motor e recebe cada recompensa.
- [x] A recompensa de cada etapa é exatamente a da sequência A; nível 36 é recusado.
- [ ] NPCs/guardiões/boss existem no mundo e dropam as provas (bloqueado por template).
- [ ] Chave consumida no uso; portão abre e fecha; chave volta a dropar se perdida.
- [ ] Buff do NPC aplicado e reaplicável.
- [ ] Boss dropa Emblema; recompensa final uma vez por personagem.
- [ ] Nenhum pacote inventado; todo valor de balanceamento vem do snapshot.

## Riscos / rollback

- Risco: recompensa duplicada → mitigação: `UNIQUE (character_id, quest_id)` + teste de replay.
- Risco: seed com ids errados → mitigação: conferir `item_catalog` antes (ids 451/452/453/524/682).
- Rollback: desligar o NPC da quest no snapshot (nenhuma migration destrutiva).
