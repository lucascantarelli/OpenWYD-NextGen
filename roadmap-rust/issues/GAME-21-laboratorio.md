# GAME-21 — Laboratório `wyd-sim` (milhões de iterações)

> **ID canônico:** `GAME-21`
> **ID histórico:** `FASE-10-3`
> **Estado 2026-11-12:** ⬜ · ver `roadmap-rust/index.md`.

**Fase:** 10 · **Depende de:** GAME-19 ✅, GAME-20 · **Bloqueia:** FASE-15 · **Esforço:** M

## Objetivo

Operar o `wyd-sim` como laboratório capaz de simular milhões de combates em
milissegundos para validar balanceamento e caçar overflows/underflows.

## Contexto / evidência

`pre-release-rust.md` FASE 10 §10.3. Base: `damage_after_absorb` (saturating) em
`crates/wyd-sim/src/lib.rs`.

## Passos

1. Implementar harness de simulação em lote (classes × atributos × cenários).
2. Caçar overflow/underflow, divisão por zero e percentuais duplicados com
   propriedades (proptest/quickcheck via ADR).
3. Validar escalonamento entre classes e a correção da GAME-20 em lote.
4. Publicar relatório de balanceamento (somente com evidência); `make check`.

## Arquivos

- Criados: harness, cenários e testes de propriedade em `crates/wyd-sim/`.
- Alterados: `crates/wyd-sim/Cargo.toml` (harness numérico via ADR).
- Proibido tocar: `crates/wyd-core/src/`, balanceamento por opinião.

## Testes / verificação

```bash
cargo test -p wyd-sim
make check
```

Saída esperada: `test result: ok`; relatório com N simulações, tempo e
divergências encontradas.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-sim - laboratório e propriedades.
- .agents/skills/rust-code-quality - saturação em vez de wrap.

## Documentação de referencia

- pre-release-rust.md (FASE 10 §10.3).
- roadmap-rust/changelog.md (GAME-19).
- roadmap-rust/issues/GAME-20-velocidade.md.

## Critérios de aceite

- [ ] Milhões de iterações em tempo hábil (números anexados); bugs
  classificados por severidade/impacto.
- [ ] Nenhum balanceamento sem evidência de simulação.

## Riscos / rollback

- Risco: simulação lenta inviabilizando o lab → mitigação: release/batches e
  medição desde a primeira versão.
- Rollback: `git checkout -- crates/wyd-sim/`.
