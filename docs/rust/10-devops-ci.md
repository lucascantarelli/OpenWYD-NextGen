# DevOps e CI do WYD (Rust) — pipeline, imagens, stack e deploy local

> **Norma desta área.** Fonte executável: `.github/workflows/rust.yml`, `Makefile`,
> `docker/Dockerfile.*`, `docker-compose*.yml` e `docker/monitoring/`. Este documento é a
> **auditoria de 2026-11-12** e o contrato do que o pipeline precisa provar; o backlog derivado
> está em `roadmap-rust/index.md` §DevOps e CI (`DEVOPS-01`…`DEVOPS-10`).
> `AGENTS.md` §3 continua sendo o comando do gate; aqui fica **o que o gate cobre, o que ele não
> cobre e por quê**. Doc desatualizada é defeito (`docs/rust/02-execucao.md` §5): mudou o
> workflow, muda esta página na mesma entrega.

## 0. Método e limite desta análise

**Lido integralmente:** `.github/workflows/rust.yml`, `Makefile`, `.dockerignore`, `deny.toml`,
`rust-toolchain.toml`, os quatro `docker/Dockerfile.*`, os cinco `docker-compose*.yml`,
`docker/monitoring/**` (Prometheus, alertas, alvos, Loki, Alloy, provisioning) ,
`tools/dev/{bench-gate,monitoring-smoke,check-docs-consistency}.mjs`,
`tools/content/check-issue-status.mjs`, `tools/dev/check-skills.mjs`, `pre-release-rust.md`
(FASE 5/15), `docs/rust/02-execucao.md`, `docs/rust/05-stack-docker.md`.

**Medido no código, não no host:** `SCHEMA_VERSION = 50` (`crates/wyd-db/src/lib.rs`) contra 50
arquivos de migration; ausência de `alerting:`/Alertmanager em todo o repositório; ausência de
`.github/dependabot.yml`; `tests/sistema/*` pulando sem `WYD_TEST_DSN`.

**Não executado nesta rodada:** nenhum `docker build`, nenhum `docker compose up`, nenhum
`cargo bench`, nenhum `act`/runner. Portanto **tempo de pipeline, tamanho de imagem e
flakiness não foram remedidos aqui** — os números citados são os registrados em
`docs/rust/02-execucao.md` §6 e nas evidências de QA-02/QA-03. Onde a conclusão depende de
medição futura, está marcado **hipótese**.

## 1. Mapa do pipeline (`rust.yml`)

Seis jobs, todos em `ubuntu-24.04`, `permissions: contents: read`, `concurrency` por `ref`:

| Job | O que realmente prova | Custo | Lacuna |
|---|---|---|---|
| `rust` | `check-docs-consistency` + `fmt --all --check` + `clippy --all-targets -D warnings` + `build` + `test --all-targets` | 1 build completo do workspace | não roda `migrations-check`, `issue-status-check`, `skills-check`, `tools-test` |
| `seed-boot` | Postgres 17 efêmero → `--migrate` (série inteira) → `sistema_seed` | 1 build do `wyd-admin` | único ponto com banco; nenhum outro teste de sistema |
| `actionlint` | os próprios workflows (imagem `rhysd/actionlint:1.7.7`) | baixo | não valida compose/Dockerfile |
| `bench` | `bench --no-run` + `bench-gate.mjs` + bench `limits` do `wyd-net` | 2 compilações do mesmo bench | limiares calibrados em host Windows (2,2×–5,3× de folga) |
| `release` | `build --release` do workspace + imagem do launcher + `docker build --check` do portal | 1 build release completo | `Dockerfile.gateway`/`Dockerfile.server` nunca são construídos; sem `WYD_VERSION`/`WYD_COMMIT` |
| `supply-chain` | `cargo audit` + `cargo deny check` | baixo | não roda por tempo, só por commit |

