# Issue 03-1 — Tick em frequência fixa determinística

**Fase:** 3 · **Depende de:** FASE-08 · **Bloqueia:** 03-2, 03-3 · **Esforço:** M

## Objetivo

Implementar o loop de tick 20–50 Hz com intervalo fixo e ordem de sistemas
documentada, sem alocação no caminho quente.

## Contexto / evidência

`pre-release-rust.md` FASE 3 §3.1: tick determinístico; §5 do `start-rust.md`:
loop determinístico + arena allocators. Base: `crates/wyd-core/src/lib.rs`
(`TICK_HZ=20`, `tick_interval_ms()`). Legado: `Map.cpp`/`Actor.cpp` com travas
globais (ver `.deprecated-go/Source/`).

## Passos

1. Implementar `crates/wyd-core/src/tick/scheduler.rs` (árvore §10): acumulador de tempo, intervalo
   fixo, ordem de sistemas documentada em comentário.
2. Proibir alocação no quente (arena/`bumpalo` via ADR ou estruturas estáticas).
3. Teste de determinismo: mesmo seed + mesma sequência de inputs = mesmo estado
   (hash do estado após N ticks).
4. `cargo test -p wyd-core` + `make check`.

## Arquivos

- Criados: `crates/wyd-core/src/tick/scheduler.rs`, `tests/determinism.rs`.
- Alterados: `crates/wyd-core/src/lib.rs`, `crates/wyd-core/Cargo.toml`
  (deps via ADR).
- Proibido tocar: `crates/wyd-protocol/src/`, `.deprecated-go/`, I/O ou lock
  global no tick.

## Testes / verificação

```bash
cargo test -p wyd-core
make check
```

Saída esperada: `test result: ok` incluindo teste de determinismo (hash igual em
duas execuções); `make check` verde.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-core - tick e determinismo.
- .agents/skills/rust-code-quality - sem alocação quente, sem unsafe.
- .agents/skills/software-architecture - ADR de alocação.

## Documentação de referencia

- pre-release-rust.md (FASE 3 §3.1, §0.2 Hot Path).
- start-rust.md (§5 blueprint).
- docs/arquitetura/adr/0001-hot-path-monolitico.md.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Tick 20–50 Hz com intervalo fixo; ordem de sistemas documentada.
- [ ] Teste de determinismo verde; zero alocação/I-O/lock no quente.

## Riscos / rollback

- Risco: deriva temporal acumulada → mitigação: acumulador com clamp + teste de
  longa duração.
- Rollback: `git checkout -- crates/wyd-core/`.
