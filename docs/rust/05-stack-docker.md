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
| `wyd-server` | raiz de composição do monólito de um canal (CPSock/status/sessão/seleção) | **Sim** — binário `wyd-server`, um container por canal (`wyd-canal-1/2`, ADR-0022 §7) |
| `wyd-launcher` | CLI/exe do cliente (GUI Tauri, ADR-0007) | **Nunca** — exe do lado do cliente, fora do compose |
| `wyd-portal/` | BFF Next.js sobre o backend HTTP/JSON (ADR-0008) | Sim — serviço `portal` |

Binários existentes: `wyd-launcher` (fora do compose), `wyd-gateway` (`crates/wyd-admin/src/main.rs`)
e `wyd-server` (`crates/wyd-server/src/main.rs`), todos pela ADR-0022. Sem binário não há serviço: o
compose não antecipa o que a issue dona não entregou. **O motor (`wyd-core`) ainda não roda dentro
do `wyd-server`** (o tick é a FASE-03-7): o processo hoje entrega conexão, login, seleção e status,
não simulação de mundo.

## 2. O que sobe hoje (`docker compose up`)

| Serviço | Imagem | Papel |
|---|---|---|
| `db` | `postgres:17-alpine` (oficial) | Postgres 17; sobe **vazio** (imagem oficial, sem Dockerfile). Sem one-shot de roles no stack novo: superusuário de dev — **não usar em produção**. Schema aplicado no boot do gateway/canal (ADR-0024) |
| `server` (por canal: `wyd-canal-1`, `wyd-canal-2`) | `docker/Dockerfile.server` (multi-stage, distroless non-root) | Monólito de jogo: CPSock `:8281`, status `:8080` (publicado como `:80`) e `/metrics` `:9898`. **Um container por canal**, mesma imagem `wyd/server:${TAG}` e mesmo Postgres; canal vem de `server_channel` (migration `0029`) e de `WYD_SERVER_CHANNEL`. Lotação e
   manutenção vêm de `channel_capacity` (migration `0040`, seed 1000 = `MAX_USER` do legado
   por canal 1–10), com sobrescrita por processo via `WYD_SERVER_MAX_PLAYERS` /
   `WYD_SERVER_MAINTENANCE` (vazio = vale o banco; ver `docker-compose.yml`). O **papel** do canal
   (`normal`/`verde`, FASE-03-10) vem de `channel_role` (migration `0041`; seed `1 = normal`,
   `2 = verde`) e é sobrescrito por `WYD_SERVER_CHANNEL_ROLE` (vazio = banco); o `/health` publica
   `role`, `pvp` e `shop`. O canal 1 publica o `serv00.htm` com a lista de canais; o 2 usa IP próprio porque o cliente fixa a porta 8281. Roda o motor do core na thread `wyd-tick` a `WYD_TICK_HZ` (20–50, default 20) desde a FASE-03-7 |
| `gateway` | `docker/Dockerfile.gateway` (multi-stage, distroless non-root) | Gateway HTTP/JSON do portal (`:7600`, ADR-0022): `GET /health` faz `SELECT 1` no Postgres, `GET /metrics` publica `wyd_gateway_requests_total{result}` + `wyd_gateway_uptime_seconds`, `POST /admin/v1/{servico}/{metodo}` serve o contrato já existente. `--healthcheck` é o próprio binário (imagem sem shell). Sobe depois do `db` saudável |
| `portal` | `docker/Dockerfile.portal` (multi-stage, non-root, standalone) | BFF Next.js. Não fala SQL (invariante 7): depende do `gateway` saudável. `/api/health` chama `item_catalog/list_items` e devolve 200 com a contagem real do catálogo; 502 só quando o gateway está fora |
| `prometheus`, `grafana`, `node-exporter` | imagens oficiais com tag fixa (mesmo `docker-compose.yml`) | **sobem no `up` normal** desde 2026-11-08 (monitoramento é parte da stack): Prometheus `:9090`, Grafana `:3001`, node-exporter `:9100`, tudo em loopback. A aba `/monitoramento` do portal lê o Prometheus pela rede `wyd` e é o **índice** do monitoramento, com subpáginas `/monitoramento/servidor` (motor: ticks/s, p99, entidades por canal), `/monitoramento/grafana` (dashboards em kiosk), `/monitoramento/prometheus` (alvos/regras de alerta) e `/monitoramento/logs` (Loki). FASE-12-1 ✅ — dashboards/alertas/runbooks em `docker/monitoring/` e `docs/rust/06-observabilidade-runbooks.md` |
| `loki` | `grafana/loki:3.5.5` | **Logs duráveis** (2026-11-09): grava em volume `wyd-loki-data` com retenção de 14 d (`docker/monitoring/loki/loki.yml`). Sem porta no host — só o portal (server-side) e o Grafana consultam `loki:3100`. Resolve o problema do `json-file`: o stdout do Docker morre quando o container é recriado, o Loki não |
| `alloy` | `grafana/alloy:v1.9.2` | **Coletor oficial da Grafana**: descobre os containers pelo socket do Docker (**read-only**, só este serviço) e empurra para o Loki (`docker/monitoring/alloy/config.alloy`), rotulando `service`/`container`. Substitui o sidecar caseiro de logs |