**Nenhum cache em cinco jobs que compilam.** Não há `Swatinem/rust-cache`, `actions/cache` de
`~/.cargo`/`target` nem `cache-to=type=gha` nos `docker build`. Cada execução recompila as
dependências do zero — o custo real de runner é ~5 compilações por push (**hipótese** de
economia de 40–60% com cache, a medir em `DEVOPS-05`). Paralelismo entre jobs existe; **paralelismo
dentro do job `rust` não** (fmt/clippy/build/test em série, quando clippy e test compilam quase o
mesmo grafo).

Nenhum job declara `timeout-minutes`: um teste que pendura em socket/DB segura o runner pelo
default de 6 h. Nenhum `path`-filter em `pull_request` (só em `push`), então PR de documentação
paga o pipeline inteiro. `concurrency` com `cancel-in-progress` cancela também em `main`, o que
derruba a garantia de "todo commit de `main` verificado".

## 2. Paridade gate local × CI — o defeito de processo mais caro

`make check` roda **nove** verificações; o CI roda **uma** delas (`docs-check`):

| Verificação | `make check` | CI | Consequência |
|---|---|---|---|
| `cargo fmt/clippy/build/test` | ✅ | ✅ | ok |
| `migrations-check` (ordem/FK/down) | ✅ | ❌ | migration inválida entra verde |
| `issue-status-check` (índice × painel × arquivo) | ✅ | ❌ | roadmap deriva sem gate |
| `skills-check` (frontmatter, 500 linhas, caminhos) | ✅ | ❌ | skill quebrada não é detectada |
| `tools-test` (`node --test apps/tools`) | ✅ | ❌ | ferramenta de dev regride |
| `docs-check` (links/crates) | ✅ | ✅ | ok |
| `portal-test` (`tsc --noEmit` + `vitest`) | ❌ | ❌ | portal sem gate nenhum |
| `bench-gate` | ❌ | ✅ | só no CI |
| `monitoring-rules` (promtool) | ❌ | ❌ | regra de alerta quebrada passa |

O resultado é **verde falso nos dois sentidos**: CI verde com `make check` vermelho (docs,
issues, migrations) e `make check` verde com CI vermelho (bench-gate). O `Makefile` chama
`ci` de "espelho local do `.github/workflows/rust.yml`"; ele não é (não inclui `bench-gate`,
`monitoring-rules`, `portal-test` nem build de imagem). Rastreio: `DEVOPS-04`.

**`make local-rebuild` está quebrado hoje.** A rotina oficial de fechamento de issue
(`docs/rust/02-execucao.md` §5, `03-progresso.md` "Rotina de fechamento") exige
`"schema":49` no `/health`, mas `SCHEMA_VERSION = 50` e a série tem 50 migrations. Quem seguir a
rotina ao pé da letra vê falha em `local-rebuild` sem nenhum defeito no código. O gate precisa
**derivar** o número (ler `SCHEMA_VERSION` ou o `/health` que ele já devolve), não fixá-lo.

## 3. Achados por severidade

P0 = risco de segredo/dado ou alerta que não chega a ninguém. P1 = verde falso, custo ou
cobertura fantasma. P2 = higiene/manutenção.

