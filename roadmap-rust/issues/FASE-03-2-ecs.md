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

## Fechamento (2026-11-09, revisão)

Os três itens que a nota de 2026-11-07 deixava abertos foram entregues e provados:
`src/systems/` existe (`movement`, `combat`, `skill`, `inventory`, `npc`, `compose`), a ADR do
ECS é a **ADR-0004** (SoA manual, decisão mantida) e o bench baseline está no bloco
“fechamento: lote tudo-ou-nada + baseline SoA” (10 000 entidades, p50 3 381,70 µs). Não há
item aberto: o que falta para o mundo existir é o **processo** rodar o tick (`FASE-03-7`), não
o ECS.

## Critérios de aceite

- [x] 4 tipos de entidade como componentes contíguos; ADR da escolha do ECS.
- [x] Bench baseline anexado; sem `unsafe` fora de interop comprovada.

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

## Progresso 2026-11-08 — fechamento: lote tudo-ou-nada + baseline SoA

- **Lote na fronteira fria do armazém:** `ComponentStore::spawn_batch(&[SpawnSpec])` valida
  **todos** os `Stats` antes da primeira escrita (um inválido no meio do lote devolve
  `CoreError::InvalidStats` e o armazém fica como estava — nem índice novo) e
  `despawn_batch(&[Entity])` devolve quantos ids eram válidos, idempotente como `despawn`.
  `SpawnSpec { kind, position, stats }` é dado de entrada; inventário nasce vazio porque item
  inicial é entrega de conteúdo, não default do armazém. Os quatro tipos (`Player`, `Npc`,
  `Monster`, `GroundItem`) seguem como `Vec<EntityKind>` contíguo ao lado de `Vec<Position>`,
  `Vec<Stats>` e `Vec<Inventory>`.
- **Escolha do ECS:** ADR-0004 já decidia SoA manual até o bench mandar o contrário; o baseline
  agora existe, então a decisão deixa de ser por intuição.
- **Baseline medido** (`cargo bench -p wyd-core --bench tick`, release, 10 000 entidades por
  rodada, 40 rodadas de spawn + uma passada contígua em `positions()` + despawn): **p50 3 381,70 µs
  · p99 11 427,80 µs · 383,867 ns por entidade**, p99 = **22,856%** do orçamento de um tick de
  50 ms (teto do cenário: 50%). Estado final conferido: 0 entidades vivas, 10 000 índices no
  armazém (índices são reciclados, não vazam).
- **Testes:** `spawn_em_lote_e_tudo_ou_nada` (lote recusado não cria entidade nem índice) e
  `despawn_em_lote_conta_apenas_ids_vivos` (60 de 100, repetição do lote morto devolve 0, resto
  devolve 40), em `crates/wyd-core/src/ecs/components.rs`; `cargo test -p wyd-core` → todos os
  alvos `ok`, **91 testes na lib** incluídos.
- **Nada aberto nesta issue:** `systems/` já cobria movimento/combate/skill/inventário/NPC (03-5/03-6).
