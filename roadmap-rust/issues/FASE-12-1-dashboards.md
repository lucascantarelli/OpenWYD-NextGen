# Issue 12-1 — Dashboards Grafana + alertas (Rust, banco, produção)

**Fase:** 12 · **Depende de:** 06-1 · **Bloqueia:** FASE-15 · **Esforço:** M
**Estado:** ✅ entregue em 2026-11-08 (stack sobe, regras testadas com `promtool test rules`; alvo do motor vazio porque não há binário de servidor)

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

## O que foi entregue

| Peça | Conteúdo |
|---|---|
| `docker-compose.monitoring.yml` | perfil `monitoring`: Prometheus v3.5.0, Grafana 12.2.0, node-exporter v1.9.1 (tags fixas, portas só em loopback), volumes `wyd-prometheus-data`/`wyd-grafana-data` |
| `monitoring/prometheus/prometheus.yml` | scrape de `prometheus` e `node`; job `wyd-server` por `file_sd` em `targets/wyd-server.json` (hoje `[]`: sem binário de servidor não há `/metrics` — nenhum alerta por ausência estrutural) |
| `monitoring/prometheus/alerts.yml` | 6 regras: `WydServidorForaDoAr`, `WydTickP99Alto`, `WydPacotesRecusados`, `WydHandshakeFalhando`, `WydRssAlto`, `WydSemAmostraDeMemoria` — só métricas do contrato 06-1 |
| `monitoring/prometheus/alerts.test.yml` | teste determinístico de disparo (e resolução) via `promtool test rules` |
| `monitoring/grafana/` | provisioning de datasource (`uid=prometheus`) + provider de dashboards + `dashboards/wyd-saude.json` (7 painéis: tick P50/P95/P99, ticks/s, entidades, pacotes, recusas por motivo, sessões, RSS) |
| `docs/rust/06-observabilidade-runbooks.md` | operação (`make monitoring-*`) e runbook por alerta |
| `tools/dev/monitoring-smoke.mjs` + `make monitoring-check` | smoke com exit code: `/-/ready`, alvos `up`, 6 regras, datasource e dashboard |

Prova de execução real (host Windows, Docker 29.9.2):

```text
$ docker compose -f docker-compose.yml -f docker-compose.monitoring.yml --profile monitoring config --quiet   → 0
$ promtool check config prometheus.yml   → SUCCESS (6 rules found)
$ promtool check rules alerts.yml         → SUCCESS: 6 rules found
$ promtool test rules alerts.test.yml     → SUCCESS
$ node tools/dev/monitoring-smoke.mjs
OK    prometheus /-/ready 200
OK    alvo prometheus up (http://localhost:9090/metrics)
OK    alvo node up (http://node-exporter:9100/metrics)
OK    regras carregadas: WydServidorForaDoAr, WydTickP99Alto, WydPacotesRecusados, WydHandshakeFalhando, WydRssAlto, WydSemAmostraDeMemoria
OK    grafana /api/health ok
OK    datasource prometheus provisionada (uid=prometheus)
OK    dashboard provisionado: WYD — saúde do motor (Rust) (7 painéis)
smoke de monitoramento: tudo verde
```

Lacuna declarada: painéis ficam sem série até existir binário de servidor de jogo
(`docs/rust/05-stack-docker.md` §3); alerta de falha de migration não existe porque a 06-1
não expõe métrica de migration (criar métrica é mudança de contrato, não desta issue).

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-obs - dashboards e alertas.
- .agents/skills/wyd-rust-toolchain - `make check`.

## Documentação de referencia

- pre-release-rust.md (FASE 12).
- docs/rust/05-stack-docker.md §4 (stack legada removida; ver FASE-12-1) - referência legada.
- roadmap-rust/issues/FASE-06-1-telemetria.md - métricas disponíveis.

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] Stack sobe do zero (`make monitoring-up`) com Prometheus/Grafana provisionados.
- [x] Alertas testados: disparo e resolução em `promtool test rules` (saída acima).
- [ ] Dados **reais** do motor nos painéis — bloqueado até existir binário de servidor (FASE-03/04/06 → serviço `tmserver`).

## Riscos / rollback

- Risco: query contra métrica inexistente → mitigação: contrato 06-1 como
  única fonte.
- Rollback: `git checkout -- monitoring/ docs/`.
