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
| `loki` | `grafana/loki:3.5.5` | **Logs duráveis** (2026-11-09): grava em volume `wyd-loki-data` com retenção de 14 d (`monitoring/loki/loki.yml`). Sem porta no host — só o portal (server-side) e o Grafana consultam `loki:3100`. Resolve o problema do `json-file`: o stdout do Docker morre quando o container é recriado, o Loki não |
| `alloy` | `grafana/alloy:v1.9.2` | **Coletor oficial da Grafana**: descobre os containers pelo socket do Docker (**read-only**, só este serviço) e empurra para o Loki (`monitoring/alloy/config.alloy`), rotulando `service`/`container`. Substitui o sidecar caseiro de logs |

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
3. **Coleta/visualização** (FASE-12-1 ✅): no próprio `docker-compose.yml`
   (Prometheus v3.5.0 + Grafana 12.2.0 + node-exporter v1.9.1 + Loki 3.5.5 + Alloy v1.9.2,
   imagens oficiais com tag fixa), configs em `monitoring/` (`prometheus.yml`,
   `alerts.yml`, `alerts.test.yml`, alvos por arquivo, `loki/loki.yml`,
   `alloy/config.alloy`, provisioning e dashboards `wyd-saude.json` + `wyd-logs.json`).
   Alvos do motor ficam vazios por arquivo até existir binário de servidor — nenhum
   alerta por ausência estrutural. Runbooks: `docs/rust/06-observabilidade-runbooks.md`.
   `make monitoring-up` (atalho — o `up` normal já sobe tudo),
   `make monitoring-rules` (promtool: config + regras + teste de disparo).
4. **Logs duráveis** (2026-11-09): `containers → Alloy (socket read-only) → Loki
   (volume `wyd-loki-data`, 14 d) → BFF do portal`. O sidecar caseiro `logpipe`, o
   modo `docker compose logs -f` e a própria tela `/logs` (com o SSE `/api/logs/<container>`
   e `/api/logs/status`) foram **removidos**: o stdout do Docker é por container e
   sumia a cada recriação, e o portal não deve ter o socket do Docker. O portal consulta
   `LOKI_URL` server-side (`wyd-portal/src/server/loki.ts`) só para o agregado do
   `/monitoramento`; a leitura linha a linha é o Grafana (datasource Loki provisionado,
   dashboard `wyd-logs`) — e `docker compose logs` para o stdout vivo.
   **Auditoria de login:** `query_range` sem `start` cobre **só 1 h**; use `since=12h`
   (ou `start`/`end` em ns) para procurar handshake antigo — o Loki retém 14 d. Um login
   de 3 h atrás passa despercebido por consulta sem janela explícita.
5. **Saúde do BFF**: `/api/health` do portal (200 com contagem do catálogo,
   502 sem backend) + volume de auditoria `portal-audit` (`/var/log/wyd-portal`).
6. **Gates**: `cargo fmt --check` + `clippy -D warnings` + `cargo test
   --workspace` (skill `wyd-rust-toolchain`), `npx tsc --noEmit` + `vitest` no portal e
   `docker compose config` (base e prod). O portal nunca monta o socket do Docker.

## 5. Banco da stack (conteúdo real, não schema vazio)

O `db` do compose sobe **sem** schema. Aplicar a série é passo explícito e idempotente-por-reset:

```bash
make stack-up        # db + gateway + portal (db fica healthy)
make db-migrate      # node tools/dev/db-migrate.mjs
# ou, para recriar do zero: node tools/dev/db-migrate.mjs --reset
```

`tools/dev/db-migrate.mjs` aplica os 30 arquivos em ordem no container `wyd-db` (usuário/banco de
`POSTGRES_USER`/`POSTGRES_DB`). Ele **não** é o runner de ciclo (`tools/content/migrate-cycle.mjs`, container
efêmero): aqui o alvo é o banco que o gateway consome. Sem ele, o gateway não acha `content_version`
no boot e falha com erro tipado — de propósito, porque health sem catálogo mentiria.

**Seed de staff (2026-11-08, revisto em 2026-11-09):** roda no **boot do gateway** sempre que
`PORTAL_ADMIN_PASSWORD` está definida (`PORTAL_ADMIN_USER`/`PORTAL_ADMIN_PASSWORD` do `.env`).
Se a conta já existe com papel `admin` e a senha confere com o hash, não regrava; se mudou,
gera argon2id, faz upsert e relê para confirmar. `WYD_SEED_STAFF=1` só torna a **ausência** de
senha um erro fatal. `--seed-staff` continua no binário para uso manual/CI.
No dev, o gateway monta o `.env` do host em `/run/wyd/env` (`WYD_ENV_FILE`, read-only) e o seed lê
esse arquivo **a cada boot**: editar `PORTAL_ADMIN_*` e rodar `docker compose restart gateway` já
reaplica a senha, sem esperar recriação do container. Em produção, o mesmo mount é opt-in pelo
overlay `docker-compose.prod-env.yml`:

