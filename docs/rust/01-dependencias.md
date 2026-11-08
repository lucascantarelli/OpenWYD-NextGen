# Levantamento de dependências Rust (esqueleto offline-verificável)

> Decisão deste passe: **workspace sem dependências externas** para manter
> `cargo build/test --workspace` verificável sem rede e `Cargo.lock` consistente.
> As crates reais abaixo ficam **catalogadas aqui** e entram com ADR + vendor/
> lockfile em passe posterior (FASE-02/03, com rede declarada).

## Decisão

- Hoje: zero deps externas; apenas `wyd-core`/`wyd-net` → `wyd-protocol` via `path`.
- `Cargo.lock` commitado reflete exatamente esse grafo; nenhuma entrada remota.
- Adicionar dep exige: justificativa, `cargo update -p`, `cargo deny/audit`,
  CI verde e doc da motivação (regra da skill `wyd-rust-toolchain`).

## Catálogo (fonte: `pre-release-rust.md` §§0.2–0.3 + FASES)

| Necessidade | Candidatos | Status |
|---|---|---|
| ECS/DOD | `hecs` ou `bevy_ecs` | pendente (FASE-03) |
| Arena allocator | `bumpalo` | pendente (FASE-03) |
| Fixed-point | `fixed` (avaliar) | pendente (FASE-03/10) |
| Async runtime | `tokio` (+ `tokio-uring` no Linux) | **adotado no `wyd-db` na FASE-04-3** (ADR-0021); `tokio-uring`/`wyd-net` seguem pendentes |
| Concorrência lock-free | `crossbeam-channel` | pendente (FASE-03) |
| Parse/serialização | `nom`, `zerocopy`/`rkyv` | pendente (FASE-08) |
| Persistência | `sqlx` 0.8 (adotado), `refinery`, `redis` | `sqlx` **adotado na FASE-04-3** (ADR-0021: pool + adapters `pg`); `refinery`/`redis` pendentes |
| Scripting sandbox | `wasmtime` | pendente (FASE-03) |
| Observabilidade | `tracing` 0.1.44 + `tracing-subscriber` 0.3.23; exporter Prometheus próprio | **adotado na FASE-06-1** (ADR-0018) |
| Qualidade/supply-chain | `cargo-audit`, `cargo-deny`, `cargo-tarpaulin`, `miri` | pendente (FASE-05/15) |
| Launcher UI | Tauri (ou CLI puro) | pendente (FASE-08) |

## Notas datadas (FASE-14-1, 2026-11-07 — sem troca sem ADR + PoC)

- **Transporte Cold Path (ADR-0008, 2026-11-07):** HTTP/1.1 + JSON em loopback entram
  **sem dependência externa** (`std::net` + `json.rs` próprio). `tonic`/`prost`/`protoc`
  **não** entram na árvore; `tokio`/`serde_json`/`axum` ficam como otimização opcional da
  ADR-0006 (troca só do transporte, contrato e BFF intactos). Não reabrir gRPC sem medição
  (novo cliente ou streaming real).


- `hecs`/`bevy_ecs` (ECS): **manter como candidato**; ADR-0004 congela SoA manual até revisão com números. PoC futura fora de `crates/`: spawn + iteração de 10k entidades, SoA manual vs crate, com `cargo bench`.
- `tokio`/`tokio-uring` (async): **`tokio` adotado no `wyd-db`** (FASE-04-3, ADR-0021), junto de
  `sqlx` 0.8 com `default-features = false` + `runtime-tokio`/`postgres`/`tls-rustls-ring`.
  `tokio-uring` e o runtime do `wyd-net` continuam pendentes, com a mesma regra: runtime entra como
  implementação de porta, nunca dentro do tick.
- `zerocopy`/`rkyv`/`nom` (serialização): **manter como candidato**; ADR-0005 congela parse manual `repr(C)` + offsets provados. PoC futura: parse de `HEADER` manual vs `zerocopy`, sem `unsafe` novo.
- `wasmtime` (scripts): **descartar por ora**; sem caso de uso aberto nas issues; reabrir só com proposta de sistema de eventos (FASE-03/10).
- `tracing`/`prometheus` (obs): **decidido na FASE-06-1 (ADR-0018)** — `tracing` + `tracing-subscriber`
  entraram (`fmt` sem `env-filter`, para não arrastar `regex`); a crate `prometheus` foi **descartada**
  (protobuf/push gateway desnecessários) em favor de um exporter de texto Prometheus sobre atomics
  no próprio `wyd-core`. Nomes publicados: `wyd_ticks_total`, `wyd_tick_duration_microseconds`
  (histograma), `wyd_entities_active`, `process_resident_memory_bytes`, `wyd_memory_samples_total`,
  mais o fragmento de rede (`wyd_packets_total`, `wyd_packets_rejected_total{reason}`,
  `wyd_handshakes_total`, `wyd_connections_pending`, `wyd_sessions_open`). `Cargo.lock` passou de 7
  para 24 pacotes; `cargo deny`/`audit` verdes.
- `sqlx`/`refinery`/`redis`: **`sqlx` adotado** (FASE-04-3, ADR-0021 — `Cargo.lock` cresceu com o
  driver e o runtime, sem macro de query: SQL sempre com bind em `sqlx::query`); `refinery` continua
  fora (migrations seguem por `psql -f`/`migrate-cycle.mjs`) e `redis` não tem necessidade medida.
- `cargo-audit`/`cargo-deny`: **manter**; `deny.toml` + falha comprovada saem na FASE-05-1.
- Evidência deste turno: `cargo test --workspace` verde, `clippy -D warnings` verde, `git status --short -- crates/ Cargo.toml Cargo.lock` sem diff (pesquisa não tocou no workspace).

## Mapeamento legado → Rust (resumo)

CPSock/`select` → `wyd-net` (tokio/io_uring); `TMSrv`+mutexes → `wyd-core`
(thread-per-core + spatial grid); `.dat`/dbServer Go → `wyd-db` (sqlx);
`binserver`/`webserver` Go → `wyd-admin` (Cold Path); Lua → `wasmtime`;
`printf`s → `tracing`/Prometheus; `ClientPatch_v7662`/launcher → `wyd-launcher`.
