# FASE-03-11 — Drop no inventário e caixa no chão com posse

## Objetivo

Ligar o abate ao **depósito no inventário** e, só no excedente, à caixa no chão com posse
exclusiva de 30 s, liberação para qualquer jogador e remoção em 3 min.

## Contexto (já entregue)

- Sorteio por item (`Gameplay::loot_at_death`) e morte no motor (`Engine::attack_at`) estão ligados
  ao `_MSG_Attack` desde a FASE-08-2.
- Regra de depósito: `crates/wyd-core/src/systems/inventory.rs::deposit` (pilha → slot livre →
  sobra), com 5 testes.
- Janela da caixa: `crates/wyd-sim/src/formula/drop.rs::GroundLoot`/`LootAccess` (30 s exclusivo,
  3 min de vida), com 3 testes.
- Layout e direção dos pacotes provados no binário: `docs/cliente/pacotes.md` §"Caixa de item no
  chão e pickup" (`CreateItem` 0x026E S→C 30 B, `GetItem` 0x0270 C→S 28 B, `CNFGetItem` 0x0171,
  `DecayItem` 0x016F).

## Escopo

1. No abate, para cada item sorteado: `deposit` no personagem que matou; o que sobrar vira caixa
   com `GroundLoot::new(character_id, now_ms)`.
2. Tabela de caixas por canal (id → `GroundLoot` + item + posição), com faixa de ids **fora** de
   jogador (1..=1000) e de NPC.
3. Emitir `_MSG_CreateItem` (30 B packed) aos presentes na área; `_MSG_DecayItem` quando a janela
   fechar em 3 min.
4. `_MSG_GetItem` (aceitar **28 B**, com padding): validar posse pela janela, distância e slot
   destino; gravar em `character_items`; responder `_MSG_CNFGetItem` + `_MSG_DecayItem`.
5. Log auditável por caixa: `event=ground_drop` (id, item, posição, dono, expira em).

## Fora de escopo

- Drop em grupo com divisão automática (`partydif` do `game_config`) — issue futura.
- Fada prateada (agrupa/vende automático): depende do sistema de fadas.

## DoD

- Teste de integração com dois personagens: durante 30 s o segundo recebe recusa; aos 30 s o
  segundo consegue; aos 180 s a caixa não existe mais.
- Round-trip `CreateItem`/`GetItem` contra os layouts de `docs/cliente/pacotes.md`.
- `make check` verde; docs atualizados (`drops.md` §8 e `pacotes.md`).

## Aberto (bloqueia o passo 2)

Faixa de ids de caixa: `MAX_USER` 1000 e `MAX_MOB` 25000 são conhecidos, a faixa própria de item
no chão do 7662 não — **UNVERIFIED**.
