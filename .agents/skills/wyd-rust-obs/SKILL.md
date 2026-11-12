---
name: wyd-rust-obs
description: Opera observabilidade e segurança do WYD em Rust (tracing, Prometheus/Grafana, hardening de handshake, rate limiting). Use ao instrumentar tick/filas/entidades, criar dashboards e alertas ou endurecer o transporte contra DoS e pacotes malformados.
---

# Observabilidade e segurança Rust (wyd-rust-obs)

Norma: ADR-0018 (`docs/arquitetura/adr/0018-telemetria-tracing.md`),
`docs/rust/06-observabilidade-runbooks.md` e `docs/rust/05-stack-docker.md` §4.
Métricas, buckets, painéis e runbooks estão nesses documentos; não repita a lista.

## Decisões específicas

- Instrumente o motor e as filas com `tracing`/Prometheus; `println!` não é telemetria.
- Segredo (senha, PIN, DSN, token) não entra em log, métrica, trace ou label.
- Handshake e parser continuam hostis por padrão: rate limit, backpressure e validação
  estrita antes do parse; entrada de rede não causa `panic!`.
- Dashboard/alerta novo é versionado em `docker/monitoring/` e provado por
  `make monitoring-check`; não crie painel manual fora do repo.

## Prova

`make monitoring-check` para o stack de observabilidade; `make check` para o código.