| # | Sev | Achado | Evidência | Issue |
|---|---|---|---|---|
| A1 | **P0** | `.env` (segredos reais do host) **não está no `.dockerignore`**: todo `docker build` envia o arquivo ao daemon/BuildKit e o mantém no cache de contexto. Nenhum `COPY` atual o consome, mas um `COPY . .` futuro o cimenta na imagem. `wyd-portal/.env*` também não é ignorado, e `COPY wyd-portal/ ./` no estágio de build do portal o leria. | `.dockerignore`, `docker/Dockerfile.portal` | `DEVOPS-01` |
| A2 | **P0** | **Resolvido em 2026-11-12.** Alertas não notificavam ninguém: sem `alerting:` e sem Alertmanager, as regras só mudavam de estado na UI. Agora há `Prometheus → Alertmanager → slack | email` (destino real por `ALERTMANAGER_DESTINO`, segredo só do ambiente), com entrega provada pelo payload de Slack e regra de vigia para o próprio Alertmanager. | `docker/monitoring/alertmanager/`, `docker-compose.yml` | `DEVOPS-02` ✅ |
| A3 | **P0** | **Portal de produção em HTTP puro**, publicado em todas as interfaces (`3000:3000`), sem proxy/TLS no stack. Sessão de staff e cookie trafegam em claro. | `docker-compose.prod.yml` | `DEVOPS-03` |
| A4 | **P0** | Grafana herda `GF_AUTH_ANONYMOUS_ENABLED: true` (Viewer) e senha default `troque-aqui-dev` no **arquivo base**, portanto também no perfil de produção; `monitoring-smoke.mjs` embute a mesma senha. | `docker-compose.yml`, `tools/dev/monitoring-smoke.mjs` | `DEVOPS-03` |
| A5 | **P0** | `alloy` monta o socket do Docker (`:ro`). O socket em modo leitura **não** impede `GET /containers/json` + `inspect`, que devolve o `env` de todos os containers — ou seja, `POSTGRES_PASSWORD`, `SESSION_SECRET` e `WEB_API_TOKEN` são legíveis por esse container. Aceitável como trade-off, mas precisa estar escrito e com escopo explícito. | `docker-compose.yml`, `docker/monitoring/alloy/config.alloy` | `DEVOPS-03` |
| B1 | **P1** | Gate local ⊃ CI (§2). | `Makefile`, `rust.yml` | `DEVOPS-04` |
| B2 | **P1** | **Portal invisível no CI**: `tsc --noEmit`, `vitest` e `next build` não rodam; `Dockerfile.portal` só passa por `docker build --check` (lint de Dockerfile, não build). | `rust.yml` | `DEVOPS-04`, `DEVOPS-06` |
| B3 | **P1** | **Testes de sistema pulam em silêncio**: `tests/sistema/*` do `wyd-server` (login, selchar, entrada no mundo, refino/piedade, lotação) chamam `common::` que imprime `SKIP` e retorna sucesso quando não há `WYD_TEST_DSN`. No CI não há DSN para eles → verde com zero cobertura de banco no que é mais frágil. | `crates/wyd-server/tests/common/mod.rs` | `DEVOPS-07` |
| B4 | **P1** | Sem cache em cinco jobs (§1). | `rust.yml` | `DEVOPS-05` |
| B5 | **P1** | `Dockerfile.gateway` e `Dockerfile.server` **nunca** são construídos no CI — podem apodrecer sem nenhum sinal. | `rust.yml` (`release`) | `DEVOPS-06` |
| B6 | **P1** | `docker build` do CI não passa `--build-arg WYD_VERSION/WYD_COMMIT`: as labels OCI saem `0.1.0`/`unknown`, ou seja, a "proveniência" da imagem de CI é mentira. | `rust.yml` | `DEVOPS-06` |
| B7 | **P1** | **Supply-chain só por commit**: `cargo audit` não roda por tempo; não há `.github/dependabot.yml` nem Renovate. Advisory publicado depois do último push nunca é visto. | `rust.yml`, `.github/` | `DEVOPS-08` |
| B8 | **P1** | **Sem cobertura medida**: `@vitest/coverage-v8` instalado e sem script; nenhum `tarpaulin`/`llvm-cov` no Rust. | `wyd-portal/package.json`, `rust.yml` | `DEVOPS-08` |
| B9 | **P1** | `make local-rebuild` fixa `schema:49` contra `SCHEMA_VERSION = 50` (§2). | `Makefile` | `DEVOPS-04` |
| B10 | **P1** | `docker build -f docker/Dockerfile.launcher` estava **quebrado**: `crates/wyd-launcher/src/identidade.rs` usa `include_str!("../../../docs/arquitetura/identidade.tokens")` e nenhum `COPY` trazia o arquivo (`couldn't read`). A única imagem que o CI constrói falhava, e nenhum gate local constrói imagem. | `docker/Dockerfile.launcher`, `crates/wyd-launcher/src/identidade.rs` | `DEVOPS-01` ✅ / `DEVOPS-06` |
| C1 | P2 | `.PHONY` não cobre `clean-target`, `clean-tmp`, `tools-test`, `tools-inventario`, `tools-zip-check`, `client-zip`: arquivo/diretório homônimo desliga o target em silêncio. | `Makefile` | `DEVOPS-04` |
| C2 | P2 | `make help` imprime 4 linhas fixas, ignorando os `##` que documentam **todos** os targets. A documentação existe; a UX não a usa. | `Makefile` | `DEVOPS-04` |
| C3 | P2 | `make fmt` usa `cargo fmt --check`; o CI usa `--all --check`. | `Makefile` | `DEVOPS-04` |
| C4 | P2 | `bench` compila o mesmo bench duas vezes (`--no-run` + o `spawnSync` do `bench-gate.mjs`). | `rust.yml`, `tools/dev/bench-gate.mjs` | `DEVOPS-05` |
| C5 | P2 | Limiares do `bench-gate` calibrados no host do dono com 2,2×–5,3× de folga; o p99 é 4× ruidoso na mesma máquina. Risco de flake em runner compartilhado. | `tools/dev/bench-gate.mjs` | `DEVOPS-05` |
| C6 | P2 | `TAG:-dev` em `docker-compose.prod.yml`: produção pode subir a imagem `dev` sem erro. O próprio §7 de `05-stack-docker.md` proíbe tag sem TAG. | `docker-compose.prod.yml` | `DEVOPS-03` |
| C7 | P2 | Imagens base por tag flutuante (`rust:1-bookworm`, `gcr.io/distroless/cc-debian12:nonroot`) e ações por major tag (`actions/checkout@v5`). Reproducibilidade e supply-chain de action ficam "quase". | `docker/Dockerfile.*`, `rust.yml` | `DEVOPS-01` |
| C8 | P2 | `rust-toolchain.toml` diz `channel = "stable"` — não é pin: dois meses depois, `clippy -D warnings` pode quebrar sem uma linha alterada. A skill `wyd-rust-toolchain` afirma "stable é pinado". | `rust-toolchain.toml` | `DEVOPS-05` |
| C9 | P2 | Sem `logging` limitado em gateway, portal, prometheus, grafana, loki, alloy, node-exporter e mkdocs (só os dois canais têm `max-size`). Disco do host pode encher. | `docker-compose.yml` | `DEVOPS-03` |
| C10 | P2 | Sem `healthcheck` em prometheus, grafana, loki, alloy e node-exporter: `docker compose up --wait` não gateia a cadeia de observabilidade. | `docker-compose.yml` | `DEVOPS-03` |
| C11 | P2 | Sem `security_opt: no-new-privileges`, sem `cap_drop`, sem `read_only` em nenhum serviço. | `docker-compose.yml` | `DEVOPS-03` |
| C12 | P2 | Sem limites de recurso em prod para gateway, canais, prometheus, grafana, loki, alloy (só `db`, `portal`, `mkdocs`). | `docker-compose.prod.yml` | `DEVOPS-03` |
| C13 | P2 | `docker-compose.reaper.yml` é overlay de um serviço, sem cabeçalho e sem entrada na documentação. | `docker-compose.reaper.yml` | `DEVOPS-03` |
| C14 | P2 | `--web.enable-lifecycle` no arquivo base (portanto em prod): reload por HTTP sem auth, contido só pela ausência de porta publicada. | `docker-compose.yml` | `DEVOPS-03` |
| C15 | P2 | `.env` inteiro montado no gateway (dev e `prod-env`), embora ele só precise de `PORTAL_ADMIN_*`. | `docker-compose.override.yml` | `DEVOPS-03` |
| C16 | P2 | **Resolvido em 2026-11-12.** Regras de host (disco/CPU/RAM/`up`), de portal (sonda `blackbox`) e de vigia do Alertmanager foram acrescentadas; total de 17 regras com teste de disparo e resolução — 18 desde o alerta de crescimento de RSS. | `docker/monitoring/prometheus/alerts.yml` | `DEVOPS-02` ✅ |
| C17 | P2 | Nenhum versionamento de release: sem `[workspace.package] version`, sem tag, sem release notes; o único sinal de versão é o build-arg default `0.1.0`. | `Cargo.toml`, `.github/` | `DEVOPS-10` |
| C18 | P2 | Comentário do `[profile.release]` cita `323.072 → 223.232 bytes`; `02-execucao.md` §6 registra `296.448`. Duas fontes para o mesmo número. | `Cargo.toml`, `docs/rust/02-execucao.md` | `DEVOPS-10` |

