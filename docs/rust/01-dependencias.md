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
| Async runtime | `tokio` (+ `tokio-uring` no Linux) | pendente (FASE-08) |
| Concorrência lock-free | `crossbeam-channel` | pendente (FASE-03) |
| Parse/serialização | `nom`, `zerocopy`/`rkyv` | pendente (FASE-08) |
| Persistência | `sqlx` (+ `sqlx-cli`), `refinery`, `redis` | pendente (FASE-04) |
| Scripting sandbox | `wasmtime` | pendente (FASE-03) |
| Observabilidade | `tracing`, `prometheus` | pendente (FASE-06) |
| Qualidade/supply-chain | `cargo-audit`, `cargo-deny`, `cargo-tarpaulin`, `miri` | pendente (FASE-05/15) |
| Launcher UI | Tauri (ou CLI puro) | pendente (FASE-08) |

## Notas datadas (FASE-14-1, 2026-11-07 — sem troca sem ADR + PoC)

- **Transporte Cold Path (ADR-0008, 2026-11-07):** HTTP/1.1 + JSON em loopback entram
  **sem dependência externa** (`std::net` + `json.rs` próprio). `tonic`/`prost`/`protoc`
  **não** entram na árvore; `tokio`/`serde_json`/`axum` ficam como otimização opcional da
  ADR-0006 (troca só do transporte, contrato e BFF intactos). Não reabrir gRPC sem medição
  (novo cliente ou streaming real).


- `hecs`/`bevy_ecs` (ECS): **manter como candidato**; ADR-0004 congela SoA manual até revisão com números. PoC futura fora de `crates/`: spawn + iteração de 10k entidades, SoA manual vs crate, com `cargo bench`.
- `tokio`/`tokio-uring` (async): **manter como candidato**; ADR-0006 adia async real até rede vendor. PoC futura: echo TCP com `tokio` vs std, latência P50/P99 local.
- `zerocopy`/`rkyv`/`nom` (serialização): **manter como candidato**; ADR-0005 congela parse manual `repr(C)` + offsets provados. PoC futura: parse de `HEADER` manual vs `zerocopy`, sem `unsafe` novo.
- `wasmtime` (scripts): **descartar por ora**; sem caso de uso aberto nas issues; reabrir só com proposta de sistema de eventos (FASE-03/10).
- `tracing`/`prometheus` (obs): **manter**; bloqueado até FASE-06-1 definir nomes de métricas; sem `println!` no Hot Path até lá.
- `sqlx`/`refinery`/`redis`: **manter**; bloqueado até FASE-04-3 com rede + ADR; até lá `psql -f` + `tests/constraints.rs`.
- `cargo-audit`/`cargo-deny`: **manter**; `deny.toml` + falha comprovada saem na FASE-05-1.
- Evidência deste turno: `cargo test --workspace` verde, `clippy -D warnings` verde, `git status --short -- crates/ Cargo.toml Cargo.lock` sem diff (pesquisa não tocou no workspace).

## Mapeamento legado → Rust (resumo)

CPSock/`select` → `wyd-net` (tokio/io_uring); `TMSrv`+mutexes → `wyd-core`
(thread-per-core + spatial grid); `.dat`/dbServer Go → `wyd-db` (sqlx);
`binserver`/`webserver` Go → `wyd-admin` (Cold Path); Lua → `wasmtime`;
`printf`s → `tracing`/Prometheus; `ClientPatch_v7662`/launcher → `wyd-launcher`.