Comandos: `make stack-config` (valida base+prod sem subir), `make stack-up`,
`make stack-down`, `make stack-logs`. Imagens `wyd/<serviço>:${TAG:-dev}`, containers com o
nome do projeto (`wyd-<serviço>-<n>` — sem `container_name` fixo, para permitir stack paralela
com `-p`), rede/volume `wyd_*`.

## 3. Alvo (quando cada binário existir, pela issue dona)

| Serviço futuro | Origem | Issue que libera |
|---|---|---|
| tick do mundo dentro do canal (ECS + cena) | `wyd-core` no `wyd-server` | FASE-03-7/03-8/03-9 |
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
   `wyd_connections_pending`, `wyd_sessions_open`. Motivos de recusa declarados (FASE-06-3
   acrescenta `canal_lotado` e `manutencao`, contados em `NetMetrics::rejected_full` /
   `rejected_maintenance`): `rate`, `ip`, `total`, `timeout`, `canal_lotado`, `manutencao`.
   Zero segredo em log/métrica.
2. **Exposição**: `MetricsServer` serve `GET /metrics` em texto Prometheus 0.0.4
   (endereço decidido pelo binário, default `127.0.0.1:9898`); o fragmento do `wyd-net`
   entra por `Metrics::render(extra)`. Dashboard só consulta o que 06-1 expõe.
3. **Coleta/visualização** (FASE-12-1 ✅): no próprio `docker-compose.yml`
   (Prometheus v3.5.0 + Grafana 12.2.0 + node-exporter v1.9.1 + Loki 3.5.5 + Alloy v1.9.2,
   imagens oficiais com tag fixa), configs em `docker/monitoring/` (`prometheus.yml`,
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

## 5. Banco da stack (schema aplicado no boot, sem passo manual)

O `db` sobe **vazio** e o schema entra no **boot do processo** (ADR-0024): `wyd-gateway` e
`wyd-server` chamam `wyd_db::migrate::run` antes de abrir o pool. Restart é no-op; banco novo é
migrado na primeira subida. Não existe job/serviço de migration — `docker compose up -d` não
deixa container parado.

```bash
docker compose up -d --build --wait   # caminho único: build + up + saúde (não exige make)
make stack-up        # up completo (gateway e canais migram no próprio boot)
make local-rebuild   # atalho + provas de contrato (/login 200, /entregas 307, /health 31)
make db-migrate      # = docker compose run --rm gateway --migrate
make db-reset        # = docker compose run --rm gateway --migrate --reset
```

Regras do runner (`crates/wyd-db/src/migrate.rs`), as mesmas em qualquer caminho:

- **ledger** `schema_migrations(version, checksum, applied_at)`: reexecutar não reaplica;
- **advisory lock de sessão**: gateway e canal subindo juntos não migram ao mesmo tempo;
- **uma transação por arquivo**: falhou, nada daquele arquivo fica;
- **checksum** do bloco `up`: arquivo já aplicado que mudou de conteúdo é erro, não skip;
- **baseline**: volume anterior ao ledger com schema completo é marcado como aplicado
  (o `--reset` recria do zero);
- **integridade**: ao fim exige `item_catalog` (o gateway depende dele).

As imagens copiam `crates/wyd-db/migrations` para `/migrations` (distroless não tem o repo);
`MIGRATIONS_DIR` sobrescreve o caminho. O runner **não** substitui
`tools/content/migrate-cycle.mjs` (`make migrations-apply`), que prova up → down → up num
Postgres efêmero.

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
sobe um Postgres efêmero, aplica só `0001_contas` e roda `cargo test -p wyd-admin --test sistema_seed`
em toda pull request. Não existe mais serviço one-shot `staff-seed` no compose: um job que termina deixa
um container parado em `docker compose ps` (estado normal de job, mas vira ruído e some do `up`
se alguém filtrar por "Up"). Seed no boot do processo que já tem `DATABASE_URL` e argon2id é
idempotente, não precisa de imagem extra e não roda no `initdb` do Postgres — o `entrypoint`
do banco só executa em volume vazio (`/docker-entrypoint-initdb.d`), então nunca serviria um
volume já existente, que é o caso desta stack.

Contagens observadas após aplicar (2026-11-10, série em **36 migrations**): **74 tabelas** —
73 do schema de jogo (`0029` trouxe `server_channel`, `0030` `character_session`, `0031` a
moderação, `0032` `character_scores`, `0033` `character_short_skill`, `0034` `account_wallet`,
`0035` `character_learned_skill` e `0036` `character_special`) + a ledger `schema_migrations`
do runner (ADR-0024),
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
