# Issue 03 — Engine `wyd-core`: tick, ECS e spatial grid

**Fase:** 3 · **Depende de:** FASE-01, FASE-08 · **Bloqueia:** FASE-06, FASE-10 · **Esforço:** L

## Objetivo

Implementar o game loop determinístico (20–50 Hz) com estado em ECS e particionamento
espacial, sem GC, sem mutex global e sem I/O bloqueante no tick.

## Contexto / evidência

O legado (`TMSrv`, `Map.cpp`, `Actor.cpp` em `.deprecated-go/Source/`) usa ponteiros
brutos e travas globais; a tentativa Go usou goroutine única com locks proibidos
(`AGENTS.md` legado). O blueprint Rust exige monolito Hot Path por nó de mundo,
thread-per-core e dados contíguos (`start-rust.md` §§2, 5; `pre-release-rust.md`
FASE 3). Esqueleto atual: `crates/wyd-core/src/lib.rs` (TICK_HZ=20) — verificado
com `cargo test -p wyd-core`.

## Passos

1. Modelar componentes contíguos (`Position`, `Stats`, `Inventory`) em
   `crates/wyd-core/src/ecs/components.rs` (estrutura de dados, sem lógica).
2. Implementar o tick em `crates/wyd-core/src/tick/scheduler.rs`: intervalo fixo
   (`tick_interval_ms`), ordem de sistemas documentada, sem alocação no quente.
3. Implementar a grade espacial em `crates/wyd-core/src/spatial/grid.rs` (visibilidade e
   interest management O(1)).
4. Validação de schema/dados estáticos no boot com erro claro (falha explícita,
   nunca default silencioso).
5. Testes: unitários por sistema + teste de determinismo (mesmo seed, mesmo
   resultado) + `make check`.

## Arquivos

- Criados (árvore normativa em `estrutura-e-padroes.md` §10): `src/error.rs`,
  `src/ecs/components.rs`, `src/tick/scheduler.rs`, `src/spatial/grid.rs`,
  `src/ports/clock.rs`, testes em `tests/determinism.rs`.
- Alterados: `crates/wyd-core/src/lib.rs`, `crates/wyd-core/Cargo.toml`
  (somente deps aprovadas via ADR — ver `docs/rust/01-dependencias.md`).
- Proibido tocar: `crates/wyd-protocol/src/` (contrato é da FASE-08),
  `.deprecated-go/`, `client/`, `wyd-portal/`.

## Testes / verificação

```bash
cargo test -p wyd-core
make check
```

Saída esperada: `test result: ok` em todos os targets de `wyd-core`; `make check`
(Fmt+Clippy+Build+Test) verde.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-core - engine, ECS, tick, spatial grid.
- .agents/skills/rust-code-quality - ownership, sem unsafe, testes.
- .agents/skills/software-architecture - Hot/Cold, fronteiras, ADR.
- .agents/skills/solid-design-patterns - sistemas coesos, traits pequenos.

## Documentação de referencia

- docs/rust/00-visao-estrutura.md - invariantes do núcleo.
- docs/rust/01-dependencias.md - candidatos ECS/arena (hecs/bevy_ecs, bumpalo).
- docs/arquitetura/adr/0001-hot-path-monolitico.md - decisão do monolito.
- pre-release-rust.md (FASE 3) - escopo da engine.
- .deprecated-go/docs/migration/game-rules.md - fórmulas de referência.
- .deprecated-go/docs/migration/data-formats.md - alinhamentos de save.

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Tick 20–50 Hz determinístico; teste de determinismo verde.
- [x] Zero `unsafe` fora de interop comprovada; `clippy -D warnings` limpo.
- [x] Nenhum I/O bloqueante, lock global ou alocação no caminho quente.
- [x] Evidência (saída de `cargo test -p wyd-core`) anexada.

**Status:** ✅ (2026-11-07)

## Resultados

- `error.rs` (`CoreError`: `BadTickHz`, `InvalidStats`, `OutOfBounds`,
  `InvalidGrid`, `TimeWentBackwards` — falha explícita, sem default silencioso).
- `ecs/components.rs`: `Position`/`Stats` (com `validate()`)/`Item`/`Inventory`
  (arrays fixos 16+64)/`EntityKind` (4 espécies)/`Entity` (índice+geração) +
  `ComponentStore` SoA com spawn validado e reuso de slot por geração.
- `tick/scheduler.rs`: `SchedulerConfig::validated` (20–50 Hz),
  acumulador com resto + clamp anti-espiral (`MAX_STEPS_PER_ADVANCE`),
  ordem canônica documentada (Movement → Combat → Visibility).
- `spatial/grid.rs`: célula fixa configurável, `insert`/`remove`/`relocate`,
  `for_each_in_radius` sem alloc, fora-dos-limites como `Err`.
- `ports/clock.rs`: trait `Clock` + `SystemClock` + `ManualClock` + `elapsed_ms`
  (volta no tempo é erro, não `elapsed` negativo).
- `tests/determinism.rs` (3 testes): mesmo seed → mesmo hash FNV; fatiamento
  7 ms × 286 = 50 ms × 40 (40 ticks); guerra com 5000 entidades — 8,6 ms de
  inserts, consulta raio-256 em 79 µs (máquina do dev, debug).
- Gate: `fmt` + `clippy -D warnings` + `cargo test --workspace` verdes
  (`wyd-core`: 12 lib + 3 integração); zero `unsafe`; `Cargo.toml` intocado
  (sem `hecs`/`bevy_ecs`/`bumpalo`: exigem rede/vendor + ADR — SoA manual;
  revisitar em FASE-13-2).
- Escopo consciente: sistemas de jogo (`systems/`), validador de dados
  estáticos (03-4, depende FASE-04) e pontos de autoridade (03-5) ficam nas
  sub-issues; árvore §10 restante inalterada.

## Riscos / rollback

- Risco: ECS inadequado à carga (guerras) → mitigação: benchmark antes de trocar
  (`cargo bench`), ADR para a escolha (hecs vs bevy_ecs).
- Rollback: reverter a crate (`git checkout -- crates/wyd-core/`); o esqueleto
  mínimo continua compilando.