## 4. Makefile — contrato

O `Makefile` é a interface do humano e do agente; ele **não** duplica o CI nem o substitui.
Regra: um target, um efeito observável, e nenhum target "verde por ausência".

- **Gate** — `make check` é o gate de issue (`AGENTS.md` §3). O que entra nele precisa ser
  determinístico e sem Docker. `portal-test` deve entrar (é `npx`, não Docker) e `bench-gate`
  **não** deve (é medição, não correção) — mas `make ci` precisa cobrir os dois lados.
- **Espelho** — `make ci` existe para reproduzir o workflow. Se ele divergir, o nome mente:
  ou ele passa a incluir `bench-gate` + `monitoring-rules` + `portal-test` + `stack-config`, ou
  deixa de se chamar espelho.
- **Fechamento** — `make local-rebuild` é a rotina do dono antes de fechar issue. Ele não pode
  fixar número de schema (§2); deriva de `SCHEMA_VERSION`.
- **Documentação** — `make help` deve ler os `##` do próprio arquivo (`grep -E '^[a-z].*:.*## '`),
  não uma lista fixa.
- **Higiene** — todo target declarado em `.PHONY`; target sem `.PHONY` é bug latente (C1).

## 5. Imagens e contexto de build — contrato

O padrão atual já é **bom**: multi-stage, runtime **distroless** sem shell, `USER nonroot`, 
`HEALTHCHECK` executando o próprio binário (`--healthcheck`, que checa o banco, não o processo) e
labels OCI de versão/commit. Não troque esse desenho por conveniência.

