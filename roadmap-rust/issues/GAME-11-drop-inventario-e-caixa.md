# GAME-11 — Drop no inventário e caixa no chão com posse

> **ID canônico:** `GAME-11`
> **ID histórico:** `FASE-03-11`
> **Estado 2026-11-12:** 🚧 · depósito no inventário entregue; falta a caixa no chão. Ver `roadmap-rust/index.md`.

## Objetivo

Ligar o abate ao **depósito no inventário** e, só no excedente, à caixa no chão com posse
exclusiva de 30 s, liberação para qualquer jogador e remoção em 3 min.

## Contexto (já entregue)

- Sorteio por item (`Gameplay::loot_at_death`) e morte no motor (`Engine::attack_at`) estão ligados
  ao `_MSG_Attack` desde a CLIENT-02.
- Regra de depósito: `crates/wyd-core/src/systems/inventory.rs::deposit` (pilha → slot livre →
  sobra), com 5 testes.
- Janela da caixa: `crates/wyd-sim/src/formula/drop.rs::GroundLoot`/`LootAccess` (30 s exclusivo,
  3 min de vida), com 3 testes.
- Layout e direção dos pacotes provados no binário: `docs/cliente/pacotes.md` §"Caixa de item no
  chão e pickup" (`CreateItem` 0x026E S→C 30 B, `GetItem` 0x0270 C→S 28 B, `CNFGetItem` 0x0171,
  `DecayItem` 0x016F).

## Estado real (2026-11-12)

- [x] Abate entrega o item no inventário do matador pelo `Gameplay::loot_at_death` → `inventory::deposit` (teto de pilha vindo de `item_stack_cap`); provado no Campo (`integracao_campo_drops`, 3/3).
- [x] Regra pura da caixa (`GroundLoot`/`LootAccess`, 30 s exclusivo/3 min) com testes em `wyd-sim`.
- [ ] Handler/tabela `CreateItem`/`GetItem`/`DecayItem` por canal; o `gameplay/service.rs` ainda declara que caixa e pickup não têm handler.
- [ ] Faixa de ids de item no chão do 7662 (UNVERIFIED) — captura CLIENT-06.

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

- [x] Depósito no inventário no abate, com teto de pilha e prova no pipeline real.
- [ ] Teste de integração com dois personagens: durante 30 s o segundo recebe recusa; aos 30 s o
  segundo consegue; aos 180 s a caixa não existe mais.
- [ ] Round-trip `CreateItem`/`GetItem` contra os layouts de `docs/cliente/pacotes.md`.
- [ ] `make check` verde; docs atualizados (`drops.md` §8 e `pacotes.md`).

## Aberto (bloqueia o passo 2)

Faixa de ids de caixa: `MAX_USER` 1000 e `MAX_MOB` 25000 são conhecidos, a faixa própria de item
no chão do 7662 não — **UNVERIFIED**.

## Skills relacionadas

- `wyd-rust-core` — estado do chão/inventário no Hot Path.
- `wyd-rust-db` — persistência da posse e transações.
- `wyd-rust-net` — `CreateItem`/`GetItem`/`DecayItem` no fio.
- `rust-code-quality` / `rust-test-quality` — implementação e testes de contrato.

## Gates

```bash
cargo test -p wyd-core inventory
cargo test -p wyd-sim drop
cargo test -p wyd-server --test integracao_campo_drops
make check
```
