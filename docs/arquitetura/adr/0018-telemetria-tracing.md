# ADR-0018 — Telemetria: `tracing` + exporter Prometheus próprio

**Status:** aceito · **Data:** 2026-11-08 · **Fase:** 06-1 · **Depende de:** ADR-0006 (deps diferidas),
ADR-0017 (perfil de release/imagem)

## Contexto

O motor não tinha observabilidade: nenhum log estruturado, nenhuma métrica, e a issue 06-1 pede
tick P50/P95/P99, pacotes/s, entidades ativas e memória **sem `println!` como telemetria**, com
overhead medido e possibilidade de desligar a exportação. O `Cargo.lock` era fechado em 7 pacotes
(só crates do workspace + `wyd-protocol`): qualquer dependência nova precisa de decisão explícita,
`cargo deny`/`audit` verdes e custo em tamanho comprovado (ADR-0017).

Restrições vigentes: Hot Path sem alocação/lock no tick; `panic = "abort"`, `strip = "symbols"`,
LTO thin; imagem distroless `cc-debian12:nonroot` (sem shell); `unsafe` só em interop comprovada;
DAG de crates `protocolo ← net ← core` (net **não** depende de core).

## Decisão

1. **`tracing` 0.1.44 + `tracing-subscriber` 0.3.23** em `wyd-core`, com `default-features = false`
   e features `fmt`, `ansi`, `std`. Sem `env-filter`: ele arrasta `regex`/`matchers` (peso e superfície
   de auditoria) e filtro por alvo não é requisito da issue. Nível vem de `WYD_LOG` (fallback
   `RUST_LOG`), valores `error|warn|info|debug|trace|off`, default `info`; saída `key=value` sem ANSI.
2. **Sem crate de métricas.** O registro é `crates/wyd-core/src/metrics/`: contadores e histograma em
   `AtomicU64` `Relaxed`, buckets fixos de 0,5 ms a 1 s, P50/P95/P99 pelo limite superior do bucket.
   O exporter é `std::net::TcpListener` em thread própria servindo `GET /metrics` em texto
   Prometheus 0.0.4 (`MetricsServer`), com 404 no resto e `Connection: close`.
3. **Fragmento por crate.** `wyd-net` mantém seus contadores (pacotes, recusas por motivo, handshakes,
   pendentes, sessões) e publica texto Prometheus; o binário concatena com `Metrics::render(extra)`
   — evita inverter o DAG criando uma crate de métricas compartilhada.
4. **Contadores de rede atrás da feature `metrics`** (`wyd-net`), **desligada no build default**:
   o `fetch_add` por pacote custou +8,8 ns sobre um consumo de balde de 2,7 ns (medido), regressão
   fora do ruído no microbench. O daemon liga `metrics`; o build/bench default fica no caminho puro.
5. **Memória** só em Linux (`/proc/self/statm`); em outras plataformas o exporter publica
   `process_resident_memory_bytes 0` + `wyd_memory_samples_total 0` (ausência declarada, sem FFI
   `unsafe` e sem número inventado).

## Nomes publicados (contrato da 06-1)

| Métrica | Tipo | Origem |
|---|---|---|
| `wyd_ticks_total` | counter | core: cada tick concluído |
| `wyd_tick_duration_microseconds` | histogram | core: duração do tick inteiro (µs) |
| `wyd_entities_active` | gauge | core: entidades vivas no último tick |
| `process_resident_memory_bytes` | gauge | core: RSS amostrado (Linux) |
| `wyd_memory_samples_total` | counter | core: amostras de memória |
| `wyd_packets_total` | counter | net (feature `metrics`): aceitos pelo balde |
| `wyd_packets_rejected_total{reason}` | counter | net: `rate`, `ip`, `total`, `timeout` |
| `wyd_handshakes_total` | counter | net: handshakes admitidos |
| `wyd_connections_pending` | gauge | net: pendentes no porte |
| `wyd_sessions_open` | gauge | net: conexões abertas |

