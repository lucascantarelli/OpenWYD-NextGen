# GAME-19 — Combate: personagem causa 1 de dano e não mata nem o NPC de treino

> **ID canônico:** `GAME-19`
> **ID histórico:** `GAME-19`
> **Estado 2026-11-13:** ⬜ · **reaberta** · ver `roadmap-rust/index.md`.

**Tipo:** Bug · **Esforço:** M · **Depende de:** `GAME-08` (score do personagem) · **Bloqueia:** `GAME-18` (Campo de Treinamento), onda 1

## Objetivo

O dano do jogador sai do **personagem real** (nível, atributos e equipamento) e é suficiente para
matar um monstro de treino. Hoje, no canal vivo, cada golpe tira **1**.

## Contexto / evidência (medido em 2026-11-13)

- Personagem nível 1 da conta `qacampo01`, com equipamento inicial persistido
  (`character_items` slots 0–6, arma `1139`, munição `401`), atacando `GremlinDecoy` (`npc 6307`,
  `2076,2022`) no canal 1:
  `event=attack result=hit damage=1 hp_left=599/598/597/596 ... client_damage=0/0`.
- `GremlinDecoy` tem **600 HP**; com 1 de dano são 600 golpes — o teste `sistema_campo_vivo`
  (limite de 20 s) nunca fecha o abate.
- `Mob::from_persisted` copia `equip`, mas `current_score` continua o do template/`apply_character`;
  não há ponto evidente que some o bônus do item ao `damage` consumido por
  `world::combat::TemplateDamage` (`base: attacker.attack`).
- `GAME-19` foi fechada em 2026-11-12 com o **exemplo numérico** da fórmula e testes de
  `wyd-sim` (13 passed), sem provar o dano real de um personagem no canal.

## Passos

1. Provar de onde vem `attacker.attack` no `handle_attack`: `Mob::current_score.damage` no login
   e o que `apply_character`/`apply_progression` sobrescrevem.
2. Aplicar o bônus de equipamento (arma/munição) no `Score` do personagem e cobrir com golden
   (`STRUCT_MOB` + dano esperado).
3. Repetir no canal vivo: personagem novo deve matar `GremlinDecoy` em tempo plausível e o log
   deve mostrar `damage > 1` e `killed=true`.
4. Confirmar no 7662 que a barra de HP do monstro cai e o abate é visível.

## Critérios de aceite

- [ ] Dano real > 1 para personagem com arma inicial, provado por log do canal.
- [ ] `killed=true` no `GremlinDecoy` e drop da chave 451 dentro do tempo do teste.
- [ ] Exemplo numérico da `GAME-19` continua batendo com o teste de `wyd-sim`.
- [ ] Print do abate no cliente.
