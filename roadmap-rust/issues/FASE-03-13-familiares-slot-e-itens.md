# FASE-03-13 — Familiares: slot, Nyerds, Imp, Griupan e Trajes

## Objetivo

Implementar o slot Familiar com os itens que o ocupam, começando pelo único efeito provado
(Nyerds) e deixando Imp/Griupan bloqueados até haver fonte do bônus.

## Contexto

Dossiê completo em `docs/wyd/sistemas/familiares.md`. Resumo do que tem fonte:

| Item | Fato | Nível |
|---|---|---|
| Slot Familiar | equipa Nyerds, Imp, Griupan e Trajes; também recebe Círculo Divino Puro (reset) | B |
| Nyerds | comprada no NPC Seller; **refinável** | A/B |
| Nyerds | restaura **100% de HP e MP** ao renascer (MP adicionado no WYD Global, 2024) | A |
| Imp | dropado por boss de dungeon | C |
| Griupan | **composto** (guia "Criando Griupan") | C |
| Boneco de neve | **UNVERIFIED** — nenhuma fonte WYD; o que existe é Traje de Boneco de Madeira | — |

## Escopo

1. **Ler o índice do slot Familiar** no build 7662 (`Equip[16]`), como parte da FASE-08-6. Sem
   isso, nada de código assume posição.
2. `ItemRules::equip_slot` passa a mapear `kind = familiar` para o índice lido; o slot aceita
   Nyerds/Imp/Griupan/Traje e recusa o resto (com teste).
3. Nyerds: item no catálogo, refinável pelo caminho existente, e efeito de respawn
   (HP/MP 100%) aplicado na rotina de renascimento — depende da FASE-04 (persistência de HP/MP).
4. Traje: apenas como cosmético de duração; sem efeito de combate.

## Fora de escopo

- Bônus passivo de Imp/Griupan: **bloqueado** por falta de fonte (invariante 5).
- Quests de familiar: nenhuma confirmada.
- "Boneco de neve": não implementar sem fonte.

## DoD

- Teste: equipar Nyerds no slot lido, recusar arma no mesmo slot.
- Teste: renascer com Nyerds restaura HP e MP ao máximo; sem Nyerds, não.
- `make check` verde e `familiares.md` sem nenhum item marcado como provado sem fonte.

## Aberto

Índice do slot no 7662 (binário), bônus de Imp/Griupan (fonte), receita do Griupan (fonte).
