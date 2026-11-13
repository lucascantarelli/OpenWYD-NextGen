# Sistema — NPCs e interações

> **Paridade: P5 na IA, P4 no spawn, P2 em diálogo/loja, P0 em serviços (teleporte).**
> O NPC existe no mundo e pensa; ele ainda não conversa como no legado.

## 1. Escopo

NPCs, diálogos, opções, serviços, lojas, teleportes, missões, entrega de item, recompensa,
condições, distância, estados, eventos, mensagens, scripts.

## 2. Evidências

| Camada | Evidência |
|---|---|
| Legado | `CMob.cpp`, `CNPCGene.cpp`, `_MSG_Quest.cpp`, `_MSG_REQShopList/ReqBuy/Buy/Sell.cpp` |
| Cliente | `CreateMob` (232 B), `0x028B` Quest (20 B), `0x027B/0x0379/0x037A/0x0398` |
| Rust | `core/npc/{brain,behavior,budget,route,state}.rs`, `systems/npc.rs`, `world/npc.rs`, `gameplay::quest_engage` |
| Banco | `0044_seed_quest_campo`, `0046_seed_campo_treinamento`, `npc_definition` |
| Testes | `integracao/npc`, `sistema/{cena_mobs,campo_vivo,loja_viva}`, `campo_drops` |
| Doc | `docs/wyd/sistemas/mundo.md`, `GAME-18` |

## 3. Regras confirmadas

| Regra | Valor |
|---|---|
| Tick de IA | 5 Hz com orçamento por tick |
| Modelo | FSM + utilidade (não é o `CMob` do legado) |
| Spawn | NPC nasce com o mapa; morte remove |
| Rotas | `npc/route.rs` |
| Merchant do Campo | responde `0x028B` |
| Guardiões/boss | no pipeline de morte/loot; buff do `Cabuncle` ausente |
| Loja | existe teste de “loja viva” (ver `DIV-09`) |

## 4. Fluxo de interação (quando completo)

```
jogador clica NPC → Action → valida distância/estado → abre diálogo (0x028B/0x0101)
  → opção (quest/loja/teleporte) → valida condição → efeito → resposta
```

Hoje só o primeiro e o último passo existem, e só para o merchant do Campo.

## 5. Estado por item

| Item | Status | Paridade |
|---|---|---|
| Spawn/despawn | Implementado e testado | P4 |
| IA (FSM/utilidade) | Paridade confirmada (modelo próprio) | P5 |
| Diálogo genérico | parcial | **P2** |
| Loja | parcial + divergência doc | P3 |
| Teleporte por NPC | ausente | P0 |
| Script de NPC | ausente | P0 |
| Missão por NPC | parcial (Campo) | P2 |

## 6. Lacunas e ações

1. Modelo de diálogo dirigido por dados (nó/opção/condição), não por código.
2. Reconciliar loja: doc ⬜ × teste ✅ (`DIV-09`); cobrir preço/estoque/saldo.
3. Serviço de teleporte reaproveitando `ReqtEleport` (`PAR-MOV-03`).
4. Distância de interação: validar no servidor (hoje não há evidência).

## 7. Riscos

| Risco | Nota |
|---|---|
| NPC sem template | cliente desenha mob genérico |
| Diálogo hardcoded | cada NPC vira código |
| Loja sem transação | item grátis/duplicação |
| Distância não validada | interagir de longe |
