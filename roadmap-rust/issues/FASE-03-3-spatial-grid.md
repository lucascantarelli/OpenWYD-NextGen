# Issue 03-3 — Spatial grid e interest management O(1)

**Fase:** 3 · **Depende de:** 03-1 · **Bloqueia:** 03-5 · **Esforço:** M

## Objetivo

Particionar o mapa em grade espacial com visibilidade e gerenciamento de
interesse em tempo O(1) por consulta.

## Contexto / evidência

`pre-release-rust.md` FASE 3 §3.1: spatial grid para visibilidade e interest
management O(1); §0.2: thread-per-core com zonas isoladas e canais lock-free
(`crossbeam-channel`).

## Passos

1. Implementar `crates/wyd-core/src/spatial/grid.rs` (árvore §10): célula fixa, inserção/remoção/
   consulta de vizinhança.
2. Definir fronteiras de zona (thread-per-core futuro) e protocolo de travessia
   entre células/zonas.
3. Testes: entidades em bordas de célula, travessia, densidade alta (guerra).
4. `cargo test -p wyd-core` + `make check`.

## Arquivos

- Criados: `crates/wyd-core/src/spatial/grid.rs`, testes de vizinhança/travessia.
- Alterados: `crates/wyd-core/src/lib.rs`.
- Proibido tocar: `crates/wyd-protocol/src/`.

## Testes / verificação

```bash
cargo test -p wyd-core grid
make check
```

Saída esperada: `test result: ok` nos testes de grade (bordas, travessia,
densidade).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-core - grid e interest management.
- .agents/skills/rust-code-quality - testes de borda.

## Documentação de referencia

- pre-release-rust.md (FASE 3 §3.1, §0.2 thread-per-core).
- crates/wyd-sim/src/formula/ + roadmap-rust/issues/FASE-10-1-equacoes.md - regras de visibilidade legada.

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

**Status:** ✅ (2026-11-07, entregue dentro da FASE-03: `spatial/grid.rs` + teste de guerra em `tests/determinism.rs` — 8,6 ms inserts, 79 µs consulta).

## Critérios de aceite

- [x] Consulta de vizinhança O(1); travessia entre zonas definida e testada.
- [x] Cenário de alta densidade (guerra) sem degradação anormal (números anexados).

## Riscos / rollback

- Risco: tamanho de célula errado para o mapa real → mitigação: parâmetro
  configurável + teste com dimensões do mapa legado.
- Rollback: `git checkout -- crates/wyd-core/src/spatial/grid.rs`.