O que falta para o padrão ser contrato:

1. **Contexto mínimo.** `.dockerignore` é a fronteira de segredo do build. `.env`, `.env.*`,
   `wyd-portal/.env*`, `dist/`, `site/`, `*.zip`, `tmp/`, `roadmap-rust/`, `docs/`, `.agents/`
   não têm por que chegar ao daemon.
2. **Nada de `COPY . .`.** Os quatro Dockerfiles copiam caminhos explícitos; manter assim é
   parte da defesa contra A1.
3. **Proveniência completa.** Toda imagem construída no CI recebe `WYD_VERSION` (tag ou
   `0.0.0-<sha>`) e `WYD_COMMIT`; o `Dockerfile.portal` ganha `ARG`/labels equivalentes
   (hoje só tem título/descrição/origem).
4. **Reprodutibilidade.** Base por versão exata (`rust:1.9x-bookworm`) e, para release, por
   digest (`@sha256:`). `rust:1-bookworm` move sozinho.
5. **Cache entre execuções.** `cache-to=type=gha,mode=max` (ou cache de `target/` no
   `rust-cache`); sem isso os `--mount=type=cache` só ajudam o host, nunca o runner.
6. **Scan.** Trivy (ou equivalente) na imagem construída, falhando em `CRITICAL` com exceção
   explícita e datada; SBOM gerado no release.
7. **Smoke do artefato.** Build não é prova: o job que constrói uma imagem roda um comando que
   só passa com a imagem funcionando (`--healthcheck`, `--help`, `check`).

### Executado em 2026-11-12 (`DEVOPS-01`) — o que ficou medido

| Imagem | Base runtime | Tamanho | Usuário | Prova |
|---|---|---:|---|---|
| `wyd/launcher:devops01` | distroless `cc-debian12:nonroot@sha256:9dac0a79…` | 38.653.196 B | `nonroot:nonroot` | `--help` (exit 2 por contrato), 1.792 entradas, único binário de app |
| `wyd/gateway:devops01` | idem | 51.403.706 B | `nonroot:nonroot` | labels de release + `HEALTHCHECK --healthcheck` |
| `wyd/server:devops01` | idem | 51.450.041 B | `nonroot:nonroot` | idem |
| `wyd/portal:devops01` | `node:22-alpine` | 320.204.079 B | `node` | labels `version`/`revision` (antes ausentes) |

