---
name: wyd-rust-obs
description: Opera observabilidade e segurança do WYD em Rust (tracing, Prometheus/Grafana, hardening de handshake, rate limiting). Use ao instrumentar tick/filas/entidades, criar dashboards e alertas ou endurecer o transporte contra DoS e pacotes malformados.
---

# Observabilidade e segurança Rust (wyd-rust-obs)

Norma: ADR-0018 (`docs/arquitetura/adr/0018-telemetria-tracing.md`),
`docs/rust/06-observabilidade-runbooks.md`, `docs/rust/05-stack-docker.md` §4 e
`docs/rust/10-devops-ci.md` §7 (cadeia de alerta e notificação).
Métricas, buckets, painéis e runbooks estão nesses documentos; não repita a lista.

## Decisões específicas

- Instrumente o motor e as filas com `tracing`/Prometheus; `println!` não é telemetria.
- Segredo (senha, PIN, DSN, token) não entra em log, métrica, trace ou label.
- Handshake e parser continuam hostis por padrão: rate limit, backpressure e validação
  estrita antes do parse; entrada de rede não causa `panic!`.
- Dashboard/alerta novo é versionado em `docker/monitoring/` e provado por
  `make monitoring-check`; não crie painel manual fora do repo.
- **Regra sem destino não é alerta.** A cadeia termina em pessoa avisada
  (`Prometheus → regras → Alertmanager → slack/email`). O destino real é escolhido por
  `ALERTMANAGER_DESTINO` e o render roda no boot — segredo vem do ambiente, nunca do arquivo.
  `alert-sink` não é destino de operação: é bootstrap (sem `SLACK_API_URL`) e caminho de prova
  (`prova=local`); se a rota padrão apontar para ele, o alerta **não** está entregue.
- Toda regra nova nasce com caso de disparo e de resolução em `alerts.test.yml` e com âncora de
  runbook; alerta sem runbook é dívida, não cobertura.

## Prova

`make monitoring-check` para o stack de observabilidade (inclui pronto, alvos, regras,
Alertmanager, datasource, dashboard e a rota padrão apontando para destino real);
`make monitoring-alertas` quando a entrega da notificação for tocada (prova rota real + entrega);
`make monitoring-rules` valida a config **renderizada** do Alertmanager sem subir a stack;
`make check` para o código.