```bash
TAG=1.0.0 docker compose -f docker-compose.yml -f docker-compose.prod.yml \
  -f docker-compose.prod-env.yml up -d
```

Sem o overlay (segredos injetados pelo gerenciador de segredos), o fallback é o ambiente do
processo; nesse caso mudanças no `.env` exigem `docker compose up -d`.

**Smoke do overlay (sem derrubar o dev):** com a stack de dev no ar, `make prod-env-smoke` valida o
compose de produção com o overlay e roda um container one-off (`docker compose run --rm --no-deps
gateway --seed-staff`). Ele usa o `.env` montado e o banco da stack, e só sai com `0` se o seed
confirmar o hash (`verified=true`). Não sobe portal/canais, não publica porta e não para o stack.

O job de CI `seed-boot`
sobe um Postgres efêmero, aplica só `0001_contas` e roda `cargo test -p wyd-admin --test seed`
em toda pull request. Não existe mais serviço one-shot `staff-seed` no compose: um job que termina deixa
um container parado em `docker compose ps` (estado normal de job, mas vira ruído e some do `up`
se alguém filtrar por "Up"). Seed no boot do processo que já tem `DATABASE_URL` e argon2id é
idempotente, não precisa de imagem extra e não roda no `initdb` do Postgres — o `entrypoint`
do banco só executa em volume vazio (`/docker-entrypoint-initdb.d`), então nunca serviria um
volume já existente, que é o caso desta stack.

Contagens observadas após aplicar (2026-11-09, banco recriado do zero com as 30 migrations):
**68 tabelas** (a `0029` trouxe `server_channel`, a `0030` trouxe `character_session`),
`item_catalog` 3.220, `server_channel` 3 linhas (canal 0 = status, 1 = normal, 2 = verde),
`content_version` 1.

**Contas de teste (2026-11-09):** além da conta default `admin` do `.env`, o banco recriado tem
`test1` (admin), `test2`/`test3` (moderator) e `test4`/`test5` (player), todas com a senha
`test123` (hash argon2id individual, gerado com os mesmos parâmetros do gateway). A senha de
teste não passa da política de força mínima do seed (`MIN_PASSWORD_LEN = 8`), então o cadastro foi
feito por SQL com hash válido — o portal continua aplicando a política para contas reais.

## 6. Canais (dois mundos, um banco)

| Canal | Container | Host do cliente | Papel |
|---|---|---|---|
| 1 | `wyd-canal-1` | `192.168.100.3:8281` | normal (cidadania, guildas, torre) |
| 2 | `wyd-canal-2` | `192.168.100.4:8281` | verde/novato — mesma cidade e mesmo banco |

O endereço de cada canal vem de `WYD_CANAL1_IP`/`WYD_CANAL2_IP` (`.env`) e cada canal é publicado
**também** em loopback (`127.0.0.1` no canal 1, `127.0.0.2` no canal 2), de propósito: um cliente
já aberto pode estar com o `serverlist.bin` de uma configuração anterior e não deve precisar de
restart para achar o canal. Em 2026-11-09 a máquina perdeu os IPs de LAN `192.168.100.3/.4`
(passou a ter `192.168.100.7`) e as publicações ficaram órfãs; a recuperação foi **local e sem
rota nova**: os dois IPs voltaram como alias `/32` no adaptador de loopback do Windows
(`netsh interface ipv4 add address "Loopback Pseudo-Interface 1" 192.168.100.3 255.255.255.255`,
idem `.4`) — máscara /32 porque `192.168.100.0/24` no loopback quebraria a LAN. O
`client/serverlist.bin` acompanha o registro canônico `tools/client/serverlist-servers.txt`
regenerado pelo launcher (`cargo run -p wyd-launcher -- serverlist … client/`), nunca edição do
`.bin`; se a máquina trocar de LAN de novo, é o registro + `WYD_CANAL*_IP` que mudam, não o
binário do cliente.

Regras: canal novo = linha em `server_channel` + container + célula no `serverlist.bin` (launcher),
**sem build**. O status `:80` do canal 1 é a única porta pública de status e serve **todos** os canais;
verde × normal é política de operação (qual IP o jogador novo recebe), não uma coluna do banco.

## 7. Proibido

- Copiar da stack arquivada: nomes `openwyd-*`/`W2PP_*`/`OPENWYD_*`, topologia,
  portas internas e variáveis do Go não entram no stack novo.
- Portal falando SQL, segredo em log/métrica/imagem, `latest` sem TAG,
  porta de banco publicada na prod.