Contexto de build (imagem de prova com `COPY . /ctx`): **638 arquivos**, sem `.env`, `.env.local`
nem `wyd-portal/.env*`; sem `docs/` amplo (só o token de identidade, que é entrada de build);
`crates/`, `apps/launcher-ui/` e `wyd-portal/package-lock.json` presentes. `docker export` da
imagem do launcher: **1.792 entradas**, zero `.env/.pcap/.pem/.key`, zero shell/compilador.
`make stack-config` exit 0 nos três modos.

**O limite do pin da base:** `rust-toolchain.toml` declara `channel = "stable"` e o builder
baixou **1.100.0** dentro da base 1.99 — fixar a imagem **não** fixa o compilador. Enquanto
`DEVOPS-05` não pinar a versão no `rust-toolchain.toml`, o build de imagem continua dependendo
de rede e de qual `stable` existir no dia. O tamanho do launcher subiu de 37.847.971 B (baseline
QA-03) para 38.653.196 B (+2,1%), coerente com a troca de compilador, não com a mudança de
contexto.

Kubernetes/Swarm **não** entram agora: a escala do projeto é "um processo por canal, N
containers, um banco", e o compose já expressa isso. Orquestrador aqui seria superfície nova sem
problema novo — decisão registrada, não dívida.

## 6. Compose e superfície de produção — contrato

Perfil de produção = `docker-compose.yml` + `docker-compose.prod.yml` (**sem** o override de dev).
Regras que precisam valer no arquivo, não na cabeça de quem opera:

- **TLS obrigatório**: nenhum serviço administrativo (portal) fala HTTP puro com o mundo. Proxy
  reverso no stack ou bind só em loopback + túnel — escolha explícita em `DEVOPS-03`.
- **`TAG` obrigatório**: `image: wyd/portal:${TAG:?defina TAG}` em prod, como já se faz com
  `POSTGRES_PASSWORD`.
- **Observabilidade fora do perfil público**: anônimo do Grafana é comodidade de dev; em prod
  deve ser desligado ou o Grafana não sobe.
- **Log com teto em todo serviço** (`max-size`/`max-file`), inclusive portal e gateway.
- **Healthcheck em todo serviço que participa de `--wait`**.
- **Limite de recurso em todo serviço de prod**; `security_opt: [no-new-privileges:true]` e
  `cap_drop: [ALL]` onde o binário não precisa de capability.
- **Socket do Docker**: único consumidor é o `alloy`, `:ro`, e a consequência (leitura de `env`
  de containers = segredos) fica escrita no arquivo e no runbook.
- **Overlay de teste** (`reaper`) ganha cabeçalho dizendo o que faz, por que existe e que não é
  produção.

## 7. Alertas e notificação — contrato

A cadeia existe e está provada desde 2026-11-12 (`DEVOPS-02`):

```text
motor/gateway/host → Prometheus (scrape) → regras (alerts.yml) → Alertmanager → destino
```

**Implementado:** `alertmanager` (`prom/alertmanager:v0.28.1`) com rota, `group_by: [alertname,
canal]`, `repeat_interval: 4h` e três `inhibit_rules` (servidor fora → alertas do canal; disco
cheio → CPU/RAM; gateway fora → portal fora). O `prometheus.yml` ganhou o bloco `alerting:`
(v2, `alertmanager:9093`). O destino padrão é **real**: `slack` (default) ou `email`, escolhidos por
`ALERTMANAGER_DESTINO`. O segredo não fica em arquivo versionado — o container renderiza a
config no boot e escreve `/tmp/wyd-secrets/slack_api_url`/`smtp_password` a partir do ambiente
(`SLACK_API_URL`, `SMTP_PASSWORD`). O serviço **`alert-sink`** continua existindo com dois
papéis: receptor do caminho de prova (`prova=local`) e **bootstrap** da integração Slack
enquanto não houver webhook — a notificação é registrada em
`/var/log/wyd-alerts/alertas.jsonl` (volume `alerts-out`) e o build não quebra. Definir
`SLACK_API_URL` não muda rota nem inibição: o mesmo POST passa a ir para o Slack.

