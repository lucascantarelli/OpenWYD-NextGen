# Issue 06-1 — Telemetria: tracing + Prometheus no motor

**Fase:** 6 · **Depende de:** FASE-03 · **Bloqueia:** FASE-12 · **Esforço:** M
**Estado:** ✅ entregue em 2026-11-08 (ADR-0018)

## Objetivo

Instrumentar o motor com `tracing` e expor tick P50/P95/P99, pacotes/s, entidades ativas e
memória, sem `println!` como telemetria.

## O que foi entregue

| Peça | Conteúdo |
|---|---|
| `crates/wyd-core/src/telemetry.rs` | `init_from_env()`, `init_with_level()`, `level_from_env()`/`level_from_value()`: `WYD_LOG` (fallback `RUST_LOG`) em `error|warn|info|debug|trace|off`, default `info`; saída `key=value` sem ANSI; `info,alvo=debug` aceita o primeiro termo |
| `crates/wyd-core/src/metrics/registry.rs` | Registro em atomics `Relaxed` (sem lock/alocação): `wyd_ticks_total`, histograma de tick (10 buckets, 0,5 ms–1 s), `wyd_entities_active`, `process_resident_memory_bytes`, `wyd_memory_samples_total`; P50/P95/P99 e média por bucket |
| `.../metrics/exporter.rs` | `MetricsServer` (`std::net::TcpListener`, thread própria): `GET /metrics` → 200 `text/plain; version=0.0.4`, resto → 404; `Metrics::render(extra)` concatena fragmento de outra crate |
| `.../metrics/mem.rs` | Amostragem de RSS via `/proc/self/statm` (Linux); `None` nas demais plataformas (sem FFI `unsafe`) |
| `crates/wyd-net/src/metrics.rs` | `NET`: pacotes aceitos, recusas por motivo (`rate/ip/total/timeout`), handshakes, pendentes e sessões; fragmento Prometheus próprio. **Feature `metrics` (default off)** |
| `crates/wyd-net/src/limits/rate.rs` | Instrumentação nos pontos de decisão (`try_consume`, `admit`, `complete`, `release`, `sweep`) |
| `Makefile` | `telemetry-smoke`: teste de telemetria com `--nocapture` + teste de rede com `--features metrics` |
| `docs/arquitetura/adr/0018-*` | Escolha de deps, exporter próprio, fragmentos e custo |

## Prova de execução real (log + `/metrics` colados)

```text
$ WYD_LOG=debug cargo test -p wyd-core --test telemetry -- --nocapture
nivel efetivo: debug
exporter em http://127.0.0.1:49383/metrics
2026-11-08T10:55:30.088912Z DEBUG tick concluido tick=0 micros=2365
2026-11-08T10:55:30.111608Z DEBUG tick concluido tick=10 micros=2571
2026-11-08T10:55:30.137036Z DEBUG tick concluido tick=20 micros=2609
2026-11-08T10:55:30.163149Z DEBUG tick concluido tick=30 micros=2640
2026-11-08T10:55:30.185912Z  INFO resumo de telemetria ticks=40 p50_us=5000 p95_us=5000 p99_us=5000 rss_bytes=0
--- GET /metrics ---
HTTP/1.1 200 OK
Content-Type: text/plain; version=0.0.4; charset=utf-8
Content-Length: 1257
Connection: close
# TYPE wyd_ticks_total counter
wyd_ticks_total 40
# TYPE wyd_tick_duration_microseconds histogram
wyd_tick_duration_microseconds_bucket{le="500"} 0
wyd_tick_duration_microseconds_bucket{le="2500"} 15
wyd_tick_duration_microseconds_bucket{le="5000"} 40
... (buckets até 1000000)
wyd_tick_duration_microseconds_bucket{le="+Inf"} 40
wyd_tick_duration_microseconds_sum 99023
wyd_tick_duration_microseconds_count 40
# TYPE wyd_entities_active gauge
wyd_entities_active 104
# TYPE process_resident_memory_bytes gauge
process_resident_memory_bytes 0
wyd_memory_samples_total 0
test telemetria_mede_tick_e_serve_metrics ... ok
```

Nível por ambiente (mesmo binário de teste, processos distintos):

```text
$ WYD_LOG=debug ...   → 4 linhas DEBUG + resumo INFO
$ WYD_LOG=info  ...   → 0 linhas DEBUG (grep -c = 0)
$ WYD_LOG=lixo  ...   → nivel efetivo: info (fallback)
```

Rede (fragmento real, tráfego de 3 consumos + 3 handshakes + recusas):

```text
$ cargo test -p wyd-net --features metrics --test metrics -- --nocapture
wyd_packets_total 3
wyd_packets_rejected_total{reason="rate"} 1
wyd_packets_rejected_total{reason="ip"} 1
wyd_packets_rejected_total{reason="total"} 0
wyd_packets_rejected_total{reason="timeout"} 0
wyd_handshakes_total 3
wyd_connections_pending 3
wyd_sessions_open 3
```

## Medições

### Dependências e tamanho

