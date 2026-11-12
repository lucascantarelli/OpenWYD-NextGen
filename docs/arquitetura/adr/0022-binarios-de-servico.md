# ADR-0022 — Binários de serviço: `wyd-gateway` e `tmserver`, imagem, healthcheck e `/metrics`

- **Estado:** aceita
- **Data:** 2026-11-08
- **Decisores:** dono do projeto + agente (per instrução do dono: infra em Docker antes do teste de cliente)
- **Governa:** FASE-07-1 (gateway), FASE-03/04/06 (monólito de jogo), FASE-05-2 (release/imagem), FASE-12-1 (observabilidade)

## Contexto

O workspace só tem um `[[bin]]` (`crates/wyd-launcher`). Por isso não existe nada escutando
`:7600` — o `wyd-portal` sobe `unhealthy`/502 — e não existe nada escutando `:8281`/`:80`, o que
inviabiliza o teste de cliente e deixa o alvo `wyd-server` do Prometheus vazio. As regras do
repositório proíbem arquivo novo fora da árvore da ADR-0002 sem decisão registrada
(`AGENTS.md` §7) e proíbem "serviço sem binário" (`docs/rust/05-stack-docker.md` §1).

Já existe tudo o que o gateway precisa como **biblioteca**, e isso não se reimplementa:

- transporte HTTP/1.1 + JSON em loopback com roteador `servico/metodo` (`crates/wyd-admin/src/transport/`, ADR-0008);
- portas de persistência com adapter `sqlx` (`wyd_db::PgBackend`, ADR-0021);
- contrato declarado pelo portal em `wyd-portal/src/server/admin-client.ts` (paridade cobrada por `crates/wyd-admin/tests/contrato/transporte.rs`);
- telemetria do motor e nomes de métrica fechados (`docs/rust/05-stack-docker.md` §4, ADR-0018).

## Decisão

Adotamos **dois binários de serviço** — `wyd-gateway` (Cold Path, a partir de `wyd-admin`) e
`tmserver` (monólito Hot Path, a partir de `wyd-core`+`wyd-net`+`wyd-protocol`+`wyd-db`) — cada
um com imagem `wyd/<serviço>:${TAG}`, container `wyd-<serviço>`, healthcheck feito pelo próprio
binário (`--healthcheck`) e `/metrics` em texto Prometheus 0.0.4, porque só o binário consegue
provar dependência real (banco, socket) numa imagem distroless sem shell.

### 1. Nomes e papéis

| Binário | Crate fonte | Papel | Porta (container) | Status |
|---|---|---|---|---|
| `wyd-gateway` | `crates/wyd-admin` (`src/main.rs`) | BFF HTTP/JSON do portal; nunca estado vivo | `:7600` | **esta entrega** |
| `tmserver` | monólito | CPSock de jogo + status `serv00.htm` | `:8281` + `:80` | contrato fixado aqui, código na FASE-03/04/06 |

Convenções (valem para os dois): imagem `wyd/<serviço>:${TAG:-dev}`; container
`wyd-<serviço>`; **nome do serviço no compose = nome DNS** usado pelos outros serviços
(`gateway`, `tmserver`) — o `wyd-portal` continua com `WEB_API_ADDR=gateway:7600`; tag nunca é
`latest`; imagem de runtime distroless non-root, sem shell, sem toolchain.

### 2. Configuração por ambiente, nunca por arquivo versionado

Só variável de ambiente (o container não tem arquivo de config):

| Variável | Serviço | Default | Obrigatória |
|---|---|---|---|
| `WYD_GATEWAY_BIND` | gateway | `127.0.0.1:7600` no host, `0.0.0.0:7600` no container | não |
| `DATABASE_URL` | gateway | — | **sim** (DSN nunca versionado) |
| `WEB_API_TOKEN` | gateway | — | **sim** (processo→processo; não é cookie de sessão) |
| `WYD_LOG` | ambos | `info` | não |

`WEB_API_TOKEN` casa com o `WEB_API_TOKEN` que `wyd-portal/src/server/admin-client.ts` envia como
`Authorization: Bearer`. Sem token o gateway **não sobe** (fail-closed): um backend admin sem
autenticação é pior que backend ausente.

