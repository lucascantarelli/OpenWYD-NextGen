# Stack Docker do WYD (Rust) — mapa de serviços e integração

Onde cada crate vira container, o que sobe hoje, o que falta e por qual issue.
Regra-mãe: **arquivo Go/C++ removido em 2026-11 — nunca fonte** (nomes, portas, variáveis e topologia legadas não entram aqui; fonte canônica é `docs/` + `crates/`)
(nomes, portas, variáveis e topologia de lá não entram aqui).

## 1. Crates → deployáveis

| Crate | Papel | Vira serviço? |
|---|---|---|
| `wyd-core` | motor Hot Path (tick, ECS) | Não — lib do monólito tmserver (FASE-03) |
| `wyd-net` | transporte CPSock | Não — lib do monólito tmserver (FASE-06) |
| `wyd-protocol` | wire/parsers | Não — lib (FASE-08) |
| `wyd-sim` | laboratório matemático | Não — lib, sem runtime (FASE-10) |
| `wyd-db` | persistência (migrations, delivery) | Não — lib usada pelos binários (FASE-04) |
| `wyd-admin` | Cold Path (auth, billing, gateway) | Futuro binário gateway (FASE-07-1) |
| `wyd-launcher` | CLI/exe do cliente (GUI Tauri, ADR-0007) | **Nunca** — exe do lado do cliente, fora do compose |
| `wyd-portal/` | BFF Next.js sobre o backend HTTP/JSON (ADR-0008) | Sim — serviço `portal` |

Só existe binário para `wyd-launcher` (único `[[bin]]` do workspace).
Sem binário, não há serviço: o compose não antecipa o que a issue dona
não entregou.

## 2. O que sobe hoje (`docker compose up`)

| Serviço | Imagem | Papel |
|---|---|---|
| `db` | `postgres:17-alpine` (oficial) | Postgres 17; schema via `crates/wyd-db/migrations/` (FASE-04). Sem one-shot de roles no stack novo: sobe com superusuário de dev — **não usar em produção** |
| `portal` | `docker/Dockerfile.portal` (multi-stage, non-root, standalone) | BFF Next.js. Não fala SQL (invariante 7): depende do gateway Rust (`gateway:7600`, FASE-07-1), hoje ausente — a UI sobe e `/api/health` responde 502. Sinal honesto, não bug |

Comandos: `make stack-config` (valida base+prod sem subir), `make stack-up`,
`make stack-down`, `make stack-logs`. Imagens `wyd/<serviço>:${TAG:-dev}`,
containers `wyd-<serviço>`, rede/volume `wyd_*`.

## 3. Alvo (quando cada binário existir, pela issue dona)

| Serviço futuro | Origem | Issue que libera |
|---|---|---|
| `gateway` (`:7600`, nome DNS já reservado no compose/`.env`) | binário a partir de `wyd-admin` | FASE-07-1 |
| `tmserver` (CPSock `:8281` + status `:80`; canais e ServerIndex no desenho) | monólito `wyd-core`+`wyd-net`+`wyd-protocol` | FASE-03/04/06 |
| `launcher` | — | Fora do compose (exe). Imagem `docker/Dockerfile.launcher` só como artefato de release/smoke (FASE-05-2): `docker build` + `docker run --rm` |

Cada novo serviço entra com: `docker/Dockerfile.<svc>`, registro no
`docker-compose.yml` (sem portas; porta só no override de dev; limites no
prod), linha nesta tabela e `make stack-config` verde nos dois modos.

## 4. Observabilidade (integração, sem código nesta rodada)

Cadeia normativa (skill `wyd-rust-obs`, FASE-06-1 → FASE-12-1):

1. **Instrumentação no motor** (FASE-06-1 ✅, ADR-0018): `tracing` + histograma de tick
   (`crates/wyd-core/src/metrics/`, `telemetry.rs`), contadores de rede em
   `crates/wyd-net/src/metrics.rs` (feature `metrics`, desligada no build default por custo
   medido). Nomes fechados: `wyd_ticks_total`, `wyd_tick_duration_microseconds`,
   `wyd_entities_active`, `process_resident_memory_bytes`, `wyd_memory_samples_total`,
   `wyd_packets_total`, `wyd_packets_rejected_total{reason}`, `wyd_handshakes_total`,
   `wyd_connections_pending`, `wyd_sessions_open`. Zero segredo em log/métrica.
2. **Exposição**: `MetricsServer` serve `GET /metrics` em texto Prometheus 0.0.4
   (endereço decidido pelo binário, default `127.0.0.1:9898`); o fragmento do `wyd-net`
   entra por `Metrics::render(extra)`. Dashboard só consulta o que 06-1 expõe.
3. **Coleta/visualização** (FASE-12/12-1): profile `monitoring` no compose
   (Prometheus + Grafana + node-exporter, imagens oficiais) com configs em
   `monitoring/` — diretório ainda não existe; será autorado na 12-1, sem
   copiar queries do legado.
4. **Saúde do BFF**: `/api/health` do portal (200 com contagem do catálogo,
   502 sem backend) + volume de auditoria `portal-audit` (`/var/log/wyd-portal`).
5. **Gates**: `cargo fmt --check` + `clippy -D warnings` + `cargo test
   --workspace` (skill `wyd-rust-toolchain`) e `docker compose config`
   (base e prod). Sem logpipe/sidecar no desenho novo até a FASE-12 definir.

## 5. Proibido

- Copiar da stack arquivada: nomes `openwyd-*`/`W2PP_*`/`OPENWYD_*`, topologia,
  portas internas e variáveis do Go não entram no stack novo.
- Portal falando SQL, segredo em log/métrica/imagem, `latest` sem TAG,
  porta de banco publicada na prod.
