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
| ECS/DOD | `hecs` ou `bevy_ecs` | avaliado na PROC-23: **SoA mantido**; `hecs` 2,18× mais lento no lote e 8,58× na varredura |
| Arena allocator | `bumpalo` | pendente (FASE-03) |
| Fixed-point | `fixed` (avaliar) | pendente (FASE-03/10) |
| Async runtime | `tokio` (+ `tokio-uring` no Linux) | **adotado no `wyd-db` na PROC-16** (ADR-0021); `tokio-uring` **não adotado** (PoC WSL2 17–43× mais lento; revisar em Linux nativo com lote) |
| Concorrência lock-free | `crossbeam-channel` | pendente (FASE-03) |
| Parse/serialização | `nom`, `zerocopy`/`rkyv` | avaliado na PROC-23: **parse manual mantido**; `zerocopy` 1,79× mais rápido no HEADER, ganho de ~1,1 ns/pacote |
| Persistência | `sqlx` 0.8 (adotado), `refinery`, `redis` | `sqlx` **adotado na PROC-16** (ADR-0021: pool + adapters `pg`); `refinery`/`redis` pendentes |
| Checksum de migration | `sha2` 0.10 | **adotado no `wyd-db`** (ADR-0024): hash do bloco `up` no ledger `schema_migrations`; já estava no grafo via `rustls` |
| Scripting sandbox | `wasmtime` | **descartado por ora** (PROC-23: ~10,4 MB RSS e nenhum caso de uso aberto) |
| Observabilidade | `tracing` 0.1.44 + `tracing-subscriber` 0.3.23; exporter Prometheus próprio | **adotado na QA-04** (ADR-0018) |
| Qualidade/supply-chain | `cargo-audit`, `cargo-deny`, `cargo-tarpaulin`, `miri` | pendente (FASE-05/15) |
| Hash de senha | `argon2` 0.5 | **adotado no `wyd-admin`** em 2026-11-08 (AGENTS §3.6): argon2id padrão, só na borda de login (`auth/password.rs`); o hash nunca sai da resposta |
| Launcher UI | Tauri (ou CLI puro) | **`tauri` 2 adotado em `apps/launcher-ui`** (2026-11-12, CLIENT-08 + ADR-0007): janela WebView2, `tauri-build` para recursos; só `core:default` nas capabilities, sem plugins/updater nesta frente |
| Assinatura de pacote (launcher) | `ed25519-dalek` 2.x | **adotado no `wyd-launcher`** em 2026-11-11 (revisão do ADR-0003): Ed25519 verify-only, chave pública embutida; `default-features = false` + `std` |

## Notas datadas (PROC-23, 2026-11-07 — sem troca sem ADR + PoC)

### PoCs medidas em 2026-11-12

Ambiente: Windows 11 + WSL2 Linux 6.6, Rust 1.99.1, release. Harness descartável em
`target/proc23-poc`; detalhes e limitações em
[`evidencias/PROC-23-pocs-2026-11-12.md`](evidencias/PROC-23-pocs-2026-11-12.md).

- **ECS:** 10k entidades, 40 rodadas, mediana de 5 execuções. SoA **1.979,50 µs** no lote
  (spawn + varredura + despawn) contra **4.321,40 µs** do `hecs` 0.11.2 (2,18× mais lento);
  varredura mutável **1,90 µs** contra **16,30 µs** (8,58× mais lento). Manter ADR-0004.
- **Serialização:** HEADER de 12 B, 200 milhões de parses. Manual **2,53 ns** contra
  **1,42 ns** do `zerocopy` 0.8.63 (1,79×); a ~1,1 ns/pacote o ganho é desprezível no
  orçamento e não paga dependência nova. Manter ADR-0005.
- **io_uring:** 10k leituras de 4 KiB em cache, uma operação por submit, `tokio-uring` 0.5.0.
  Síncrono **2,49–11,69 µs/op** contra **84,24–214,75 µs/op** (17–43× mais lento); Docker
  padrão bloqueia `io_uring_setup` com `EPERM` e exigiu `seccomp=unconfined`. WSL2 não é bare
  metal; **não adotar** e revisar só com PoC Linux nativa e submissão em lote.
- **wasmtime:** 50.0.1, módulo trivial. Compilação 1,62–14,21 ms, instanciação
  0,011–0,131 ms, chamada 27,5–74,7 ns e RSS ~10,4 MB. Sem caso de uso; manter descartado.

- **Transporte Cold Path (ADR-0008, 2026-11-07):** HTTP/1.1 + JSON em loopback entram
  **sem dependência externa** (`std::net` + `json.rs` próprio). `tonic`/`prost`/`protoc`
  **não** entram na árvore; `tokio`/`serde_json`/`axum` ficam como otimização opcional da
  ADR-0006 (troca só do transporte, contrato e BFF intactos). Não reabrir gRPC sem medição
  (novo cliente ou streaming real).


