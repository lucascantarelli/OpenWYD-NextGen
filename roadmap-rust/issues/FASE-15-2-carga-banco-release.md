# Issue 15-2 — Carga de rede + integridade do banco + build de produção

**Fase:** 15 · **Depende de:** 15-1, 12-1 · **Bloqueia:** release · **Esforço:** M

## Objetivo

Validar carga de rede (P99 vs baseline 06-1), integridade do banco do zero e
build de produção com smoke, liberando somente com tudo verde.

## Contexto / evidência

`pre-release-rust.md` FASE 15 + §16 (ordem item 8, critério global). Baseline:
métricas 06-1 e dashboards 12-1. Gates: 05-2 (release/imagem).

## Passos

1. Carga contra `wyd-net`: pacotes/s, P99 e rejeições vs baseline (regressão
   de P99 veta o release).
2. Banco: migrations do zero + integridade + seeds (reexecuta 04-2/04-3).
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
- roadmap-rust/issues/FASE-06-1-telemetria.md - baseline.
- roadmap-rust/issues/FASE-12-1-dashboards.md - painéis.
- roadmap-rust/issues/FASE-15-1-suite-concorrencia.md - suite base.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] P99 sem regressão vs baseline; banco íntegro do zero; imagem + smoke OK.
- [ ] Relatório de release publicado com evidências.

## Riscos / rollback

- Risco: liberar com regressão silenciosa → mitigação: baseline como
  comparador obrigatório.
- Rollback: tag anterior + imagem anterior (procedimento no relatório).
