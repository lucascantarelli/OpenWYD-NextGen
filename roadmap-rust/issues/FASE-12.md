# Issue 12 — Dashboard e visualização do estado do projeto

**Fase:** 12 · **Depende de:** FASE-06 · **Bloqueia:** — · **Esforço:** M

## Objetivo

Dashboards Grafana versionados (saúde do Rust, migrations, telemetria de produção)
com alertas para regressão de P99 e falha de migration.

## Contexto / evidência

Métricas nascem na FASE-06 (`tracing`/Prometheus no motor); sem dashboard
versionado, cada ambiente improvisa. Referência legada: `docs/rust/05-stack-docker.md §4 (stack legada removida; ver FASE-12-1)`
(Prometheus + Grafana) e `logpipe/`. Destino: paineis como código no repo,
provisionados automaticamente (`pre-release-rust.md` FASE 12).

## Passos

1. Definir as queries (tick P50/P95/P99, pacotes/s, entidades, memória, filas,
   status de migrations) contra o que a FASE-06 expõe — sem inventar métrica.
2. Criar dashboards JSON provisionados + alertas (P99, falha de migration).
3. Documentar runbook mínimo por alerta (o que olhar, onde, como voltar atrás).
4. Subir stack de observabilidade local e validar com carga mínima.
5. `make check`.

## Arquivos

- Criados: dashboards/alertas versionados (ex.: `monitoring/rust-*.json`,
  regras de alerta), runbooks em `docs/`.
- Alterados: composição local de monitoring (somente o necessário para subir).
- Proibido tocar: `crates/*/src/` além da exposição de métricas da FASE-06,
  segredos.

## Testes / verificação

```bash
make check
```

Saída esperada: gates verdes; dashboards carregam sem erro de query (print ou
JSON de provisionamento anexado); alerta de teste dispara e resolve.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-obs - métricas, dashboards, alertas.
- .agents/skills/wyd-rust-toolchain - `make check`.

## Documentação de referencia

- pre-release-rust.md (FASE 12) - escopo dos painéis.
- docs/rust/05-stack-docker.md §4 (stack legada removida; ver FASE-12-1) - dashboards e exporters legados.
- roadmap-rust/issues/FASE-06.md - métricas expostas pelo motor.

## Subtasks

- [ ] Codificar issue
- [ ] Testar issue
- [ ] Revisar issue
- [ ] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [ ] Dashboards versionados sobem do zero e mostram dados reais.
- [ ] Alertas de P99 e de migration funcionam (teste de disparo anexado).

## Riscos / rollback

- Risco: dashboard contra métrica inexistente → mitigação: queries só contra o
  que a FASE-06 expõe (contrato fechado).
- Rollback: `git checkout -- monitoring/ docs/`.
