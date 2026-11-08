# Issue 03-2 — ECS: componentes contíguos das entidades

**Fase:** 3 · **Depende de:** 03-1 · **Bloqueia:** 03-5, FASE-10 · **Esforço:** L

## Objetivo

Modelar Players, NPCs, Monsters e Items no chão como dados contíguos
(`Vec<Position>`, `Vec<Stats>`, `Vec<Inventory>`), sem hierarquia OOP.

## Contexto / evidência

`pre-release-rust.md` FASE 3 §3.1 + §0.2 (DOD/ECS, `hecs`/`bevy_ecs` via ADR);
`start-rust.md` §2: OOP com ponteiros brutos gera cache misses — abandonar
`Player`/`Mob` como objetos espalhados no heap.

## Passos

1. Definir componentes em `crates/wyd-core/src/ecs/components.rs` (árvore §10; posição, stats,
   inventário, flags) — dados, sem lógica.
2. Escolher o ECS via ADR (hecs vs bevy_ecs) e registrar em
   `docs/arquitetura/adr/`.
3. Sistemas mínimos (spawn, despawn, movimento) operando em lote sobre os
   componentes.
4. Testes: spawn/despawn em massa + iteração contígua; `make check`.

## Arquivos

- Criados: `crates/wyd-core/src/ecs/components.rs`, `src/systems/` (spawn, despawn, movimento), testes.
- Alterados: `crates/wyd-core/src/lib.rs`, `Cargo.toml` (ECS via ADR).
- Proibido tocar: lógica de fórmulas (FASE-10), OOP no core.

## Testes / verificação

```bash
cargo test -p wyd-core
cargo bench -p wyd-core 2>&1 | tail -n 5
make check
```

Saída esperada: `test result: ok`; bench de iteração publicado como baseline.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-core - ECS e DOD.
- .agents/skills/rust-code-quality - ownership, testes.
- .agents/skills/software-architecture - ADR do ECS.
- .agents/skills/solid-design-patterns - sistemas coesos, traits pequenas.

## Documentação de referencia

- pre-release-rust.md (FASE 3 §3.1, §0.3 de-para).
- docs/rust/01-dependencias.md - candidatos ECS.
- docs/arquitetura/adr/ - ADR da escolha.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

> Nota (2026-11-07): a base de dados (componentes + `ComponentStore` SoA) foi
> entregue na FASE-03; restam `src/systems/`, a ADR do ECS e o bench baseline
> (exigem decisão com rede/vendor e carga real — fora do escopo desta entrega).
>
> Decisão registrada: ADR-0004 (`docs/arquitetura/adr/0004-ecs-manual-diferido.md`) —
> SoA manual até o bench mandar o contrário; `hecs`/`bevy_ecs`/`bumpalo` diferidos.

## Critérios de aceite

- [ ] 4 tipos de entidade como componentes contíguos; ADR da escolha do ECS.
- [ ] Bench baseline anexado; sem `unsafe` fora de interop comprovada.

## Riscos / rollback

- Risco: ECS inadequado à carga de guerras → mitigação: bench antes/depois;
  trocar exige nova ADR.
- Rollback: `git checkout -- crates/wyd-core/`.

## Progresso 2026-11-08 — gema como transação do inventário

- **Caminho único:** `ComponentStore::apply_gem(entity, slot, gem_item)` opera no inventário
  real da entidade; `Inventory::{plan_gem, consume_gem, commit_gem, apply_gem_from_bag}` ficaram
  `pub(crate)` (fora do crate só existe o caminho com entidade viva, estoque e evento).
- **Transação:** `plan_gem` valida sem escrever → estoque conferido (`GemNotInBag`) → espaço na
  fila (`EventQueueFull`) → `DomainEvent::GemEffectChanged` publicado → `consume_gem` retira uma
  unidade → `commit_gem` grava o efeito. Toda recusa acontece antes da primeira escrita.
- **Fonte:** `docs/wyd/sistemas/refino.md` §3 (gema aplicada sobre o item; vale para Anct e
  +10+; efeito trocável; 40 por nível acima de +9; drop 8% e EXP 2% fixos).
- **Provisório:** D-12 (a gema é consumida — a fonte descreve aplicar sobre o item, não diz
  quantas unidades; assumimos uma) e D-13 (reaplicar a mesma gema é permitido: o efeito não é
  definitivo, então "esgotada" é estoque, não "já aplicada"), em
  `docs/arquitetura/matematica-sim.md` §4. ADR-0012 registra a decisão.
- **Prova:** `crates/wyd-core/tests/gem.rs` (6 testes de API pública) + `gem_tests` em
  `components.rs` (11): consumo unitário, recusa sem mutação parcial, evento único por
  aplicação e zero nas recusas, troca de efeito carregando o delta, fila cheia sem consumir, e
  **Anct +9 (40 de perfuração) estritamente melhor que comum +9 (recusado, 0)**.
- **Ainda aberto:** `systems/` (spawn/despawn/movimento em lote), ADR de ECS (hecs vs bevy_ecs)
  e bench de iteração contígua.
