# QA-10 — Carga de rede + integridade do banco + build de produção

> **ID canônico:** `QA-10`
> **ID histórico:** `FASE-15-2`
> **Estado 2026-11-12:** ⬜ · ver `roadmap-rust/index.md`.

**Fase:** 15 · **Depende de:** QA-09, QA-08 · **Bloqueia:** release · **Esforço:** M

## Objetivo

Validar carga de rede (P99 vs baseline QA-04), integridade do banco do zero e
build de produção com smoke, liberando somente com tudo verde.

## Contexto / evidência

`pre-release-rust.md` FASE 15 + §16 (ordem item 8, critério global). Baseline:
métricas QA-04 e dashboards QA-08. Gates: QA-03 (release/imagem).

## Passos

1. Carga contra `wyd-net`: pacotes/s, P99 e rejeições vs baseline (regressão
   de P99 veta o release).
2. Banco: migrations do zero + integridade + seeds (reexecuta PROC-15/PROC-16).
3. `cargo build --release --workspace` (LTO) + imagem + smoke do binário.
4. Publicar o relatório de release (números, versões, evidências).

## Arquivos

- Criados: relatório de release em `docs/`.
- Alterados: ajustes mínimos exigidos (cada um com teste).
- Proibido tocar: `client/`, segredos.

## Testes / verificação

```bash
make check
cargo build --release --workspace 2>&1 | tail -n 3
docker build -f docker/Dockerfile.launcher -t wyd/launcher:release . && docker build -f docker/Dockerfile.portal -t wyd/portal:release .
```

Saída esperada: tudo verde; relatório com P99, pacotes/s e cobertura anexado;
imagem builda e binário faz smoke.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-toolchain - release e relatório.
- .agents/skills/wyd-rust-obs - P99 e regressões.
- .agents/skills/wyd-rust-db - integridade do banco.
- .agents/skills/software-architecture - conformidade Hot/Cold.

## Documentação de referencia

- pre-release-rust.md (FASE 15, §16).
- roadmap-rust/changelog.md - baseline.
- `roadmap-rust/changelog.md` — QA-08 concluída (dashboards/painéis).
- roadmap-rust/issues/QA-09-suite-concorrencia.md - suite base.

## Critérios de aceite

- [ ] P99 sem regressão vs baseline; banco íntegro do zero; imagem + smoke OK.
- [ ] Relatório de release publicado com evidências.

## Riscos / rollback

- Risco: liberar com regressão silenciosa → mitigação: baseline como
  comparador obrigatório.
- Rollback: tag anterior + imagem anterior (procedimento no relatório).