- `hecs`/`bevy_ecs` (ECS): **manter como candidato, sem adoção**; ADR-0004 congela SoA manual. A PoC de 2026-11-12 mediu 10k entidades e o `hecs` ficou 2,18× mais lento no lote e 8,58× na varredura mutável.
- `tokio`/`tokio-uring` (async): **`tokio` adotado no `wyd-db`** (PROC-16, ADR-0021), junto de
  `sqlx` 0.8 com `default-features = false` + `runtime-tokio`/`postgres`/`tls-rustls-ring`.
  `tokio-uring` foi medido e **não adotado** (17–43× mais lento no WSL2/Docker); o runtime do
  `wyd-net` continua pendente, com a mesma regra: runtime entra como implementação de porta,
  nunca dentro do tick.
- `zerocopy`/`rkyv`/`nom` (serialização): **manter como candidato, sem adoção**; ADR-0005 congela parse manual `repr(C)` + offsets provados. A PoC de 2026-11-12 mediu `HEADER` manual (2,53 ns) contra `zerocopy` (1,42 ns), sem justificar a dependência.
- `wasmtime` (scripts): **descartar por ora**; a PoC de 2026-11-12 mediu ~10,4 MB de RSS e chamada de 27,5–74,7 ns, mas não há caso de uso aberto; reabrir só com proposta de sistema de eventos (FASE-03/10).
- `tracing`/`prometheus` (obs): **decidido na QA-04 (ADR-0018)** — `tracing` + `tracing-subscriber`
  entraram (`fmt` sem `env-filter`, para não arrastar `regex`); a crate `prometheus` foi **descartada**
  (protobuf/push gateway desnecessários) em favor de um exporter de texto Prometheus sobre atomics
  no próprio `wyd-core`. Nomes publicados: `wyd_ticks_total`, `wyd_tick_duration_microseconds`
  (histograma), `wyd_entities_active`, `process_resident_memory_bytes`, `wyd_memory_samples_total`,
  mais o fragmento de rede (`wyd_packets_total`, `wyd_packets_rejected_total{reason}`,
  `wyd_handshakes_total`, `wyd_connections_pending`, `wyd_sessions_open`). `Cargo.lock` passou de 7
  para 24 pacotes; `cargo deny`/`audit` verdes.
- `sqlx`/`refinery`/`redis`: **`sqlx` adotado** (PROC-16, ADR-0021 — `Cargo.lock` cresceu com o
  driver e o runtime, sem macro de query: SQL sempre com bind em `sqlx::query`); `refinery` continua
  fora (migrations aplicadas pelo runner `wyd_db::migrate`, ADR-0024) e `redis` não tem necessidade medida.
- `sha2` (ADR-0024): dependência direta no `wyd-db` só para o checksum do bloco `up`; já existia no
  grafo como transitiva de `rustls`, então não adiciona superfície nova de supply-chain.
- `cargo-audit`/`cargo-deny`: **manter**; `deny.toml` + falha comprovada saem na QA-02.
- Evidência da nota de 2026-11-07: `cargo test --workspace` verde, `clippy -D warnings` verde, `Cargo.toml`/`Cargo.lock` sem diff. As PoCs de 2026-11-12 também não os alteraram; `crates/` já tinha alterações do trabalho em andamento, registradas na evidência dedicada.

### Nota de 2026-11-11 — assinatura do pacote (revisão do ADR-0003)

O HMAC simétrico do `wyd-launcher` foi substituído por **Ed25519** (regra da skill
`wyd-rust-toolchain`: dependência entra com justificativa, `cargo deny`/`audit` e doc).
`ed25519-dalek 2.x` (`BSD-3-Clause`, já na allowlist do `deny.toml`) entra só com `std`,
sem `fast` nem `zeroize`/`batch`: o binário **verifica**, nunca assina, então não carrega
segredo. A chave pública fica em `distribuicao::CHAVE_PUBLICA_HEX`; a privada só no CI
(`tools/launcher/sign-package.mjs`). O `Cargo.lock` ganha `curve25519-dalek`, `ed25519`,
`signature` e `cpufeatures` (já presente via `sha2`). Ganho: espelhar o binário não permite
re-assinar pacote, e `verify_strict` recusa assinatura malleável. Custo: rotação de chave
exige launcher novo.

## Mapeamento legado → Rust (resumo)

CPSock/`select` → `wyd-net` (tokio; io_uring não adotado na PROC-23); `TMSrv`+mutexes → `wyd-core`
(thread-per-core + spatial grid); `.dat`/dbServer Go → `wyd-db` (sqlx);
`binserver`/`webserver` Go → `wyd-admin` (Cold Path); Lua → `wasmtime`;
`printf`s → `tracing`/Prometheus; `ClientPatch_v7662`/launcher → `wyd-launcher`.
