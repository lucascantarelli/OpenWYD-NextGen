# QA-09 — Suite completa + concorrência (tarpaulin/miri)

> **ID canônico:** `QA-09`
> **ID histórico:** `FASE-15-1`
> **Estado 2026-11-12:** ⬜ · ver `roadmap-rust/index.md`.

**Fase:** 15 · **Depende de:** QA-02 ✅, GAME-21 · **Bloqueia:** QA-10 · **Esforço:** M

## Objetivo

Executar a validação fim a fim da suite Rust (all-targets) e da concorrência
com `tarpaulin`/`miri` nos módulos críticos, sem data races.

## Contexto / evidência

`pre-release-rust.md` FASE 15 + §16: suite completa, concorrência, cobertura das
fórmulas críticas. Módulos críticos: protocolo, tick, persistência.

## Passos

1. `cargo test --all-targets` (unitários, integração, doc-tests) — tudo verde.
2. Concorrência sob carga (threads) nos módulos críticos; `miri` onde aplicável.
3. `tarpaulin` nas fórmulas críticas (cobertura anexada).
4. `make check` + relatório parcial.

## Arquivos

- Criados: relatório parcial em `docs/` (números, cobertura).
- Alterados: ajustes mínimos exigidos (cada um com teste).
- Proibido tocar: `client/`; nenhum `#[ignore]` para
  esconder falha.

## Testes / verificação

```bash
make check
cargo test --all-targets 2>&1 | tail -n 5
```

Saída esperada: tudo `ok`/`Finished` sem warnings; cobertura das fórmulas anexada.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - gates e cobertura.
- .agents/skills/wyd-rust-sim - fórmulas críticas.
- .agents/skills/rust-code-quality - revisão sem concessões.

## Documentação de referencia

- pre-release-rust.md (FASE 15, §16).
- docs/rust/01-dependencias.md - tarpaulin/miri.
- roadmap-rust/changelog.md - gates.

## Critérios de aceite

- [ ] `test --all-targets` verde; miri/tarpaulin nos críticos com evidência.

## Riscos / rollback

- Risco: data race só sob carga específica → mitigação: carga variada
  (threads × entidades × pacotes) documentada.
- Rollback: `git checkout -- crates/ docs/`.
