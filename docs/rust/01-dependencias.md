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

## Mapeamento legado → Rust (resumo)

CPSock/`select` → `wyd-net` (tokio/io_uring); `TMSrv`+mutexes → `wyd-core`
(thread-per-core + spatial grid); `.dat`/dbServer Go → `wyd-db` (sqlx);
`binserver`/`webserver` Go → `wyd-admin` (Cold Path); Lua → `wasmtime`;
`printf`s → `tracing`/Prometheus; `ClientPatch_v7662`/launcher → `wyd-launcher`.
