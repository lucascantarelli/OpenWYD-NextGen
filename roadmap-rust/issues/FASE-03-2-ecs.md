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
- Proibido tocar: lógica de fórmulas (FASE-10), `.deprecated-go/`, OOP no core.

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

## Critérios de aceite

- [ ] 4 tipos de entidade como componentes contíguos; ADR da escolha do ECS.
- [ ] Bench baseline anexado; sem `unsafe` fora de interop comprovada.

## Riscos / rollback

- Risco: ECS inadequado à carga de guerras → mitigação: bench antes/depois;
  trocar exige nova ADR.
- Rollback: `git checkout -- crates/wyd-core/`.