**Cobertura acrescentada:** host (`node-exporter`: disco < 10 %, CPU > 90 %/15 min, RAM > 95 %/15
min, `up{job="node"}==0`), portal (`probe_success` via `blackbox-exporter` em
`/api/health`, porque o BFF não publica `/metrics`), vigia do vigia (`up{job="alertmanager"}==0`)
e `WydWatchdog` (heartbeat `vector(1)` sempre em firing, que prova a cadeia inteira). As 9 regras
originais continuam como estavam; o total passou a **17**.

**Medido em 2026-11-12:** `promtool check config`/`check rules` **SUCCESS, 17 rules**; `promtool
test rules` **SUCCESS** (8 cenários, incluindo disparo *e* resolução do portal); `amtool
check-config` **SUCCESS** (3 inhibit rules, **3 receivers**); os alvos `alertmanager` e
`wyd-portal` ficaram `up` no Prometheus; `WydWatchdog` apareceu no Alertmanager em
`state=active`; `amtool config routes test` mostra alerta crítico → **slack** e `prova=local` →
`local-sink`; e `make monitoring-alertas` entregou a notificação pela **integração Slack** com
payload de Slack (`channel=#wyd-alertas`, `attachments`, título `[FIRING] WYD <alerta>`), em
bootstrap para o receptor local até existir `SLACK_API_URL`. Com o webhook do workspace no
`.env`, o container sai do bootstrap e a entrega real foi medida pelo contador da integração:
`alertmanager_notifications_total{integration="slack"}` +1, `failed_total=0`, sem `level=error`.

**Stack completa e ciclo de resolução (2026-11-12):** `make stack-up` subiu os **14 containers**
(`db`, `gateway`, `server-canal-1/2`, `portal`, `prometheus`, `grafana`, `alertmanager`,
`alert-sink`, `blackbox-exporter`, `node-exporter`, `loki`, `alloy`, `mkdocs`), todos `healthy`;
`make monitoring-check` **exit 0** com os 7 alvos `up` (inclusive gateway, os dois canais e o
portal via `blackbox`), regras carregadas (17 naquele momento, **18** depois do
`WydRssCrescendo`), `datasource`/dashboard do Grafana e rota padrão →
`slack`. `make monitoring-alertas` provou **disparo e resolução** (`failed_total=0`) e, durante o
boot, quatro alertas **de regra** (portal, gateway e os dois canais fora do ar) dispararam e
resolveram sozinhos — cada transição virou notificação. A rotação da webhook foi validada sem
expor segredo: o hash de `/tmp/wyd-secrets/slack_api_url` no container passou a bater com o do
`.env`, o token antigo saiu do arquivo e não aparece em nenhum arquivo versionado, e a prova
repetida entregou disparo + `RESOLVED` (`notifications_total` +2, `failed_total=0`). A subida
também revelou um alerta **verdadeiro**: `WydSemAmostraDeMemoria` disparava porque o runtime nunca
chamava `mem::sample_rss_bytes()`; a correção (thread de 1 Hz no boot, fora do tick) está no
changelog — depois dela `wyd_memory_samples_total` mede 132/136 amostras e RSS de 36,9/36,0 MB,
e o alerta resolve. `make check` na mesma stack: **exit 0**, 811 testes, 0 falhas.

**Monitoramento do próprio CI**: o GitHub já guarda duração/falha por run; o que não existe é
alerta. Não vale montar Prometheus para isso — vale `timeout-minutes`, retenção de artefato e,
quando houver runner local (`DEVOPS-09`), notificação do resultado no mesmo destino do
Alertmanager.

## 8. Versionamento e release — contrato

- **Fonte única de versão**: `[workspace.package] version` no `Cargo.toml` raiz, herdada pelas
  crates, espelhada no `package.json` do portal. Enquanto não existir, "0.1.0" é texto repetido
  em três lugares.
- **Tag `vX.Y.Z`** em `main` é o que dispara release; a imagem recebe `WYD_VERSION=X.Y.Z` e
  `WYD_COMMIT=<sha completo>`.