| Item | Antes | Depois |
|---|---|---|
| `Cargo.lock` (pacotes) | 7 | **24** (`tracing` 0.1.44, `tracing-core` 0.1.36, `tracing-subscriber` 0.3.23, `sharded-slab` 0.1.7, `thread_local` 1.1.10, `nu-ansi-term` 0.50.3, `once_cell`, `lazy_static`, `pin-project-lite`, …) |
| `target/release/wyd-launcher` | 296.448 bytes | **296.448 bytes** (não linka `wyd-core`) |
| imagem `wyd/launcher:check` (reconstruída) | 37.847.971 bytes | **37.847.971 bytes**, 20 camadas, `nonroot:nonroot`, base `gcr.io/distroless/cc-debian12:nonroot` |
| smoke no container (`check /jogo /jogo/manifest.toml`) | exit 0 | **exit 0** (`1 íntegros, 0 com problema`) |
| perfil de release | LTO thin, cgu=1, strip, abort | **inalterado** (nenhuma dep exigiu unwind) |

### Bench (5 execuções de tick, 3 de cada config de rede; host compartilhado)

| Bench | Baseline (05-1/05-2) | Esta passada |
|---|---|---|
| Tick IA (`cargo bench -p wyd-core`) | 85,6–189,7 µs (p99 139,9–397,5) | **81,27–82,22 µs (p99 88–116)** — sem regressão |
| `TokenBucket::try_consume` (default) | 2,7–2,8 ns | **2,7 ns** — idêntico |
| `TokenBucket` com `--features metrics` | — | **24,3–24,8 ns** (custo medido do contador por pacote) |
| `ConnectionGate::admit+complete` | 100,1–100,8 ns | **98,1–98,2 ns** |

### Gates (árvore final, exit code real)

```text
cargo fmt --all --check                                 0
cargo test --workspace                                  0 — 32 alvos, 0 falhas
cargo test -p wyd-net --features metrics                0
cargo clippy --workspace --all-targets -D warnings      0 — 0 erro/warning (default e --features metrics)
node check-migrations.mjs                               0 — 24 migrations, 59 tabelas, 140 INSERTs
node gen-catalogo-banco.mjs --check                     0 — catálogo em dia
node verify-schema.mjs                                  0 — 59 tabelas, 457 colunas, 59 PKs
cargo audit                                             0 — 24 crate dependencies, sem advisory
cargo deny check                                        0 — advisories/bans/licenses/sources ok
cargo build --release --workspace                       0 — launcher 296.448 bytes
```

## Lacunas declaradas

- **RSS:** `process_resident_memory_bytes` só é amostrado em Linux (`/proc/self/statm`); o ensaio
  rodou no host Windows e o exporter publicou `0` com `wyd_memory_samples_total 0`. A leitura em
  `/proc` é coberta por teste unitário condicional, mas **não houve execução em container Linux**;
  a prova de valor não-zero fica para o binário do servidor (FASE-06/12) em produção.
- **P50/P95/P99 por bucket:** resolução de 0,5 ms–1 s; o tick de ~2,5 ms do ensaio cai no bucket de
  5 ms (daí `p50=p95=p99=5000`). Suficiente para o orçamento de 50 ms; precisão fina exigiria
  buckets novos (não é requisito da issue).
- **Contadores de rede off por default:** o daemon precisa compilar com `--features metrics`; o
  build/bench default mede o caminho sem instrumentação (decisão e custo no ADR-0018).
- **Sem push/OTLP/coletor:** a FASE-12 faz scrape; nenhum Prometheus/Grafana subiu nesta passada.

## Skills Relacionadas do freebuff

- .agents/skills/wyd-rust-obs - métricas e instrumentação.
- .agents/skills/wyd-rust-core - pontos de instrumentação no tick.
- .agents/skills/wyd-rust-net - contadores de rede.

## Documentação de referencia

- pre-release-rust.md (FASE 6 §6.1).
- docs/rust/01-dependencias.md - tracing/prometheus (status atualizado).
- docs/rust/05-stack-docker.md §4 - cadeia de observabilidade.
- docs/arquitetura/adr/0018-telemetria-tracing.md

## Subtasks

- [x] Codificar issue
- [x] Testar issue
- [x] Revisar issue
- [x] Documentar issue
- [ ] Versionar issue, mergear branch e fechar issue (árvore só com alterações locais, por instrução)
- [ ] Limpar branch (se aplicável)

## Critérios de aceite

- [x] P50/P95/P99 do tick + pacotes/s observáveis; zero segredo em log.
- [x] Overhead medido (`cargo bench` antes/depois anexado) e eliminado do default por feature.

## Riscos / rollback

- Risco: overhead da telemetria no tick → mitigado: contadores atômicos fora do laço de decisão
  (core) e feature-flag no `wyd-net`; medido em 2,7 ns (sem) / 24,5 ns (com).
- Rollback: remover `tracing`/`tracing-subscriber` do `Cargo.toml` do core, apagar
  `crates/wyd-core/src/metrics/` + `telemetry.rs`, `crates/wyd-net/src/metrics.rs` e as chamadas em
  `limits/rate.rs`; ADR-0018 marcado como substituído.
