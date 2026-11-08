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
| `wyd-admin` | Cold Path (auth, billing, gateway) | **Sim** — binário `wyd-gateway` (`:7600`, ADR-0022) |
| `wyd-launcher` | CLI/exe do cliente (GUI Tauri, ADR-0007) | **Nunca** — exe do lado do cliente, fora do compose |
| `wyd-portal/` | BFF Next.js sobre o backend HTTP/JSON (ADR-0008) | Sim — serviço `portal` |

Binários existentes: `wyd-launcher` (fora do compose) e `wyd-gateway` (`crates/wyd-admin/src/main.rs`,
ADR-0022). Sem binário, não há serviço: o compose não antecipa o que a issue dona não entregou —
`tmserver` continua bloqueado pela FASE-03/04/06.

## 2. O que sobe hoje (`docker compose up`)

| Serviço | Imagem | Papel |
|---|---|---|
| `db` | `postgres:17-alpine` (oficial) | Postgres 17; schema via `crates/wyd-db/migrations/` (FASE-04). Sem one-shot de roles no stack novo: sobe com superusuário de dev — **não usar em produção** |
| `server` (por canal: `wyd-canal-1`, `wyd-canal-2`) | `docker/Dockerfile.server` (multi-stage, distroless non-root) | Monólito de jogo: CPSock `:8281`, status `:8080` (publicado como `:80`) e `/metrics` `:9898`. **Um container por canal**, mesma imagem `wyd/server:${TAG}` e mesmo Postgres; canal vem de `server_channel` (migration `0029`) e de `WYD_SERVER_CHANNEL`. O canal 1 publica o `serv00.htm` com a lista de canais; o 2 usa IP próprio porque o cliente fixa a porta 8281 |
| `gateway` | `docker/Dockerfile.gateway` (multi-stage, distroless non-root) | Gateway HTTP/JSON do portal (`:7600`, ADR-0022): `GET /health` faz `SELECT 1` no Postgres, `GET /metrics` publica `wyd_gateway_requests_total{result}` + `wyd_gateway_uptime_seconds`, `POST /admin/v1/{servico}/{metodo}` serve o contrato já existente. `--healthcheck` é o próprio binário (imagem sem shell). Sobe depois do `db` saudável |
| `portal` | `docker/Dockerfile.portal` (multi-stage, non-root, standalone) | BFF Next.js. Não fala SQL (invariante 7): depende do `gateway` saudável. `/api/health` chama `item_catalog/list_items` e devolve 200 com a contagem real do catálogo; 502 só quando o gateway está fora |
| `prometheus`, `grafana`, `node-exporter` | imagens oficiais com tag fixa (mesmo `docker-compose.yml`) | **sobem no `up` normal** desde 2026-11-08 (monitoramento é parte da stack): Prometheus `:9090`, Grafana `:3001`, node-exporter `:9100`, tudo em loopback. A aba `/monitoramento` do portal lê o Prometheus pela rede `wyd`. FASE-12-1 ✅ — dashboards/alertas/runbooks em `monitoring/` e `docs/rust/06-observabilidade-runbooks.md` |

Comandos: `make stack-config` (valida base+prod sem subir), `make stack-up`,
`make stack-down`, `make stack-logs`. Imagens `wyd/<serviço>:${TAG:-dev}`,
containers `wyd-<serviço>`, rede/volume `wyd_*`.

## 3. Alvo (quando cada binário existir, pela issue dona)

| Serviço futuro | Origem | Issue que libera |
|---|---|---|
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
3. **Coleta/visualização** (FASE-12-1 ✅): override
   no próprio `docker-compose.yml` (Prometheus v3.5.0 +
   Grafana 12.2.0 + node-exporter v1.9.1, imagens oficiais com tag fixa), configs em
   `monitoring/` (`prometheus.yml`, `alerts.yml`, `alerts.test.yml`, alvos por
   arquivo, provisioning e dashboard `wyd-saude.json`). Alvos do motor ficam vazios
   por arquivo até existir binário de servidor — nenhum alerta por ausência estrutural.
   Runbooks: `docs/rust/06-observabilidade-runbooks.md`. `make monitoring-up` (atalho —
   o `up` normal já sobe tudo),
   `make monitoring-rules` (promtool: config + regras + teste de disparo).
4. **Saúde do BFF**: `/api/health` do portal (200 com contagem do catálogo,
   502 sem backend) + volume de auditoria `portal-audit` (`/var/log/wyd-portal`).
5. **Gates**: `cargo fmt --check` + `clippy -D warnings` + `cargo test
   --workspace` (skill `wyd-rust-toolchain`) e `docker compose config`
   (base e prod). Sem logpipe/sidecar no desenho novo até a FASE-12 definir.

## 5. Banco da stack (conteúdo real, não schema vazio)

O `db` do compose sobe **sem** schema. Aplicar a série é passo explícito e idempotente-por-reset:

```bash
make stack-up        # db + gateway + portal (db fica healthy)
make db-migrate      # node tools/dev/db-migrate.mjs
# ou, para recriar do zero: node tools/dev/db-migrate.mjs --reset
```

`tools/dev/db-migrate.mjs` aplica os 28 arquivos em ordem no container `wyd-db` (usuário/banco de
`POSTGRES_USER`/`POSTGRES_DB`). Ele **não** é o runner de ciclo (`tools/content/migrate-cycle.mjs`, container
efêmero): aqui o alvo é o banco que o gateway consome. Sem ele, o gateway não acha `content_version`
no boot e falha com erro tipado — de propósito, porque health sem catálogo mentiria.

**Seed de staff (2026-11-08):** roda no **boot do gateway** quando `WYD_SEED_STAFF=1`
(`PORTAL_ADMIN_USER`/`PORTAL_ADMIN_PASSWORD` do `.env`); `--seed-staff` continua no binário para
uso manual/CI. Não existe mais serviço one-shot `staff-seed` no compose: um job que termina deixa
um container parado em `docker compose ps` (estado normal de job, mas vira ruído e some do `up`
se alguém filtrar por "Up"). Seed no boot do processo que já tem `DATABASE_URL` e argon2id é
idempotente, não precisa de imagem extra e não roda no `initdb` do Postgres — o `entrypoint`
do banco só executa em volume vazio (`/docker-entrypoint-initdb.d`), então nunca serviria um
volume já existente, que é o caso desta stack.

Contagens observadas após aplicar (2026-11-08): **67 tabelas** (a `0029` trouxe `server_channel`),
`item_catalog` 3.220, `server_channel` 3 linhas (canal 0 = status, 1 = normal, 2 = verde),
`content_version` 1.

## 6. Canais (dois mundos, um banco)

| Canal | Container | Host do cliente | Papel |
|---|---|---|---|
| 1 | `wyd-canal-1` | `192.168.100.3:8281` | normal (cidadania, guildas, torre) |
| 2 | `wyd-canal-2` | `192.168.100.4:8281` | verde/novato — mesma cidade e mesmo banco |

Regras: canal novo = linha em `server_channel` + container + célula no `serverlist.bin` (launcher),
**sem build**. O status `:80` do canal 1 é a única porta pública de status e serve **todos** os canais;
verde × normal é política de operação (qual IP o jogador novo recebe), não uma coluna do banco.

## 7. Proibido

- Copiar da stack arquivada: nomes `openwyd-*`/`W2PP_*`/`OPENWYD_*`, topologia,
  portas internas e variáveis do Go não entram no stack novo.
- Portal falando SQL, segredo em log/métrica/imagem, `latest` sem TAG,
  porta de banco publicada na prod.
