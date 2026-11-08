# Issue 12-1 — Dashboards Grafana + alertas (Rust, banco, produção)

**Fase:** 12 · **Depende de:** 06-1 · **Bloqueia:** FASE-15 · **Esforço:** M

## Objetivo

Versionar dashboards provisionados (saúde do Rust, migrations, telemetria de
produção) com alertas de P99 e falha de migration.

## Contexto / evidência

`pre-release-rust.md` FASE 12. Métricas fonte: 06-1. Referência:
`docs/rust/05-stack-docker.md §4 (stack legada removida; ver FASE-12-1)` (Prometheus + Grafana) e `logpipe/`.

## Passos

1. Definir queries somente contra métricas que 06-1 expõe (contrato fechado).
2. Criar dashboards JSON provisionados + regras de alerta (P99, migration).
3. Escrever runbook mínimo por alerta (o que olhar, onde, como voltar atrás).
4. Subir stack local, validar com carga mínima e disparo de teste.

## Arquivos

- Criados: dashboards/alertas versionados (ex.: `monitoring/rust-*.json`),
  runbooks em `docs/`.
- Alterados: composição local de monitoring (mínimo para subir).
- Proibido tocar: `crates/*/src/` além do exposto em 06-1, segredos.

## Testes / verificação

```bash
make check
```

Saída esperada: gates verdes; dashboards carregam sem erro de query; alerta de
teste dispara e resolve (evidência anexada).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-obs - dashboards e alertas.
- .agents/skills/wyd-rust-toolchain - `make check`.

## Documentação de referencia

- pre-release-rust.md (FASE 12).
- docs/rust/05-stack-docker.md §4 (stack legada removida; ver FASE-12-1) - referência legada.
- roadmap-rust/issues/FASE-06-1-telemetria.md - métricas disponíveis.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Dashboards sobem do zero com dados reais; alertas testados (disparo anexado).

## Riscos / rollback

- Risco: query contra métrica inexistente → mitigação: contrato 06-1 como
  única fonte.
- Rollback: `git checkout -- monitoring/ docs/`.
