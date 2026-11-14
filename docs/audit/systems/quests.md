# Sistema — Quests e missões

> **Paridade: P3 no Campo de Treinamento; P0 em todo o resto.** Existe um motor mínimo funcional
> para uma quest; story line, Kefra, runas e Arch não têm nenhuma linha.

## 1. Escopo

Estrutura, IDs, tipos, pré-requisitos, aceitar/recusar/abandonar, progresso, objetivos,
contadores, eventos, NPCs, itens/monstros necessários, recompensas, conclusão, repetição, limites,
diárias, encadeadas, persistência, estados, interface.

## 2. Evidências

| Camada | Evidência |
|---|---|
| Legado | `_MSG_Quest.cpp` (2.753 linhas), Go `runequest*.go`, `newbiequest.go`, `kefra*.go` |
| Cliente | `0x028B` Quest (20 B); diálogo por `0x0101` |
| Rust | `cpsock::handle_quest`, `gameplay::{quest_engage,apply_quest_reward}`, `payload/quest.rs` |
| Banco | `0044_seed_quest_campo` (`stage`, `completed`, `reward_mask`), `0046/0049_campo` |
| Testes | `integracao_campo_treinamento` **7/7**, `campo_drops`, `cena_armia` |
| Doc | `docs/wyd/sistemas/quests.md`, `GAME-18` |

## 3. Regras confirmadas (Campo de Treinamento)

| Regra | Valor |
|---|---|
| Payload | `_MSG_Quest` 20 B |
| Estado | `character_quest(stage, completed, reward_mask)` |
| Recompensa | sequência A (682; `Equip[6]` 50+ReqLvl; `Equip[1..7]`+sanc 3; sorteio 682/481/652) |
| Teto de nível | 35 |
| Prova consumida | sim |
| Portões | `CAMP_GATES` com `EF_KEYID` 2/3/4; chave consumida; fecha em 30 s |
| Acionamento | `_MSG_Action` no tile → `Engine::use_key_at` |
| Bots | 4 raças, campanha de 4 etapas (3/3) |
| Etapas | 4 |

## 4. Modelo de dados atual

```
character_quest: character_id, quest_id, stage, completed, reward_mask
```

Não há: objetivo tipado (matar/coletar/falar), contador genérico, pré-requisito, cadeia,
repetição, diária, limite por conta.

## 5. Estado por item

| Item | Status | Paridade |
|---|---|---|
| Campo de Treinamento | Implementado, sem prova visual | P3 |
| Motor genérico de quest | ausente | P0 |
| Story line | ausente | P0 |
| Kefra/runas/Arch | ausente | P0 |
| Diárias/repetição | ausente | P0 |
| Indicador visual no cliente | não provado | P1 |

## 6. Lacunas e ações

1. Generalizar: `quest_definition` + `quest_objective` (tipo, alvo, quantidade).
2. Eventos de progresso: matar, coletar, falar, entrar em mapa.
3. Recompensa declarativa (item/EXP/ouro/flags), com `reward_mask` como histórico.
4. Cadeia/pré-requisito e repetição com cooldown.
5. Prova visual do Campo (`GAME-18`) antes de expandir.

## 7. Riscos

| Risco | Nota |
|---|---|
| Motor por código | cada quest vira `if` |
| Recompensa duplicada | precisa de idempotência por etapa |
| Condição sem evento | progresso “congela” |
| Quest de servidor privado | conteúdo não oficial |
