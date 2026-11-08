# Issue 06-1 — Telemetria: tracing + Prometheus no motor

**Fase:** 6 · **Depende de:** FASE-03 · **Bloqueia:** FASE-12 · **Esforço:** M

## Objetivo

Instrumentar o motor com `tracing` e expor tick P50/P95/P99, pacotes/s,
entidades ativas e memória, sem `println!` como telemetria.

## Contexto / evidência

`pre-release-rust.md` FASE 6 §6.1. Legado: `printf`s e `monitoring/` Go
(referência em `docs/rust/05-stack-docker.md §4 (stack legada removida; ver FASE-12-1)`).

## Passos

1. Adicionar `tracing` (+ subscriber) e `prometheus`/`metrics` via ADR (ver ADR-0006 — diferidos até rede/vendor).
2. Instrumentar `wyd-core`: histograma de latência de tick, entidades ativas,
   memória amostrada (fora do quente).
3. Instrumentar `wyd-net`: pacotes/s, rejeições, tamanho de filas.
4. Carga mínima demonstrando métricas variando (log anexado, sem segredos).
5. `make check`.

## Arquivos

- Criados: módulos `metrics`/`telemetry` + testes em `wyd-core` e `wyd-net`.
- Alterados: `Cargo.toml` das crates (deps via ADR), `docs/rust/01-dependencias.md`.
- Proibido tocar: segredos em log/métrica.

## Testes / verificação

```bash
cargo test -p wyd-core -p wyd-net
make check
```

Saída esperada: `test result: ok`; métricas exibem tick e pacotes/s sob carga.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-obs - métricas e instrumentação.
- .agents/skills/wyd-rust-core - pontos de instrumentação no tick.
- .agents/skills/wyd-rust-net - contadores de rede.

## Documentação de referencia

- pre-release-rust.md (FASE 6 §6.1).
- docs/rust/01-dependencias.md - tracing/prometheus.
- docs/rust/05-stack-docker.md §4 (stack legada removida; ver FASE-12-1) - referência legada.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] P50/P95/P99 do tick + pacotes/s observáveis; zero segredo em log.
- [ ] Overhead medido (`cargo bench` antes/depois anexado).

## Riscos / rollback

- Risco: overhead da telemetria no tick → mitigação: amostragem + contadores
  atômicos; feature-flag para desligar exportação.
- Rollback: `git checkout -- crates/`.
