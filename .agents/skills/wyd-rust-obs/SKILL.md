---
name: wyd-rust-obs
description: Opera observabilidade e segurança do OpenWYD em Rust (tracing, Prometheus/Grafana, hardening de handshake, rate limiting). Use ao instrumentar tick/filas/entidades, criar dashboards e alertas ou endurecer o transporte contra DoS e pacotes malformados.
---

# Observabilidade e segurança Rust (wyd-rust-obs)

Referências: `pre-release-rust.md` FASE 6, `roadmap-rust/issues/FASE-06.md`,
`roadmap-rust/issues/FASE-12.md`.

## Regras

1. **Métricas do motor primeiro.** Latência de tick (P50/P95/P99), pacotes/s,
   entidades ativas, memória, tamanho de filas — via `tracing` + Prometheus,
   nunca `println!` como telemetria.
2. **Segredo nunca em log.** Sem senhas, PINs, DSNs ou tokens em logs, métricas
   ou traces (ver `.gitleaks.toml`).
3. **Segurança no handshake.** Rate limiting, backpressure e sanitização estrita
   do parser; input de rede nunca causa `panic!`.
4. **Dashboards versionados.** Painéis e alertas (regressão de P99, falha de
   migration) vivem no repo e sobem pelo pipeline.

## Antes de mudar

Confira `docs/analysis/banco-de-dados.md` para métricas de banco e a FASE-05
para onde os gates de CI validam a instrumentação.