Config: `WYD_LOG` (nível) e, no binário, `WYD_METRICS_ADDR` (default `127.0.0.1:9898`, decidido no
`main`, não no core). Nenhum segredo/endereço de cliente entra em log ou métrica.

## Custo medido

| Item | Antes | Depois |
|---|---|---|
| `Cargo.lock` (pacotes) | 7 | 24 (`tracing`, `tracing-core`, `tracing-subscriber`, `sharded-slab`, `thread_local`, `nu-ansi-term`, `smallvec`, `once_cell`, `lazy_static`, `pin-project-lite`…) |
| `target/release/wyd-launcher` | 296.448 bytes | **296.448 bytes** (launcher não linka `wyd-core`; telemetria entra no futuro binário do servidor) |
| imagem `wyd/launcher:check` (distroless) | 37.847.971 bytes | **37.847.971 bytes** (mesma base, mesmo conteúdo) |
| perfil | LTO thin, cgu=1, strip, abort | **inalterado** — nenhuma dep exige unwind ou mudança de perfil |
| `TokenBucket::try_consume` | 2,7–2,8 ns/op | **2,7 ns/op sem a feature** (idêntico) / **24,3–24,8 ns/op com `metrics`** |
| `ConnectionGate::admit+complete` | 100,1–100,8 ns | **98,1–98,2 ns** (porta com contadores no-op; 1×/handshake) |
| tick IA (`cargo bench -p wyd-core`) | 85,6–189,7 µs/tick | **81,27–82,22 µs/tick** (p99 88–116; a instrumentação não está no laço do bench) |

## Alternativas descartadas

| Alternativa | Por que não |
|---|---|
| crate `prometheus` 0.14 | Registry global com lock, `protobuf` e push gateway no grafo, para um formato que cabe em ~60 linhas de `format!`; custo de binário/dep sem contrapartida |
| crate `metrics` 0.24 + `metrics-exporter-prometheus` | Fachada + exporter separado + `once_cell`/`quanta` extra; amarra o contrato a uma API externa em vez do texto Prometheus |
| `opentelemetry`/OTLP | Exige coletor, protobuf/gRPC e runtime async (FASE-08); a FASE-12 coleta por scrape |
| `log` + `env_logger` | Sem spans/campos estruturados nem integração com histogramas; `tracing` é o padrão do ecossistema e o catálogo já o previa |
| crate de métricas compartilhada (`wyd-metrics`) | Inverteria o DAG (net → nova crate → core) e criaria nó novo na árvore do ADR-0002; fragmentos resolvem com menos peças |
| Contadores de rede sempre ligados | +8,8 ns/pacote medidos no bench; a issue prevê feature-flag para desligar a exportação |

## Consequências

- Logs e `/metrics` existem no motor e em `wyd-net`; o binário do servidor só precisa ligar
  `telemetry::init_from_env()`, criar `Metrics`, subir `MetricsServer` e concatenar o fragmento de rede.
- Resolução de P50/P95/P99 é a do bucket (0,5 ms/1/2,5/5/10/25/50/100/250/1000 ms); para o orçamento
  de 50 ms do tick é suficiente; precisão fina exigiria buckets adicionais.
- `Cargo.lock` cresceu de 7 para 24 pacotes — barreira de entrada para novas deps continua valendo
  (`cargo deny`/`audit` no gate).
- Build default do `wyd-net` não paga custo por pacote; monitoração de rede exige `--features metrics`
  no artefato do servidor (documentado em `docs/rust/02-execucao.md §7`).

## Evidência

`cargo test -p wyd-core --test telemetry -- --nocapture` (nível `debug`) imprime os eventos
`tracing` e a resposta HTTP real de `GET /metrics` com `wyd_ticks_total 40`,
`wyd_tick_duration_microseconds_count 40`, buckets cumulativos e `wyd_entities_active 104`;
`cargo test -p wyd-net --features metrics --test metrics` cobre pacotes/recusas/gauges.
Gates e números da passada em `roadmap-rust/issues/FASE-06-1-telemetria.md`.