### 3. Healthcheck = dependência real, não "processo vivo"

Cada binário expõe `GET /health` e sabe se auto-checar:

- `/health` do gateway executa `SELECT 1` no pool (`PgBackend::ping`) e devolve `200` com
  `{"ok":true,"database":"up","schema":28}`; banco fora = `503`. Não existe caminho em que
  responda 200 sem falar com o banco;
- healthcheck do container: `["CMD", "/usr/local/bin/wyd-gateway", "--healthcheck"]` — o binário
  abre TCP em `127.0.0.1:7600`, faz `GET /health` e sai `0` só em `200`. Isso funciona em
  distroless (sem `curl`/`wget`/shell) e não depende de o container "estar de pé".

O healthcheck do `wyd-portal` continua o do Next (`/api/health`), que agora passa porque o upstream
responde de verdade — `/api/health` chama `item_catalog/listItems`, não um `ping` fictício.

### 4. `/metrics` e nomes de métrica

- Formato Prometheus 0.0.4 em `GET /metrics` (mesmo contrato do exporter da FASE-06-1, sem
  dependência do core no Cold Path).
- **Gateway** publica (declarado aqui, primeira vez):
  `wyd_gateway_requests_total{result="ok|business|transport|unauthorized"}` (counter) e
  `wyd_gateway_uptime_seconds` (gauge).
- **Monólito** publica os nomes já fechados na FASE-06-1 (`wyd_ticks_total`,
  `wyd_tick_duration_microseconds`, `wyd_entities_active`, `wyd_packets_total`,
  `wyd_packets_rejected_total{reason}`, `wyd_handshakes_total`, `wyd_connections_pending`,
  `wyd_sessions_open`, `process_resident_memory_bytes`) — **nenhum nome novo**. A FASE-03-7
  acrescentou só `wyd_tick_budget_microseconds` (§7), porque o orçamento do tick depende de
  `WYD_TICK_HZ` (20–50 Hz) e um limiar fixo mentiria no alerta.
- Scrape: um job por serviço no `docker/monitoring/prometheus/prometheus.yml`; o job `wyd-server` segue
  por descoberta de arquivo (`targets/wyd-server.json`), com um alvo por canal e o label `canal`
  que os painéis agrupam.

### 5. Runtime async só na borda do Cold Path

O binário do gateway usa `tokio` (`rt-multi-thread`) para pool `sqlx` e `SELECT 1`; o roteador
continua síncrono (ADR-0006). O tick do jogo não ganha `await`, `mutex` global nem canal novo:
nada desta ADR toca o Hot Path.

## Consequências

- **Positivas:** o portal deixa de responder 502 porque o upstream existe; a observabilidade passa a
  ter série real (gateway) sem inventar métrica; o teste de cliente fica a um binário de distância
  (`tmserver`), com contrato já decidido; imagem distroless non-root com healthcheck honesto.
- **Negativas:** mais uma imagem para construir (primeiro build compila `sqlx`/`tokio`); o catálogo
  que o gateway serve é uma **foto carregada no boot** (sem push de atualização — reinício aplica
  conteúdo novo); auditoria vai para stdout estruturado em vez de `audit_log` (gap declarado).
- **Invariantes afetados:**
  - serviço só existe com binário e imagem próprios (nada de processo `latest` sem tag);
  - healthcheck tem de tocar dependência real (banco, socket), nunca só o processo;
  - `/metrics` só publica nome declarado (aqui ou na ADR-0018/FASE-06-1);
  - gateway nunca escreve estado de personagem vivo (invariante 7 do `AGENTS.md`).

## Alternativas consideradas

| Alternativa | Por que foi descartada |
|---|---|
| Servir o contrato direto do Next (route handlers) | portal viraria backend; viola BFF puro e o contrato em Rust (ADR-0008) |
| Healthcheck com `wget`/`curl` em imagem alpine | exige shell/pacote na imagem e checa só HTTP, não o banco; distroless + `--healthcheck` prova mais com menos |
| Reimplementar HTTP/JSON no binário | duplicaria `transport/` e quebraria a paridade de contrato cobrada por teste |
| Publicar métricas do gateway com nomes do motor (`wyd_ticks_total`) | mentiria sobre o que o processo faz; nome novo fica declarado e separado por `job` |
| Injetar `sqlx` direto no handler (SQL no `wyd-admin`) | SQL mora no adapter `wyd-db`; o handler consome a foto tipada |

