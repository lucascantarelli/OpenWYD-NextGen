# Issue 06 — Observabilidade, monitoramento e segurança nativa

**Fase:** 6 · **Depende de:** FASE-03 · **Bloqueia:** FASE-12 · **Esforço:** M

## Objetivo

Instrumentar o motor com `tracing` + Prometheus (tick P50/P95/P99, pacotes/s,
entidades, memória, filas) e endurecer o handshake (DoS, rate limiting,
sanitização estrita do parser).

## Contexto / evidência

O legado usa `printf`s; a fase Go tinha `logpipe`/`node-exporter` em
`.deprecated-go/monitoring/` e `logpipe/`. Destino Rust: sem `println!` como
telemetria, métricas do motor desde o dia um (`pre-release-rust.md` FASE 6,
skill `wyd-rust-obs`). Sem instrumentação, regressão de P99 só aparece no
jogador reclamando.

## Passos

1. Adicionar `tracing` (+ `tracing-subscriber`) e `prometheus`/`metrics` via ADR
   e `docs/rust/01-dependencias.md`.
2. Expor no `wyd-core`: latência de tick (histograma P50/P95/P99), contagem de
   entidades ativas e uso de memória (amostrado, fora do quente).
3. Expor no `wyd-net`: pacotes/s, rejeições por validação, tamanho de filas,
   rate-limit ativado.
4. Hardening: limite de conexões por IP, timeout de handshake, buffer máximo
   (`MAX_PACKET_BYTES` já existe em `crates/wyd-net/src/lib.rs`).
5. `make check` + teste de carga mínimo que mostre métricas variando.

## Arquivos

- Criados: módulos `metrics`/`telemetry` em `wyd-core` e `wyd-net`, testes.
- Alterados: `crates/wyd-core/Cargo.toml`, `crates/wyd-net/Cargo.toml`
  (deps via ADR), `docs/rust/01-dependencias.md`.
- Proibido tocar: segredos em logs/métricas, `.deprecated-go/`, `client/`.

## Testes / verificação

```bash
cargo test -p wyd-core -p wyd-net
make check
```

Saída esperada: `test result: ok`; endpoint/relatório de métricas exibe tick e
pacotes/s (trecho do log anexado, sem segredos).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-obs - métricas, dashboards, hardening.
- .agents/skills/wyd-rust-core - pontos de instrumentação no tick.
- .agents/skills/wyd-rust-net - contadores de rede e rate-limit.
- .agents/skills/rust-code-quality - amostragem fora do quente, sem panic.

## Documentação de referencia

- docs/rust/01-dependencias.md - tracing/prometheus.
- pre-release-rust.md (FASE 6) - telemetria e segurança.
- .deprecated-go/monitoring/ - dashboards e exporters legados (referência).

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Tick P50/P95/P99 e pacotes/s observáveis com carga mínima.
- [ ] Handshake resiste a flood básico (limite documentado e testado).
- [ ] Zero segredo em log/métrica (gitleaks/grep de DSN anexado).

## Riscos / rollback

- Risco: overhead da telemetria no tick → mitigação: amostragem e contadores
  atômicos; medir com `cargo bench` antes/depois.
- Rollback: feature-flag para desligar exportação; `git checkout -- crates/`.
