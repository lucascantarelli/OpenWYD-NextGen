# GAME-09 — Cena autoritativa: NPC não clica e mundo não desenha o esperado

> **ID canônico:** `GAME-09`
> **ID histórico:** `GAME-09`
> **Estado 2026-11-13:** ⬜ · **reaberta** · ver `roadmap-rust/index.md`.

**Tipo:** Bug · **Esforço:** M · **Depende de:** `CLIENT-06` (captura de `_MSG_CreateMob`/`_MSG_Action`) · **Bloqueia:** `GAME-18`, onda 1

## Objetivo

A cena entregue ao 7662 tem que ser **clicável e coerente**: o jogador vê os NPCs, clica e recebe
o painel/quest correspondente. Hoje o relato é “npc não clica”.

## Contexto / evidência

- **Relato do dono (2026-11-13):** “npc não clica”.
- A cena é por proximidade (`world::scene_around`, raio 24) e o `_MSG_CreateMob` leva
  `HEADER.ID = ESCENE_FIELD` (30000), `MobID` e posição; o clique volta como `_MSG_Action` com
  `TargetX/TargetY` e o servidor resolve NPC por tile (`handle_action` → `store.talk`).
- Teste de sistema `sistema_campo_vivo` de 2026-11-13 **encontrou** o `GremlinDecoy` e o
  Primeiro Treinador na cena e o aceite da quest respondeu “aceita” via bot — ou seja, o caminho
  de fio funciona; o que falha é o **cliente** (render/clique) ou o id/posição enviados.
- A issue foi fechada com testes de bytes (`scene`, `visibility`, `cena`), sem prova visual.

## Passos

1. Capturar no 7662 um clique em NPC (`_MSG_Action`) e o `_MSG_MessagePanel`/quest de resposta;
   registrar `TargetX/TargetY`, `MobID` clicado e o NPC resolvido no log do canal.
2. Comparar o `_MSG_CreateMob` real com o esperado pelo cliente: `MobID`, `CreateType`,
   `Merchant`, `Class`, nome e posição — um id fora da faixa faz o cliente ignorar o clique.
3. Verificar a ordem/rajada da cena por proximidade (o cliente pode descartar pacotes fora de
   `ESCENE_FIELD` ou com `ID` de jogador).
4. Corrigir só com captura; sem captura, registrar como `UNVERIFIED` e manter o caminho atual.

## Critérios de aceite

- [ ] Print de NPC clicado abrindo painel/quest no 7662, com log do canal no mesmo instante.
- [ ] `MobID`/`CreateType` do `CreateMob` conferidos com a captura.
- [ ] Cena por proximidade documentada (raio 24) sem “piscar” NPC ao andar.