## Rastreabilidade

- Issues consumidoras: `roadmap-rust/issues/FASE-07-1-gateway.md`, `FASE-03/04/06` (monólito), `FASE-05-2`, `FASE-12-1`.
- Docs atualizadas nesta ADR: `docs/rust/05-stack-docker.md`, `docs/rust/06-observabilidade-runbooks.md`, `docs/arquitetura/estrutura-e-padroes.md` §13, `docs/rust/03-progresso.md`.
- Evidência: `docker compose ps` (gateway + portal `healthy`), `curl :7600/health`, `curl :7600/metrics`,
  chamada real `POST /admin/v1/item_catalog/list_items`, query `up{job="wyd-gateway"}` no Prometheus.

## 7. Canais: uma imagem, um container por canal (2026-11-08)

Canal no WYD é **processo**, não campo de configuração de um processo grande: dois canais
significam dois mundos independentes sobre o **mesmo banco** (cidadania, guildas e torre são
linhas de banco compartilhadas; `docs/wyd/canais-e-cidadania.md` §5). Decidimos:

- **imagem única** `wyd/server:${TAG}` para todos os canais (build único, `-p wyd-server`);
- **um container por canal**: `wyd-canal-1` (normal) e `wyd-canal-2` (verde/novato), diferenciados
  por `WYD_SERVER_CHANNEL` e pelo IP publicado — o cliente disca `8281` **fixo** no host da célula
  do `serverlist.bin`, então canal novo exige IP novo, nunca porta nova;
- **serviço de compose** `server` = canal 1 (base); o canal 2 entra no override de dev com IP
  próprio (`WYD_CANAL2_IP`) e `env_file` compartilhado; produção publica por IP no override de prod;
- **canal é dado, não build**: `server_channel` (migration `0029`) guarda grupo, canal, host, URL de
  status e `enabled`. Registrar canal novo = uma linha + um container; nenhum binário recompilado;
- **status `:80` é do canal 1** e publica o CSV de **todos** os canais habilitados (é o que popula a
  coluna Canal do cliente); os demais canais expõem o status só na rede interna, para `/health`;
- **métricas novas declaradas aqui** (o resto vem do contrato da FASE-06-1):
  `wyd_server_uptime_seconds`, `wyd_server_channels_published`, `wyd_status_requests_total`,
  `wyd_status_health_failures_total`.`wyd_ticks_total`/`wyd_tick_duration_microseconds`/`wyd_entities_active` **não** eram
  publicados enquanto o loop de `wyd-core` não rodava no processo — publicá-los zerados seria
  mentira. Desde a **FASE-03-7** (2026-11-09) o `Engine` roda na thread `wyd-tick`
  (`crates/wyd-server/src/world.rs`) e os três passam a vir do registro real do motor,
  anexado ao `/metrics` antes do primeiro scrape. O mesmo registro publica
  `wyd_tick_budget_microseconds` (`1e6 / WYD_TICK_HZ`), que o painel de orçamento e as regras
  `WydTickP99Alto`/`WydTickAcimaDoOrcamento` comparam com o P99 de cada canal.

Consequências: custo por canal é um container (barato), a capacidade de “canal verde” é
política de operação (qual IP recebe jogador novo), e a lista de canais passa a ser auditável no
banco em vez de escondida no `.bin`.

## Revisão

- Revisar quando: o `tmserver` existir (mesmo rito de healthcheck/`/metrics`), o catálogo ganhar
  push de conteúdo, ou o runtime async deixar de ser diferido (ADR-0006).
- Sinais de que a decisão envelheceu: healthcheck verde com banco fora; métrica do gateway
  confundida com métrica de jogo; build da imagem passando de ~10 min sem cache.