- **Release notes** derivadas do `roadmap-rust/changelog.md` + commits da tag — o changelog do
  roadmap é histórico de execução, não nota de release.
- **Imutabilidade**: imagem publicada não é reconstruída com a mesma tag; rollback usa
  tag/digest anterior (`DEVOPS-10` + procedimento em `02-execucao.md` §6).

## 9. Deploy local automático (runner auto-hospedado) — resposta direta

**Sim, é possível e é o caminho certo para este projeto**, com uma condição: o runner **não**
pode executar código de pull request. Ele roda na máquina que tem o Docker, o `.env` e o
`client/` — um PR malicioso ali é execução arbitrária no host.

Desenho recomendado (`DEVOPS-09`):

1. **Runner de repositório**, escopo repo (não organização), labels `self-hosted,wyd-local`,
   conta Windows dedicada e sem privilégio de administrador.
2. **Job `deploy-local`** com `needs: [rust, seed-boot, portal, actionlint]`,
   `if: github.event_name == 'push' && github.ref == 'refs/heads/develop'`,
   `runs-on: [self-hosted, wyd-local]`, `environment: local`.
3. **Passos**: `docker compose build` → `docker compose up -d --wait --wait-timeout 180` →
   provas (`/login` 200, `/entregas` 307, `gateway/health` 200 com `schema` igual a
   `SCHEMA_VERSION`) → log do deploy como artefato. Falhou prova, o job falha e o deploy não é
   dado como feito.
4. **Trava de segurança**: nenhum workflow `pull_request` nesse runner (ou, se houver,
   `environment` com aprovação manual). Segredos continuam no `.env` do host, **nunca** em
   `secrets` do GitHub, e o runner não recebe permissão de escrita no repositório.
5. **Trava operacional**: o job não roda `down -v`, não faz `DROP`, não toca em `client/` e não
   reinicia a stack durante teste humano (`AGENTS.md` §0/§7). Se o teste do dono estiver em
   andamento, o deploy espera — `workflow_dispatch` manual é o freio.
6. **Alternativa sem runner** (menos privilégio, menos exatidão): `git pull` + `docker compose
   build` por tarefa agendada na máquina, com o mesmo bloco de provas. Não sabe se o CI passou;
   por isso é fallback, não desenho.

## 10. Backlog derivado

Ordem de execução recomendada (severidade × esforço):

| Ordem | Issue | Sev | Esforço | Por quê agora |
|---:|---|---|---|---|
| 1 | `DEVOPS-01` segredos no contexto de build + hardening de imagem | P0 | S | **✅ fechada em 2026-11-12** (§5): fecha A1/C7 e o B10 descoberto no caminho |
| 2 | `DEVOPS-02` Alertmanager e notificação real | P0 | M | **✅ fechada em 2026-11-12** (§7): fecha A2/C16, com entrega medida |
| 3 | `DEVOPS-03` superfície de produção do compose | P0 | M | fecha A3/A4/A5/C6–C15 |
| 4 | `DEVOPS-04` paridade gate local × CI | P1 | S | fecha B1/B9/C1–C3; impede verde falso |
| 5 | `DEVOPS-05` cache, timeouts e custo do CI | P1 | S/M | fecha B4/C4/C5/C8 |
| 6 | `DEVOPS-06` build + scan + proveniência das quatro imagens | P1 | M | fecha B2/B5/B6 |
| 7 | `DEVOPS-07` banco e portal no CI (fim do skip silencioso) | P1 | M | fecha B3 |
| 8 | `DEVOPS-08` supply-chain contínua e cobertura | P1 | M | fecha B7/B8 |
| 9 | `DEVOPS-09` runner local + deploy pós-CI | P1 | M | resposta ao pedido de CD local |
| 10 | `DEVOPS-10` versionamento e release | P2 | S | fecha C17/C18 |

## 11. Limites

Nada aqui substitui medição: tempo de pipeline com/sem cache, tamanho das quatro imagens e taxa
de flake do `bench-gate` em runner compartilhado são **hipóteses** até `DEVOPS-05`/`DEVOPS-06`
publicarem números. `docker build`/`compose up` não foram executados nesta análise.
